package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

type MergeController struct {
	gdb *gorm.DB
}

func NewMergeController() *MergeController {
	return &MergeController{gdb: database.GetDB()}
}

// MergeProfiles merges the source profile into the target profile. The whole
// operation (identity/address/demand/participant/fulfillment migration plus the
// source soft-delete) runs in a single transaction so a partial failure rolls
// back cleanly instead of leaving data half-migrated.
func (c *MergeController) MergeProfiles(input dto.MergeProfilesInput) (*dto.MergeProfilesResult, error) {
	var result *dto.MergeProfilesResult
	ctx := appContext
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		mergeUC := app.NewProfileMergeUseCase(
			repos.CustomerProfile,
			repos.Address,
			repos.DemandRepo,
			repos.WaveRepo,
			repos.FulfillRepo,
		)
		r, mergeErr := mergeUC.MergeProfiles(ctx, input)
		if mergeErr != nil {
			return mergeErr
		}
		result = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
