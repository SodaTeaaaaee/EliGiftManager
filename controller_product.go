package main

import (
	"fmt"
	"strings"

	dbpkg "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
			ti := TagInfo{TagName: t.TagName, Quantity: t.Quantity, TagType: tt, Platform: t.Platform}
			if t.WaveMemberID != nil {
				ti.WaveMemberID = *t.WaveMemberID
			}
			tagInfos = append(tagInfos, ti)
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

// ---- Level tag operations ----

// UpsertLevelTag creates or updates a level tag identified by (product_id, platform, tag_name, tag_type='level').
// OnConflict uses idx_prod_platform_tag.  Triggers ReconcileWave on success.
func (c *ProductController) UpsertLevelTag(productID uint, memberPlatform string, levelName string, quantity int) error {
	memberPlatform = strings.TrimSpace(memberPlatform)
	levelName = strings.TrimSpace(levelName)

	if productID == 0 || memberPlatform == "" || levelName == "" {
		return fmt.Errorf("upsert level tag failed: productID, memberPlatform, and levelName are required")
	}
	if quantity < 0 {
		quantity = 0
	}

	tag := model.ProductTag{
		ProductID: productID,
		Platform:  memberPlatform,
		TagName:   levelName,
		TagType:   "level",
		Quantity:  quantity,
	}

	if err := c.db().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "product_id"}, {Name: "platform"}, {Name: "tag_name"}, {Name: "tag_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"quantity", "updated_at"}),
	}).Create(&tag).Error; err != nil {
		return fmt.Errorf("upsert level tag failed: %w", err)
	}

	// Trigger ReconcileWave if the product belongs to a wave.
	var product model.Product
	if err := c.db().First(&product, productID).Error; err == nil && product.WaveID != nil {
		var wc WaveController
		if _, err := wc.ReconcileWave(*product.WaveID); err != nil {
			return fmt.Errorf("reconcile wave after level tag upsert failed: %w", err)
		}
	}

	return nil
}

// RemoveLevelTag deletes a level tag by (product_id, platform, tag_name, tag_type='level').
// Triggers ReconcileWave on success.
func (c *ProductController) RemoveLevelTag(productID uint, platform string, tagName string) error {
	platform = strings.TrimSpace(platform)
	tagName = strings.TrimSpace(tagName)

	if productID == 0 || platform == "" || tagName == "" {
		return fmt.Errorf("remove level tag failed: productID, platform, and tagName are required")
	}

	result := c.db().Where("product_id = ? AND platform = ? AND tag_name = ? AND tag_type = 'level'",
		productID, platform, tagName).Delete(&model.ProductTag{})
	if result.Error != nil {
		return fmt.Errorf("remove level tag failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("remove level tag failed: tag not found")
	}

	var product model.Product
	if err := c.db().First(&product, productID).Error; err == nil && product.WaveID != nil {
		var wc WaveController
		if _, err := wc.ReconcileWave(*product.WaveID); err != nil {
			return fmt.Errorf("reconcile wave after level tag removal failed: %w", err)
		}
	}

	return nil
}

// ---- User tag operations ----

// UpsertUserTag creates or updates a user tag identified by (product_id, wave_member_id, tag_type='user').
// Platform and TagName are filled from the WaveMember snapshot for display.
// OnConflict uses idx_prod_wm_tag.  Triggers ReconcileWave on success.
func (c *ProductController) UpsertUserTag(productID uint, waveMemberID uint, quantity int) error {
	if productID == 0 || waveMemberID == 0 {
		return fmt.Errorf("upsert user tag failed: productID and waveMemberID are required")
	}
	// Look up WaveMember to fill Platform and TagName for display.
	var wm model.WaveMember
	if err := c.db().First(&wm, waveMemberID).Error; err != nil {
		return fmt.Errorf("upsert user tag failed: wave member (id=%d) not found: %w", waveMemberID, err)
	}

	tag := model.ProductTag{
		ProductID:    productID,
		Platform:     wm.Platform,
		TagName:      wm.PlatformUID,
		TagType:      "user",
		Quantity:     quantity,
		WaveMemberID: &waveMemberID,
	}

	if err := c.db().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "product_id"}, {Name: "wave_member_id"}, {Name: "tag_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"quantity", "platform", "tag_name", "updated_at"}),
	}).Create(&tag).Error; err != nil {
		return fmt.Errorf("upsert user tag failed: %w", err)
	}

	var product model.Product
	if err := c.db().First(&product, productID).Error; err == nil && product.WaveID != nil {
		var wc WaveController
		if _, err := wc.ReconcileWave(*product.WaveID); err != nil {
			return fmt.Errorf("reconcile wave after user tag upsert failed: %w", err)
		}
	}

	return nil
}

