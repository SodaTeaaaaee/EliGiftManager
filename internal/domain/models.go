package domain

// ---- CustomerProfile ----

type CustomerProfile struct {
	ID          uint
	DisplayName string
	ProfileType string
	ExtraData   string
	CreatedAt   string
	UpdatedAt   string
}

// ---- CustomerIdentity ----

type CustomerIdentity struct {
	ID                uint
	CustomerProfileID uint
	IdentityPlatform  string
	IdentityValue     string
	IdentityType      string
	IsPrimary         bool
	ExtraData         string
	CreatedAt         string
	UpdatedAt         string
}

// ---- DemandDocument ----

type DemandDocument struct {
	ID                   uint
	Kind                 string
	CaptureMode          string
	SourceChannel        string
	SourceSurface        string
	IntegrationProfileID *uint
	SourceDocumentNo     string
	SourceCustomerRef    string
	CustomerProfileID    *uint
	SourceCreatedAt      string
	SourcePaidAt         string
	Currency             string
	AuthoritySnapshotAt  string
	RawPayload           string
	ExtraData            string
	CreatedAt            string
	UpdatedAt            string
}

// ---- DemandLine ----

type DemandLine struct {
	ID                    uint
	DemandDocumentID      uint
	SourceLineNo          int
	LineType              string
	ObligationTriggerKind string
	EntitlementAuthority  string
	RecipientInputState   string
	RoutingDisposition    string
	RoutingReasonCode     string
	EligibilityContextRef string
	ProductMasterID       *uint
	ExternalTitle         string
	RequestedQuantity     int
	EntitlementCode       string
	GiftLevelSnapshot     string
	RecipientInputPayload string
	RawPayload            string
	ExtraData             string
	CreatedAt             string
	UpdatedAt             string
}

// ---- Wave ----

type Wave struct {
	ID               uint
	WaveNo           string
	Name             string
	WaveType         string
	LifecycleStage   string
	ProgressSnapshot string
	Notes            string
	LevelTags        string
	CreatedAt        string
	UpdatedAt        string
}

// ---- WaveParticipantSnapshot ----
// Does not have UpdatedAt per V2 spec.

type WaveParticipantSnapshot struct {
	ID                 uint
	WaveID             uint
	CustomerProfileID  uint
	SnapshotType       string
	IdentityPlatform   string
	IdentityValue      string
	DisplayName        string
	GiftLevel          string
	SourceDocumentRefs string
	SourceProfileRefs  string
	ExtraData          string
	CreatedAt          string
}

// ---- FulfillmentLine ----

type FulfillmentLine struct {
	ID                        uint
	WaveID                    uint
	CustomerProfileID         *uint
	WaveParticipantSnapshotID *uint
	ProductID                 *uint
	DemandDocumentID          *uint
	DemandLineID              *uint
	CustomerAddressID         *uint
	Quantity                  int
	AllocationState           string
	AddressState              string
	SupplierState             string
	ChannelSyncState          string
	LineReason                string
	GeneratedBy               string
	ExtraData                 string
	CreatedAt                 string
	UpdatedAt                 string
}

// ---- AllocationPolicyRule ----

type AllocationPolicyRule struct {
	ID                   uint
	WaveID               uint
	ProductID            uint
	SelectorPayload      string
	ProductTargetRef     string
	ContributionQuantity int
	RuleKind             string
	Priority             int
	Active               bool
	CreatedAt            string
	UpdatedAt            string
}

// ---- SupplierOrder ----

type SupplierOrder struct {
	ID                   uint
	WaveID               uint
	SupplierPlatform     string
	TemplateID           string
	BatchNo              string
	ExternalOrderNo      string
	SubmissionMode       string
	SubmittedAt          string
	Status               string
	RequestPayload       string
	ResponsePayload      string
	BasisHistoryNodeID   string
	BasisProjectionHash  string
	BasisPayloadSnapshot string
	ExtraData            string
	CreatedAt            string
	UpdatedAt            string
}

// ---- SupplierOrderLine ----

type SupplierOrderLine struct {
	ID                uint
	SupplierOrderID   uint
	FulfillmentLineID uint
	SupplierLineNo    int
	SupplierSKU       string
	SubmittedQuantity int
	AcceptedQuantity  int
	Status            string
	ExtraData         string
	CreatedAt         string
	UpdatedAt         string
}

// ---- WaveDemandAssignment ----

type WaveDemandAssignment struct {
	ID               uint
	WaveID           uint
	DemandDocumentID uint
	AcceptedAt       string
	AcceptedBy       string
	ExtraData        string
	CreatedAt        string
	UpdatedAt        string
}

// ---- Shipment ----

