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
	result, err := c.templateUC.CreateDocumentTemplate(input)
	if err != nil {
		return dto.DocumentTemplateDTO{}, err
	}
	return *result, nil
}

func (c *TemplateController) ListDocumentTemplates() ([]dto.DocumentTemplateDTO, error) {
	return c.templateUC.ListDocumentTemplates()
}

func (c *TemplateController) BindTemplateToProfile(input dto.BindTemplateToProfileInput) (dto.ProfileTemplateBindingDTO, error) {
	result, err := c.templateUC.BindTemplateToProfile(input)
	if err != nil {
		return dto.ProfileTemplateBindingDTO{}, err
	}
	return *result, nil
}

func (c *TemplateController) ListBindingsByProfile(profileID uint) ([]dto.ProfileTemplateBindingDTO, error) {
	return c.templateUC.ListBindingsByProfile(profileID)
}

func (c *TemplateController) GetDefaultTemplateForProfile(profileID uint, docType string) (dto.DocumentTemplateDTO, error) {
	result, err := c.templateUC.GetDefaultTemplateForProfile(profileID, docType)
	if err != nil {
		return dto.DocumentTemplateDTO{}, err
	}
	if result == nil {
		return dto.DocumentTemplateDTO{}, nil
	}
	return *result, nil
}
