package app_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	application "github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

func TestCustomerSplitExecuteLedgerHistoryIdempotencyAndReopen(t *testing.T) {
	db, dbPath := newSplitExecutionTestDB(t)
	ctx := context.Background()
	fixture := seedSplitFixture(t, db, "happy")
	planInput := fixture.fullPlan()
	executor := application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(db))
	preview, err := executor.PreviewSplit(ctx, planInput)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.CanExecute || preview.PlanToken == "" || preview.ImmutableHistory.WillRewrite {
		t.Fatalf("unexpected split preview: blockers=%+v preview=%+v", preview.Blockers, preview)
	}
	if preview.Counts.Identities != 1 || preview.Counts.Addresses != 1 || preview.Counts.DemandDocuments != 1 ||
		preview.Counts.NameObservations != 1 || preview.Counts.NameEvents != 1 || preview.Counts.Origins != 1 ||
		preview.Counts.ProfileMutations != 2 {
		t.Fatalf("unexpected split counts: %+v", preview.Counts)
	}
	if len(preview.ImmutableHistory.WaveParticipantSnapshotIDs) != 1 || len(preview.ImmutableHistory.FulfillmentLineIDs) != 1 {
		t.Fatalf("immutable history was not surfaced: %+v", preview.ImmutableHistory)
	}
	executeInput := dto.ExecuteCustomerSplitInput{OperationKey: "split-happy-operation", PlanToken: preview.PlanToken,
		ExpectedSourceRowVersion: preview.SourceRowVersion, ExpectedTargetRowVersion: preview.TargetRowVersion,
		ActorRef: "operator:test", DecisionReason: "separate two customers", Plan: planInput}
	var executed *dto.ExecuteCustomerSplitResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		var executeErr error
		executed, executeErr = application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(tx)).ExecuteSplit(ctx, executeInput)
		return executeErr
	}); err != nil {
		t.Fatal(err)
	}
	if executed.Status != domain.SplitRecordStatusCompleted || executed.TargetProfileID == 0 || executed.DirectUndoSupported ||
		executed.ReverseOperationKind != domain.SplitReverseOperationManualMerge {
		t.Fatalf("unexpected split receipt: %+v", executed)
	}
	replayed, err := executor.ExecuteSplit(ctx, executeInput)
	if err != nil || !replayed.IdempotentReplay || replayed.SplitID != executed.SplitID {
		t.Fatalf("split idempotent replay failed: result=%+v err=%v", replayed, err)
	}
	conflict := executeInput
	conflict.DecisionReason = "different command"
	if _, err := executor.ExecuteSplit(ctx, conflict); err == nil || !strings.Contains(err.Error(), "idempotency conflict") {
		t.Fatalf("expected split idempotency conflict, got %v", err)
	}

	assertSplitOwnership(t, db, fixture, executed.TargetProfileID)
	var moved []persistence.SplitMovedEntity
	if err := db.Where("split_record_id = ?", executed.SplitID).Order("move_order").Find(&moved).Error; err != nil {
		t.Fatal(err)
	}
	if len(moved) != 10 {
		t.Fatalf("exact split ledger rows=%d want=10 rows=%+v", len(moved), moved)
	}
	for _, row := range moved {
		if row.BeforeSnapshot == "" || row.AfterSnapshot == "" || row.AfterStateHash == "" || row.SnapshotVersion != 1 {
			t.Fatalf("incomplete exact split ledger row: %+v", row)
		}
	}
	var waveSnapshot persistence.WaveParticipantSnapshot
	if err := db.First(&waveSnapshot, fixture.waveSnapshotID).Error; err != nil || waveSnapshot.CustomerProfileID != fixture.source.ID {
		t.Fatalf("wave history was rewritten: row=%+v err=%v", waveSnapshot, err)
	}
	var fulfillment persistence.FulfillmentLine
	if err := db.First(&fulfillment, fixture.fulfillmentLineID).Error; err != nil || fulfillment.CustomerProfileID == nil || *fulfillment.CustomerProfileID != fixture.source.ID {
		t.Fatalf("fulfillment history was rewritten: row=%+v err=%v", fulfillment, err)
	}

	history := application.NewCustomerSplitHistoryUseCase(infra.NewSplitExecutionStore(db))
	page, err := history.ListSplitHistory(ctx, dto.CustomerSplitHistoryQuery{ProfileID: executed.TargetProfileID, Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].OperationType != "split" || page.Items[0].DirectUndoSupported {
		t.Fatalf("unexpected split history: page=%+v err=%v", page, err)
	}
	detail, err := history.GetSplitHistory(ctx, executed.SplitID)
	if err != nil || len(detail.MovedEntities) != len(moved) || len(detail.Events) != 1 || detail.ReverseGuidance == "" {
		t.Fatalf("unexpected split history detail: detail=%+v err=%v", detail, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("reopen split database: %v", err)
	}
	defer closeSplitTestDB(t, reopened)
	reopenedDetail, err := application.NewCustomerSplitHistoryUseCase(infra.NewSplitExecutionStore(reopened)).GetSplitHistory(ctx, executed.SplitID)
	if err != nil || reopenedDetail.SplitID != executed.SplitID || len(reopenedDetail.MovedEntities) != len(moved) {
		t.Fatalf("split history did not survive restart: detail=%+v err=%v", reopenedDetail, err)
	}
}

func TestCustomerSplitPreviewBlockers(t *testing.T) {
	db, _ := newSplitExecutionTestDB(t)
	defer closeSplitTestDB(t, db)
	fixture := seedSplitFixture(t, db, "blockers")
	ctx := context.Background()

	other := persistence.CustomerProfile{DisplayName: "Other", ProfileType: "member", Status: domain.CustomerProfileStatusActive,
		RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	duplicateIdentity := persistence.CustomerIdentity{CustomerProfileID: other.ID, IdentityPlatform: "bilibili",
		IdentityValue: fixture.selectedIdentity.IdentityValue, IdentityType: persistence.IdentityType(domain.IdentityTypePlatformUID),
		Namespace: "bilibili", NormalizedValue: fixture.selectedIdentity.NormalizedValue, IsPrimary: true}
	if err := db.Create(&duplicateIdentity).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&persistence.WaveDemandAssignment{WaveID: 999, DemandDocumentID: fixture.selectedDemand.ID}).Error; err != nil {
		t.Fatal(err)
	}
	duplicateOrigins := []persistence.CustomerProfileOrigin{
		{CustomerProfileID: fixture.source.ID, OriginKind: "manual", ExternalRef: "collision"},
		{CustomerProfileID: fixture.source.ID, OriginKind: "manual", ExternalRef: "collision"},
	}
	if err := db.Create(&duplicateOrigins).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := db.Exec(`INSERT INTO customer_merge_records
(source_profile_id,target_profile_id,payload,status,created_at,completed_at) VALUES (?,?,?,?,?,?)`,
		9001, fixture.source.ID, `{}`, domain.MergeRecordStatusCompleted, now, now).Error; err != nil {
		t.Fatal(err)
	}
	plan := dto.CustomerSplitPreviewInput{SourceProfileID: fixture.source.ID,
		TargetStrategy: domain.SplitTargetStrategyRestoreMerged, NewProfileDisplayName: "Blocked target",
		TargetPrimaryIdentityIDs: []uint{fixture.remainingIdentity.ID}, TargetDefaultAddressID: &fixture.remainingAddress.ID,
		Selection: dto.CustomerSplitSelection{IdentityIDs: []uint{fixture.selectedIdentity.ID},
			DemandDocumentIDs: []uint{fixture.selectedDemand.ID}, OriginIDs: []uint{duplicateOrigins[0].ID, duplicateOrigins[1].ID}}}
	preview, err := application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(db)).PreviewSplit(ctx, plan)
	if err != nil {
		t.Fatal(err)
	}
	if preview.CanExecute {
		t.Fatalf("blocked split preview was executable: %+v", preview)
	}
	codes := splitBlockerCodeSet(preview.Blockers)
	for _, code := range []string{"split_restore_merged_not_supported", "split_merge_graph_active",
		"split_strong_identity_ambiguous", "split_demand_document_assigned", "split_origin_collision",
		"split_invalid_target_primary_identity", "split_invalid_target_default_address"} {
		if !codes[code] {
			t.Errorf("missing blocker %q in %+v", code, preview.Blockers)
		}
	}
	addressOnly, err := application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(db)).PreviewSplit(ctx,
		dto.CustomerSplitPreviewInput{SourceProfileID: fixture.source.ID, NewProfileDisplayName: "No anchor",
			Selection: dto.CustomerSplitSelection{AddressIDs: []uint{fixture.selectedAddress.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	if !splitBlockerCodeSet(addressOnly.Blockers)["split_target_ownership_anchor_required"] {
		t.Fatalf("address-only selection became an ownership anchor: %+v", addressOnly.Blockers)
	}
	blankIdentity := persistence.CustomerIdentity{CustomerProfileID: fixture.source.ID, IdentityPlatform: "bilibili",
		IdentityType: persistence.IdentityType(domain.IdentityTypePlatformUID), Namespace: "bilibili"}
	if err := db.Create(&blankIdentity).Error; err != nil {
		t.Fatal(err)
	}
	blankOnly, err := application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(db)).PreviewSplit(ctx,
		dto.CustomerSplitPreviewInput{SourceProfileID: fixture.source.ID, NewProfileDisplayName: "Blank identity",
			Selection: dto.CustomerSplitSelection{IdentityIDs: []uint{blankIdentity.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	if !splitBlockerCodeSet(blankOnly.Blockers)["split_target_ownership_anchor_required"] {
		t.Fatalf("blank identity became an ownership anchor: %+v", blankOnly.Blockers)
	}
}

func TestCustomerSplitRejectsStaleTokenAndSourceRevision(t *testing.T) {
	db, _ := newSplitExecutionTestDB(t)
	defer closeSplitTestDB(t, db)
	fixture := seedSplitFixture(t, db, "stale")
	ctx := context.Background()
	plan := dto.CustomerSplitPreviewInput{SourceProfileID: fixture.source.ID, NewProfileDisplayName: "Separated",
		TargetPrimaryIdentityIDs: []uint{fixture.selectedIdentity.ID},
		Selection:                dto.CustomerSplitSelection{IdentityIDs: []uint{fixture.selectedIdentity.ID}}}
	executor := application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(db))
	preview, err := executor.PreviewSplit(ctx, plan)
	if err != nil || !preview.CanExecute {
		t.Fatalf("preview: %+v err=%v", preview, err)
	}
	base := dto.ExecuteCustomerSplitInput{OperationKey: "split-stale", PlanToken: "tampered",
		ExpectedSourceRowVersion: preview.SourceRowVersion, ExpectedTargetRowVersion: 0,
		ActorRef: "operator:test", DecisionReason: "stale test", Plan: plan}
	if err := db.Transaction(func(tx *gorm.DB) error {
		_, executeErr := application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(tx)).ExecuteSplit(ctx, base)
		return executeErr
	}); err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("expected token mismatch, got %v", err)
	}
	if err := db.Model(&persistence.CustomerProfile{}).Where("id = ?", fixture.source.ID).
		UpdateColumn("row_version", gorm.Expr("row_version + 1")).Error; err != nil {
		t.Fatal(err)
	}
	base.OperationKey = "split-stale-version"
	base.PlanToken = preview.PlanToken
	if err := db.Transaction(func(tx *gorm.DB) error {
		_, executeErr := application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(tx)).ExecuteSplit(ctx, base)
		return executeErr
	}); err == nil || !strings.Contains(err.Error(), "row version") {
		t.Fatalf("expected source row-version mismatch, got %v", err)
	}
}

func TestCustomerSplitTransactionRollsBackAfterLedgerAndMutations(t *testing.T) {
	db, _ := newSplitExecutionTestDB(t)
	defer closeSplitTestDB(t, db)
	fixture := seedSplitFixture(t, db, "rollback")
	ctx := context.Background()
	plan := fixture.fullPlan()
	executor := application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(db))
	preview, err := executor.PreviewSplit(ctx, plan)
	if err != nil || !preview.CanExecute {
		t.Fatalf("preview: %+v err=%v", preview, err)
	}
	operationKey := "split-rollback-operation"
	if err := db.Create(&persistence.CustomerSplitOperationEvent{SplitRecordID: 999,
		EventKey: "split:" + operationKey + ":completed", OperationKey: "preexisting", EventType: "test",
		Status: "test", CreatedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatal(err)
	}
	var profileCountBefore int64
	if err := db.Model(&persistence.CustomerProfile{}).Count(&profileCountBefore).Error; err != nil {
		t.Fatal(err)
	}
	input := dto.ExecuteCustomerSplitInput{OperationKey: operationKey, PlanToken: preview.PlanToken,
		ExpectedSourceRowVersion: preview.SourceRowVersion, ActorRef: "operator:test", DecisionReason: "force rollback", Plan: plan}
	err = db.Transaction(func(tx *gorm.DB) error {
		_, executeErr := application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(tx)).ExecuteSplit(ctx, input)
		return executeErr
	})
	if err == nil || !strings.Contains(err.Error(), "completion event") {
		t.Fatalf("expected late audit-event failure, got %v", err)
	}
	var profileCountAfter, recordCount, movedCount int64
	_ = db.Model(&persistence.CustomerProfile{}).Count(&profileCountAfter).Error
	_ = db.Model(&persistence.CustomerSplitRecord{}).Count(&recordCount).Error
	_ = db.Model(&persistence.SplitMovedEntity{}).Count(&movedCount).Error
	if profileCountAfter != profileCountBefore || recordCount != 0 || movedCount != 0 {
		t.Fatalf("split transaction leaked state: profiles=%d/%d records=%d moved=%d", profileCountBefore, profileCountAfter, recordCount, movedCount)
	}
	var identity persistence.CustomerIdentity
	if err := db.First(&identity, fixture.selectedIdentity.ID).Error; err != nil || identity.CustomerProfileID != fixture.source.ID {
		t.Fatalf("selected identity survived rollback: row=%+v err=%v", identity, err)
	}
}

func TestCustomerSplitConcurrentCASAllowsOneExecution(t *testing.T) {
	db, _ := newSplitExecutionTestDB(t)
	defer closeSplitTestDB(t, db)
	fixture := seedSplitFixture(t, db, "concurrent")
	ctx := context.Background()
	plans := []dto.CustomerSplitPreviewInput{
		{SourceProfileID: fixture.source.ID, NewProfileDisplayName: "Concurrent A",
			TargetPrimaryIdentityIDs: []uint{fixture.selectedIdentity.ID}, Selection: dto.CustomerSplitSelection{IdentityIDs: []uint{fixture.selectedIdentity.ID}}},
		{SourceProfileID: fixture.source.ID, NewProfileDisplayName: "Concurrent B",
			TargetPrimaryIdentityIDs: []uint{fixture.remainingIdentity.ID}, Selection: dto.CustomerSplitSelection{IdentityIDs: []uint{fixture.remainingIdentity.ID}}},
	}
	executor := application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(db))
	previews := make([]*dto.CustomerSplitPreviewResult, len(plans))
	for i := range plans {
		var err error
		previews[i], err = executor.PreviewSplit(ctx, plans[i])
		if err != nil || !previews[i].CanExecute {
			t.Fatalf("preview %d: %+v err=%v", i, previews[i], err)
		}
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	results := make([]*dto.ExecuteCustomerSplitResult, 2)
	for i := range plans {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = db.Transaction(func(tx *gorm.DB) error {
				var executeErr error
				results[i], executeErr = application.NewCustomerSplitExecutor(infra.NewSplitExecutionStore(tx)).ExecuteSplit(ctx,
					dto.ExecuteCustomerSplitInput{OperationKey: fmt.Sprintf("split-concurrent-%d", i), PlanToken: previews[i].PlanToken,
						ExpectedSourceRowVersion: previews[i].SourceRowVersion, ActorRef: "operator:test",
						DecisionReason: "concurrent CAS", Plan: plans[i]})
				return executeErr
			})
		}()
	}
	wg.Wait()
	successes := 0
	for i := range errs {
		if errs[i] == nil && results[i] != nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent split successes=%d results=%+v errors=%v", successes, results, errs)
	}
	var recordCount int64
	if err := db.Model(&persistence.CustomerSplitRecord{}).Count(&recordCount).Error; err != nil || recordCount != 1 {
		t.Fatalf("concurrent split records=%d err=%v", recordCount, err)
	}
}

