package main

import (
	"fmt"
	"strings"

	dbpkg "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/model"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProductController struct{}

func (c *ProductController) db() *gorm.DB { return dbpkg.GetDB() }

// ListProducts 查询波次步骤页使用的 Product 快照（带 wave_id 过滤的页面不需要此接口，
// 它们应使用 ListProductsWithTags）。此接口保留用于全局商品库浏览等非波次场景。
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
				tt = "identity"
			}
			ti := TagInfo{TagName: t.TagName, Quantity: t.Quantity, TagType: tt, Platform: t.Platform, MatchMode: t.MatchMode}
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

// ---- Level tag operations (compatibility wrappers) ----

// UpsertLevelTag is a backward-compatible wrapper that delegates to UpsertIdentityTag
// with matchMode="gift_level".  Old "level" tagType is normalised to "identity" on write.
func (c *ProductController) UpsertLevelTag(productID uint, memberPlatform string, levelName string, quantity int) error {
	return c.UpsertIdentityTag(productID, memberPlatform, levelName, "gift_level", quantity)
}

// RemoveLevelTag is a backward-compatible wrapper that delegates to RemoveIdentityTag
// with matchMode="gift_level".
func (c *ProductController) RemoveLevelTag(productID uint, platform string, tagName string) error {
	return c.RemoveIdentityTag(productID, platform, tagName, "gift_level")
}

// ---- Identity tag operations ----

// UpsertIdentityTag creates or updates an identity tag identified by the 4-column
// partial unique index (product_id, platform, tag_name, match_mode) WHERE tag_type='identity'.
// matchMode defaults to "gift_level" when empty; "user_member" is rejected.
// Triggers ReconcileWave on success.
func (c *ProductController) UpsertIdentityTag(productID uint, platform, tagName, matchMode string, quantity int) error {
	var err error
	platform, tagName, matchMode, err = service.NormalizeIdentityTag(platform, tagName, matchMode)
	if err != nil {
		return fmt.Errorf("upsert identity tag failed: %w", err)
	}

	if productID == 0 {
		return fmt.Errorf("upsert identity tag failed: productID is required")
	}
	if quantity < 0 {
		quantity = 0
	}

	tag := model.ProductTag{
		ProductID: productID,
		Platform:  platform,
		TagName:   tagName,
		MatchMode: matchMode,
		TagType:   "identity",
		Quantity:  quantity,
	}

	if err := c.db().Clauses(clause.OnConflict{
		Columns:     []clause.Column{{Name: "product_id"}, {Name: "platform"}, {Name: "tag_name"}, {Name: "match_mode"}},
		TargetWhere: identityTagConflictTargetWhere(),
		DoUpdates: clause.AssignmentColumns([]string{"quantity", "updated_at"}),
	}).Create(&tag).Error; err != nil {
		return fmt.Errorf("upsert identity tag failed: %w", err)
	}

	// Trigger ReconcileWave if the product belongs to a wave.
	var product model.Product
	if err := c.db().First(&product, productID).Error; err == nil && product.WaveID != nil {
		var wc WaveController
		if _, err := wc.ReconcileWave(*product.WaveID); err != nil {
			return fmt.Errorf("reconcile wave after identity tag upsert failed: %w", err)
		}
	}

	return nil
}

// RemoveIdentityTag deletes an identity tag by (product_id, platform, tag_name, match_mode)
// WHERE tag_type='identity'. matchMode is validated via NormalizeIdentityTag; user_member
// and unknown values are rejected. Triggers ReconcileWave on success.
func (c *ProductController) RemoveIdentityTag(productID uint, platform, tagName, matchMode string) error {
	var err error
	platform, tagName, matchMode, err = service.NormalizeIdentityTag(platform, tagName, matchMode)
	if err != nil {
		return fmt.Errorf("remove identity tag failed: %w", err)
	}

	if productID == 0 {
		return fmt.Errorf("remove identity tag failed: productID is required")
	}

	result := c.db().Where("product_id = ? AND platform = ? AND tag_name = ? AND match_mode = ? AND tag_type = 'identity'",
		productID, platform, tagName, matchMode).Delete(&model.ProductTag{})
	if result.Error != nil {
		return fmt.Errorf("remove identity tag failed: %w", result.Error)
	}

	// Idempotent: trigger ReconcileWave even if the tag didn't exist.
	var product model.Product
	if err := c.db().First(&product, productID).Error; err == nil && product.WaveID != nil {
		var wc WaveController
		if _, err := wc.ReconcileWave(*product.WaveID); err != nil {
			return fmt.Errorf("reconcile wave after identity tag removal failed: %w", err)
		}
	}

	return nil
}

