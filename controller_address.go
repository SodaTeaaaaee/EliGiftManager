package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

type AddressController struct {
	uc app.AddressManagementUseCase
}

func NewAddressController() *AddressController {
	gdb := database.GetDB()
	addressRepo := infra.NewAddressRepository(gdb)
	fulfillmentRepo := infra.NewFulfillmentRepository(gdb)
	return &AddressController{
		uc: app.NewAddressManagementUseCase(addressRepo, fulfillmentRepo),
	}
}

func (c *AddressController) CreateAddress(input dto.CreateAddressInput) (*dto.CustomerAddressDTO, error) {
	return c.uc.CreateAddress(input)
}

func (c *AddressController) UpdateAddress(input dto.UpdateAddressInput) (*dto.CustomerAddressDTO, error) {
	return c.uc.UpdateAddress(input)
}

func (c *AddressController) DeleteAddress(id uint) error {
	return c.uc.DeleteAddress(id)
}

func (c *AddressController) GetAddress(id uint) (*dto.CustomerAddressDTO, error) {
	return c.uc.GetAddress(id)
}

func (c *AddressController) ListAddressesByProfile(profileID uint) ([]dto.CustomerAddressDTO, error) {
	return c.uc.ListAddressesByProfile(profileID)
}

func (c *AddressController) BindAddressToLine(input dto.BindAddressInput) (*dto.CustomerAddressDTO, error) {
	return c.uc.BindAddressToLine(input)
}

func (c *AddressController) UnbindAddressFromLine(fulfillmentLineID uint) error {
	return c.uc.UnbindAddressFromLine(fulfillmentLineID)
}