type splitFixture struct {
	source               *domain.CustomerProfile
	selectedIdentity     *domain.CustomerIdentity
	remainingIdentity    *domain.CustomerIdentity
	selectedAddress      *domain.CustomerAddress
	remainingAddress     *domain.CustomerAddress
	selectedDemand       *domain.DemandDocument
	remainingDemand      *domain.DemandDocument
	selectedObservation  *domain.CustomerNameObservation
	remainingObservation *domain.CustomerNameObservation
	selectedOrigin       *domain.CustomerProfileOrigin
	remainingOrigin      *domain.CustomerProfileOrigin
	waveSnapshotID       uint
	fulfillmentLineID    uint
}

func (f splitFixture) fullPlan() dto.CustomerSplitPreviewInput {
	return dto.CustomerSplitPreviewInput{SourceProfileID: f.source.ID, TargetStrategy: domain.SplitTargetStrategyCreateNew,
		TargetPrimaryIdentityIDs: []uint{f.selectedIdentity.ID}, TargetDefaultAddressID: &f.selectedAddress.ID,
		TargetDisplayNameObservationID: &f.selectedObservation.ID, SourceDisplayNameResolution: "auto_remaining",
		Selection: dto.CustomerSplitSelection{IdentityIDs: []uint{f.selectedIdentity.ID}, AddressIDs: []uint{f.selectedAddress.ID},
			DemandDocumentIDs: []uint{f.selectedDemand.ID}, NameObservationIDs: []uint{f.selectedObservation.ID},
			OriginIDs: []uint{f.selectedOrigin.ID}}}
}