// ---- User tag operations ----

// UpsertUserTag creates or updates a user tag identified by (product_id, wave_member_id, tag_type='user').
// Platform and TagName are filled from the WaveMember snapshot for display.
// MatchMode is set to "user_member".
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
		MatchMode:    "user_member",
		TagType:      "user",
		Quantity:     quantity,
		WaveMemberID: &waveMemberID,
	}

	if err := c.db().Clauses(clause.OnConflict{
		Columns:     []clause.Column{{Name: "product_id"}, {Name: "wave_member_id"}},
		TargetWhere: userTagConflictTargetWhere(),
		DoUpdates: clause.AssignmentColumns([]string{"quantity", "platform", "tag_name", "match_mode", "updated_at"}),
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
// MatchMode is not involved in the WHERE clause (user tags are unique by wave_member_id).
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

	// Idempotent: trigger ReconcileWave even if the tag didn't exist.
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

// AssignProductTag is a backward-compatible wrapper that delegates to UpsertIdentityTag
// (matchMode="gift_level") or UpsertUserTag based on tagType.  For user tags it resolves
// the wave member from the product's wave + (platform, platformUid).
func (c *ProductController) AssignProductTag(productID uint, platform, tagName string, quantity int, tagType string) error {
	platform = strings.TrimSpace(platform)
	tagName = strings.TrimSpace(tagName)
	tagType = strings.TrimSpace(tagType)

	if platform == "" || tagName == "" || productID == 0 {
		return fmt.Errorf("assign product tag failed: invalid parameters")
	}
	if tagType == "" {
		tagType = "identity"
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

	// Old "level" tagType is normalised to identity with matchMode="gift_level".
	return c.UpsertIdentityTag(productID, platform, tagName, "gift_level", quantity)
}

// RemoveProductTag is a backward-compatible wrapper that delegates to RemoveIdentityTag
// (matchMode="gift_level") or RemoveUserTag based on tagType.
func (c *ProductController) RemoveProductTag(productID uint, platform, tagName, tagType string) error {
	platform, tagName, tagType = strings.TrimSpace(platform), strings.TrimSpace(tagName), strings.TrimSpace(tagType)
	if platform == "" || tagName == "" || productID == 0 {
		return fmt.Errorf("remove product tag failed: productId, platform, and tagName are required")
	}
	if tagType == "" {
		tagType = "identity"
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

	// Old "level" tagType is normalised to identity with matchMode="gift_level".
	return c.RemoveIdentityTag(productID, platform, tagName, "gift_level")
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

// ListProductMasters 查询全局商品库 ProductMaster（与波次无关）。
func (c *ProductController) ListProductMasters(page, pageSize int, keyword, platform string) (ProductMasterListPayload, error) {
	db := c.db()
	if db == nil {
		return ProductMasterListPayload{}, fmt.Errorf("database not available")
	}
	page, pageSize = normalizePagination(page, pageSize)

	q := db.Model(&model.ProductMaster{})
	if platform = strings.TrimSpace(platform); platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR factory_sku LIKE ? OR factory LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return ProductMasterListPayload{}, err
	}
	var masters []model.ProductMaster
	if err := q.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&masters).Error; err != nil {
		return ProductMasterListPayload{}, err
	}
	items := make([]ProductMasterItem, 0, len(masters))
	for _, m := range masters {
		items = append(items, ProductMasterItem{
			ID:         m.ID,
			Platform:   m.Platform,
			Factory:    m.Factory,
			FactorySKU: m.FactorySKU,
			Name:       m.Name,
			CoverImage: m.CoverImage,
			ExtraData:  m.ExtraData,
			UpdatedAt:  m.UpdatedAt,
		})
	}
	platforms := make([]string, 0)
	db.Model(&model.ProductMaster{}).Distinct().Order("platform ASC").Pluck("platform", &platforms)
	return ProductMasterListPayload{Items: items, Total: total, Platforms: platforms}, nil
}

func (c *ProductController) GetProductMasterImages(masterID uint) ([]model.ProductMasterImage, error) {
	db := c.db()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var images []model.ProductMasterImage
	if err := db.Where("product_master_id = ?", masterID).Order("sort_order ASC").Find(&images).Error; err != nil {
		return nil, fmt.Errorf("get product master images failed: %w", err)
	}
	return images, nil
}
