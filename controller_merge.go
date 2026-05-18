package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

type MergeController struct {
	mergeUC app.ProfileMergeUseCase
}

func NewMergeController() *MergeController {
	gdb := database.GetDB()
	return &MergeController{
		mergeUC: app.NewProfileMergeUseCase(
			infra.NewProfileRepository(gdb),
			infra.NewAddressRepository(gdb),
			infra.NewDemandRepository(gdb),
			infra.NewWaveRepository(gdb),
			infra.NewFulfillmentRepository(gdb),
		),
	}
}

func (c *MergeController) MergeProfiles(input dto.MergeProfilesInput) (*dto.MergeProfilesResult, error) {
	return c.mergeUC.MergeProfiles(input)
}
