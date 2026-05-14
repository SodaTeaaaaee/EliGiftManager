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
