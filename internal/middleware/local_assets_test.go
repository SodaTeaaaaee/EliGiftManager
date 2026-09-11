package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestMiddleware(t *testing.T) (http.Handler, string) {
	t.Helper()

	assetsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(assetsDir, "products"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "avatar.png"), []byte("png-bytes"), 0o644); err != nil {
		t.Fatalf("write avatar: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "products", "sku-1.jpg"), []byte("jpg-bytes"), 0o644); err != nil {
		t.Fatalf("write sku image: %v", err)
	}

	secret := filepath.Join(filepath.Dir(assetsDir), "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler := localAssetsMiddleware("/local-images/", func() (string, error) {
		return assetsDir, nil
	})(next)

	return handler, assetsDir
}

func TestLocalAssetsMiddleware_PassesNonPrefixRequestsThrough(t *testing.T) {
	t.Parallel()

	handler, _ := newTestMiddleware(t)
	for _, path := range []string{"/", "/index.html", "/assets/app.js", "/local-images"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusTeapot {
			t.Errorf("GET %s: code = %d, want %d (must reach next handler)", path, rec.Code, http.StatusTeapot)
		}
	}
}

func TestLocalAssetsMiddleware_ServesFilesUnderAssetsDir(t *testing.T) {
	t.Parallel()

	handler, _ := newTestMiddleware(t)

	for path, wantBody := range map[string]string{
		"/local-images/avatar.png":          "png-bytes",
		"/local-images/products/sku-1.jpg":  "jpg-bytes",
		"/local-images/products//sku-1.jpg": "jpg-bytes",
		"/local-images/./avatar.png":        "png-bytes",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: code = %d, want 200", path, rec.Code)
			continue
		}
		if got := rec.Body.String(); got != wantBody {
			t.Errorf("GET %s: body = %q, want %q", path, got, wantBody)
		}
	}

	// In-scope dot-dot segments (still inside the assets dir after cleaning)
	// are rejected by http.ServeFile's own invalid-URL-path guard with 400.
	// Hostile dot-dot escapes are covered by the traversal test below.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/local-images/products/../avatar.png", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET /local-images/products/../avatar.png: code = %d, want 400", rec.Code)
	}
}

func TestLocalAssetsMiddleware_BlocksDirectoryTraversal(t *testing.T) {
	t.Parallel()

	handler, assetsDir := newTestMiddleware(t)

	// httptest.NewRequest keeps the raw path (including "..") on r.URL.Path,
	// which is exactly the hostile shape the guard must survive.
	for _, path := range []string{
		"/local-images/../secret.txt",
		"/local-images/..%5Csecret.txt",
		"/local-images/%2e%2e/secret.txt",
		"/local-images/products/../../../secret.txt",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusForbidden {
			t.Errorf("GET %s: code = %d, want 403", path, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "secret") {
			t.Errorf("GET %s: response leaked outside content: %q", path, rec.Body.String())
		}
	}

	// Sanity: the traversal target really exists next to the assets dir.
	if _, err := os.Stat(filepath.Join(filepath.Dir(assetsDir), "secret.txt")); err != nil {
		t.Fatalf("secret fixture missing: %v", err)
	}
}

func TestLocalAssetsMiddleware_MissingFileIsNotFound(t *testing.T) {
	t.Parallel()

	handler, _ := newTestMiddleware(t)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/local-images/nope.png", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestLocalAssetsMiddleware_ResolverErrorIsInternal(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("prefix request must not reach the next handler")
	})
	handler := localAssetsMiddleware("/local-images/", func() (string, error) {
		return "", fmt.Errorf("boom")
	})(next)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/local-images/avatar.png", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("code = %d, want 500", rec.Code)
	}
}
