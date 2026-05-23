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
	settingsSvc := service.NewSettingsService()
	return &CustomerProfileController{
		uc: app.NewCustomerProfileUseCase(profileRepo, addressRepo, settingsSvc, gdb),
	}
}

func (c *CustomerProfileController) ListCustomerProfiles(keyword, platform string, missingAddressOnly bool) ([]dto.CustomerProfileDTO, error) {
	return c.uc.ListCustomerProfiles(keyword, platform, missingAddressOnly)
}

func (c *CustomerProfileController) GetCustomerProfile(id uint) (*dto.CustomerProfileDTO, error) {
	return c.uc.GetCustomerProfile(id)
}

func (c *CustomerProfileController) CreateCustomerProfile(input dto.CreateCustomerProfileInput) (*dto.CustomerProfileDTO, error) {
	return c.uc.CreateCustomerProfile(input)
}

func (c *CustomerProfileController) UpdateCustomerProfile(input dto.UpdateCustomerProfileInput) (*dto.CustomerProfileDTO, error) {
	return c.uc.UpdateCustomerProfile(input)
}

func (c *CustomerProfileController) DeleteCustomerProfile(id uint) error {
	return c.uc.DeleteCustomerProfile(id)
}

func (c *CustomerProfileController) AddCustomerIdentity(input dto.CreateCustomerIdentityInput) (*dto.CustomerIdentityDTO, error) {
	return c.uc.AddCustomerIdentity(input)
}

func (c *CustomerProfileController) DeleteCustomerIdentity(id uint) error {
	return c.uc.DeleteCustomerIdentity(id)
}

func (c *CustomerProfileController) GetMergeSuggestions() ([]dto.MergeSuggestionDTO, error) {
	return c.uc.GetMergeSuggestions()
}

func (c *CustomerProfileController) DismissMergeSuggestion(id uint) error {
	return c.uc.DismissMergeSuggestion(id)
}

func (c *CustomerProfileController) SaveSettings(settings dto.SystemSettingsDTO) error {
	return c.uc.SaveSettings(settings)
}

func (c *CustomerProfileController) GetSettings() (dto.SystemSettingsDTO, error) {
	return c.uc.GetSettings()
}
