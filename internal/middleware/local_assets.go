package middleware

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

// LocalAssetsMiddleware returns an http middleware for the Wails v3 asset
// server (application.AssetOptions.Middleware) that intercepts requests for
// local static assets before the SPA fallback can swallow them.
func LocalAssetsMiddleware(prefix string) func(http.Handler) http.Handler {
	return localAssetsMiddleware(prefix, service.ResolveAssetsDir)
}

// localAssetsMiddleware is the testable core of LocalAssetsMiddleware: the
// assets directory resolver is injected so tests never touch the real data
// directory.
func localAssetsMiddleware(prefix string, resolveAssetsDir func() (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
			serveLocalAsset(w, r, resolveAssetsDir, strings.TrimPrefix(r.URL.Path, prefix))
		})
	}
}

func serveLocalAsset(w http.ResponseWriter, r *http.Request, resolveAssetsDir func() (string, error), relPath string) {
	assetsDir, err := resolveAssetsDir()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	cleanPath := filepath.Clean(relPath)
	filePath := filepath.Join(assetsDir, cleanPath)

	// Directory traversal protection: ensure the resolved path is still
	// underneath the assets directory.
	expectedPrefix := filepath.Clean(assetsDir) + string(filepath.Separator)
	if !strings.HasPrefix(filepath.Clean(filePath)+string(filepath.Separator), expectedPrefix) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	http.ServeFile(w, r, filePath)
}
