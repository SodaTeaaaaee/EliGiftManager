package app_test

import (
	"context"
	"strings"
	"testing"
	"time"

	application "github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAuditedMergeExecuteHistoryDryRunUndoAndIdempotency(t *testing.T) {
	t.Parallel()
	db := newMergeExecutionTestDB(t)
	ctx := context.Background()
	profiles := infra.NewProfileRepository(db)
	addresses := infra.NewAddressRepository(db)
	demands := infra.NewDemandRepository(db)
	observations := infra.NewCustomerNameObservationRepository(db)
	events := infra.NewCustomerNameEventRepository(db)
	origins := infra.NewCustomerProfileOriginRepository(db)

	source := &domain.CustomerProfile{DisplayName: "Source Stable", ProfileType: "member", Status: domain.CustomerProfileStatusActive,
		RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	target := &domain.CustomerProfile{DisplayName: "Target Recipient", ProfileType: "member", Status: domain.CustomerProfileStatusActive,
		RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	if err := profiles.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	if err := profiles.Create(ctx, target); err != nil {
		t.Fatal(err)
	}
	sourceIdentity := &domain.CustomerIdentity{CustomerProfileID: source.ID, IdentityPlatform: "mail", IdentityValue: "s@example.test",
		IdentityType: string(domain.IdentityTypeEmail), Namespace: "mail", NormalizedValue: "s@example.test", IsPrimary: true}
	targetIdentity := &domain.CustomerIdentity{CustomerProfileID: target.ID, IdentityPlatform: "mail", IdentityValue: "t@example.test",
		IdentityType: string(domain.IdentityTypeEmail), Namespace: "mail", NormalizedValue: "t@example.test", IsPrimary: true}
	if err := profiles.CreateIdentity(ctx, sourceIdentity); err != nil {
		t.Fatal(err)
	}
	if err := profiles.CreateIdentity(ctx, targetIdentity); err != nil {
		t.Fatal(err)
	}
	sourceAddress := &domain.CustomerAddress{CustomerProfileID: source.ID, Label: "source", RecipientName: "Source", IsDefault: true}
	targetAddress := &domain.CustomerAddress{CustomerProfileID: target.ID, Label: "target", RecipientName: "Target", IsDefault: true}
	if err := addresses.Create(ctx, sourceAddress); err != nil {
		t.Fatal(err)
	}
	if err := addresses.Create(ctx, targetAddress); err != nil {
		t.Fatal(err)
	}
	document := &domain.DemandDocument{Kind: "gift", CustomerProfileID: &source.ID, SourceDocumentNo: "merge-doc"}
	if err := demands.Create(ctx, document); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	sourceObservation := &domain.CustomerNameObservation{CustomerProfileID: source.ID, Name: "Source Stable", NormalizedName: "sourcestable",
		SourceEventKey: "source-name", EpisodeKey: "source-name", ObservationCount: 1,
		NameKind: domain.CustomerNameKindStableIdentityNickname, TrustScore: 1, ObservedAt: &now,
		FirstSeenAt: &now, LastSeenAt: &now, IsActive: true}
	targetObservation := &domain.CustomerNameObservation{CustomerProfileID: target.ID, Name: "Target Recipient", NormalizedName: "targetrecipient",
		SourceEventKey: "target-name", EpisodeKey: "target-name", ObservationCount: 1,
		NameKind: domain.CustomerNameKindRecipient, TrustScore: .5, ObservedAt: &now,
		FirstSeenAt: &now, LastSeenAt: &now, IsActive: true}
	if err := observations.Create(ctx, sourceObservation); err != nil {
		t.Fatal(err)
	}
	if err := observations.Create(ctx, targetObservation); err != nil {
		t.Fatal(err)
	}
	created, err := events.CreateIfAbsent(ctx, &domain.CustomerNameEvent{EventKey: "source-name", CustomerProfileID: source.ID,
		ObservationID: &sourceObservation.ID, EventKind: "observed", NewName: sourceObservation.Name,
		ReasonCode: sourceObservation.NameKind, Payload: `{}`, CreatedAt: now})
	if err != nil || !created {
		t.Fatalf("create source name event: created=%v err=%v", created, err)
	}
	if _, err := origins.CreateIfAbsent(ctx, &domain.CustomerProfileOrigin{CustomerProfileID: source.ID,
		OriginKind: domain.CustomerOriginKindRetailOrder, ExternalRef: "source-order", FirstSeenAt: &now, LastSeenAt: &now}); err != nil {
		t.Fatal(err)
	}

	policy := persistence.MergePolicy{PolicyKey: "test", Name: "Test", IsActive: true,
		DefaultAction: domain.MergePolicyActionSuggestOnly, RowVersion: 1}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatal(err)
	}
	revision := persistence.MergePolicyRevision{MergePolicyID: policy.ID, Revision: 1,
		Action: domain.MergePolicyActionSuggestOnly, Rules: `{"schemaVersion":1,"executionMode":"suggest_only"}`,
		Checksum: "policy-v1", SchemaVersion: 1, CreatedAt: now}
	if err := db.Create(&revision).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&policy).Updates(map[string]any{"current_revision_id": revision.ID}).Error; err != nil {
		t.Fatal(err)
	}
	candidate := persistence.MergeCandidate{SourceProfileID: source.ID, TargetProfileID: target.ID,
		Status: domain.MergeCandidateStatusPending, MergePolicyRevisionID: &revision.ID,
		CanonicalPairKey: "pair", EvidenceHash: "evidence-v1", PolicyVersion: 1, Blockers: "[]", RowVersion: 1}
	if err := db.Create(&candidate).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&persistence.MergeEvidence{MergeCandidateID: candidate.ID, EvidenceKind: "review",
		EvidenceKey: "evidence", Polarity: "positive", ExplanationCode: "reviewed", ValueHash: "masked-hash",
		MaskedValue: "***", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	executor := application.NewCustomerMergeExecutor(infra.NewMergeExecutionStore(db))
	preview, err := executor.PreviewMerge(ctx, dto.CustomerMergePreviewInput{SourceProfileID: source.ID,
		TargetProfileID: target.ID, CandidateID: &candidate.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.CanExecute || preview.PreviewToken == "" {
		t.Fatalf("unexpected blocked preview: %+v", preview.Blockers)
	}
	if preview.Counts.NameEvents != 1 || preview.Counts.Origins != 1 || preview.Counts.ProfileMutations != 2 {
		t.Fatalf("unexpected exact preview counts: %+v", preview.Counts)
	}
	executeInput := dto.ExecuteCustomerMergeInput{OperationKey: "merge-operation-1", PreviewToken: preview.PreviewToken,
		SourceProfileID: source.ID, TargetProfileID: target.ID, ExpectedSourceRowVersion: preview.SourceRowVersion,
		ExpectedTargetRowVersion: preview.TargetRowVersion, CandidateID: &candidate.ID,
		ExpectedCandidateRowVersion: preview.CandidateRowVersion, ExpectedEvidenceHash: preview.EvidenceHash,
		ExpectedPolicyVersion: preview.PolicyVersion, ExpectedPolicyRevisionID: preview.PolicyRevisionID,
		ActorRef: "operator:test", DecisionReason: "reviewed evidence"}
	var executed *dto.ExecuteCustomerMergeResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		var txErr error
		executed, txErr = application.NewCustomerMergeExecutor(infra.NewMergeExecutionStore(tx)).ExecuteMerge(ctx, executeInput)
		return txErr
	}); err != nil {
		t.Fatal(err)
	}
	if executed.Status != domain.MergeRecordStatusCompleted || !executed.UndoDryRunRequired {
		t.Fatalf("unexpected execute result: %+v", executed)
	}
	replayed, err := executor.ExecuteMerge(ctx, executeInput)
	if err != nil || !replayed.IdempotentReplay || replayed.MergeID != executed.MergeID {
		t.Fatalf("idempotent replay failed: %+v err=%v", replayed, err)
	}
	conflict := executeInput
	conflict.DecisionReason = "different command"
	if _, err := executor.ExecuteMerge(ctx, conflict); err == nil || !strings.Contains(err.Error(), "idempotency conflict") {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}

	var sourceRow, targetRow persistence.CustomerProfile
	sourceRow = persistence.CustomerProfile{}
	if err := db.Unscoped().First(&sourceRow, source.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Unscoped().First(&targetRow, target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if sourceRow.Status != domain.CustomerProfileStatusMerged || sourceRow.MergedIntoProfileID == nil || *sourceRow.MergedIntoProfileID != target.ID || !sourceRow.DeletedAt.Valid {
		t.Fatalf("source profile was not authoritatively merged: %+v", sourceRow)
	}
	if targetRow.DisplayName != sourceObservation.Name || targetRow.RowVersion != 2 {
		t.Fatalf("unexpected target projection: %+v", targetRow)
	}
	var movedIdentity persistence.CustomerIdentity
	if err := db.First(&movedIdentity, sourceIdentity.ID).Error; err != nil {
		t.Fatal(err)
	}
	if movedIdentity.CustomerProfileID != target.ID || movedIdentity.IsPrimary {
		t.Fatalf("target primary did not win: %+v", movedIdentity)
	}
	var executedCandidate persistence.MergeCandidate
	if err := db.First(&executedCandidate, candidate.ID).Error; err != nil {
		t.Fatal(err)
	}
	if executedCandidate.Status != domain.MergeCandidateStatusExecuted || executedCandidate.RowVersion != 2 {
		t.Fatalf("candidate was not CAS executed: %+v", executedCandidate)
	}

	history := application.NewCustomerMergeHistoryUseCase(infra.NewMergeExecutionStore(db))
	page, err := history.ListMergeHistory(ctx, dto.CustomerMergeHistoryQuery{ProfileID: target.ID, Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].Counts.NameEvents != 1 {
		t.Fatalf("unexpected history: %+v err=%v", page, err)
	}
	if err := db.Model(&persistence.CustomerAddress{}).Where("id = ?", sourceAddress.ID).Update("label", "edited-after-merge").Error; err != nil {
		t.Fatal(err)
	}
	undoService := application.NewCustomerMergeUndoService(infra.NewMergeExecutionStore(db))
	dryRun, err := undoService.DryRunUndo(ctx, dto.CustomerMergeUndoDryRunInput{MergeID: executed.MergeID})
	if err != nil {
		t.Fatal(err)
	}
	if !dryRun.Eligible || dryRun.EligibilityToken == "" {
		t.Fatalf("unexpected undo blockers: %+v", dryRun.Blockers)
	}
	undoInput := dto.ExecuteCustomerMergeUndoInput{MergeID: executed.MergeID, UndoOperationKey: "undo-operation-1",
		EligibilityToken: dryRun.EligibilityToken, ExpectedSourceRowVersion: dryRun.SourceRowVersion,
		ExpectedTargetRowVersion: dryRun.TargetRowVersion, ActorRef: "operator:test", Reason: "verified undo"}
	var undone *dto.ExecuteCustomerMergeUndoResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		var txErr error
		undone, txErr = application.NewCustomerMergeUndoService(infra.NewMergeExecutionStore(tx)).ExecuteUndo(ctx, undoInput)
		return txErr
	}); err != nil {
		t.Fatal(err)
	}
	if undone.Status != domain.MergeRecordStatusUndone || undone.RestoreCounts.NameEvents != 1 {
		t.Fatalf("unexpected undo result: %+v", undone)
	}
	sourceRow = persistence.CustomerProfile{}
	if err := db.Unscoped().First(&sourceRow, source.ID).Error; err != nil {
		t.Fatal(err)
	}
	if sourceRow.Status != domain.CustomerProfileStatusActive || sourceRow.MergedIntoProfileID != nil || sourceRow.DeletedAt.Valid {
		t.Fatalf("source not restored: %+v", sourceRow)
	}
	var restoredAddress persistence.CustomerAddress
	if err := db.First(&restoredAddress, sourceAddress.ID).Error; err != nil {
		t.Fatal(err)
	}
	if restoredAddress.CustomerProfileID != source.ID || restoredAddress.Label != "edited-after-merge" {
		t.Fatalf("owner-only restore overwrote content: %+v", restoredAddress)
	}
	replayedUndo, err := undoService.ExecuteUndo(ctx, undoInput)
	if err != nil || !replayedUndo.IdempotentReplay {
		t.Fatalf("undo idempotent replay failed: %+v err=%v", replayedUndo, err)
	}
	if err := db.First(&executedCandidate, candidate.ID).Error; err != nil {
		t.Fatal(err)
	}
	if executedCandidate.Status != domain.MergeCandidateStatusStale {
		t.Fatalf("undo must stale candidate, got %+v", executedCandidate)
	}
}

func newMergeExecutionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:merge-execution-" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&persistence.CustomerProfile{}, &persistence.CustomerIdentity{}, &persistence.CustomerAddress{},
		&persistence.DemandDocument{}, &persistence.WaveDemandAssignment{}, &persistence.CustomerNameObservation{},
		&persistence.CustomerNameEvent{}, &persistence.CustomerProfileOrigin{}, &persistence.MergePolicy{},
		&persistence.MergePolicyRevision{}, &persistence.MergeCandidate{}, &persistence.MergeEvidence{},
		&persistence.CustomerMergeRecord{}, &persistence.MergeMovedEntity{}, &persistence.CustomerMergeOperationEvent{}); err != nil {
		t.Fatal(err)
	}
	seedEnabledFeaturePolicyForFocusedDB(t, db)
	return db
}
