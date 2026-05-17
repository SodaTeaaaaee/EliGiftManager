package dto

type WaveStepStateDTO struct {
	StepKey        string `json:"stepKey"`
	Status         string `json:"status"`
	PrimaryCount   int    `json:"primaryCount"`
	SecondaryCount int    `json:"secondaryCount"`
}

type WaveWorkspaceGuidanceDTO struct {
	Code          string `json:"code"`
	Severity      string `json:"severity"`
	TargetStepKey string `json:"targetStepKey"`
	Count         int    `json:"count"`
}

type WaveWorkspaceBasisSummaryDTO struct {
	DriftedCount        int  `json:"driftedCount"`
	RequiredReviewCount int  `json:"requiredReviewCount"`
	HasDriftedBasis     bool `json:"hasDriftedBasis"`
	HasRequiredReview   bool `json:"hasRequiredReview"`
}

type WaveWorkspaceSnapshotDTO struct {
	Wave                      WaveDTO                      `json:"wave"`
	ProjectedLifecycleStage   string                       `json:"projectedLifecycleStage"`
	Overview                  WaveOverviewDTO              `json:"overview"`
	StepStates                []WaveStepStateDTO           `json:"stepStates"`
	Guidance                  []WaveWorkspaceGuidanceDTO   `json:"guidance"`
	BasisSummary              WaveWorkspaceBasisSummaryDTO `json:"basisSummary"`
	HistoryHeadNodeID         uint                         `json:"historyHeadNodeId"`
	HistoryHeadProjectionHash string                       `json:"historyHeadProjectionHash"`
	RecentHistory             []HistoryNodeDTO             `json:"recentHistory"`
}

type WaveFulfillmentRowDTO struct {
	FulfillmentLineID         uint   `json:"fulfillmentLineId"`
	WaveID                    uint   `json:"waveId"`
	WaveParticipantSnapshotID *uint  `json:"waveParticipantSnapshotId"`
	CustomerProfileID         *uint  `json:"customerProfileId"`
	ParticipantType           string `json:"participantType"`
	ParticipantDisplay        string `json:"participantDisplay"`
	ParticipantBadge          string `json:"participantBadge"`
	ProductID                 *uint  `json:"productId"`
	ProductDisplay            string `json:"productDisplay"`
	DemandDocumentID          *uint  `json:"demandDocumentId"`
	DemandLineID              *uint  `json:"demandLineId"`
	DemandKind                string `json:"demandKind"`
	DemandSourceSummary       string `json:"demandSourceSummary"`
	Quantity                  int    `json:"quantity"`
	AllocationState           string `json:"allocationState"`
	AddressState              string `json:"addressState"`
	SupplierState             string `json:"supplierState"`
	ChannelSyncState          string `json:"channelSyncState"`
	LineReason                string `json:"lineReason"`
	GeneratedBy               string `json:"generatedBy"`
	BasisDriftStatus          string `json:"basisDriftStatus"`
	ReviewRequirement         string `json:"reviewRequirement"`
	ReviewReasonSummary       string `json:"reviewReasonSummary"`
}

type WaveParticipantRowDTO struct {
	WaveParticipantSnapshotID uint   `json:"waveParticipantSnapshotId"`
	WaveID                    uint   `json:"waveId"`
	CustomerProfileID         uint   `json:"customerProfileId"`
	SnapshotType              string `json:"snapshotType"`
	DisplayName               string `json:"displayName"`
	IdentityPlatform          string `json:"identityPlatform"`
	IdentityValue             string `json:"identityValue"`
	GiftLevel                 string `json:"giftLevel"`
	SourceSummary             string `json:"sourceSummary"`
	FulfillmentLineCount      int    `json:"fulfillmentLineCount"`
	ReadyFulfillmentCount     int    `json:"readyFulfillmentCount"`
}

type ListDemandInboxInput struct {
	Assignment string `json:"assignment"`
	DemandKind string `json:"demandKind"`
}

type DemandInboxRowDTO struct {
	DemandDocumentID        uint   `json:"demandDocumentId"`
	Kind                    string `json:"kind"`
	CaptureMode             string `json:"captureMode"`
	SourceChannel           string `json:"sourceChannel"`
	SourceSurface           string `json:"sourceSurface"`
	SourceDocumentNo        string `json:"sourceDocumentNo"`
	CustomerProfileID       *uint  `json:"customerProfileId"`
	IntegrationProfileID    *uint  `json:"integrationProfileId"`
	IntegrationProfileLabel string `json:"integrationProfileLabel"`
	Assigned                bool   `json:"assigned"`
	AssignedWaveID          *uint  `json:"assignedWaveId"`
	AssignedWaveLabel       string `json:"assignedWaveLabel"`
	TotalLineCount          int    `json:"totalLineCount"`
	AcceptedCount           int    `json:"acceptedCount"`
	ReadyAcceptedCount      int    `json:"readyAcceptedCount"`
	WaitingInputCount       int    `json:"waitingInputCount"`
	DeferredCount           int    `json:"deferredCount"`
	ExcludedCount           int    `json:"excludedCount"`
	CreatedAt               string `json:"createdAt"`
}
