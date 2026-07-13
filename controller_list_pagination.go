package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

type ListPaginationController struct {
	uc *app.ListPaginationUseCase
}

func NewListPaginationController() *ListPaginationController {
	gdb := database.GetDB()
	pageRepo := infra.NewListPaginationRepository(gdb)
	return &ListPaginationController{uc: app.NewListPaginationUseCase(
		pageRepo, pageRepo, pageRepo, pageRepo,
		infra.NewWaveRepository(gdb),
		infra.NewIntegrationProfileRepository(gdb),
	)}
}

func (c *ListPaginationController) ListCustomerProfilesPage(input dto.CustomerProfilePageFilterInput) (dto.CustomerProfilePageResult, error) {
	return c.uc.ListCustomerProfilesPage(appContext, input)
}

func (c *ListPaginationController) ListCustomerIdentityPlatforms() ([]string, error) {
	return c.uc.ListCustomerIdentityPlatforms(appContext)
}

func (c *ListPaginationController) ListProductMastersPage(input dto.ProductMasterPageFilterInput) (dto.ProductMasterPageResult, error) {
	return c.uc.ListProductMastersPage(appContext, input)
}

func (c *ListPaginationController) ListDemandInboxRowsPage(input dto.DemandInboxFilterInput) (dto.DemandInboxPageResult, error) {
	return c.uc.ListDemandInboxRowsPage(appContext, input)
}

func (c *ListPaginationController) ListShipmentsByWavePage(input dto.ShipmentByWavePageFilterInput) (dto.ShipmentPageResult, error) {
	return c.uc.ListShipmentsByWavePage(appContext, input)
}
