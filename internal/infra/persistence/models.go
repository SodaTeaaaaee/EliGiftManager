package persistence

import (
	"time"

	"gorm.io/gorm"
)

// Soft-delete design: all models that embed gorm.Model inherit gorm.DeletedAt,
// which GORM uses to automatically exclude soft-deleted rows from queries (WHERE
// deleted_at IS NULL). No explicit scope, callback, or plugin is needed.
// Intentional hard deletes (e.g. patch executor restore, cleanup by wave) use
// db.Unscoped().Delete() with comments explaining why the row must be fully absent.
// The few models that do NOT embed gorm.Model (ShipmentLine, ChannelSyncItem) have
// no DeletedAt and are never soft-deleted — they are append-only join/link rows.

// ---- CustomerProfile ----

type CustomerProfile struct {
	gorm.Model
	DisplayName              string      `gorm:"not null"`
	ProfileType              ProfileType `gorm:"type:text;not null;default:'member'"`
	Status                   string      `gorm:"type:text;not null;default:'active';index"`
	MergedIntoProfileID      *uint       `gorm:"index"`
	RowVersion               uint64      `gorm:"not null;default:1"`
	DisplayNameMode          string      `gorm:"type:text;not null;default:'auto';index"`
	DisplayNameObservationID *uint       `gorm:"index"`
	ExtraData                string      `gorm:"type:text"` // JSON
}

func (CustomerProfile) TableName() string { return "customer_profiles" }

// ---- CustomerMergeRecord ----

type CustomerMergeRecord struct {
	ID                     uint   `gorm:"primaryKey;autoIncrement"`
	SourceProfileID        uint   `gorm:"not null;index"`
	TargetProfileID        uint   `gorm:"not null;index"`
	MergeCandidateID       *uint  `gorm:"index"`
	MergePolicyRevisionID  *uint  `gorm:"index"`
	MergeMode              string `gorm:"type:text;not null;default:'manual'"`
	DecisionSource         string `gorm:"type:text;not null;default:''"`
	DecisionReason         string `gorm:"type:text;not null;default:''"`
	ActorRef               string `gorm:"type:text;not null;default:''"`
	CorrelationID          string `gorm:"type:text;not null;default:'';index"`
	SourceRowVersion       uint64 `gorm:"not null;default:0"`
	TargetRowVersion       uint64 `gorm:"not null;default:0"`
	EvidenceSnapshot       string `gorm:"type:text;not null;default:''"`
	Payload                string `gorm:"type:text;not null"`
	RowVersion             uint64 `gorm:"not null;default:1"`
	OperationKey           string `gorm:"type:text;not null;default:''"`
	CommandHash            string `gorm:"type:text;not null;default:''"`
	PreviewHash            string `gorm:"type:text;not null;default:''"`
	MovePlanHash           string `gorm:"type:text;not null;default:''"`
	Status                 string `gorm:"type:text;not null;default:'completed'"`
	DependsOnMergeRecordID *uint
	SourceRowVersionAfter  uint64 `gorm:"not null;default:0"`
	TargetRowVersionAfter  uint64 `gorm:"not null;default:0"`
	SourceProfileSnapshot  string `gorm:"type:text;not null;default:''"`
	TargetProfileSnapshot  string `gorm:"type:text;not null;default:''"`
	CompletedAt            *time.Time
	UndoOperationKey       string `gorm:"type:text;not null;default:''"`
	LastUndoPlanHash       string `gorm:"type:text;not null;default:''"`
	LastUndoCheckedAt      *time.Time
	UndoneBy               string     `gorm:"type:text;not null;default:''"`
	UndoReason             string     `gorm:"type:text;not null;default:''"`
	UndoneSourceRowVersion uint64     `gorm:"not null;default:0"`
	UndoneTargetRowVersion uint64     `gorm:"not null;default:0"`
	CreatedAt              time.Time  `gorm:"not null"`
	UndoneAt               *time.Time `gorm:"index"`
}

func (CustomerMergeRecord) TableName() string { return "customer_merge_records" }

// ---- CustomerIdentity ----

