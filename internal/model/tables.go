package model

import "time"

// Member represents the normalized platform member record.
type Member struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	Platform    string           `gorm:"size:100;not null;uniqueIndex:idx_members_platform_uid" json:"platform"`
	PlatformUID string           `gorm:"size:255;not null;uniqueIndex:idx_members_platform_uid" json:"platformUid"`
	ExtraData   string           `gorm:"type:text;not null;default:'{}'" json:"extraData"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	Nicknames   []MemberNickname `gorm:"foreignKey:MemberID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"nicknames"`
	Addresses   []MemberAddress  `gorm:"foreignKey:MemberID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"addresses"`
}

// MemberNickname stores nickname history for a member.
type MemberNickname struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MemberID  uint      `gorm:"not null;index" json:"memberId"`
	Nickname  string    `gorm:"size:255;not null;index" json:"nickname"`
	CreatedAt time.Time `json:"createdAt"`
	Member    Member    `gorm:"foreignKey:MemberID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"member"`
}

// MemberAddress stores member shipping address history.
type MemberAddress struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	MemberID      uint      `gorm:"not null;index" json:"memberId"`
	RecipientName string    `gorm:"size:255;not null" json:"recipientName"`
	Phone         string    `gorm:"size:64;not null" json:"phone"`
	Address       string    `gorm:"type:text;not null" json:"address"`
	IsDefault     bool      `gorm:"not null;default:false;index" json:"isDefault"`
	IsDeleted     bool      `gorm:"not null;default:false;index" json:"isDeleted"`
	IsTestAddress bool      `gorm:"not null;default:false;index" json:"isTestAddress"`
	CreatedAt     time.Time `json:"createdAt"`
	Member        Member    `gorm:"foreignKey:MemberID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"member"`
}

// ProductMaster represents the global product registry indexed by (platform, factory_sku).
// It stores the canonical product attributes independent of any wave.
type ProductMaster struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Platform   string    `gorm:"size:100;not null;uniqueIndex:idx_product_master_platform_sku" json:"platform"`
	Factory    string    `gorm:"size:100;not null" json:"factory"`
	FactorySKU string    `gorm:"size:255;not null;uniqueIndex:idx_product_master_platform_sku" json:"factorySku"`
	Name       string    `gorm:"size:255;not null" json:"name"`
	CoverImage string    `gorm:"type:text" json:"coverImage"`
	ExtraData  string    `gorm:"type:text;not null;default:'{}'" json:"extraData"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// Product represents a wave-specific snapshot of a product, linked to ProductMaster.
type Product struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Platform        string         `gorm:"size:100;not null;index;uniqueIndex:idx_product_wave_platform_sku" json:"platform"`
	Factory         string         `gorm:"size:100;not null" json:"factory"`
	FactorySKU      string         `gorm:"size:255;not null;index;uniqueIndex:idx_product_wave_platform_sku" json:"factorySku"`
	Name            string         `gorm:"size:255;not null" json:"name"`
	CoverImage      string         `gorm:"type:text" json:"coverImage"`
	WaveID          *uint          `gorm:"index;uniqueIndex:idx_product_wave_platform_sku" json:"waveId"`
	ExtraData       string         `gorm:"type:text;not null;default:'{}'" json:"extraData"`
	ProductMasterID *uint          `gorm:"index" json:"productMasterId"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	Tags            []ProductTag   `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tags"`
	Images          []ProductImage `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"images"`
}

// ProductImage stores multi-image associations for a product.
type ProductImage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProductID uint      `gorm:"not null;index" json:"productId"`
	Path      string    `gorm:"type:text;not null" json:"path"`
	SortOrder int       `gorm:"not null;default:0" json:"sortOrder"`
	SourceDir string    `gorm:"size:100;not null;default:''" json:"sourceDir"`
	CreatedAt time.Time `json:"createdAt"`
	Product   Product   `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"product"`
}

// ProductMasterImage stores global image associations for a ProductMaster.
type ProductMasterImage struct {
	ID              uint          `gorm:"primaryKey" json:"id"`
	ProductMasterID uint          `gorm:"not null;index;uniqueIndex:idx_pmi_master_path" json:"productMasterId"`
	Path            string        `gorm:"type:text;not null;uniqueIndex:idx_pmi_master_path" json:"path"`
	SortOrder       int           `gorm:"not null;default:0" json:"sortOrder"`
	SourceDir       string        `gorm:"size:100;not null;default:''" json:"sourceDir"`
	CreatedAt       time.Time     `json:"createdAt"`
	ProductMaster   ProductMaster `gorm:"foreignKey:ProductMasterID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"productMaster"`
}