func seedSplitFixture(t *testing.T, db *gorm.DB, prefix string) splitFixture {
	t.Helper()
	ctx := context.Background()
	profiles := infra.NewProfileRepository(db)
	addresses := infra.NewAddressRepository(db)
	demands := infra.NewDemandRepository(db)
	observations := infra.NewCustomerNameObservationRepository(db)
	events := infra.NewCustomerNameEventRepository(db)
	origins := infra.NewCustomerProfileOriginRepository(db)
	source := &domain.CustomerProfile{DisplayName: prefix + " selected name", ProfileType: "member",
		Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModePinned}
	if err := profiles.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	selectedIdentity := &domain.CustomerIdentity{CustomerProfileID: source.ID, IdentityPlatform: "bilibili",
		IdentityValue: prefix + "-uid-a", IdentityType: string(domain.IdentityTypePlatformUID), Namespace: "bilibili",
		NormalizedValue: prefix + "-uid-a", VerificationStatus: "observed", IsPrimary: true}
	remainingIdentity := &domain.CustomerIdentity{CustomerProfileID: source.ID, IdentityPlatform: "bilibili",
		IdentityValue: prefix + "-uid-b", IdentityType: string(domain.IdentityTypePlatformUID), Namespace: "bilibili",
		NormalizedValue: prefix + "-uid-b", VerificationStatus: "observed", IsPrimary: false}
	if err := profiles.CreateIdentity(ctx, selectedIdentity); err != nil {
		t.Fatal(err)
	}
	if err := profiles.CreateIdentity(ctx, remainingIdentity); err != nil {
		t.Fatal(err)
	}
	selectedAddress := &domain.CustomerAddress{CustomerProfileID: source.ID, Label: prefix + " selected", IsDefault: true}
	remainingAddress := &domain.CustomerAddress{CustomerProfileID: source.ID, Label: prefix + " remaining", IsDefault: false}
	if err := addresses.Create(ctx, selectedAddress); err != nil {
		t.Fatal(err)
	}
	if err := addresses.Create(ctx, remainingAddress); err != nil {
		t.Fatal(err)
	}
	selectedDemand := &domain.DemandDocument{Kind: "gift", CustomerProfileID: &source.ID, SourceDocumentNo: prefix + "-selected"}
	remainingDemand := &domain.DemandDocument{Kind: "gift", CustomerProfileID: &source.ID, SourceDocumentNo: prefix + "-remaining"}
	if err := demands.Create(ctx, selectedDemand); err != nil {
		t.Fatal(err)
	}
	if err := demands.Create(ctx, remainingDemand); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	selectedObservation := &domain.CustomerNameObservation{CustomerProfileID: source.ID, Name: prefix + " selected name",
		NormalizedName: prefix + "selectedname", SourceEventKey: prefix + "-selected-name", EpisodeKey: prefix + "-selected-name",
		ObservationCount: 1, NameKind: domain.CustomerNameKindManual, TrustScore: 1, ObservedAt: &now,
		FirstSeenAt: &now, LastSeenAt: &now, IsPinned: true, IsActive: true}
	remainingObservation := &domain.CustomerNameObservation{CustomerProfileID: source.ID, Name: prefix + " remaining name",
		NormalizedName: prefix + "remainingname", SourceEventKey: prefix + "-remaining-name", EpisodeKey: prefix + "-remaining-name",
		ObservationCount: 1, NameKind: domain.CustomerNameKindStableIdentityNickname, TrustScore: .8, ObservedAt: &now,
		FirstSeenAt: &now, LastSeenAt: &now, IsActive: true}
	if err := observations.Create(ctx, selectedObservation); err != nil {
		t.Fatal(err)
	}
	if err := observations.Create(ctx, remainingObservation); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&persistence.CustomerProfile{}).Where("id = ?", source.ID).
		UpdateColumn("display_name_observation_id", selectedObservation.ID).Error; err != nil {
		t.Fatal(err)
	}
	source.DisplayNameObservationID = &selectedObservation.ID
	for _, observation := range []*domain.CustomerNameObservation{selectedObservation, remainingObservation} {
		created, err := events.CreateIfAbsent(ctx, &domain.CustomerNameEvent{EventKey: observation.SourceEventKey,
			CustomerProfileID: source.ID, ObservationID: &observation.ID, EventKind: "observed", NewName: observation.Name,
			ReasonCode: observation.NameKind, Payload: `{}`, CreatedAt: now})
		if err != nil || !created {
			t.Fatalf("create name event: created=%v err=%v", created, err)
		}
	}
	integrationID := uint(42)
	selectedOrigin := &domain.CustomerProfileOrigin{CustomerProfileID: source.ID, OriginKind: domain.CustomerOriginKindRetailOrder,
		SourceIntegrationProfileID: &integrationID, ExternalRef: prefix + "-order-a", FirstSeenAt: &now, LastSeenAt: &now}
	remainingOrigin := &domain.CustomerProfileOrigin{CustomerProfileID: source.ID, OriginKind: domain.CustomerOriginKindRetailOrder,
		SourceIntegrationProfileID: &integrationID, ExternalRef: prefix + "-order-b", FirstSeenAt: &now, LastSeenAt: &now}
	if created, err := origins.CreateIfAbsent(ctx, selectedOrigin); err != nil || !created {
		t.Fatalf("create selected origin: created=%v err=%v", created, err)
	}
	if created, err := origins.CreateIfAbsent(ctx, remainingOrigin); err != nil || !created {
		t.Fatalf("create remaining origin: created=%v err=%v", created, err)
	}
	waveSnapshot := persistence.WaveParticipantSnapshot{WaveID: 700, CustomerProfileID: source.ID,
		SnapshotType: "member", DisplayName: source.DisplayName, CreatedAt: now}
	if err := db.Create(&waveSnapshot).Error; err != nil {
		t.Fatal(err)
	}
	fulfillment := persistence.FulfillmentLine{WaveID: 700, CustomerProfileID: &source.ID, Quantity: 1}
	if err := db.Create(&fulfillment).Error; err != nil {
		t.Fatal(err)
	}
	return splitFixture{source: source, selectedIdentity: selectedIdentity, remainingIdentity: remainingIdentity,
		selectedAddress: selectedAddress, remainingAddress: remainingAddress, selectedDemand: selectedDemand,
		remainingDemand: remainingDemand, selectedObservation: selectedObservation, remainingObservation: remainingObservation,
		selectedOrigin: selectedOrigin, remainingOrigin: remainingOrigin, waveSnapshotID: waveSnapshot.ID,
		fulfillmentLineID: fulfillment.ID}
}

