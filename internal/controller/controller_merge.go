package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
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
// operation (identity/address/unassigned-demand migration, source soft-delete,
// and merge-record creation) runs in a single transaction so a partial failure rolls
// back cleanly instead of leaving data half-migrated.
func (c *MergeController) MergeProfiles(input dto.MergeProfilesInput) (*dto.MergeProfilesResult, error) {
	var result *dto.MergeProfilesResult
	ctx := appContext
	if err := infra.NewCustomerResolutionFeaturePolicyRepository(c.gdb).RequireFeature(ctx, domain.CustomerResolutionFeatureMergeExecution); err != nil {
		return nil, err
	}
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		executor := app.NewCustomerMergeExecutor(repos.MergeExecution)
		preview, previewErr := executor.PreviewMerge(ctx, dto.CustomerMergePreviewInput{
			SourceProfileID: input.SourceProfileID, TargetProfileID: input.TargetProfileID,
		})
		if previewErr != nil {
			return previewErr
		}
		if !preview.CanExecute {
			return fmt.Errorf("merge blocked; use PreviewCustomerMerge for blocker details")
		}
		operationKey, keyErr := newLegacyMergeOperationKey()
		if keyErr != nil {
			return keyErr
		}
		executed, executeErr := executor.ExecuteMerge(ctx, dto.ExecuteCustomerMergeInput{
			OperationKey: operationKey, PreviewToken: preview.PreviewToken,
			SourceProfileID: input.SourceProfileID, TargetProfileID: input.TargetProfileID,
			ExpectedSourceRowVersion: preview.SourceRowVersion, ExpectedTargetRowVersion: preview.TargetRowVersion,
			ActorRef: "legacy-manual", DecisionReason: "legacy MergeProfiles adapter",
		})
		if executeErr != nil {
			return executeErr
		}
		result = &dto.MergeProfilesResult{MigratedIdentityCount: executed.Counts.Identities,
			MigratedAddressCount: executed.Counts.Addresses, UpdatedDemandDocs: executed.Counts.DemandDocuments,
			MergeID: executed.MergeID, UndoAvailable: false}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *MergeController) PreviewCustomerMerge(input dto.CustomerMergePreviewInput) (*dto.CustomerMergePreviewResult, error) {
	executor := app.NewCustomerMergeExecutor(infra.NewMergeExecutionStore(c.gdb))
	return executor.PreviewMerge(appContext, input)
}

func (c *MergeController) ExecuteCustomerMerge(input dto.ExecuteCustomerMergeInput) (*dto.ExecuteCustomerMergeResult, error) {
	var result *dto.ExecuteCustomerMergeResult
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		executor := app.NewCustomerMergeExecutor(infra.NewMergeExecutionStore(tx))
		var executeErr error
		result, executeErr = executor.ExecuteMerge(appContext, input)
		return executeErr
	})
	return result, err
}

func (c *MergeController) ListCustomerMergeHistory(query dto.CustomerMergeHistoryQuery) (*dto.CustomerMergeHistoryPage, error) {
	useCase := app.NewCustomerMergeHistoryUseCase(infra.NewMergeExecutionStore(c.gdb))
	return useCase.ListMergeHistory(appContext, query)
}

func (c *MergeController) GetCustomerMergeHistory(mergeID uint) (*dto.CustomerMergeHistoryDetail, error) {
	useCase := app.NewCustomerMergeHistoryUseCase(infra.NewMergeExecutionStore(c.gdb))
	return useCase.GetMergeHistory(appContext, mergeID)
}

func newLegacyMergeOperationKey() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("create merge operation key: %w", err)
	}
	return "legacy-merge-" + hex.EncodeToString(buffer), nil
}
