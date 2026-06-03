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
	ctx := appContext
	return c.uc.CreateAddress(ctx, input)
}

func (c *AddressController) UpdateAddress(input dto.UpdateAddressInput) (*dto.CustomerAddressDTO, error) {
	ctx := appContext
	return c.uc.UpdateAddress(ctx, input)
}

func (c *AddressController) DeleteAddress(id uint) error {
	ctx := appContext
	return c.uc.DeleteAddress(ctx, id)
}

func (c *AddressController) GetAddress(id uint) (*dto.CustomerAddressDTO, error) {
	ctx := appContext
	return c.uc.GetAddress(ctx, id)
}

func (c *AddressController) ListAddressesByProfile(profileID uint) ([]dto.CustomerAddressDTO, error) {
	ctx := appContext
	return c.uc.ListAddressesByProfile(ctx, profileID)
}

func (c *AddressController) BindAddressToLine(input dto.BindAddressInput) (*dto.CustomerAddressDTO, error) {
	ctx := appContext
	return c.uc.BindAddressToLine(ctx, input)
}

func (c *AddressController) UnbindAddressFromLine(fulfillmentLineID uint) error {
	ctx := appContext
	return c.uc.UnbindAddressFromLine(ctx, fulfillmentLineID)
}
