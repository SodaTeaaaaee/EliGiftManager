package service

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetStore_StoreBytes_DedupSameContent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewAssetStoreAt(root)
	data := []byte("identical-product-image-bytes")

	rel1, err := store.StoreBytes(data, ".PNG")
	if err != nil {
		t.Fatalf("first StoreBytes: %v", err)
	}
	rel2, err := store.StoreBytes(data, ".png")
	if err != nil {
		t.Fatalf("second StoreBytes: %v", err)
	}
	if rel1 != rel2 {
		t.Fatalf("expected same rel path, got %q and %q", rel1, rel2)
	}

	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	wantRel := path.Join("products", hash[:2], hash+".png")
	if rel1 != wantRel {
		t.Fatalf("rel path = %q, want %q", rel1, wantRel)
	}

	abs := store.AbsPath(rel1)
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("stored file missing at %q: %v", abs, err)
	}

	// Exactly one content file under products/ (no leftover .tmp).
	var files []string
	err = filepath.WalkDir(filepath.Join(root, "products"), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if err != nil {
		t.Fatalf("walk products: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected exactly 1 file under products/, got %d: %v", len(files), files)
	}
	if strings.HasSuffix(files[0], ".tmp") {
		t.Fatalf("leftover tmp file: %s", files[0])
	}
}

func TestAssetStore_StoreFile_UsesSourceExt(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "photo.JPG")
	data := []byte("jpeg-bytes-for-store-file")
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	store := NewAssetStoreAt(root)
	rel, err := store.StoreFile(src)
	if err != nil {
		t.Fatalf("StoreFile: %v", err)
	}
	if !strings.HasSuffix(rel, ".jpg") {
		t.Fatalf("expected lowercase .jpg ext, got %q", rel)
	}

	got, err := os.ReadFile(store.AbsPath(rel))
	if err != nil {
		t.Fatalf("read stored: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("stored content mismatch")
	}
}

func TestAssetStore_URLPath_Posix(t *testing.T) {
	t.Parallel()

	store := NewAssetStoreAt(t.TempDir())
	rel := path.Join("products", "ab", "abcd.png")
	got := store.URLPath(rel)
	want := "/local-images/products/ab/abcd.png"
	if got != want {
		t.Fatalf("URLPath = %q, want %q", got, want)
	}
}

func TestAssetStore_AbsPath_JoinsRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewAssetStoreAt(root)
	rel := "products/ab/abcd.png"
	got := store.AbsPath(rel)
	want := filepath.Join(root, "products", "ab", "abcd.png")
	if got != want {
		t.Fatalf("AbsPath = %q, want %q", got, want)
	}
}

func TestNormalizeExt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want string
	}{
		{".PNG", ".png"},
		{"jpg", ".jpg"},
		{"", ""},
		{"  .JPEG  ", ".jpeg"},
	}
	for _, tc := range cases {
		if got := normalizeExt(tc.in); got != tc.want {
			t.Fatalf("normalizeExt(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