type CustomerIdentity struct {
	gorm.Model
	CustomerProfileID          uint         `gorm:"index;not null"`
	IdentityPlatform           string       `gorm:"not null;index:idx_identity_platform_value,priority:1"`
	IdentityValue              string       `gorm:"not null;index:idx_identity_platform_value,priority:2"`
	IdentityType               IdentityType `gorm:"type:text;not null"`
	Namespace                  string       `gorm:"type:text;not null;default:'';index"`
	NormalizedValue            string       `gorm:"type:text;not null;default:'';index"`
	NormalizationVersion       string       `gorm:"type:text;not null;default:''"`
	Authority                  string       `gorm:"type:text;not null;default:'';index"`
	VerificationStatus         string       `gorm:"type:text;not null;default:'unverified';index"`
	SourceIntegrationProfileID *uint        `gorm:"index"`
	ResolutionStatus           string       `gorm:"type:text;not null;default:'unresolved';index"`
	FirstSeenAt                *time.Time
	LastSeenAt                 *time.Time
	IsPrimary                  bool   `gorm:"not null;default:false"`
	ExtraData                  string `gorm:"type:text"` // JSON
}

func (CustomerIdentity) TableName() string { return "customer_identities" }

// ---- CustomerAddress ----

type CustomerAddress struct {
	gorm.Model
	CustomerProfileID    uint `gorm:"index;not null"`
	Label                string
	RecipientName        string
	Phone                string
	NormalizedPhone      string `gorm:"type:text;not null;default:'';index"`
	AddressFingerprint   string `gorm:"type:text;not null;default:'';index"`
	NormalizationVersion string `gorm:"type:text;not null;default:''"`
	QualityStatus        string `gorm:"type:text;not null;default:'unknown';index"`
	Country              string
	Province             string
	City                 string
	District             string
	AddressLine1         string
	AddressLine2         string
	PostalCode           string
	IsDefault            bool                    `gorm:"not null;default:false"`
	IsTest               bool                    `gorm:"not null;default:false"`
	ValidationStatus     AddressValidationStatus `gorm:"type:text;not null;default:'unvalidated'"`
	ValidationDetail     string                  `gorm:"type:text"` // JSON
	ExtraData            string                  `gorm:"type:text"` // JSON
}

func (CustomerAddress) TableName() string { return "customer_addresses" }

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
	BoundProfileSnapshot string `gorm:"type:text"` // JSON snapshot of execution-relevant profile fields at wave assignment time
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
	WaveID                      uint  `gorm:"index;not null"`
	FactoryIntegrationProfileID *uint `gorm:"index"`
	SupplierPlatform            string
	TemplateID                  string
	BatchNo                     string
	ExternalOrderNo             string
	SubmissionMode              SubmissionMode `gorm:"type:text;not null;default:'csv'"`
	SubmittedAt                 *time.Time
	Status                      SupplierOrderStatus `gorm:"type:text;not null;default:'draft'"`
	RequestPayload              string              `gorm:"type:text"` // JSON
	ResponsePayload             string              `gorm:"type:text"` // JSON
	BasisHistoryNodeID          string
	BasisProjectionHash         string
	BasisPayloadSnapshot        string `gorm:"type:text"` // JSON
	ExtraData                   string `gorm:"type:text"` // JSON
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
	SupplierOrderLineID uint `gorm:"index"`
	FulfillmentLineID   uint
	Quantity            int `gorm:"not null;default:0"`
	CreatedAt           time.Time
}

func (ShipmentLine) TableName() string { return "shipment_lines" }

// ---- ChannelSyncJob ----

type ChannelSyncJob struct {
	gorm.Model
	WaveID               uint                 `gorm:"index;not null"`
	IntegrationProfileID uint                 `gorm:"index"`
	Direction            ChannelSyncDirection `gorm:"type:text;not null;default:'push_tracking'"`
	Status               ChannelSyncJobStatus `gorm:"type:text;not null;default:'pending'"`
	BasisHistoryNodeID   string
	BasisProjectionHash  string
	BasisPayloadSnapshot string `gorm:"type:text"`
	RequestPayload       string `gorm:"type:text"`
	ResponsePayload      string `gorm:"type:text"`
	ErrorMessage         string `gorm:"type:text"`
	StartedAt            *time.Time
	FinishedAt           *time.Time
}

func (ChannelSyncJob) TableName() string { return "channel_sync_jobs" }

// ---- ChannelSyncItem ----

