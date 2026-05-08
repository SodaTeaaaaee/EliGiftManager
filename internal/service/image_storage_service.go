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

// normalizeProductNameForImageMatch converts a product name or image filename
// (after extension/suffix removal) into a canonical form for matching. It
// replaces all Windows-reserved filename characters with '_', converts fullwidth
// punctuation to halfwidth, strips common separators, and lowercases the result.
// Both the product name from the database and the image filename must be
// normalized before comparison.
func normalizeProductNameForImageMatch(s string) string {
	s = strings.TrimSpace(s)
	// Fullwidth→halfwidth first, so the resulting ASCII chars are caught by the
	// Windows-illegal-char replacement below (e.g. 全角：→: → then : → _).
	s = strings.NewReplacer(
		"（", "(", "）", ")",
		"：", ":", "？", "?",
	).Replace(s)
	s = strings.NewReplacer(
		"<", "_", ">", "_", ":", "_", "\"", "_",
		"/", "_", "\\", "_", "|", "_", "?", "_", "*", "_",
	).Replace(s)
	// Strip separators — consistent with normalizeDynamicKey in dynamic_parser.go.
	s = strings.NewReplacer("_", "", "-", "", " ", "").Replace(s)
	return strings.ToLower(s)
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

// ProcessCoverImages scans the extractDir subdirectories specified by coverDir
// and detailDir for image files, copies them into the content-addressable
// data/assets/ tree, and creates ProductImage rows matched by product name.
//
// When both coverDir and detailDir are empty the entire extractDir is walked
// (backward compatibility with single-directory archive layouts).
//
// Images matched from coverDir are eligible to become the product's CoverImage
// (first match wins).  Images from detailDir are stored as ProductImage rows
// with SourceDir set to the directory basename.
//
// Returns the total number of product-image associations created.
func ProcessCoverImages(db *gorm.DB, extractDir, coverDir, detailDir string) (int, error) {
	// Backward compat: if no subdirectories specified, scan the entire tree.
	if coverDir == "" && detailDir == "" {
		return processImageSubdir(db, extractDir, filepath.Base(extractDir), true)
	}

	matched := 0

	if coverDir != "" {
		srcDir := filepath.Join(extractDir, coverDir)
		if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
			count, err := processImageSubdir(db, srcDir, filepath.Base(coverDir), true)
			if err != nil {
				return matched, err
			}
			matched += count
		}
	}

	if detailDir != "" {
		srcDir := filepath.Join(extractDir, detailDir)
		if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
			count, err := processImageSubdir(db, srcDir, filepath.Base(detailDir), false)
			if err != nil {
				return matched, err
			}
			matched += count
		}
	}

	return matched, nil
}

// processImageSubdir scans a single directory for image files, copies them to
// the assets tree, and matches them against products by normalized name.
// When allowCover is true the first matched image for a product may become its
// CoverImage; otherwise only ProductImage rows are created.
func processImageSubdir(db *gorm.DB, scanDir, sourceDir string, allowCover bool) (int, error) {
	assetsDir, err := ResolveAssetsDir()
	if err != nil {
		return 0, err
	}

	// Load all products once and index by normalized name for O(1) lookup.
	var allProducts []model.Product
	db.Select("id, name, cover_image").Find(&allProducts)
	normalizedIndex := make(map[string][]model.Product, len(allProducts))
	for _, p := range allProducts {
		norm := normalizeProductNameForImageMatch(p.Name)
		if norm != "" {
			normalizedIndex[norm] = append(normalizedIndex[norm], p)
		}
	}

	if len(normalizedIndex) == 0 {
		return 0, nil
	}

	matched := 0
	_ = filepath.Walk(scanDir, func(path string, info os.FileInfo, err error) error {
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

		relativePath := hash[:2] + "/" + hash + ext

		productName := stripSuffix(filepath.Base(path))
		if productName == "" {
			return nil
		}

		normalized := normalizeProductNameForImageMatch(productName)
		matchedProducts, ok := normalizedIndex[normalized]
		if !ok {
			return nil
		}

		for _, p := range matchedProducts {
			var maxOrder int
			db.Model(&model.ProductImage{}).Where("product_id = ?", p.ID).
				Select("COALESCE(MAX(sort_order), -1)").Scan(&maxOrder)
			nextOrder := maxOrder + 1

			pi := model.ProductImage{ProductID: p.ID, Path: relativePath, SourceDir: sourceDir}
			db.Where("product_id = ? AND path = ?", p.ID, relativePath).
				Attrs(model.ProductImage{SortOrder: nextOrder}).
				FirstOrCreate(&pi)

			matched++
			if allowCover && p.CoverImage == "" {
				_ = db.Model(&p).Update("cover_image", relativePath)
			}
		}

		return nil
	})

	return matched, nil
}
