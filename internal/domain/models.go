package domain

import "time"

type Platform struct {
	ID        uint
	Key       string
	Name      string
	Kind      string
	Notes     string
	ExtraData string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CustomerProfile struct {
	ID          uint
	DisplayName string
	Notes       string
	ExtraData   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PlatformIdentity struct {
	ID                uint
	CustomerProfileID *uint
	PlatformID        uint
	IdentityType      string
	IdentityValue     string
	NormalizedValue   string
	ExtraData         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type RecipientAddress struct {
	ID                uint
	CustomerProfileID uint
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
	IsDefault         bool
	ExtraData         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AddressSnapshot struct {
	SourceAddressID *uint  `json:"source_address_id,omitempty"`
	RecipientName   string `json:"recipient_name"`
	Phone           string `json:"phone"`
	Country         string `json:"country"`
	Province        string `json:"province"`
	City            string `json:"city"`
	District        string `json:"district"`
	AddressLine1    string `json:"address_line1"`
	AddressLine2    string `json:"address_line2"`
	PostalCode      string `json:"postal_code"`
}

func (a AddressSnapshot) Usable() bool {
	return a.RecipientName != "" && a.AddressLine1 != ""
}

type ProductItem struct {
	ID                uint
	Name              string
	FactoryPlatformID uint
	FactorySKU        string
	Notes             string
	ExtraData         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ProductAlias struct {
	ID                uint
	ProductItemID     uint
	PlatformID        uint
	ExternalProductID string
	Title             string
	Spec              string
	ExtraData         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ProductBundleComponent struct {
	ID            uint
	AliasID       uint
	ProductItemID uint
	Quantity      int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type TemplateConfig struct {
	ID           uint
	PlatformID   uint
	DocumentType string
	Direction    string
	Name         string
	Version      int
	Builtin      bool
	MappingJSON  string
	LayoutJSON   string
	Notes        string
	ExtraData    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CarrierMapping struct {
	ID           uint
	PlatformID   uint
	ExternalCode string
	InternalCode string
	InternalName string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type InputDocument struct {
	ID              uint
	PlatformID      uint
	DocumentType    string
	Direction       string
	OriginalName    string
	RawPayload      string
	TemplateID      *uint
	TemplateVersion int
	ImportedAt      time.Time
	ExtraData       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type InputFact struct {
	ID                 uint
	DocumentID         *uint
	PlatformID         uint
	Kind               string
	StableExternalID   string
	CustomerProfileID  *uint
	PlatformIdentityID *uint
	MembershipLevel    string
	SourceDocumentNo   string
	SourceCreatedAt    *time.Time
	ExtraData          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type InputFactLine struct {
	ID            uint
	FactID        uint
	SourceLineNo  int
	ExternalSKU   string
	ExternalTitle string
	ExternalSpec  string
	ProductItemID *uint
	Quantity      int
	WaveID        *uint
	ExtraData     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DuplicateObservation struct {
	ID             uint
	DocumentID     uint
	ExistingFactID uint
	Verdict        string
	Reason         string
	Decided        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Wave struct {
	ID          uint
	WaveNo      string
	Name        string
	Notes       string
	CloseResult string
	CloseNote   string
	ClosedAt    *time.Time
	ReopenedAt  *time.Time
	ExtraData   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type EntitlementSelector struct {
	Type       string `json:"type"`
	PlatformID uint   `json:"platform_id,omitempty"`
	Level      string `json:"level,omitempty"`
	InstanceID *uint  `json:"instance_id,omitempty"`
}

type EntitlementRule struct {
	ID        uint
	WaveID    uint
	ProductID uint
	Selector  EntitlementSelector
	Quantity  int
	Active    bool
	ExtraData string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EntitlementException struct {
	ID         uint
	WaveID     uint
	ProductID  uint
	InstanceID uint
	Quantity   int
	Note       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type EntitlementInstance struct {
	ID                 uint
	WaveID             uint
	InputFactLineID    uint
	CustomerProfileID  *uint
	PlatformIdentityID *uint
	MembershipLevel    string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type FulfillmentResult struct {
	ID                    uint
	WaveID                uint
	SourceKind            string
	EntitlementInstanceID *uint
	InputFactLineID       *uint
	InputFactID           *uint
	CustomerProfileID     *uint
	ProductItemID         *uint
	Quantity              int
	Address               AddressSnapshot
	Frozen                bool
	AddressPinned         bool
	ExtraData             string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type ExecutionQuantityLink struct {
	ID                  uint
	FulfillmentResultID uint
	SupplierOrderLineID uint
	Quantity            int
	CreatedAt           time.Time
}

type SupplierOrder struct {
	ID                uint
	WaveID            uint
	FactoryPlatformID uint
	Status            string
	ExportedAt        *time.Time
	VoidedAt          *time.Time
	ExportPayload     string
	ExtraData         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type SupplierOrderLine struct {
	ID              uint
	SupplierOrderID uint
	ProductItemID   uint
	FactorySKU      string
	Quantity        int
	TrackingID      string
	TrackingRetired bool
	ExtraData       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type RetiredTrackingID struct {
	ID         uint
	TrackingID string
	WaveID     uint
	RetiredAt  time.Time
}

type Shipment struct {
	ID          uint
	TrackingID  string
	CarrierCode string
	CarrierName string
	TrackingNo  string
	ShippedAt   *time.Time
	Quantity    int
	ExtraData   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ChannelWritebackItem struct {
	ID           uint
	InputFactID  uint
	ShipmentID   uint
	TrackingNo   string
	CarrierCode  string
	Status       string
	ErrorMessage string
	Payload      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AppSettings struct {
	ID                     uint
	Locale                 string
	Theme                  string
	Density                string
	DuplicateRecordMinutes int
	DuplicateAskDays       int
	UpdatedAt              time.Time
}

var SemanticDictionary = []string{
	"customer.display_name",
	"identity.platform",
	"identity.value",
	"identity.type",
	"membership.level",
	"source.document_no",
	"source.line_no",
	"source.created_at",
	"product.alias_id",
	"product.alias_title",
	"product.alias_spec",
	"product.factory_sku",
	"product.name",
	"quantity",
	"recipient.name",
	"recipient.phone",
	"recipient.country",
	"recipient.province",
	"recipient.city",
	"recipient.district",
	"recipient.address_line1",
	"recipient.address_line2",
	"recipient.postal_code",
	"tracking.id",
	"shipment.tracking_no",
	"shipment.carrier_code",
	"shipment.carrier_name",
	"shipment.shipped_at",
	"shipment.quantity",
}

var NamedTransformers = []string{
	"trim",
	"strip_quotes",
	"parseDate",
	"mapEnum",
	"normalizePhone",
	"splitSkuQuantity",
	"joinAddress",
}
