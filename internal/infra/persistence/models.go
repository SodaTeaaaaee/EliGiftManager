package persistence

import "time"

type Platform struct {
	ID        uint   `gorm:"primaryKey"`
	Key       string `gorm:"uniqueIndex;not null"`
	Name      string `gorm:"not null"`
	Kind      string `gorm:"not null"`
	Notes     string
	ExtraData string `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Platform) TableName() string { return "platforms" }

type CustomerProfile struct {
	ID          uint   `gorm:"primaryKey"`
	DisplayName string `gorm:"not null"`
	Notes       string
	ExtraData   string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (CustomerProfile) TableName() string { return "customer_profiles" }

type PlatformIdentity struct {
	ID                uint   `gorm:"primaryKey"`
	CustomerProfileID *uint  `gorm:"index"`
	PlatformID        uint   `gorm:"uniqueIndex:idx_identity_unique,priority:1;not null"`
	IdentityType      string `gorm:"uniqueIndex:idx_identity_unique,priority:2;not null"`
	IdentityValue     string `gorm:"not null"`
	NormalizedValue   string `gorm:"uniqueIndex:idx_identity_unique,priority:3;not null"`
	ExtraData         string `gorm:"type:text"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (PlatformIdentity) TableName() string { return "platform_identities" }

type RecipientAddress struct {
	ID                uint `gorm:"primaryKey"`
	CustomerProfileID uint `gorm:"index;not null"`
	Label             string
	RecipientName     string
	Phone             string
	Country           string
	Province          string
	City              string
	District          string
	AddressLine1      string
	AddressLine2      string
	PostalCode        string
	IsDefault         bool   `gorm:"not null;default:false"`
	ExtraData         string `gorm:"type:text"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (RecipientAddress) TableName() string { return "recipient_addresses" }

type ProductItem struct {
	ID                uint   `gorm:"primaryKey"`
	Name              string `gorm:"not null"`
	FactoryPlatformID uint   `gorm:"uniqueIndex:idx_product_factory_sku,priority:1;not null"`
	FactorySKU        string `gorm:"uniqueIndex:idx_product_factory_sku,priority:2;not null"`
	Notes             string
	ExtraData         string `gorm:"type:text"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (ProductItem) TableName() string { return "product_items" }

type ProductAlias struct {
	ID                uint   `gorm:"primaryKey"`
	ProductItemID     uint   `gorm:"index;not null"`
	PlatformID        uint   `gorm:"uniqueIndex:idx_alias_platform_ext,priority:1;not null"`
	ExternalProductID string `gorm:"uniqueIndex:idx_alias_platform_ext,priority:2;not null"`
	Title             string
	Spec              string
	ExtraData         string `gorm:"type:text"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (ProductAlias) TableName() string { return "product_aliases" }

type ProductBundleComponent struct {
	ID            uint `gorm:"primaryKey"`
	AliasID       uint `gorm:"index;not null"`
	ProductItemID uint `gorm:"not null"`
	Quantity      int  `gorm:"not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (ProductBundleComponent) TableName() string { return "product_bundle_components" }

type QuantitySplitRule struct {
	ID          uint   `gorm:"primaryKey"`
	WaveID      uint   `gorm:"uniqueIndex:idx_qsplit_wave_platform_key,priority:1;not null"`
	PlatformID  uint   `gorm:"uniqueIndex:idx_qsplit_wave_platform_key,priority:2;not null"`
	ExternalKey string `gorm:"uniqueIndex:idx_qsplit_wave_platform_key,priority:3;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (QuantitySplitRule) TableName() string { return "quantity_split_rules" }

type QuantitySplitComponent struct {
	ID            uint `gorm:"primaryKey"`
	RuleID        uint `gorm:"index;not null"`
	ProductItemID uint `gorm:"not null"`
	Quantity      int  `gorm:"not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (QuantitySplitComponent) TableName() string { return "quantity_split_components" }

type TemplateConfig struct {
	ID           uint   `gorm:"primaryKey"`
	PlatformID   uint   `gorm:"index;not null"`
	DocumentType string `gorm:"not null"`
	Direction    string `gorm:"not null"`
	Name         string `gorm:"not null"`
	Version      int    `gorm:"not null;default:1"`
	Builtin      bool   `gorm:"not null;default:false"`
	MappingJSON  string `gorm:"type:text"`
	LayoutJSON   string `gorm:"type:text"`
	Notes        string
	ExtraData    string `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (TemplateConfig) TableName() string { return "template_configs" }

type CarrierMapping struct {
	ID           uint   `gorm:"primaryKey"`
	PlatformID   uint   `gorm:"index;not null"`
	ExternalCode string `gorm:"not null"`
	InternalCode string `gorm:"not null"`
	InternalName string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (CarrierMapping) TableName() string { return "carrier_mappings" }

type InputDocument struct {
	ID              uint `gorm:"primaryKey"`
	PlatformID      uint `gorm:"index;not null"`
	DocumentType    string
	Direction       string
	OriginalName    string
	RawPayload      string `gorm:"type:text"`
	TemplateID      *uint
	TemplateVersion int
	ImportedAt      time.Time
	ExtraData       string `gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (InputDocument) TableName() string { return "input_documents" }

type InputFact struct {
	ID                 uint   `gorm:"primaryKey"`
	DocumentID         *uint  `gorm:"index"`
	PlatformID         uint   `gorm:"index;not null"`
	Kind               string `gorm:"not null"`
	StableExternalID   string `gorm:"index"`
	CustomerProfileID  *uint  `gorm:"index"`
	PlatformIdentityID *uint  `gorm:"index"`
	MembershipLevel    string
	SourceDocumentNo   string
	SourceCreatedAt    *time.Time
	RevisesID          *uint      `gorm:"index"`
	RevisionAppliedAt  *time.Time `gorm:"index"`
	ExtraData          string     `gorm:"type:text"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (InputFact) TableName() string { return "input_facts" }

type InputFactLine struct {
	ID            uint `gorm:"primaryKey"`
	FactID        uint `gorm:"index;not null"`
	SourceLineNo  int  `gorm:"not null"`
	ExternalSKU   string
	ExternalTitle string
	ExternalSpec  string
	ProductItemID *uint
	Quantity      int    `gorm:"not null"`
	WaveID        *uint  `gorm:"index"`
	ExtraData     string `gorm:"type:text"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (InputFactLine) TableName() string { return "input_fact_lines" }

type DuplicateObservation struct {
	ID             uint   `gorm:"primaryKey"`
	DocumentID     uint   `gorm:"index;not null"`
	ExistingFactID uint   `gorm:"index;not null"`
	Verdict        string `gorm:"not null"`
	Reason         string
	Decided        bool   `gorm:"not null;default:false"`
	ExtraData      string `gorm:"type:text"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (DuplicateObservation) TableName() string { return "duplicate_observations" }

type Wave struct {
	ID          uint   `gorm:"primaryKey"`
	WaveNo      string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"not null"`
	Notes       string
	CloseResult string `gorm:"not null;default:'open'"`
	CloseNote   string
	ClosedAt    *time.Time
	ReopenedAt  *time.Time
	ExtraData   string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Wave) TableName() string { return "waves" }

type EntitlementRule struct {
	ID           uint   `gorm:"primaryKey"`
	WaveID       uint   `gorm:"index;not null"`
	ProductID    uint   `gorm:"index;not null"`
	SelectorJSON string `gorm:"type:text;not null"`
	Quantity     int    `gorm:"not null"`
	Active       bool   `gorm:"not null;default:true"`
	ExtraData    string `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (EntitlementRule) TableName() string { return "entitlement_rules" }

type EntitlementException struct {
	ID         uint `gorm:"primaryKey"`
	WaveID     uint `gorm:"index;not null"`
	ProductID  uint `gorm:"not null"`
	InstanceID uint `gorm:"index;not null"`
	Quantity   int  `gorm:"not null"`
	Note       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (EntitlementException) TableName() string { return "entitlement_exceptions" }

type EntitlementInstance struct {
	ID                 uint `gorm:"primaryKey"`
	WaveID             uint `gorm:"uniqueIndex:idx_instance_wave_line,priority:1;not null"`
	InputFactLineID    uint `gorm:"uniqueIndex:idx_instance_wave_line,priority:2;not null"`
	CustomerProfileID  *uint
	PlatformIdentityID *uint
	MembershipLevel    string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (EntitlementInstance) TableName() string { return "entitlement_instances" }

type FulfillmentResult struct {
	ID                    uint   `gorm:"primaryKey"`
	WaveID                uint   `gorm:"index;not null"`
	SourceKind            string `gorm:"not null"`
	EntitlementInstanceID *uint  `gorm:"index"`
	InputFactLineID       *uint  `gorm:"index"`
	InputFactID           *uint  `gorm:"index"`
	CustomerProfileID     *uint  `gorm:"index"`
	ProductItemID         *uint
	Quantity              int
	AddressJSON           string `gorm:"type:text"`
	Frozen                bool   `gorm:"not null;default:false;index"`
	AddressPinned         bool   `gorm:"not null;default:false"`
	ExtraData             string `gorm:"type:text"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (FulfillmentResult) TableName() string { return "fulfillment_results" }

type ExecutionQuantityLink struct {
	ID                  uint `gorm:"primaryKey"`
	FulfillmentResultID uint `gorm:"index;not null"`
	SupplierOrderLineID uint `gorm:"index;not null"`
	Quantity            int  `gorm:"not null"`
	ConfigVersion       int  `gorm:"not null;default:0"`
	CreatedAt           time.Time
}

func (ExecutionQuantityLink) TableName() string { return "execution_quantity_links" }

type SupplierOrder struct {
	ID                uint   `gorm:"primaryKey"`
	WaveID            uint   `gorm:"index;not null"`
	FactoryPlatformID uint   `gorm:"index;not null"`
	Status            string `gorm:"not null;default:'generated'"`
	TemplateID        uint   `gorm:"not null;default:0"`
	TemplateVersion   int    `gorm:"not null;default:0"`
	ExportedAt        *time.Time
	VoidedAt          *time.Time
	ExportPayload     string `gorm:"type:text"`
	ExtraData         string `gorm:"type:text"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (SupplierOrder) TableName() string { return "supplier_orders" }

type SupplierOrderLine struct {
	ID              uint `gorm:"primaryKey"`
	SupplierOrderID uint `gorm:"index;not null"`
	ProductItemID   uint `gorm:"not null"`
	FactorySKU      string
	Quantity        int    `gorm:"not null"`
	TrackingID      string `gorm:"uniqueIndex"`
	TrackingRetired bool   `gorm:"not null;default:false"`
	ExtraData       string `gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (SupplierOrderLine) TableName() string { return "supplier_order_lines" }

type RetiredTrackingID struct {
	ID         uint      `gorm:"primaryKey"`
	TrackingID string    `gorm:"uniqueIndex;not null"`
	WaveID     uint      `gorm:"index;not null"`
	RetiredAt  time.Time `gorm:"not null"`
}

func (RetiredTrackingID) TableName() string { return "retired_tracking_ids" }

type Shipment struct {
	ID          uint   `gorm:"primaryKey"`
	TrackingID  string `gorm:"index;not null"`
	CarrierCode string
	CarrierName string
	TrackingNo  string
	ShippedAt   *time.Time
	Quantity    int
	ExtraData   string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Shipment) TableName() string { return "shipments" }

type ChannelWritebackItem struct {
	ID              uint `gorm:"primaryKey"`
	InputFactID     uint `gorm:"index;not null"`
	ShipmentID      uint `gorm:"index;not null"`
	TrackingNo      string
	CarrierCode     string
	Status          string `gorm:"not null;default:'pending'"`
	TemplateID      uint   `gorm:"not null;default:0"`
	TemplateVersion int    `gorm:"not null;default:0"`
	ErrorMessage    string
	Payload         string `gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (ChannelWritebackItem) TableName() string { return "channel_writeback_items" }

type AppSettings struct {
	ID                     uint `gorm:"primaryKey"`
	Locale                 string
	Theme                  string
	Density                string
	DuplicateRecordMinutes int `gorm:"not null;default:10"`
	DuplicateAskDays       int `gorm:"not null;default:10"`
	UpdatedAt              time.Time
}

func (AppSettings) TableName() string { return "app_settings" }

func AllModels() []any {
	return []any{
		&Platform{},
		&CustomerProfile{},
		&PlatformIdentity{},
		&RecipientAddress{},
		&ProductItem{},
		&ProductAlias{},
		&ProductBundleComponent{},
		&QuantitySplitRule{},
		&QuantitySplitComponent{},
		&TemplateConfig{},
		&CarrierMapping{},
		&InputDocument{},
		&InputFact{},
		&InputFactLine{},
		&DuplicateObservation{},
		&Wave{},
		&EntitlementRule{},
		&EntitlementException{},
		&EntitlementInstance{},
		&FulfillmentResult{},
		&ExecutionQuantityLink{},
		&SupplierOrder{},
		&SupplierOrderLine{},
		&RetiredTrackingID{},
		&Shipment{},
		&ChannelWritebackItem{},
		&AppSettings{},
	}
}
