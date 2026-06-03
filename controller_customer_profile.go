package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/service"
)

type CustomerProfileController struct {
	uc *app.CustomerProfileUseCase
}

func NewCustomerProfileController() *CustomerProfileController {
	gdb := database.GetDB()
	profileRepo := infra.NewProfileRepository(gdb)
	addressRepo := infra.NewAddressRepository(gdb)
	suggestionRepo := infra.NewMergeSuggestionRepository(gdb)
	settingsSvc := service.NewSettingsService()
	return &CustomerProfileController{
		uc: app.NewCustomerProfileUseCase(profileRepo, addressRepo, settingsSvc, suggestionRepo),
	}
}

func (c *CustomerProfileController) ListCustomerProfiles(keyword, platform string, missingAddressOnly bool) ([]dto.CustomerProfileDTO, error) {
	ctx := appContext
	return c.uc.ListCustomerProfiles(ctx, keyword, platform, missingAddressOnly)
}

func (c *CustomerProfileController) GetCustomerProfile(id uint) (*dto.CustomerProfileDTO, error) {
	ctx := appContext
	return c.uc.GetCustomerProfile(ctx, id)
}

func (c *CustomerProfileController) CreateCustomerProfile(input dto.CreateCustomerProfileInput) (*dto.CustomerProfileDTO, error) {
	ctx := appContext
	return c.uc.CreateCustomerProfile(ctx, input)
}

func (c *CustomerProfileController) UpdateCustomerProfile(input dto.UpdateCustomerProfileInput) (*dto.CustomerProfileDTO, error) {
	ctx := appContext
	return c.uc.UpdateCustomerProfile(ctx, input)
}

func (c *CustomerProfileController) DeleteCustomerProfile(id uint) error {
	ctx := appContext
	return c.uc.DeleteCustomerProfile(ctx, id)
}

func (c *CustomerProfileController) AddCustomerIdentity(input dto.CreateCustomerIdentityInput) (*dto.CustomerIdentityDTO, error) {
	ctx := appContext
	return c.uc.AddCustomerIdentity(ctx, input)
}

func (c *CustomerProfileController) DeleteCustomerIdentity(id uint) error {
	ctx := appContext
	return c.uc.DeleteCustomerIdentity(ctx, id)
}

func (c *CustomerProfileController) GetMergeSuggestions() ([]dto.MergeSuggestionDTO, error) {
	ctx := appContext
	return c.uc.GetMergeSuggestions(ctx)
}

func (c *CustomerProfileController) DismissMergeSuggestion(id uint) error {
	ctx := appContext
	return c.uc.DismissMergeSuggestion(ctx, id)
}

func (c *CustomerProfileController) SaveSettings(settings dto.SystemSettingsDTO) error {
	ctx := appContext
	return c.uc.SaveSettings(ctx, settings)
}

func (c *CustomerProfileController) GetSettings() (dto.SystemSettingsDTO, error) {
	ctx := appContext
	return c.uc.GetSettings(ctx)
}
