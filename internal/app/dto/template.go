package dto

type CreateDocumentTemplateInput struct {
	TemplateKey  string `json:"templateKey"`
	DocumentType string `json:"documentType"`
	Format       string `json:"format"`
	MappingRules string `json:"mappingRules"`
	ExtraData    string `json:"extraData"`
}

type DocumentTemplateDTO struct {
	ID           uint   `json:"id"`
	TemplateKey  string `json:"templateKey"`
	DocumentType string `json:"documentType"`
	Format       string `json:"format"`
	MappingRules string `json:"mappingRules"`
	ExtraData    string `json:"extraData"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type BindTemplateToProfileInput struct {
	IntegrationProfileID uint   `json:"integrationProfileId"`
	DocumentType         string `json:"documentType"`
	TemplateID           uint   `json:"templateId"`
	IsDefault            bool   `json:"isDefault"`
}

type ProfileTemplateBindingDTO struct {
	ID                   uint   `json:"id"`
	IntegrationProfileID uint   `json:"integrationProfileId"`
	DocumentType         string `json:"documentType"`
	TemplateID           uint   `json:"templateId"`
	IsDefault            bool   `json:"isDefault"`
	CreatedAt            string `json:"createdAt"`
}
