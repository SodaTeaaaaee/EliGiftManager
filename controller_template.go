package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	db "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

type TemplateController struct {
	templateUC app.TemplateManagementUseCase
}

func NewTemplateController() *TemplateController {
	gdb := db.GetDB()
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	return &TemplateController{
		templateUC: app.NewTemplateManagementUseCase(templateRepo, bindingRepo, profileRepo),
	}
}

func (c *TemplateController) CreateDocumentTemplate(input dto.CreateDocumentTemplateInput) (dto.DocumentTemplateDTO, error) {
	ctx := appContext
	result, err := c.templateUC.CreateDocumentTemplate(ctx, input)
	if err != nil {
		return dto.DocumentTemplateDTO{}, err
	}
	return *result, nil
}

func (c *TemplateController) ListDocumentTemplates() ([]dto.DocumentTemplateDTO, error) {
	ctx := appContext
	return c.templateUC.ListDocumentTemplates(ctx)
}

func (c *TemplateController) BindTemplateToProfile(input dto.BindTemplateToProfileInput) (dto.ProfileTemplateBindingDTO, error) {
	ctx := appContext
	result, err := c.templateUC.BindTemplateToProfile(ctx, input)
	if err != nil {
		return dto.ProfileTemplateBindingDTO{}, err
	}
	return *result, nil
}

func (c *TemplateController) ListBindingsByProfile(profileID uint) ([]dto.ProfileTemplateBindingDTO, error) {
	ctx := appContext
	return c.templateUC.ListBindingsByProfile(ctx, profileID)
}

func (c *TemplateController) UpdateDocumentTemplate(input dto.UpdateDocumentTemplateInput) (dto.DocumentTemplateDTO, error) {
	ctx := appContext
	result, err := c.templateUC.UpdateDocumentTemplate(ctx, input)
	if err != nil {
		return dto.DocumentTemplateDTO{}, err
	}
	return *result, nil
}

func (c *TemplateController) DeleteDocumentTemplate(id uint) error {
	ctx := appContext
	return c.templateUC.DeleteDocumentTemplate(ctx, id)
}

func (c *TemplateController) UnbindTemplate(bindingID uint) error {
	ctx := appContext
	return c.templateUC.UnbindTemplate(ctx, bindingID)
}

func (c *TemplateController) SetDefaultBinding(bindingID uint) error {
	ctx := appContext
	return c.templateUC.SetDefaultBinding(ctx, bindingID)
}

// GetDefaultTemplateForProfile returns the default template bound to the given
// profile/docType, or nil if no default binding exists. Returning a typed nil
// pointer (instead of a zero-value DTO) lets the JSON response serialize to a
// real `null`, so the frontend can distinguish "no default template" from "a
// template whose fields all happen to be empty" (plan 5.5).
func (c *TemplateController) GetDefaultTemplateForProfile(profileID uint, docType string) (*dto.DocumentTemplateDTO, error) {
	ctx := appContext
	result, err := c.templateUC.GetDefaultTemplateForProfile(ctx, profileID, docType)
	if err != nil {
		return nil, err
	}
	return result, nil
}