type Shipment struct {
	ID                   uint
	SupplierOrderID      uint
	SupplierPlatform     string
	ShipmentNo           string
	ExternalShipmentNo   string
	CarrierCode          string
	CarrierName          string
	TrackingNo           string
	Status               string
	ShippedAt            string
	BasisHistoryNodeID   string
	BasisProjectionHash  string
	BasisPayloadSnapshot string
	ExtraData            string
	CreatedAt            string
	UpdatedAt            string
}

// ---- ShipmentLine ----

type ShipmentLine struct {
	ID                  uint
	ShipmentID          uint
	SupplierOrderLineID uint
	FulfillmentLineID   uint
	Quantity            int
	CreatedAt           string
}

// ---- ChannelSyncJob ----

type ChannelSyncJob struct {
	ID                   uint
	WaveID               uint
	IntegrationProfileID uint
	Direction            string
	Status               string
	BasisHistoryNodeID   string
	BasisProjectionHash  string
	BasisPayloadSnapshot string
	RequestPayload       string
	ResponsePayload      string
	ErrorMessage         string
	StartedAt            string
	FinishedAt           string
	CreatedAt            string
	UpdatedAt            string
}

// ---- ChannelSyncItem ----

type ChannelSyncItem struct {
	ID                 uint
	ChannelSyncJobID   uint
	FulfillmentLineID  uint
	ShipmentID         uint
	ExternalDocumentNo string
	ExternalLineNo     string
	TrackingNo         string
	CarrierCode        string
	Status             string
	ErrorMessage       string
	CreatedAt          string
	UpdatedAt          string
}

// ---- IntegrationProfile ----

type IntegrationProfile struct {
	ID                        uint
	ProfileKey                string
	SourceChannel             string
	SourceSurface             string
	DemandKind                string
	InitialAllocationStrategy string
	IdentityStrategy          string
	EntitlementAuthorityMode  string
	RecipientInputMode        string
	ReferenceStrategy         string
	TrackingSyncMode          string
	ClosurePolicy             string
	SupportsPartialShipment   bool
	SupportsAPIImport         bool
	SupportsAPIExport         bool
	RequiresCarrierMapping    bool
	RequiresExternalOrderNo   bool
	AllowsManualClosure       bool
	ConnectorKey              string
	SupportedLocales          string
	DefaultLocale             string
	ExtraData                 string
	CreatedAt                 string
	UpdatedAt                 string
}

// ---- ChannelClosureDecisionRecord ----

type ChannelClosureDecisionRecord struct {
	ID                   uint
	WaveID               uint
	IntegrationProfileID uint
	FulfillmentLineID    uint
	DecisionKind         string
	ReasonCode           string
	Note                 string
	EvidenceRef          string
	OperatorID           string
	CreatedAt            string
	UpdatedAt            string
}

// ---- FulfillmentAdjustment ----

type FulfillmentAdjustment struct {
	ID                uint
	WaveID            uint
	FulfillmentLineID uint
	AdjustmentKind    string
	QuantityDelta     int
	ReasonCode        string
	OperatorID        string
	Note              string
	EvidenceRef       string
	CreatedAt         string
	UpdatedAt         string
}

// ---- DocumentTemplate ----

type DocumentTemplate struct {
	ID           uint
	TemplateKey  string
	DocumentType string
	Format       string
	MappingRules string
	ExtraData    string
	CreatedAt    string
	UpdatedAt    string
}

// ---- IntegrationProfileTemplateBinding ----

type IntegrationProfileTemplateBinding struct {
	ID                   uint
	IntegrationProfileID uint
	DocumentType         string
	TemplateID           uint
	IsDefault            bool
	CreatedAt            string
}

// ---- HistoryScope ----

type HistoryScope struct {
	ID                uint
	ScopeType         string
	ScopeKey          string
	CurrentHeadNodeID uint
	CreatedAt         string
	UpdatedAt         string
}

// ---- HistoryNode ----

type HistoryNode struct {
	ID                   uint
	HistoryScopeID       uint
	ParentNodeID         uint
	PreferredRedoChildID uint
	CommandKind          string
	CommandSummary       string
	PatchPayload         string
	InversePatchPayload  string
	CheckpointHint       bool
	ProjectionHash       string
	CreatedBy            string
	CreatedAt            string
}

// ---- HistoryCheckpoint ----

type HistoryCheckpoint struct {
	ID              uint
	HistoryScopeID  uint
	HistoryNodeID   uint
	SnapshotPayload string
	SchemaVersion   string
	CreatedAt       string
}

// ---- HistoryPin ----

type HistoryPin struct {
	ID            uint
	HistoryNodeID uint
	PinKind       string
	RefType       string
	RefID         uint
	CreatedAt     string
}
