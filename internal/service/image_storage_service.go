package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
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

// ProcessCoverImages scans zipExtractDir for image files, copies them into
// the content-addressable data/assets/ tree, and inserts matching ProductImage
// rows. If a product has no CoverImage yet, the first matched image becomes its
// cover.
//
// When imageDir is non-empty only that subdirectory of zipExtractDir is
// scanned; otherwise the entire tree is walked recursively.  Set imageDir to
// "" to cover all subdirectories such as 主图/ and 详情图/.
//
// When productIDs is non-empty, only products in that list are matched for
// image association.  Pass nil or empty slice to match all products (legacy).
//
// File-name matching is only performed when platform is "柔造".  For all other
// platforms the function returns 0, nil immediately.
//
// If a single normalized file name matches more than one product in productIDs,
// the match is skipped and a warning is logged (ambiguity).
//
// For each matched product with a non-nil ProductMasterID, a corresponding
// ProductMasterImage row is also created (upsert on product_master_id + path).
//
// Returns the total number of product-image associations created.
func ProcessCoverImages(db *gorm.DB, zipExtractDir, imageDir, platform string, productIDs []uint) (int, error) {
	// D4: Only the 柔造 platform uses file-name matching.
	if platform != "柔造" {
		return 0, nil
	}

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

		q := db.Where(
			"REPLACE(REPLACE(REPLACE(name, '*', '_'), '（', '('), '）', ')') = ?",
			normalized,
		)
		if len(productIDs) > 0 {
			q = q.Where("id IN ?", productIDs)
		}

		var matchedProducts []model.Product
		if err := q.Find(&matchedProducts).Error; err != nil {
			return nil
		}

		// Ambiguity detection: skip if more than one product in the
		// matched batch shares the same normalized name.
		if len(matchedProducts) > 1 {
			ids := make([]string, len(matchedProducts))
			for i, mp := range matchedProducts {
				ids[i] = fmt.Sprintf("%d", mp.ID)
			}
			log.Printf("[WARN] 图片匹配歧义: 文件 %q 归一化名称 %q 匹配到 %d 个商品 (ids=%s)，跳过",
				filepath.Base(path), normalized, len(matchedProducts), strings.Join(ids, ","))
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

			// Also create a ProductMasterImage for the global registry.
			if p.ProductMasterID != nil {
				var maxMasterOrder int
				db.Model(&model.ProductMasterImage{}).Where("product_master_id = ?", *p.ProductMasterID).
					Select("COALESCE(MAX(sort_order), -1)").Scan(&maxMasterOrder)
				masterNextOrder := maxMasterOrder + 1

				pmi := model.ProductMasterImage{
					ProductMasterID: *p.ProductMasterID,
					Path:            relativePath,
					SourceDir:       parentDir,
				}
				db.Where("product_master_id = ? AND path = ?", *p.ProductMasterID, relativePath).
					Attrs(model.ProductMasterImage{SortOrder: masterNextOrder}).
					FirstOrCreate(&pmi)
			}

			matched++
			if p.CoverImage == "" {
				_ = db.Model(&p).Update("cover_image", relativePath)
			}

			// Sync ProductMaster cover image — fill empty cover, never overwrite existing.
			if p.ProductMasterID != nil {
				var master model.ProductMaster
				if err := db.First(&master, *p.ProductMasterID).Error; err == nil {
					if master.CoverImage == "" {
						_ = db.Model(&master).Update("cover_image", relativePath)
					}
				}
			}
		}

		return nil
	})

	return matched, nil
}
