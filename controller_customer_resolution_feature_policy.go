package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

type CustomerResolutionFeaturePolicyController struct {
	uc *app.CustomerResolutionFeaturePolicyUseCase
}

func NewCustomerResolutionFeaturePolicyController() *CustomerResolutionFeaturePolicyController {
	repo := infra.NewCustomerResolutionFeaturePolicyRepository(database.GetDB())
	return &CustomerResolutionFeaturePolicyController{uc: app.NewCustomerResolutionFeaturePolicyUseCase(repo)}
}

func (c *CustomerResolutionFeaturePolicyController) GetCustomerResolutionFeaturePolicy() (*dto.CustomerResolutionFeaturePolicyDTO, error) {
	return c.uc.Get(appContext)
}

func (c *CustomerResolutionFeaturePolicyController) UpdateCustomerResolutionFeaturePolicy(
	input dto.UpdateCustomerResolutionFeaturePolicyInput,
) (*dto.CustomerResolutionFeaturePolicyDTO, error) {
	return c.uc.Update(appContext, input)
}
