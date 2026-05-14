package persistence

import (
	"time"

	"gorm.io/gorm"
)

// ---- CustomerProfile ----

type CustomerProfile struct {
	gorm.Model
	DisplayName string      `gorm:"not null"`
	ProfileType ProfileType `gorm:"type:text;not null;default:'member'"`
	ExtraData   string      `gorm:"type:text"` // JSON
}

func (CustomerProfile) TableName() string { return "customer_profiles" }

// ---- CustomerIdentity ----

type CustomerIdentity struct {
	gorm.Model
	CustomerProfileID uint         `gorm:"index;not null"`
	IdentityPlatform  string       `gorm:"not null"`
	IdentityValue     string       `gorm:"not null"`
	IdentityType      IdentityType `gorm:"type:text;not null"`
	IsPrimary         bool         `gorm:"not null;default:false"`
	ExtraData         string       `gorm:"type:text"` // JSON
}

func (CustomerIdentity) TableName() string { return "customer_identities" }

// ---- DemandDocument ----

type DemandDocument struct {
	gorm.Model
	Kind                 DemandKind  `gorm:"type:text;not null"`
	CaptureMode          CaptureMode `gorm:"type:text;not null"`
	SourceChannel        string
	SourceSurface        string
	IntegrationProfileID *uint
	SourceDocumentNo     string
	SourceCustomerRef    string
	CustomerProfileID    *uint `gorm:"index"`
	SourceCreatedAt      *time.Time
	SourcePaidAt         *time.Time
	Currency             string
	AuthoritySnapshotAt  *time.Time
	RawPayload           string `gorm:"type:text"` // JSON
	ExtraData            string `gorm:"type:text"` // JSON
}

func (DemandDocument) TableName() string { return "demand_documents" }

// ---- DemandLine ----

type DemandLine struct {
	gorm.Model
	DemandDocumentID      uint `gorm:"index;not null"`
	SourceLineNo          int
	LineType              DemandLineType        `gorm:"type:text;not null"`
	ObligationTriggerKind ObligationTriggerKind `gorm:"type:text"`
	EntitlementAuthority  EntitlementAuthority  `gorm:"type:text"`
	RecipientInputState   RecipientInputState   `gorm:"type:text"`
	RoutingDisposition    RoutingDisposition    `gorm:"type:text"`
	RoutingReasonCode     string
	EligibilityContextRef string
	ProductMasterID       *uint
	ExternalTitle         string
	RequestedQuantity     int
	EntitlementCode       string
	GiftLevelSnapshot     string `gorm:"type:text"` // JSON
	RecipientInputPayload string `gorm:"type:text"` // JSON
	RawPayload            string `gorm:"type:text"` // JSON
	ExtraData             string `gorm:"type:text"` // JSON
}

func (DemandLine) TableName() string { return "demand_lines" }

// ---- Wave ----

type Wave struct {
	gorm.Model
	WaveNo           string `gorm:"uniqueIndex;not null"`
	Name             string
	WaveType         WaveType `gorm:"type:text;not null;default:'mixed'"`
	LifecycleStage   string
	ProgressSnapshot string `gorm:"type:text"` // JSON
	Notes            string `gorm:"type:text"`
	LevelTags        string `gorm:"type:text"` // JSON
}

func (Wave) TableName() string { return "waves" }

// ---- WaveParticipantSnapshot ----
// Does not use gorm.Model — only CreatedAt, no UpdatedAt/DeletedAt per V2 spec.

type WaveParticipantSnapshot struct {
	ID                 uint         `gorm:"primaryKey;autoIncrement"`
	WaveID             uint         `gorm:"index;not null"`
	CustomerProfileID  uint         `gorm:"index;not null"`
	SnapshotType       SnapshotType `gorm:"type:text;not null;default:'member'"`
	IdentityPlatform   string
	IdentityValue      string
	DisplayName        string
	GiftLevel          string
	SourceDocumentRefs string `gorm:"type:text"` // JSON
	SourceProfileRefs  string `gorm:"type:text"` // JSON
	ExtraData          string `gorm:"type:text"` // JSON
	CreatedAt          time.Time
}

func (WaveParticipantSnapshot) TableName() string { return "wave_participant_snapshots" }

// ---- FulfillmentLine ----

type FulfillmentLine struct {
	gorm.Model
	WaveID                    uint  `gorm:"index;not null"`
	CustomerProfileID         *uint `gorm:"index"`
	WaveParticipantSnapshotID *uint `gorm:"index"`
	ProductID                 *uint `gorm:"index"` // nullable FK
	DemandDocumentID          *uint `gorm:"index"` // nullable FK
	DemandLineID              *uint `gorm:"index"` // nullable FK
	CustomerAddressID         *uint // nullable FK
	Quantity                  int   `gorm:"not null;default:1"`
	AllocationState           string
	AddressState              string
	SupplierState             string
	ChannelSyncState          string
	LineReason                FulfillmentLineReason `gorm:"type:text;not null"`
	GeneratedBy               string
	ExtraData                 string `gorm:"type:text"` // JSON
}

func (FulfillmentLine) TableName() string { return "fulfillment_lines" }

