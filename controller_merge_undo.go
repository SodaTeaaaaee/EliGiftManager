package main

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

type MergeUndoController struct {
	gdb *gorm.DB
}

func NewMergeUndoController() *MergeUndoController {
	return &MergeUndoController{gdb: database.GetDB()}
}

func (c *MergeUndoController) UndoCustomerMerge(input dto.UndoCustomerMergeInput) (*dto.UndoCustomerMergeResult, error) {
	var result *dto.UndoCustomerMergeResult
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		undoUC := app.NewProfileMergeUndoUseCase(
			repos.CustomerProfile,
			repos.Address,
			repos.DemandRepo,
			repos.CustomerMerge,
		)
		var undoErr error
		result, undoErr = undoUC.UndoCustomerMerge(appContext, input)
		return undoErr
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
