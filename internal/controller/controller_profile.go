package controller

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// ProfileController exposes IntegrationProfile management Wails bindings.
type ProfileController struct {
	uc app.ProfileManagementUseCase
}

func NewProfileController() *ProfileController {
	gdb := database.GetDB()
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	demandRepo := infra.NewDemandRepository(gdb)
	channelSyncRepo := infra.NewChannelSyncRepository(gdb)
	templateBindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	closureDecisionRepo := infra.NewClosureDecisionRepository(gdb)
	executorProvider := buildExecutorProvider()
	uc := app.NewProfileManagementUseCase(profileRepo, demandRepo, channelSyncRepo, templateBindingRepo, closureDecisionRepo, executorProvider)
	uc = app.WithIntegrationProfileReferenceRepo(uc, infra.NewIntegrationProfileReferenceRepository(gdb))
	return &ProfileController{uc: uc}
}

// CreateProfile creates a new integration profile.
func (c *ProfileController) CreateProfile(input dto.CreateProfileInput) (*dto.IntegrationProfileDTO, error) {
	ctx := appContext
	return c.uc.CreateProfile(ctx, input)
}

// UpdateProfile updates an existing integration profile.
func (c *ProfileController) UpdateProfile(input dto.UpdateProfileInput) (*dto.IntegrationProfileDTO, error) {
	ctx := appContext
	return c.uc.UpdateProfile(ctx, input)
}

// DeleteProfile deletes an integration profile by ID.
func (c *ProfileController) DeleteProfile(id uint) error {
	ctx := appContext
	return c.uc.DeleteProfile(ctx, id)
}

// GetProfile returns a single integration profile by ID.
func (c *ProfileController) GetProfile(id uint) (*dto.IntegrationProfileDTO, error) {
	ctx := appContext
	return c.uc.GetProfile(ctx, id)
}

// ListProfiles returns all integration profiles.
func (c *ProfileController) ListProfiles() ([]dto.IntegrationProfileDTO, error) {
	ctx := appContext
	return c.uc.ListProfiles(ctx)
}

// SeedDefaultProfiles creates default profiles if they don't already exist,
// then ensures the catalog demo profile/template/binding (idempotent).
func (c *ProfileController) SeedDefaultProfiles() ([]dto.IntegrationProfileDTO, error) {
	ctx := appContext
	profiles, err := c.uc.SeedDefaultProfiles(ctx)
	if err != nil {
		return nil, err
	}

	// Catalog demo seed (factory_rouzao_demo + catalog_rouzao_zip_demo binding).
	// Independent of membership/retail defaults so it can also be called alone later.
	gdb := database.GetDB()
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	if _, err := app.SeedCatalogDemo(ctx, profileRepo, templateRepo, bindingRepo); err != nil {
		return nil, err
	}
	// Bilibili demo seed: export_source_tracking_update + import_carrier_mapping
	// templates bound to bilibili_membership_demo (SampleData header-locked).
	if _, err := app.SeedBilibiliDemo(ctx, profileRepo, templateRepo, bindingRepo); err != nil {
		return nil, err
	}
	return profiles, nil
}
