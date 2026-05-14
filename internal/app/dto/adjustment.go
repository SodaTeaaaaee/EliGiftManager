package dto

type RecordAdjustmentInput struct {
	WaveID            uint   `json:"waveId"`
	FulfillmentLineID uint   `json:"fulfillmentLineId"`
	AdjustmentKind    string `json:"adjustmentKind"`
	QuantityDelta     int    `json:"quantityDelta"`
	ReasonCode        string `json:"reasonCode"`
	OperatorID        string `json:"operatorId"`
	Note              string `json:"note"`
	EvidenceRef       string `json:"evidenceRef"`
}

type FulfillmentAdjustmentDTO struct {
	ID                uint   `json:"id"`
	WaveID            uint   `json:"waveId"`
	FulfillmentLineID uint   `json:"fulfillmentLineId"`
	AdjustmentKind    string `json:"adjustmentKind"`
	QuantityDelta     int    `json:"quantityDelta"`
	ReasonCode        string `json:"reasonCode"`
	OperatorID        string `json:"operatorId"`
	Note              string `json:"note"`
	EvidenceRef       string `json:"evidenceRef"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}
