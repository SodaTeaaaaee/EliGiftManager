package app_test

import (
	"context"
	"path/filepath"
	"testing"

	application "github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

func TestMergeExecuteHistoryDryRunUndoSurvivesRealDatabaseRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "merge-restart.db")
	db, err := database.InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	profiles := infra.NewProfileRepository(db)
	source := &domain.CustomerProfile{DisplayName: "Restart Source", ProfileType: "member",
		Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	target := &domain.CustomerProfile{DisplayName: "Restart Target", ProfileType: "member",
		Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	if err := profiles.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	if err := profiles.Create(ctx, target); err != nil {
		t.Fatal(err)
	}
	identity := &domain.CustomerIdentity{CustomerProfileID: source.ID, IdentityPlatform: "bilibili",
		IdentityValue: "restart-source", IdentityType: string(domain.IdentityTypePlatformUID), Namespace: "bilibili",
		NormalizedValue: "restart-source", IsPrimary: true}
	if err := profiles.CreateIdentity(ctx, identity); err != nil {
		t.Fatal(err)
	}

	executor := application.NewCustomerMergeExecutor(infra.NewMergeExecutionStore(db))
	preview, err := executor.PreviewMerge(ctx, dto.CustomerMergePreviewInput{SourceProfileID: source.ID, TargetProfileID: target.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.CanExecute || preview.PreviewToken == "" {
		t.Fatalf("restart merge preview blocked: %+v", preview.Blockers)
	}
	executeInput := dto.ExecuteCustomerMergeInput{OperationKey: "merge-restart-operation",
		PreviewToken: preview.PreviewToken, SourceProfileID: source.ID, TargetProfileID: target.ID,
		ExpectedSourceRowVersion: preview.SourceRowVersion, ExpectedTargetRowVersion: preview.TargetRowVersion,
		ActorRef: "operator:restart-test", DecisionReason: "verify durable audit lifecycle"}
	var executed *dto.ExecuteCustomerMergeResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		var executeErr error
		executed, executeErr = application.NewCustomerMergeExecutor(infra.NewMergeExecutionStore(tx)).ExecuteMerge(ctx, executeInput)
		return executeErr
	}); err != nil {
		t.Fatal(err)
	}
	if executed.Status != domain.MergeRecordStatusCompleted || executed.MergeID == 0 {
		t.Fatalf("unexpected merge execution receipt: %+v", executed)
	}
	closeMergeRestartDB(t, db)

	reopened, err := database.InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMergeRestartDB(t, reopened)
	store := infra.NewMergeExecutionStore(reopened)
	history := application.NewCustomerMergeHistoryUseCase(store)
	page, err := history.ListMergeHistory(ctx, dto.CustomerMergeHistoryQuery{ProfileID: target.ID, Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].MergeID != executed.MergeID {
		t.Fatalf("merge history did not survive restart: page=%+v err=%v", page, err)
	}
	detail, err := history.GetMergeHistory(ctx, executed.MergeID)
	if err != nil || detail.Status != domain.MergeRecordStatusCompleted || len(detail.PlannedEntities) == 0 {
		t.Fatalf("merge audit detail did not survive restart: detail=%+v err=%v", detail, err)
	}
	undo := application.NewCustomerMergeUndoService(store)
	dryRun, err := undo.DryRunUndo(ctx, dto.CustomerMergeUndoDryRunInput{MergeID: executed.MergeID})
	if err != nil {
		t.Fatal(err)
	}
	if !dryRun.Eligible || dryRun.EligibilityToken == "" {
		t.Fatalf("restart undo dry-run blocked: %+v", dryRun.Blockers)
	}
	undoInput := dto.ExecuteCustomerMergeUndoInput{MergeID: executed.MergeID, UndoOperationKey: "undo-restart-operation",
		EligibilityToken: dryRun.EligibilityToken, ExpectedSourceRowVersion: dryRun.SourceRowVersion,
		ExpectedTargetRowVersion: dryRun.TargetRowVersion, ActorRef: "operator:restart-test", Reason: "verified after restart"}
	var undone *dto.ExecuteCustomerMergeUndoResult
	if err := reopened.Transaction(func(tx *gorm.DB) error {
		var undoErr error
		undone, undoErr = application.NewCustomerMergeUndoService(infra.NewMergeExecutionStore(tx)).ExecuteUndo(ctx, undoInput)
		return undoErr
	}); err != nil {
		t.Fatal(err)
	}
	if undone.Status != domain.MergeRecordStatusUndone || undone.RestoredSourceProfileID != source.ID {
		t.Fatalf("unexpected restart undo receipt: %+v", undone)
	}
	var restored persistence.CustomerProfile
	if err := reopened.Unscoped().First(&restored, source.ID).Error; err != nil {
		t.Fatal(err)
	}
	if restored.Status != domain.CustomerProfileStatusActive || restored.MergedIntoProfileID != nil || restored.DeletedAt.Valid {
		t.Fatalf("source was not restored after restart: %+v", restored)
	}
}

func closeMergeRestartDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
}
