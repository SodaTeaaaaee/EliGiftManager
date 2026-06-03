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

func (c *TemplateController) GetDefaultTemplateForProfile(profileID uint, docType string) (dto.DocumentTemplateDTO, error) {
	ctx := appContext
	result, err := c.templateUC.GetDefaultTemplateForProfile(ctx, profileID, docType)
	if err != nil {
		return dto.DocumentTemplateDTO{}, err
	}
	if result == nil {
		return dto.DocumentTemplateDTO{}, nil
	}
	return *result, nil
}
