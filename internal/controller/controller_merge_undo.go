package controller

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
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
	if err := infra.NewCustomerResolutionFeaturePolicyRepository(c.gdb).RequireFeature(appContext, domain.CustomerResolutionFeatureMergeExecution); err != nil {
		return nil, err
	}
	var result *dto.UndoCustomerMergeResult
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		record, findErr := repos.MergeExecution.FindMergeRecord(appContext, input.MergeID)
		if findErr != nil {
			return findErr
		}
		if record.MovePlanHash != "" {
			return fmt.Errorf("merge %d requires server-side undo dry-run; use DryRunCustomerMergeUndo then ExecuteCustomerMergeUndo", input.MergeID)
		}
		undoUC := app.NewProfileMergeUndoUseCase(
			repos.CustomerProfile,
			repos.Address,
			repos.DemandRepo,
			repos.CustomerMerge,
			app.CustomerMergeResolutionRepos{NameObservations: repos.NameObservation, Origins: repos.CustomerOrigin},
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

func (c *MergeUndoController) DryRunCustomerMergeUndo(input dto.CustomerMergeUndoDryRunInput) (*dto.CustomerMergeUndoDryRunResult, error) {
	service := app.NewCustomerMergeUndoService(infra.NewMergeExecutionStore(c.gdb))
	return service.DryRunUndo(appContext, input)
}

func (c *MergeUndoController) ExecuteCustomerMergeUndo(input dto.ExecuteCustomerMergeUndoInput) (*dto.ExecuteCustomerMergeUndoResult, error) {
	var result *dto.ExecuteCustomerMergeUndoResult
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		service := app.NewCustomerMergeUndoService(infra.NewMergeExecutionStore(tx))
		var undoErr error
		result, undoErr = service.ExecuteUndo(appContext, input)
		return undoErr
	})
	return result, err
}