func assertSplitOwnership(t *testing.T, db *gorm.DB, fixture splitFixture, targetID uint) {
	t.Helper()
	var selectedIdentity, remainingIdentity persistence.CustomerIdentity
	if err := db.First(&selectedIdentity, fixture.selectedIdentity.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&remainingIdentity, fixture.remainingIdentity.ID).Error; err != nil {
		t.Fatal(err)
	}
	if selectedIdentity.CustomerProfileID != targetID || !selectedIdentity.IsPrimary ||
		remainingIdentity.CustomerProfileID != fixture.source.ID || !remainingIdentity.IsPrimary {
		t.Fatalf("identity ownership/primary projection failed: selected=%+v remaining=%+v", selectedIdentity, remainingIdentity)
	}
	var selectedAddress, remainingAddress persistence.CustomerAddress
	_ = db.First(&selectedAddress, fixture.selectedAddress.ID).Error
	_ = db.First(&remainingAddress, fixture.remainingAddress.ID).Error
	if selectedAddress.CustomerProfileID != targetID || !selectedAddress.IsDefault ||
		remainingAddress.CustomerProfileID != fixture.source.ID || !remainingAddress.IsDefault {
		t.Fatalf("address ownership/default projection failed: selected=%+v remaining=%+v", selectedAddress, remainingAddress)
	}
	var selectedDemand persistence.DemandDocument
	var selectedObservation persistence.CustomerNameObservation
	var selectedEvent persistence.CustomerNameEvent
	var selectedOrigin persistence.CustomerProfileOrigin
	_ = db.First(&selectedDemand, fixture.selectedDemand.ID).Error
	_ = db.First(&selectedObservation, fixture.selectedObservation.ID).Error
	_ = db.Where("observation_id = ?", fixture.selectedObservation.ID).First(&selectedEvent).Error
	_ = db.First(&selectedOrigin, fixture.selectedOrigin.ID).Error
	if selectedDemand.CustomerProfileID == nil || *selectedDemand.CustomerProfileID != targetID ||
		selectedObservation.CustomerProfileID != targetID || selectedEvent.CustomerProfileID != targetID ||
		selectedOrigin.CustomerProfileID != targetID {
		t.Fatalf("split ownership did not move exact selected graph: demand=%+v observation=%+v event=%+v origin=%+v",
			selectedDemand, selectedObservation, selectedEvent, selectedOrigin)
	}
	var source, target persistence.CustomerProfile
	_ = db.First(&source, fixture.source.ID).Error
	_ = db.First(&target, targetID).Error
	if source.RowVersion != 2 || source.DisplayName != fixture.remainingObservation.Name || source.DisplayNameMode != domain.DisplayNameModeAuto ||
		source.DisplayNameObservationID == nil || *source.DisplayNameObservationID != fixture.remainingObservation.ID {
		t.Fatalf("source display projection failed: %+v", source)
	}
	if target.RowVersion != 1 || target.DisplayName != fixture.selectedObservation.Name || target.DisplayNameMode != domain.DisplayNameModePinned ||
		target.DisplayNameObservationID == nil || *target.DisplayNameObservationID != fixture.selectedObservation.ID {
		t.Fatalf("target display projection failed: %+v", target)
	}
}

func splitBlockerCodeSet(blockers []dto.MergeBlocker) map[string]bool {
	result := map[string]bool{}
	for _, blocker := range blockers {
		result[blocker.Code] = true
	}
	return result
}

func newSplitExecutionTestDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "split-execution.db")
	db, err := database.InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	return db, path
}

func closeSplitTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
}
