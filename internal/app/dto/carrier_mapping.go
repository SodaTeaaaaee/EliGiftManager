package dto

// CarrierMappingDTO is returned by carrier mapping queries.
type CarrierMappingDTO struct {
	ID                   uint   `json:"id"`
	IntegrationProfileID uint   `json:"integrationProfileId"`
	InternalCarrierCode  string `json:"internalCarrierCode"`
	ExternalCarrierCode  string `json:"externalCarrierCode"`
	ExternalCarrierName  string `json:"externalCarrierName"`
	IsDefault            bool   `json:"isDefault"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
}

// CreateCarrierMappingInput is the input for creating a carrier mapping.
type CreateCarrierMappingInput struct {
	IntegrationProfileID uint   `json:"integrationProfileId"`
	InternalCarrierCode  string `json:"internalCarrierCode"`
	ExternalCarrierCode  string `json:"externalCarrierCode"`
	ExternalCarrierName  string `json:"externalCarrierName"`
	IsDefault            bool   `json:"isDefault"`
}
