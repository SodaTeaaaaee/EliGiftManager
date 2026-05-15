package main

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
	return &ProfileController{
		uc: app.NewProfileManagementUseCase(profileRepo, demandRepo, channelSyncRepo, templateBindingRepo, closureDecisionRepo),
	}
}

// CreateProfile creates a new integration profile.
func (c *ProfileController) CreateProfile(input dto.CreateProfileInput) (*dto.IntegrationProfileDTO, error) {
	return c.uc.CreateProfile(input)
}

// UpdateProfile updates an existing integration profile.
func (c *ProfileController) UpdateProfile(input dto.UpdateProfileInput) (*dto.IntegrationProfileDTO, error) {
	return c.uc.UpdateProfile(input)
}

// DeleteProfile deletes an integration profile by ID.
func (c *ProfileController) DeleteProfile(id uint) error {
	return c.uc.DeleteProfile(id)
}

// GetProfile returns a single integration profile by ID.
func (c *ProfileController) GetProfile(id uint) (*dto.IntegrationProfileDTO, error) {
	return c.uc.GetProfile(id)
}

// ListProfiles returns all integration profiles.
func (c *ProfileController) ListProfiles() ([]dto.IntegrationProfileDTO, error) {
	return c.uc.ListProfiles()
}

// SeedDefaultProfiles creates default profiles if they don't already exist.
func (c *ProfileController) SeedDefaultProfiles() ([]dto.IntegrationProfileDTO, error) {
	return c.uc.SeedDefaultProfiles()
}
