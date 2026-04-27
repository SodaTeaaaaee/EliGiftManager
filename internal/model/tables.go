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
	CreatedAt     time.Time `json:"createdAt"`
	Member        Member    `gorm:"foreignKey:MemberID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"member"`
}

// Product represents the normalized product/gift record.
type Product struct {
	ID         uint         `gorm:"primaryKey" json:"id"`
	Platform   string       `gorm:"size:100;not null;index" json:"platform"`
	Factory    string       `gorm:"size:100;not null" json:"factory"`
	FactorySKU string       `gorm:"size:255;not null;index" json:"factorySku"`
	Name       string       `gorm:"size:255;not null" json:"name"`
	CoverImage string       `gorm:"type:text" json:"coverImage"`
	WaveID     *uint        `gorm:"index" json:"waveId"`
	ExtraData  string       `gorm:"type:text;not null;default:'{}'" json:"extraData"`
	CreatedAt  time.Time    `json:"createdAt"`
	UpdatedAt  time.Time    `json:"updatedAt"`
	Tags       []ProductTag   `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tags"`
	Images     []ProductImage `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"images"`
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

// ProductTag stores platform-level classification tags attached to a product.
// TagName captures the gift tier/level name (e.g. "舰长", "提督").
type ProductTag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProductID uint      `gorm:"not null;uniqueIndex:idx_prod_platform_tag" json:"productId"`
	Platform  string    `gorm:"size:100;not null;uniqueIndex:idx_prod_platform_tag" json:"platform"`
	TagName   string    `gorm:"size:255;not null;uniqueIndex:idx_prod_platform_tag" json:"tagName"`
	CreatedAt time.Time `json:"createdAt"`
	Product   Product   `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"product"`
}

// Wave 表示一次按特定规则聚合的发货波次（UI 层面称为发货任务）。
type Wave struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	WaveNo    string           `gorm:"size:64;not null;uniqueIndex" json:"waveNo"`
	Name      string           `gorm:"size:255;not null" json:"name"`
	Status    string           `gorm:"size:64;not null;default:'draft'" json:"status"`
	LevelTags string           `gorm:"type:text;not null;default:'[]'" json:"levelTags"`
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

func (Member) TableName() string         { return "members" }
func (MemberNickname) TableName() string { return "member_nicknames" }
func (MemberAddress) TableName() string  { return "member_addresses" }
func (Product) TableName() string        { return "products" }
func (Wave) TableName() string           { return "waves" }
func (DispatchRecord) TableName() string { return "dispatch_records" }
func (ProductTag) TableName() string    { return "product_tags" }
func (TemplateConfig) TableName() string { return "template_configs" }
func (ProductImage) TableName() string  { return "product_images" }
