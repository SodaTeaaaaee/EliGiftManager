package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

var suffixPattern = regexp.MustCompile(`#\d+`)

// ResolveAssetsDir returns the absolute path to data/assets/, using the same
// dev/production fallback logic as resolveDatabasePath() in app.go.
// During wails dev the binary runs from a temp directory, so we fall back to
// the working directory (project root).
func ResolveAssetsDir() (string, error) {
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		if !strings.HasPrefix(execDir, os.TempDir()) {
			return filepath.Join(execDir, "data", "assets"), nil
		}
	}
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve assets dir failed: %w", err)
	}
	return filepath.Join(workDir, "data", "assets"), nil
}

// hashFile computes the SHA-256 hash of the file at path and returns its hex
// encoding.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("hash file open failed: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file read failed: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// stripSuffix removes the file extension and any "#01" / "#02" style numbered
// suffix from filename, returning the base product name.
func stripSuffix(filename string) string {
	name := filename
	if ext := filepath.Ext(name); ext != "" {
		name = name[:len(name)-len(ext)]
	}
	name = suffixPattern.ReplaceAllString(name, "")
	return strings.TrimSpace(name)
}

// normalizeForMatch normalizes a string for SQLite LIKE matching by
// converting '*' → '_' and fullwidth parentheses to halfwidth.
func normalizeForMatch(s string) string {
	r := strings.NewReplacer("*", "_", "（", "(", "）", ")")
	return r.Replace(s)
}

// copyAssetFile copies src to dst, creating the destination directory if it
// does not exist.
func copyAssetFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("copy asset mkdir failed: %w", err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copy asset open src failed: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copy asset create dst failed: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy asset write failed: %w", err)
	}
	return out.Sync()
}

// ProcessCoverImages scans zipExtractDir (or its imageDir subdirectory) for
// image files, copies them into the content-addressable data/assets/ tree, and
// inserts matching ProductImage rows. If a product has no CoverImage yet, the
// first matched image becomes its cover.
//
// imageDir comes from the template MappingRules; when empty, the extract root
// is scanned directly.
//
// Returns the total number of product-image associations created.
func ProcessCoverImages(db *gorm.DB, zipExtractDir, imageDir string) (int, error) {
	srcDir := zipExtractDir
	if imageDir != "" {
		srcDir = filepath.Join(zipExtractDir, imageDir)
	}

	// Fall back to extract root if subdirectory not found.
	if info, err := os.Stat(srcDir); err != nil || !info.IsDir() {
		if imageDir != "" {
			srcDir = zipExtractDir
		}
	}

	assetsDir, err := ResolveAssetsDir()
	if err != nil {
		return 0, err
	}

	matched := 0
	_ = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		hash, err := hashFile(path)
		if err != nil {
			return nil
		}

		ext := filepath.Ext(path)
		targetDir := filepath.Join(assetsDir, hash[:2])
		targetPath := filepath.Join(targetDir, hash+ext)

		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			if err := copyAssetFile(path, targetPath); err != nil {
				return nil
			}
		}

		// Relative path uses forward slashes for cross-platform consistency
		// and to match the Wails asset URL convention.
		relativePath := hash[:2] + "/" + hash + ext

			parentDir := filepath.Base(filepath.Dir(path))

			productName := stripSuffix(filepath.Base(path))
		if productName == "" {
			return nil
		}

		normalized := normalizeForMatch(productName)

		var matchedProducts []model.Product
		if err := db.Where(
			"REPLACE(REPLACE(REPLACE(name, '*', '_'), '（', '('), '）', ')') = ?",
			normalized,
		).Find(&matchedProducts).Error; err != nil {
			return nil
		}

		for _, p := range matchedProducts {
			var maxOrder int
			db.Model(&model.ProductImage{}).Where("product_id = ?", p.ID).
				Select("COALESCE(MAX(sort_order), -1)").Scan(&maxOrder)
			nextOrder := maxOrder + 1

			pi := model.ProductImage{ProductID: p.ID, Path: relativePath, SourceDir: parentDir}
			db.Where("product_id = ? AND path = ?", p.ID, relativePath).
				Attrs(model.ProductImage{SortOrder: nextOrder}).
				FirstOrCreate(&pi)

			matched++
			if p.CoverImage == "" {
				_ = db.Model(&p).Update("cover_image", relativePath)
			}
		}

		return nil
	})

	return matched, nil
}
