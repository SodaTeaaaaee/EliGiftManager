package dto

type CreateChannelSyncJobInput struct {
	WaveID               uint                     `json:"waveId"`
	IntegrationProfileID uint                     `json:"integrationProfileId"`
	Direction            string                   `json:"direction"`
	Items                []CreateChannelSyncItemInput `json:"items"`
}

type CreateChannelSyncItemInput struct {
	FulfillmentLineID  uint   `json:"fulfillmentLineId"`
	ShipmentID         uint   `json:"shipmentId"`
	ExternalDocumentNo string `json:"externalDocumentNo"`
	ExternalLineNo     string `json:"externalLineNo"`
	TrackingNo         string `json:"trackingNo"`
	CarrierCode        string `json:"carrierCode"`
}

type ChannelSyncJobDTO struct {
	ID                   uint                  `json:"id"`
	WaveID               uint                  `json:"waveId"`
	IntegrationProfileID uint                  `json:"integrationProfileId"`
	Direction            string                `json:"direction"`
	Status               string                `json:"status"`
	BasisHistoryNodeID   string                `json:"basisHistoryNodeId"`
	BasisProjectionHash  string                `json:"basisProjectionHash"`
	BasisPayloadSnapshot string                `json:"basisPayloadSnapshot"`
	RequestPayload       string                `json:"requestPayload"`
	ResponsePayload      string                `json:"responsePayload"`
	ErrorMessage         string                `json:"errorMessage"`
	StartedAt            string                `json:"startedAt"`
	FinishedAt           string                `json:"finishedAt"`
	CreatedAt            string                `json:"createdAt"`
	UpdatedAt            string                `json:"updatedAt"`
	Items                []ChannelSyncItemDTO  `json:"items"`
}

// ClosureDecision enumerates the outcomes of PlanChannelClosure.
type ClosureDecision string

const (
	ClosureDecisionCreateJob      ClosureDecision = "create_job"
	ClosureDecisionManualClosure  ClosureDecision = "manual_closure"
	ClosureDecisionUnsupported    ClosureDecision = "unsupported"
)

// PlanChannelClosureInput is the input for the orchestration use case.
type PlanChannelClosureInput struct {
	WaveID               uint `json:"waveId"`
	IntegrationProfileID uint `json:"integrationProfileId"`
}

// PlanChannelClosureResult carries the high-level closure plan outcome.
type PlanChannelClosureResult struct {
	Decision           ClosureDecision       `json:"decision"`
	IntegrationProfileID uint                 `json:"integrationProfileId"`
	TrackingSyncMode   string                `json:"trackingSyncMode"`
	ClosurePolicy      string                `json:"closurePolicy"`
	Job                *ChannelSyncJobDTO    `json:"job,omitempty"`
	Items              []ChannelSyncItemDTO  `json:"items,omitempty"`
}

type ChannelSyncItemDTO struct {
	ID                 uint   `json:"id"`
	ChannelSyncJobID   uint   `json:"channelSyncJobId"`
	FulfillmentLineID  uint   `json:"fulfillmentLineId"`
	ShipmentID         uint   `json:"shipmentId"`
	ExternalDocumentNo string `json:"externalDocumentNo"`
	ExternalLineNo     string `json:"externalLineNo"`
	TrackingNo         string `json:"trackingNo"`
	CarrierCode        string `json:"carrierCode"`
	Status             string `json:"status"`
	ErrorMessage       string `json:"errorMessage"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}
