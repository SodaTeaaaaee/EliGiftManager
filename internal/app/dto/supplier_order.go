package dto

import "time"

type SupplierOrderDTO struct {
	ID                          uint       `json:"id"`
	WaveID                      uint       `json:"waveId"`
	FactoryIntegrationProfileID *uint      `json:"factoryIntegrationProfileId,omitempty"`
	SupplierPlatform            string     `json:"supplierPlatform"`
	TemplateID                  string     `json:"templateId"`
	BatchNo                     string     `json:"batchNo"`
	ExternalOrderNo             string     `json:"externalOrderNo"`
	SubmissionMode              string     `json:"submissionMode"`
	SubmittedAt                 *time.Time `json:"submittedAt" ts_type:"string"`
	Status                      string     `json:"status"`
	RequestPayload              string     `json:"requestPayload"`
	ResponsePayload             string     `json:"responsePayload"`
	BasisHistoryNodeID          string     `json:"basisHistoryNodeId"`
	BasisProjectionHash         string     `json:"basisProjectionHash"`
	BasisPayloadSnapshot        string     `json:"basisPayloadSnapshot"`
	ExtraData                   string     `json:"extraData"`
	CreatedAt                   time.Time  `json:"createdAt" ts_type:"string"`
	UpdatedAt                   time.Time  `json:"updatedAt" ts_type:"string"`
}

type SupplierOrderLineDTO struct {
	ID                uint      `json:"id"`
	SupplierOrderID   uint      `json:"supplierOrderId"`
	FulfillmentLineID uint      `json:"fulfillmentLineId"`
	SupplierLineNo    *int      `json:"supplierLineNo"`
	SupplierSKU       string    `json:"supplierSku"`
	SubmittedQuantity int       `json:"submittedQuantity"`
	AcceptedQuantity  *int      `json:"acceptedQuantity"`
	Status            string    `json:"status"`
	ExtraData         string    `json:"extraData"`
	CreatedAt         time.Time `json:"createdAt" ts_type:"string"`
	UpdatedAt         time.Time `json:"updatedAt" ts_type:"string"`
}
