package dto

import "time"

// CarrierMappingDTO is returned by carrier mapping queries.
type CarrierMappingDTO struct {
	ID                   uint      `json:"id"`
	IntegrationProfileID uint      `json:"integrationProfileId"`
	InternalCarrierCode  string    `json:"internalCarrierCode"`
	ExternalCarrierCode  string    `json:"externalCarrierCode"`
	ExternalCarrierName  string    `json:"externalCarrierName"`
	IsDefault            bool      `json:"isDefault"`
	CreatedAt            time.Time `json:"createdAt" ts_type:"string"`
	UpdatedAt            time.Time `json:"updatedAt" ts_type:"string"`
}

// CreateCarrierMappingInput is the input for creating a carrier mapping.
type CreateCarrierMappingInput struct {
	IntegrationProfileID uint   `json:"integrationProfileId"`
	InternalCarrierCode  string `json:"internalCarrierCode"`
	ExternalCarrierCode  string `json:"externalCarrierCode"`
	ExternalCarrierName  string `json:"externalCarrierName"`
	IsDefault            bool   `json:"isDefault"`
}
