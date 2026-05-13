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
	EligibilityContextRef  string
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
	ID                       uint
	WaveID                   uint
	CustomerProfileID        uint
	WaveParticipantSnapshotID uint
	ProductID                *uint
	DemandDocumentID         *uint
	DemandLineID             *uint
	CustomerAddressID        *uint
	Quantity                 int
	AllocationState          string
	AddressState             string
	SupplierState            string
	ChannelSyncState         string
	LineReason               string
	ExtraData                string
	CreatedAt                string
	UpdatedAt                string
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
