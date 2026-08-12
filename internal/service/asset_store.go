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

// AssetStage isolates files produced by an import until its database
// transaction is ready to commit. Commit publishes the staged files; Rollback
// removes both the staging directory and files newly published by this stage.
type AssetStage struct {
	finalRoot string
	stageRoot string
	store     *AssetStore
	published []string
	committed bool
	finalized bool
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

// BeginStage creates an isolated store below the final asset root.
func (s *AssetStore) BeginStage() (*AssetStage, error) {
	if s == nil || strings.TrimSpace(s.root) == "" {
		return nil, fmt.Errorf("asset store: root is empty")
	}
	parent := filepath.Join(s.root, ".staging")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, fmt.Errorf("asset stage: mkdir: %w", err)
	}
	root, err := os.MkdirTemp(parent, "catalog-")
	if err != nil {
		return nil, fmt.Errorf("asset stage: create: %w", err)
	}
	return &AssetStage{
		finalRoot: s.root,
		stageRoot: root,
		store:     NewAssetStoreAt(root),
	}, nil
}

// Store returns the isolated AssetStore used while importing.
func (s *AssetStage) Store() *AssetStore {
	if s == nil {
		return nil
	}
	return s.store
}

// Commit publishes staged files. Existing content-addressed files are reused.
func (s *AssetStage) Commit() error {
	if s == nil || s.finalized {
		return fmt.Errorf("asset stage: invalid state")
	}
	if s.committed {
		return nil
	}
	err := filepath.WalkDir(s.stageRoot, func(src string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(s.stageRoot, src)
		if relErr != nil {
			return relErr
		}
		dst := filepath.Join(s.finalRoot, rel)
		if _, statErr := os.Stat(dst); statErr == nil {
			return nil
		} else if !os.IsNotExist(statErr) {
			return statErr
		}
		if mkdirErr := os.MkdirAll(filepath.Dir(dst), 0o755); mkdirErr != nil {
			return mkdirErr
		}
		if renameErr := os.Rename(src, dst); renameErr != nil {
			return renameErr
		}
		s.published = append(s.published, dst)
		return nil
	})
	if err != nil {
		_ = s.Rollback()
		return fmt.Errorf("asset stage: publish: %w", err)
	}
	s.committed = true
	return nil
}

// Finalize keeps published files and removes the staging directory.
func (s *AssetStage) Finalize() error {
	if s == nil || s.finalized {
		return nil
	}
	s.finalized = true
	if err := os.RemoveAll(s.stageRoot); err != nil {
		return fmt.Errorf("asset stage: finalize: %w", err)
	}
	return nil
}

// Rollback removes files newly published by this stage and its staging tree.
func (s *AssetStage) Rollback() error {
	if s == nil || s.finalized {
		return nil
	}
	var firstErr error
	for i := len(s.published) - 1; i >= 0; i-- {
		if err := os.Remove(s.published[i]); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	if err := os.RemoveAll(s.stageRoot); err != nil && firstErr == nil {
		firstErr = err
	}
	s.finalized = true
	return firstErr
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