// ProductTag stores platform-level classification tags attached to a product.
//
// Two tag families are distinguished by TagType:
//
//   - identity (tag_type='identity'): match on platform-level attributes via MatchMode.
//     Unique on (product_id, platform, tag_name, match_mode) WHERE tag_type='identity'
//     (partial unique index idx_prod_identity_tag).
//     MatchMode values: gift_level / platform_all / wave_all.
//     WaveMemberID is never set for identity tags.
//     Identity tags accumulate — multiple matching tags on the same product
//     each contribute their quantity (no mutual exclusion).
//
//   - user (tag_type='user'): per-member quantity overrides that bypass matchMode.
//     TagName stores the member's PlatformUID. Unique on (product_id, wave_member_id)
//     WHERE tag_type='user' (partial unique index idx_prod_user_tag).
//     WaveMemberID points to the specific wave member; always non-nil for user tags.
type ProductTag struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	ProductID    uint        `gorm:"not null" json:"productId"`
	Platform     string      `gorm:"size:100;not null" json:"platform"`
	TagName      string      `gorm:"size:255;not null" json:"tagName"`
	MatchMode    string      `gorm:"size:20;not null;default:'gift_level'" json:"matchMode"`
	TagType      string      `gorm:"size:20;not null;default:'identity'" json:"tagType"`
	Quantity     int         `gorm:"not null;default:1" json:"quantity"`
	WaveMemberID *uint       `json:"waveMemberId"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
	WaveMember   *WaveMember `gorm:"foreignKey:WaveMemberID" json:"-"`
	Product      Product     `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"product"`
}

// Wave 表示一次按特定规则聚合的发货波次（UI 层面称为发货任务）。
type Wave struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	WaveNo    string           `gorm:"size:64;not null;uniqueIndex" json:"waveNo"`
	Name      string           `gorm:"size:255;not null" json:"name"`
	Status    string           `gorm:"size:64;not null;default:'draft'" json:"status"`
	LevelTags string           `gorm:"type:text;not null;default:'[]'" json:"levelTags"` // 波次身份标签候选集 JSON（字段名保留 level_tags）
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
	Records   []DispatchRecord `gorm:"foreignKey:WaveID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"records"`
}

// DispatchRecord represents a member/product allocation under a wave.
type DispatchRecord struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	WaveID          uint           `gorm:"not null;index" json:"waveId"`
	MemberID        uint           `gorm:"not null;index" json:"memberId"`
	ProductID       uint           `gorm:"not null;index" json:"productId"`
	MemberAddressID *uint          `gorm:"index" json:"memberAddressId"`
	Quantity        int            `gorm:"not null;default:1" json:"quantity"`
	Status          string         `gorm:"size:64;not null;index" json:"status"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	Wave            Wave           `gorm:"foreignKey:WaveID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"wave"`
	Member          Member         `gorm:"foreignKey:MemberID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"member"`
	Product         Product        `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"product"`
	MemberAddress   *MemberAddress `gorm:"foreignKey:MemberAddressID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"memberAddress"`
}

// TemplateConfig stores dynamic import/export/allocation template mappings.
type TemplateConfig struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Platform     string    `gorm:"size:100;not null;index" json:"platform"`
	Type         string    `gorm:"size:100;not null;index" json:"type"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	MappingRules string    `gorm:"type:text;not null;default:'{}'" json:"mappingRules"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// WaveMember records which members were imported into a wave.
// It serves as a snapshot of the member's key attributes at import time,
// decoupling wave-level operations (tag matching, dispatch) from the global members table.
type WaveMember struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	WaveID         uint      `gorm:"not null;uniqueIndex:idx_wave_member" json:"waveId"`
	MemberID       uint      `gorm:"not null;uniqueIndex:idx_wave_member" json:"memberId"`
	Platform       string    `gorm:"size:100;not null;default:''" json:"platform"`
	PlatformUID    string    `gorm:"size:255;not null;default:''" json:"platformUid"`
	GiftLevel      string    `gorm:"size:100;not null;default:''" json:"giftLevel"`
	LatestNickname string    `gorm:"size:255;not null;default:''" json:"latestNickname"`
	CreatedAt      time.Time `json:"createdAt"`
	Wave           Wave      `gorm:"foreignKey:WaveID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Member         Member    `gorm:"foreignKey:MemberID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (Member) TableName() string             { return "members" }
func (MemberNickname) TableName() string     { return "member_nicknames" }
func (MemberAddress) TableName() string      { return "member_addresses" }
func (ProductMaster) TableName() string      { return "product_masters" }
func (Product) TableName() string            { return "products" }
func (Wave) TableName() string               { return "waves" }
func (DispatchRecord) TableName() string     { return "dispatch_records" }
func (ProductTag) TableName() string         { return "product_tags" }
func (TemplateConfig) TableName() string     { return "template_configs" }
func (ProductImage) TableName() string       { return "product_images" }
func (ProductMasterImage) TableName() string { return "product_master_images" }
func (WaveMember) TableName() string         { return "wave_members" }