// ---- AllocationPolicyRule ----

type AllocationPolicyRule struct {
	gorm.Model
	WaveID               uint   `gorm:"index;not null"`
	ProductID            uint   `gorm:"index;not null"`
	SelectorPayload      string `gorm:"type:text"` // JSON
	ProductTargetRef     string
	ContributionQuantity int
	RuleKind             string
	Priority             int  `gorm:"not null;default:0"`
	Active               bool `gorm:"not null;default:true"`
}

func (AllocationPolicyRule) TableName() string { return "allocation_policy_rules" }

// ---- SupplierOrder ----

type SupplierOrder struct {
	gorm.Model
	WaveID               uint `gorm:"index;not null"`
	SupplierPlatform     string
	TemplateID           string
	BatchNo              string
	ExternalOrderNo      string
	SubmissionMode       SubmissionMode `gorm:"type:text;not null;default:'csv'"`
	SubmittedAt          *time.Time
	Status               SupplierOrderStatus `gorm:"type:text;not null;default:'draft'"`
	RequestPayload       string              `gorm:"type:text"` // JSON
	ResponsePayload      string              `gorm:"type:text"` // JSON
	BasisHistoryNodeID   string
	BasisProjectionHash  string
	BasisPayloadSnapshot string `gorm:"type:text"` // JSON
	ExtraData            string `gorm:"type:text"` // JSON
}

func (SupplierOrder) TableName() string { return "supplier_orders" }

// ---- SupplierOrderLine ----

type SupplierOrderLine struct {
	gorm.Model
	SupplierOrderID   uint `gorm:"index;not null"`
	FulfillmentLineID uint `gorm:"index;not null"`
	SupplierLineNo    int
	SupplierSKU       string
	SubmittedQuantity int
	AcceptedQuantity  int
	Status            string
	ExtraData         string `gorm:"type:text"` // JSON
}

func (SupplierOrderLine) TableName() string { return "supplier_order_lines" }

// ---- WaveDemandAssignment ----

type WaveDemandAssignment struct {
	gorm.Model
	WaveID           uint `gorm:"uniqueIndex:idx_wave_demand;not null"`
	DemandDocumentID uint `gorm:"uniqueIndex:idx_wave_demand;not null"`
	AcceptedAt       *time.Time
	AcceptedBy       string
	ExtraData        string `gorm:"type:text"`
}

func (WaveDemandAssignment) TableName() string { return "wave_demand_assignments" }

// ---- Shipment ----

type Shipment struct {
	gorm.Model
	SupplierOrderID      uint `gorm:"index;not null"`
	SupplierPlatform     string
	ShipmentNo           string
	ExternalShipmentNo   string
	CarrierCode          string
	CarrierName          string
	TrackingNo           string
	Status               ShipmentStatus `gorm:"type:text;not null;default:'pending'"`
	ShippedAt            *time.Time
	BasisHistoryNodeID   string
	BasisProjectionHash  string
	BasisPayloadSnapshot string `gorm:"type:text"` // JSON
	ExtraData            string `gorm:"type:text"` // JSON
}

func (Shipment) TableName() string { return "shipments" }

// ---- ShipmentLine ----

type ShipmentLine struct {
	ID                  uint `gorm:"primaryKey;autoIncrement"`
	ShipmentID          uint `gorm:"index;not null"`
	SupplierOrderLineID uint
	FulfillmentLineID   uint
	Quantity            int `gorm:"not null;default:0"`
	CreatedAt           time.Time
}

func (ShipmentLine) TableName() string { return "shipment_lines" }

// ---- ChannelSyncJob ----

type ChannelSyncJob struct {
	gorm.Model
	WaveID               uint                `gorm:"index;not null"`
	IntegrationProfileID uint                `gorm:"index"`
	Direction            ChannelSyncDirection `gorm:"type:text;not null;default:'push_tracking'"`
	Status               ChannelSyncJobStatus `gorm:"type:text;not null;default:'pending'"`
	BasisHistoryNodeID   string
	BasisProjectionHash  string
	BasisPayloadSnapshot string              `gorm:"type:text"`
	RequestPayload       string              `gorm:"type:text"`
	ResponsePayload      string              `gorm:"type:text"`
	ErrorMessage         string              `gorm:"type:text"`
	StartedAt            *time.Time
	FinishedAt           *time.Time
}

func (ChannelSyncJob) TableName() string { return "channel_sync_jobs" }

// ---- ChannelSyncItem ----

type ChannelSyncItem struct {
	ID                 uint                 `gorm:"primaryKey;autoIncrement"`
	ChannelSyncJobID   uint                 `gorm:"index;not null"`
	FulfillmentLineID  uint                 `gorm:"index"`
	ShipmentID         uint                 `gorm:"index"`
	ExternalDocumentNo string
	ExternalLineNo     string
	TrackingNo         string
	CarrierCode        string
	Status             ChannelSyncItemStatus `gorm:"type:text;not null;default:'pending'"`
	ErrorMessage       string                `gorm:"type:text"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (ChannelSyncItem) TableName() string { return "channel_sync_items" }
