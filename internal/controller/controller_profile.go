package controller

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// ProfileController exposes IntegrationProfile management Wails bindings.
type ProfileController struct {
	uc app.ProfileManagementUseCase
	// seedCatalogDemo / seedBilibiliDemo override production demo seeds in tests.
	seedCatalogDemo  func(ctx context.Context) error
	seedBilibiliDemo func(ctx context.Context) error
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
// then ensures the catalog and bilibili demo seeds (idempotent).
// A later demo-seed failure is returned; callers never see a successful
// result when catalog or bilibili seeding failed.
func (c *ProfileController) SeedDefaultProfiles() ([]dto.IntegrationProfileDTO, error) {
	ctx := appContext
	profiles, err := c.uc.SeedDefaultProfiles(ctx)
	if err != nil {
		return nil, err
	}

	seedCatalog, seedBilibili, err := c.catalogAndBilibiliSeeders()
	if err != nil {
		return nil, err
	}
	if err := seedCatalog(ctx); err != nil {
		return nil, err
	}
	if err := seedBilibili(ctx); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (c *ProfileController) catalogAndBilibiliSeeders() (catalog, bilibili func(context.Context) error, err error) {
	catalog, bilibili = c.seedCatalogDemo, c.seedBilibiliDemo
	if catalog != nil && bilibili != nil {
		return catalog, bilibili, nil
	}
	gdb := database.GetDB()
	if gdb == nil {
		return nil, nil, fmt.Errorf("seed default profiles: database is not initialized")
	}
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	if catalog == nil {
		catalog = func(ctx context.Context) error {
			_, seedErr := app.SeedCatalogDemo(ctx, profileRepo, templateRepo, bindingRepo)
			return seedErr
		}
	}
	if bilibili == nil {
		bilibili = func(ctx context.Context) error {
			_, seedErr := app.SeedBilibiliDemo(ctx, profileRepo, templateRepo, bindingRepo)
			return seedErr
		}
	}
	return catalog, bilibili, nil
}

// SeedBuiltinPlatform installs one named builtin platform (keys: "bilibili",
// "rouzao"). Idempotent if that platform is already installed. Does not seed
// membership_default or any other platform.
func (c *ProfileController) SeedBuiltinPlatform(key string) (*dto.IntegrationProfileDTO, error) {
	ctx := appContext
	gdb := database.GetDB()
	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	return app.SeedBuiltinPlatform(ctx, key, profileRepo, templateRepo, bindingRepo)
}