// RemoveUserTag deletes a user tag by (product_id, wave_member_id, tag_type='user').
// Triggers ReconcileWave on success.
func (c *ProductController) RemoveUserTag(productID uint, waveMemberID uint) error {
	if productID == 0 || waveMemberID == 0 {
		return fmt.Errorf("remove user tag failed: productID and waveMemberID are required")
	}

	result := c.db().Where("product_id = ? AND wave_member_id = ? AND tag_type = 'user'",
		productID, waveMemberID).Delete(&model.ProductTag{})
	if result.Error != nil {
		return fmt.Errorf("remove user tag failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("remove user tag failed: tag not found")
	}

	var product model.Product
	if err := c.db().First(&product, productID).Error; err == nil && product.WaveID != nil {
		var wc WaveController
		if _, err := wc.ReconcileWave(*product.WaveID); err != nil {
			return fmt.Errorf("reconcile wave after user tag removal failed: %w", err)
		}
	}

	return nil
}

// ---- Legacy wrappers (backward compat for existing Wails bindings) ----

// AssignProductTag is a backward-compatible wrapper that delegates to UpsertLevelTag
// or UpsertUserTag based on tagType.  For user tags it resolves the wave member from
// the product's wave + (platform, platformUid).
func (c *ProductController) AssignProductTag(productID uint, platform, tagName string, quantity int, tagType string) error {
	platform = strings.TrimSpace(platform)
	tagName = strings.TrimSpace(tagName)
	tagType = strings.TrimSpace(tagType)

	if platform == "" || tagName == "" || productID == 0 {
		return fmt.Errorf("assign product tag failed: invalid parameters")
	}
	if tagType == "" {
		tagType = "level"
	}

	if tagType == "user" {
		// Resolve wave + wave_member from the product and (platform, tagName=platformUid).
		var product model.Product
		if err := c.db().First(&product, productID).Error; err != nil {
			return fmt.Errorf("assign product tag failed: product not found: %w", err)
		}
		if product.WaveID == nil {
			return fmt.Errorf("无法添加指定用户分配：商品未关联任何发货任务")
		}
		var wm model.WaveMember
		if err := c.db().Where("wave_id = ? AND platform = ? AND platform_uid = ?",
			*product.WaveID, platform, tagName).First(&wm).Error; err != nil {
			return fmt.Errorf("无法添加指定用户分配：平台 [%s] UID [%s] 的会员不在当前 wave 中", platform, tagName)
		}
		return c.UpsertUserTag(productID, wm.ID, quantity)
	}

	return c.UpsertLevelTag(productID, platform, tagName, quantity)
}

// RemoveProductTag is a backward-compatible wrapper that delegates to RemoveLevelTag
// or RemoveUserTag based on tagType.
func (c *ProductController) RemoveProductTag(productID uint, platform, tagName, tagType string) error {
	platform, tagName, tagType = strings.TrimSpace(platform), strings.TrimSpace(tagName), strings.TrimSpace(tagType)
	if platform == "" || tagName == "" || productID == 0 {
		return fmt.Errorf("remove product tag failed: productId, platform, and tagName are required")
	}
	if tagType == "" {
		tagType = "level"
	}

	if tagType == "user" {
		var product model.Product
		if err := c.db().First(&product, productID).Error; err != nil {
			return fmt.Errorf("remove product tag failed: product not found: %w", err)
		}
		if product.WaveID == nil {
			return fmt.Errorf("remove product tag failed: product not in any wave")
		}
		var wm model.WaveMember
		if err := c.db().Where("wave_id = ? AND platform = ? AND platform_uid = ?",
			*product.WaveID, platform, tagName).First(&wm).Error; err != nil {
			return fmt.Errorf("remove product tag failed: wave member not found for user tag removal")
		}
		return c.RemoveUserTag(productID, wm.ID)
	}

	return c.RemoveLevelTag(productID, platform, tagName)
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