type ChannelSyncItem struct {
	ID                 uint `gorm:"primaryKey;autoIncrement"`
	ChannelSyncJobID   uint `gorm:"index;not null"`
	FulfillmentLineID  uint `gorm:"index"`
	ShipmentID         uint `gorm:"index"`
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

// ---- IntegrationProfile ----

type IntegrationProfile struct {
	gorm.Model
	ProfileKey                     string `gorm:"uniqueIndex;not null"`
	SourceChannel                  string
	SourceSurface                  string
	DemandKind                     ProfileDemandKind         `gorm:"type:text;not null;default:'membership_entitlement'"`
	InitialAllocationStrategy      InitialAllocationStrategy `gorm:"type:text;not null;default:'policy_driven'"`
	IdentityStrategy               IdentityStrategy          `gorm:"type:text;not null;default:'platform_uid'"`
	EntitlementAuthorityMode       EntitlementAuthorityMode  `gorm:"type:text;not null;default:'local_policy'"`
	RecipientInputMode             RecipientInputMode        `gorm:"type:text;not null;default:'none'"`
	ReferenceStrategy              ReferenceStrategy         `gorm:"type:text;not null;default:'member_level'"`
	TrackingSyncMode               TrackingSyncMode          `gorm:"type:text;not null;default:'manual_confirmation'"`
	ClosurePolicy                  ClosurePolicy             `gorm:"type:text;not null;default:'close_after_sync'"`
	SupportsPartialShipment        bool                      `gorm:"not null;default:false"`
	SupportsAPIImport              bool                      `gorm:"not null;default:false"`
	SupportsAPIExport              bool                      `gorm:"not null;default:false"`
	RequiresCarrierMapping         bool                      `gorm:"not null;default:false"`
	RequiresExternalOrderNo        bool                      `gorm:"not null;default:false"`
	AllowsManualClosure            bool                      `gorm:"not null;default:false"`
	SupportsExportSupplierOrder    bool                      `gorm:"not null;default:false"`
	SupportsImportProductCatalog   bool                      `gorm:"not null;default:false"`
	SupportsImportSupplierShipment bool                      `gorm:"not null;default:false"`
	ConnectorKey                   string
	FactorySupplierPlatform        string
	SupportedLocales               string `gorm:"type:text"`
	DefaultLocale                  string
	ExtraData                      string `gorm:"type:text"`
}

func (IntegrationProfile) TableName() string { return "integration_profiles" }

// ---- ChannelClosureDecisionRecord ----

type ChannelClosureDecisionRecord struct {
	gorm.Model
	WaveID               uint                       `gorm:"index;not null"`
	IntegrationProfileID uint                       `gorm:"index;not null"`
	FulfillmentLineID    uint                       `gorm:"index;not null"`
	DecisionKind         ChannelClosureDecisionKind `gorm:"type:text;not null"`
	ReasonCode           string
	Note                 string `gorm:"type:text"`
	EvidenceRef          string
	OperatorID           string
}

func (ChannelClosureDecisionRecord) TableName() string { return "channel_closure_decision_records" }

// ---- FulfillmentAdjustment ----

type FulfillmentAdjustment struct {
	gorm.Model
	WaveID                    uint   `gorm:"not null;index"`
	TargetKind                string `gorm:"not null;default:'fulfillment_line'"`
	FulfillmentLineID         *uint  `gorm:"index"`
	WaveParticipantSnapshotID *uint  `gorm:"index"`
	AdjustmentKind            string `gorm:"not null"`
	QuantityDelta             int    `gorm:"not null;default:0"`
	FromProductID             *uint  `gorm:"index"`
	ToProductID               *uint  `gorm:"index"`
	ReasonCode                string
	OperatorID                string `gorm:"not null"`
	Note                      string
	EvidenceRef               string
}

func (FulfillmentAdjustment) TableName() string { return "fulfillment_adjustments" }

// ---- DocumentTemplate ----

type DocumentTemplate struct {
	gorm.Model
	TemplateKey  string `gorm:"not null;uniqueIndex"`
	DocumentType string `gorm:"not null;index"`
	Format       string `gorm:"not null"`
	MappingRules string `gorm:"type:text"`
	ExtraData    string `gorm:"type:text"`
}

func (DocumentTemplate) TableName() string { return "document_templates" }

// ---- IntegrationProfileTemplateBinding ----

type IntegrationProfileTemplateBinding struct {
	gorm.Model
	IntegrationProfileID uint   `gorm:"not null;index:idx_binding_profile_type"`
	DocumentType         string `gorm:"not null;index:idx_binding_profile_type"`
	TemplateID           uint   `gorm:"not null;index"`
	IsDefault            bool   `gorm:"not null;default:false"`
}

func (IntegrationProfileTemplateBinding) TableName() string {
	return "integration_profile_template_bindings"
}

// ---- HistoryScope ----

type HistoryScope struct {
	gorm.Model
	ScopeType         string `gorm:"not null;uniqueIndex:idx_history_scope_type_key"`
	ScopeKey          string `gorm:"not null;uniqueIndex:idx_history_scope_type_key"`
	CurrentHeadNodeID uint   `gorm:"default:0"`
}

func (HistoryScope) TableName() string { return "history_scopes" }

// ---- HistoryNode ----

type HistoryNode struct {
	gorm.Model
	HistoryScopeID       uint   `gorm:"not null;index"`
	ParentNodeID         uint   `gorm:"index"`
	PreferredRedoChildID uint   `gorm:"default:0"`
	CommandKind          string `gorm:"not null"`
	CommandSummary       string
	PatchPayload         string `gorm:"type:text"`
	InversePatchPayload  string `gorm:"type:text"`
	CheckpointHint       bool   `gorm:"not null;default:false"`
	ProjectionHash       string
	CreatedBy            string
}

func (HistoryNode) TableName() string { return "history_nodes" }

// ---- HistoryCheckpoint ----

type HistoryCheckpoint struct {
	gorm.Model
	HistoryScopeID  uint   `gorm:"not null;index"`
	HistoryNodeID   uint   `gorm:"not null;uniqueIndex"`
	SnapshotPayload string `gorm:"type:text"`
	SchemaVersion   string `gorm:"not null"`
}

func (HistoryCheckpoint) TableName() string { return "history_checkpoints" }

// ---- HistoryPin ----

type HistoryPin struct {
	gorm.Model
	HistoryNodeID uint   `gorm:"not null;index"`
	PinKind       string `gorm:"not null"`
	RefType       string `gorm:"not null"`
	RefID         uint   `gorm:"not null"`
}

func (HistoryPin) TableName() string { return "history_pins" }

// ---- ProductMaster ----

type ProductMaster struct {
	gorm.Model
	SupplierPlatform   string `gorm:"not null;uniqueIndex:idx_pm_platform_sku"`
	FactorySKU         string `gorm:"not null;uniqueIndex:idx_pm_platform_sku"`
	SupplierProductRef string
	Name               string `gorm:"not null"`
	ProductKind        string `gorm:"not null;default:other"`
	Archived           bool   `gorm:"not null;default:false"`
	CoverImagePath     string `gorm:"type:text"`
	DetailImagePaths   string `gorm:"type:text"` // JSON []string
	ExtraData          string `gorm:"type:text"`
}

func (ProductMaster) TableName() string { return "product_masters" }

// ---- Product ----

type Product struct {
	gorm.Model
	WaveID           uint   `gorm:"not null;index;uniqueIndex:idx_product_wave_platform_sku"`
	ProductMasterID  *uint  `gorm:"index"`
	SupplierPlatform string `gorm:"not null;uniqueIndex:idx_product_wave_platform_sku"`
	FactorySKU       string `gorm:"not null;uniqueIndex:idx_product_wave_platform_sku"`
	Name             string `gorm:"not null"`
	ExtraData        string `gorm:"type:text"`
}

func (Product) TableName() string { return "products" }

// ---- CarrierMapping ----

type CarrierMapping struct {
	gorm.Model
	IntegrationProfileID uint   `gorm:"not null;index"`
	InternalCarrierCode  string `gorm:"not null"`
	ExternalCarrierCode  string `gorm:"not null"`
	ExternalCarrierName  string
	Aliases              string `gorm:"type:text"` // JSON []string
	IsDefault            bool   `gorm:"not null;default:false"`
}

func (CarrierMapping) TableName() string { return "carrier_mappings" }

// ---- MergeSuggestion ----

type MergeSuggestion struct {
	gorm.Model
	SourceProfileID uint   `gorm:"not null"`
	TargetProfileID uint   `gorm:"not null"`
	Reason          string `gorm:"not null"`
	Status          string `gorm:"not null;default:'pending'"` // pending, dismissed, merged
}

func (MergeSuggestion) TableName() string { return "merge_suggestions" }
