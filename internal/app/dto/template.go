package dto

import "time"

type CreateDocumentTemplateInput struct {
	TemplateKey  string `json:"templateKey"`
	DocumentType string `json:"documentType"`
	Format       string `json:"format"`
	MappingRules string `json:"mappingRules"`
	ExtraData    string `json:"extraData"`
}

// UpdateDocumentTemplateInput updates mutable template fields only.
// TemplateKey and DocumentType are intentionally omitted — they are immutable.
type UpdateDocumentTemplateInput struct {
	ID           uint   `json:"id"`
	Format       string `json:"format"`
	MappingRules string `json:"mappingRules"`
	ExtraData    string `json:"extraData"`
}

type DocumentTemplateDTO struct {
	ID           uint      `json:"id"`
	TemplateKey  string    `json:"templateKey"`
	DocumentType string    `json:"documentType"`
	Format       string    `json:"format"`
	MappingRules string    `json:"mappingRules"`
	ExtraData    string    `json:"extraData"`
	CreatedAt    time.Time `json:"createdAt" ts_type:"string"`
	UpdatedAt    time.Time `json:"updatedAt" ts_type:"string"`
}

// BindTemplateToProfileInput binds a template as a profile default for one
// documentType. Uniqueness is per (profile, documentType); one source platform
// may have defaults for both demand import types.
type BindTemplateToProfileInput struct {
	IntegrationProfileID uint   `json:"integrationProfileId"`
	DocumentType         string `json:"documentType"`
	TemplateID           uint   `json:"templateId"`
	IsDefault            bool   `json:"isDefault"`
}

type ProfileTemplateBindingDTO struct {
	ID                   uint      `json:"id"`
	IntegrationProfileID uint      `json:"integrationProfileId"`
	DocumentType         string    `json:"documentType"`
	TemplateID           uint      `json:"templateId"`
	IsDefault            bool      `json:"isDefault"`
	CreatedAt            time.Time `json:"createdAt" ts_type:"string"`
}
