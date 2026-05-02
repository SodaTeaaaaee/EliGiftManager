package main

import (
	"fmt"
	"strings"

	dbpkg "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
)

type ProductController struct{}

func (c *ProductController) db() *gorm.DB { return dbpkg.GetDB() }

func (c *ProductController) ListProducts(page, pageSize int, keyword, platform string) (ProductListPayload, error) {
	db := c.db()
	if db == nil {
		return ProductListPayload{}, fmt.Errorf("database not available")
	}
	page, pageSize = normalizePagination(page, pageSize)

	q := db.Model(&model.Product{})
	if platform = strings.TrimSpace(platform); platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR factory_sku LIKE ? OR factory LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return ProductListPayload{}, err
	}
	var products []model.Product
	if err := q.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&products).Error; err != nil {
		return ProductListPayload{}, err
	}
	items, err := buildProductItems(db, products)
	if err != nil {
		return ProductListPayload{}, err
	}
	platforms, err := queryProductPlatforms(db)
	if err != nil {
		return ProductListPayload{}, err
	}
	return ProductListPayload{Items: items, Total: total, Platforms: platforms}, nil
}

func (c *ProductController) UpdateProduct(product model.Product) error {
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	if product.ID == 0 {
		return fmt.Errorf("product id is required")
	}
	updates := map[string]any{"platform": strings.TrimSpace(product.Platform), "name": strings.TrimSpace(product.Name), "cover_image": product.CoverImage, "extra_data": product.ExtraData}
	if updates["platform"] == "" || updates["name"] == "" {
		return fmt.Errorf("platform and name are required")
	}
	if updates["extra_data"] == "" {
		updates["extra_data"] = "{}"
	}
	return db.Model(&model.Product{}).Where("id = ?", product.ID).Updates(updates).Error
}

func (c *ProductController) ListProductsWithTags(waveID uint, platform string, page, pageSize int) (ProductListWithTagsPayload, error) {
	db := c.db()
	if db == nil {
		return ProductListWithTagsPayload{}, fmt.Errorf("database not available")
	}
	page, pageSize = normalizePagination(page, pageSize)
	q := db.Model(&model.Product{}).Preload("Tags")
	if waveID != 0 {
		q = q.Where("wave_id = ?", waveID)
	}
	if platform = strings.TrimSpace(platform); platform != "" {
		if plats := strings.Split(platform, ","); len(plats) > 1 {
			q = q.Where("platform IN ?", plats)
		} else {
			q = q.Where("platform = ?", platform)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return ProductListWithTagsPayload{}, err
	}
	var products []model.Product
	if err := q.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&products).Error; err != nil {
		return ProductListWithTagsPayload{}, err
	}
	items := make([]ProductItemWithTags, 0, len(products))
	for _, p := range products {
		tagInfos := make([]TagInfo, 0, len(p.Tags))
		for _, t := range p.Tags {
			tt := t.TagType
			if tt == "" {
				tt = "level"
			}
			tagInfos = append(tagInfos, TagInfo{TagName: t.TagName, Quantity: t.Quantity, TagType: tt})
		}
		items = append(items, ProductItemWithTags{ID: p.ID, Platform: p.Platform, Factory: p.Factory, FactorySKU: p.FactorySKU, Name: p.Name, CoverImage: p.CoverImage, ExtraData: p.ExtraData, UpdatedAt: p.UpdatedAt, Tags: tagInfos})
	}
	platforms, _ := queryProductPlatforms(db)
	return ProductListWithTagsPayload{Items: items, Total: total, Platforms: platforms}, nil
}

func (c *ProductController) ListProductTags(platform string) ([]string, error) {
	db := c.db()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var tags []string
	if err := db.Model(&model.ProductTag{}).Where("platform = ?", strings.TrimSpace(platform)).Distinct().Order("tag_name ASC").Pluck("tag_name", &tags).Error; err != nil {
		return nil, fmt.Errorf("list product tags failed: %w", err)
	}
	return tags, nil
}

func (c *ProductController) AssignProductTag(productID uint, platform, tagName string, quantity int, tagType string) error {
	platform, tagName, tagType = strings.TrimSpace(platform), strings.TrimSpace(tagName), strings.TrimSpace(tagType)
	if platform == "" || tagName == "" || productID == 0 {
		return fmt.Errorf("assign product tag failed: productId, platform, and tagName are required")
	}
	if tagType == "" {
		tagType = "level"
	}
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	// Normalise: if we're writing a "level" tag and a legacy record exists with
	// tag_type="", migrate it to tag_type="level" first to avoid duplicates.
	if tagType == "level" {
		db.Model(&model.ProductTag{}).
			Where("product_id = ? AND platform = ? AND tag_name = ? AND tag_type = ?", productID, platform, tagName, "").
			Update("tag_type", "level")
	}

	tag := model.ProductTag{ProductID: productID, Platform: platform, TagName: tagName, TagType: tagType, Quantity: quantity}
	if err := db.Where("product_id = ? AND platform = ? AND tag_name = ? AND tag_type = ?", productID, platform, tagName, tagType).FirstOrCreate(&tag).Error; err != nil {
		return fmt.Errorf("assign product tag failed: %w", err)
	}
	if tag.Quantity != quantity || tag.TagType != tagType {
		updates := map[string]interface{}{"quantity": quantity}
		if tag.TagType != tagType {
			updates["tag_type"] = tagType
		}
		if err := db.Model(&tag).Updates(updates).Error; err != nil {
			return fmt.Errorf("assign product tag failed: update: %w", err)
		}
	}
	return nil
}

func (c *ProductController) RemoveProductTag(productID uint, platform, tagName, tagType string) error {
	platform, tagName, tagType = strings.TrimSpace(platform), strings.TrimSpace(tagName), strings.TrimSpace(tagType)
	if platform == "" || tagName == "" || productID == 0 {
		return fmt.Errorf("remove product tag failed: productId, platform, and tagName are required")
	}
	db := c.db()
	if db == nil {
		return fmt.Errorf("database not available")
	}
	result := db.Where("product_id = ? AND platform = ? AND tag_name = ? AND tag_type = ?", productID, platform, tagName, tagType).Delete(&model.ProductTag{})
	if result.Error != nil {
		return fmt.Errorf("remove product tag failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("remove product tag failed: tag not found")
	}
	return nil
}

func (c *ProductController) GetProductImages(productID uint) ([]model.ProductImage, error) {
	db := c.db()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var images []model.ProductImage
	if err := db.Where("product_id = ?", productID).Order("sort_order ASC").Find(&images).Error; err != nil {
		return nil, fmt.Errorf("get product images failed: %w", err)
	}
	return images, nil
}
