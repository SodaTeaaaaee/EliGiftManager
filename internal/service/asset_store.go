package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// AssetStore stores content-addressed product assets under a root directory.
// Layout: products/{hash[0:2]}/{hash}{ext}
type AssetStore struct {
	root string
}

// NewAssetStore creates an AssetStore rooted at ResolveAssetsDir().
func NewAssetStore() (*AssetStore, error) {
	root, err := ResolveAssetsDir()
	if err != nil {
		return nil, err
	}
	return NewAssetStoreAt(root), nil
}

// NewAssetStoreAt creates an AssetStore rooted at the given directory (for tests).
func NewAssetStoreAt(root string) *AssetStore {
	return &AssetStore{root: root}
}

// StoreBytes writes data under the content-addressed layout and returns the
// relative path (posix-style). Identical content reuses the existing file.
func (s *AssetStore) StoreBytes(data []byte, ext string) (string, error) {
	if s == nil || s.root == "" {
		return "", fmt.Errorf("asset store: root is empty")
	}

	ext = normalizeExt(ext)
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	rel := path.Join("products", hash[:2], hash+ext)
	abs := s.AbsPath(rel)

	if _, err := os.Stat(abs); err == nil {
		return rel, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("asset store: stat %q: %w", abs, err)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", fmt.Errorf("asset store: mkdir: %w", err)
	}

	tmp := abs + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", fmt.Errorf("asset store: write tmp: %w", err)
	}

	// Another writer may have landed the final file first; prefer the existing one.
	if _, err := os.Stat(abs); err == nil {
		_ = os.Remove(tmp)
		return rel, nil
	}

	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		// Race: final may now exist after a concurrent rename.
		if _, statErr := os.Stat(abs); statErr == nil {
			return rel, nil
		}
		return "", fmt.Errorf("asset store: rename: %w", err)
	}
	return rel, nil
}

// StoreFile reads srcPath and stores it via StoreBytes, using the source extension.
func (s *AssetStore) StoreFile(srcPath string) (string, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("asset store: read %q: %w", srcPath, err)
	}
	return s.StoreBytes(data, filepath.Ext(srcPath))
}

// AbsPath joins root with a relative asset path.
func (s *AssetStore) AbsPath(rel string) string {
	return filepath.Join(s.root, filepath.FromSlash(rel))
}

// URLPath returns the HTTP path served by LocalAssetsMiddleware.
func (s *AssetStore) URLPath(rel string) string {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "/")
	return "/local-images/" + rel
}

func normalizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}
