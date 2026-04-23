package model

import "time"

// Member 表示统一标准库中的会员主数据，仅保存稳定的外部平台标识。
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

// MemberNickname 表示会员昵称的历史记录。
type MemberNickname struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MemberID  uint      `gorm:"not null;index" json:"memberId"`
	Nickname  string    `gorm:"size:255;not null" json:"nickname"`
	CreatedAt time.Time `json:"createdAt"`
	Member    Member    `gorm:"foreignKey:MemberID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"member"`
}

// MemberAddress 表示会员地址的历史记录，由人工维护并支持软删除标记。
type MemberAddress struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	MemberID      uint      `gorm:"not null;index" json:"memberId"`
	RecipientName string    `gorm:"size:255;not null" json:"recipientName"`
	Phone         string    `gorm:"size:64;not null" json:"phone"`
	Address       string    `gorm:"type:text;not null" json:"address"`
	IsDeleted     bool      `gorm:"not null;default:false;index" json:"isDeleted"`
	CreatedAt     time.Time `json:"createdAt"`
	Member        Member    `gorm:"foreignKey:MemberID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"member"`
}

// Product 表示统一标准库中的工厂商品主数据。
type Product struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Factory    string    `gorm:"size:100;not null;uniqueIndex:idx_products_factory_sku" json:"factory"`
	FactorySKU string    `gorm:"size:255;not null;uniqueIndex:idx_products_factory_sku" json:"factorySku"`
	Name       string    `gorm:"size:255;not null" json:"name"`
	ImagePath  string    `gorm:"type:text" json:"imagePath"`
	ExtraData  string    `gorm:"type:text;not null;default:'{}'" json:"extraData"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// DispatchRecord 表示批次发货过程中会员与商品的分发记录。
type DispatchRecord struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	BatchName       string         `gorm:"size:255;not null;index" json:"batchName"`
	MemberID        uint           `gorm:"not null;index" json:"memberId"`
	ProductID       uint           `gorm:"not null;index" json:"productId"`
	MemberAddressID *uint          `gorm:"index" json:"memberAddressId"`
	Quantity        int            `gorm:"not null;default:1" json:"quantity"`
	Status          string         `gorm:"size:64;not null;index" json:"status"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	Member          Member         `gorm:"foreignKey:MemberID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"member"`
	Product         Product        `gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"product"`
	MemberAddress   *MemberAddress `gorm:"foreignKey:MemberAddressID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"memberAddress"`
}

// TemplateConfig 表示导入导出模板的动态映射配置。
type TemplateConfig struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Type         string    `gorm:"size:100;not null;index" json:"type"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	MappingRules string    `gorm:"type:text;not null;default:'{}'" json:"mappingRules"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// TableName 返回会员标准表名。
func (Member) TableName() string {
	return "members"
}

// TableName 返回会员昵称历史表名。
func (MemberNickname) TableName() string {
	return "member_nicknames"
}

// TableName 返回会员地址历史表名。
func (MemberAddress) TableName() string {
	return "member_addresses"
}

// TableName 返回商品标准表名。
func (Product) TableName() string {
	return "products"
}

// TableName 返回分发记录标准表名。
func (DispatchRecord) TableName() string {
	return "dispatch_records"
}

// TableName 返回模板配置表名。
func (TemplateConfig) TableName() string {
	return "template_configs"
}
