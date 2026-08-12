package app

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

func TestCustomerResolutionFeaturePolicyDefaultCASAndRestart(t *testing.T) {
	db, path := newFeaturePolicyTestDB(t)
	ctx := context.Background()
	repo := infra.NewCustomerResolutionFeaturePolicyRepository(db)
	uc := NewCustomerResolutionFeaturePolicyUseCase(repo)
	initial, err := uc.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Revision != 1 || !initial.CustomerResolutionWritesEnabled || !initial.CandidateScanEnabled ||
		!initial.MergeExecutionEnabled || !initial.SplitExecutionEnabled || !initial.ImportEvidenceEnabled ||
		!initial.CarrierRegistryWritesEnabled {
		t.Fatalf("unsafe default policy: %+v", initial)
	}
	updated, err := uc.Update(ctx, dto.UpdateCustomerResolutionFeaturePolicyInput{
		ExpectedRevision: 1, ActorRef: "operator:test", Reason: "emergency stop",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || updated.CustomerResolutionWritesEnabled || updated.CandidateScanEnabled ||
		updated.MergeExecutionEnabled || updated.SplitExecutionEnabled || updated.ImportEvidenceEnabled ||
		updated.CarrierRegistryWritesEnabled {
		t.Fatalf("CAS did not persist disabled policy: %+v", updated)
	}
	if _, err := uc.Update(ctx, dto.UpdateCustomerResolutionFeaturePolicyInput{ExpectedRevision: 1}); featurePolicyErrorCode(err) != domain.FeaturePolicyCodeRevisionConflict {
		t.Fatalf("stale CAS error=%v code=%q", err, featurePolicyErrorCode(err))
	}
	var revisions []persistence.CustomerResolutionFeaturePolicyRevision
	if err := db.Order("revision").Find(&revisions).Error; err != nil || len(revisions) != 2 || revisions[0].Revision != 1 || revisions[1].Revision != 2 {
		t.Fatalf("immutable revision history=%+v err=%v", revisions, err)
	}
	closeFeaturePolicyTestDB(t, db)
	reopened, err := database.InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer closeFeaturePolicyTestDB(t, reopened)
	persisted, err := NewCustomerResolutionFeaturePolicyUseCase(infra.NewCustomerResolutionFeaturePolicyRepository(reopened)).Get(ctx)
	if err != nil || persisted.Revision != 2 || persisted.ImportEvidenceEnabled {
		t.Fatalf("policy did not survive restart: policy=%+v err=%v", persisted, err)
	}
}

func TestCustomerResolutionFeaturePolicyMasterSwitchCascadesAcrossAllSixFeatures(t *testing.T) {
	db, _ := newFeaturePolicyTestDB(t)
	defer closeFeaturePolicyTestDB(t, db)
	ctx := context.Background()
	repo := infra.NewCustomerResolutionFeaturePolicyRepository(db)
	features := []struct {
		name string
		code string
	}{
		{domain.CustomerResolutionFeatureWrites, domain.FeaturePolicyCodeWritesDisabled},
		{domain.CustomerResolutionFeatureCandidateScan, domain.FeaturePolicyCodeCandidateScanDisabled},
		{domain.CustomerResolutionFeatureMergeExecution, domain.FeaturePolicyCodeMergeExecutionDisabled},
		{domain.CustomerResolutionFeatureSplitExecution, domain.FeaturePolicyCodeSplitExecutionDisabled},
		{domain.CustomerResolutionFeatureImportEvidence, domain.FeaturePolicyCodeImportEvidenceDisabled},
		{domain.CustomerResolutionFeatureCarrierRegistry, domain.FeaturePolicyCodeCarrierRegistryDisabled},
	}

	for _, feature := range features {
		enabled, err := repo.FeatureEnabled(ctx, feature.name)
		if err != nil || !enabled {
			t.Fatalf("default feature %s enabled=%v err=%v", feature.name, enabled, err)
		}
	}

	current, err := repo.GetFeaturePolicy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	next := *current
	next.CustomerResolutionWritesEnabled = false
	// Deliberately leave every child flag stored as true. Effective state must
	// still be false and each child must keep its own public error code.
	if updated, applied, updateErr := repo.UpdateFeaturePolicyCAS(ctx, current.Revision, &next); updateErr != nil || !applied ||
		!updated.CandidateScanEnabled || !updated.MergeExecutionEnabled || !updated.SplitExecutionEnabled ||
		!updated.ImportEvidenceEnabled || !updated.CarrierRegistryWritesEnabled {
		t.Fatalf("persist master-only shutdown: updated=%+v applied=%v err=%v", updated, applied, updateErr)
	}
	for _, feature := range features {
		enabled, featureErr := repo.FeatureEnabled(ctx, feature.name)
		if featureErr != nil || enabled {
			t.Errorf("master-off feature %s enabled=%v err=%v", feature.name, enabled, featureErr)
		}
		assertFeaturePolicyCode(t, repo.RequireFeature(ctx, feature.name), feature.code)
	}

	current, err = repo.GetFeaturePolicy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	next = *current
	next.CustomerResolutionWritesEnabled = true
	next.CandidateScanEnabled = false
	next.MergeExecutionEnabled = false
	next.SplitExecutionEnabled = false
	next.ImportEvidenceEnabled = false
	next.CarrierRegistryWritesEnabled = false
	if _, applied, err := repo.UpdateFeaturePolicyCAS(ctx, current.Revision, &next); err != nil || !applied {
		t.Fatalf("persist child-only shutdown: applied=%v err=%v", applied, err)
	}
	if enabled, err := repo.FeatureEnabled(ctx, domain.CustomerResolutionFeatureWrites); err != nil || !enabled {
		t.Fatalf("master should remain enabled when only children are off: enabled=%v err=%v", enabled, err)
	}
	for _, feature := range features[1:] {
		assertFeaturePolicyCode(t, repo.RequireFeature(ctx, feature.name), feature.code)
	}
}

func TestCustomerResolutionWritesGateBlocksIdentityNameAndOriginButKeepsReads(t *testing.T) {
	db, _ := newFeaturePolicyTestDB(t)
	defer closeFeaturePolicyTestDB(t, db)
	ctx := context.Background()
	profiles := infra.NewProfileRepository(db)
	profile := &domain.CustomerProfile{DisplayName: "Readable", ProfileType: "member", Status: domain.CustomerProfileStatusActive,
		RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	if err := profiles.Create(ctx, profile); err != nil {
		t.Fatal(err)
	}
	disableFeaturePolicy(t, db, domain.CustomerResolutionFeatureWrites)
	if rows, err := profiles.List(ctx); err != nil || len(rows) != 1 {
		t.Fatalf("ordinary profile read was blocked: rows=%+v err=%v", rows, err)
	}
	identity := NewIdentityResolutionService(profiles)
	_, err := identity.ResolveStableProfile(ctx, StableIdentityResolutionInput{Namespace: "test", IdentityPlatform: "test",
		IdentityValue: "uid-1", IdentityType: string(domain.IdentityTypePlatformUID)})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	name := NewCustomerNameObservationService(profiles, infra.NewCustomerNameObservationRepository(db), infra.NewCustomerNameEventRepository(db))
	_, err = name.Observe(ctx, ObserveCustomerNameInput{CustomerProfileID: profile.ID, Name: "Blocked", SourceEventKey: "blocked-name"})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	origins := NewCustomerResolutionService(profiles, infra.NewCustomerProfileOriginRepository(db))
	_, err = origins.ResolveRetailOrderProfile(ctx, 1, "order-1", "Blocked", time.Now())
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	var identities, observations, originRows int64
	_ = db.Model(&persistence.CustomerIdentity{}).Count(&identities).Error
	_ = db.Model(&persistence.CustomerNameObservation{}).Count(&observations).Error
	_ = db.Model(&persistence.CustomerProfileOrigin{}).Count(&originRows).Error
	if identities != 0 || observations != 0 || originRows != 0 {
		t.Fatalf("disabled resolution leaked writes identities=%d observations=%d origins=%d", identities, observations, originRows)
	}
}

func TestCustomerResolutionWritesGateBlocksEveryAddressMutationBeforeValidationOrLookup(t *testing.T) {
	db, _ := newFeaturePolicyTestDB(t)
	defer closeFeaturePolicyTestDB(t, db)
	ctx := context.Background()
	profiles := infra.NewProfileRepository(db)
	profile := &domain.CustomerProfile{DisplayName: "Address owner", ProfileType: "member",
		Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	if err := profiles.Create(ctx, profile); err != nil {
		t.Fatal(err)
	}
	addresses := infra.NewAddressRepository(db)
	address := &domain.CustomerAddress{CustomerProfileID: profile.ID, Label: "Original", RecipientName: "Original",
		IsDefault: true, ValidationStatus: "valid"}
	if err := addresses.Create(ctx, address); err != nil {
		t.Fatal(err)
	}
	wave := &domain.Wave{WaveNo: "ADDRESS-GATE", Name: "ADDRESS-GATE", WaveType: "manual", LifecycleStage: "draft"}
	if err := infra.NewWaveRepository(db).Create(ctx, wave); err != nil {
		t.Fatal(err)
	}
	profileID := profile.ID
	line := &domain.FulfillmentLine{WaveID: wave.ID, CustomerProfileID: &profileID, Quantity: 1,
		AddressState: string(domain.AddressStateMissing), LineReason: string(domain.LineReasonEntitlement)}
	fulfillments := infra.NewFulfillmentRepository(db)
	if err := fulfillments.Create(ctx, line); err != nil {
		t.Fatal(err)
	}

	disableFeaturePolicy(t, db, domain.CustomerResolutionFeatureWrites)
	management := NewAddressManagementUseCase(addresses, fulfillments)
	_, err := management.CreateAddress(ctx, dto.CreateAddressInput{CustomerProfileID: profile.ID, IsDefault: true})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	_, err = management.UpdateAddress(ctx, dto.UpdateAddressInput{ID: 999, CustomerProfileID: profile.ID,
		Label: "Blocked", IsDefault: true})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	assertFeaturePolicyCode(t, management.DeleteAddress(ctx, 999), domain.FeaturePolicyCodeWritesDisabled)
	_, err = management.UpsertAddressFromImport(ctx, 0, RecipientAddressDraft{})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	_, err = management.BindAddressToLine(ctx, dto.BindAddressInput{FulfillmentLineID: 999, CustomerAddressID: 999})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	assertFeaturePolicyCode(t, management.UnbindAddressFromLine(ctx, 999), domain.FeaturePolicyCodeWritesDisabled)

	batch := NewAddressBatchUseCase(addresses, fulfillments)
	_, err = batch.BatchBindAddressToLines(ctx, []dto.BindAddressEntry{{FulfillmentLineID: line.ID, CustomerAddressID: address.ID}})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	_, err = batch.BindDefaultAddressesForWave(ctx, 999)
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)

	if got, readErr := management.GetAddress(ctx, address.ID); readErr != nil || got.Label != "Original" || !got.IsDefault {
		t.Fatalf("disabled address read blocked or changed: address=%+v err=%v", got, readErr)
	}
	if rows, readErr := management.ListAddressesByProfile(ctx, profile.ID); readErr != nil || len(rows) != 1 {
		t.Fatalf("disabled address list blocked: rows=%+v err=%v", rows, readErr)
	}
	var addressRows int64
	if err := db.Model(&persistence.CustomerAddress{}).Count(&addressRows).Error; err != nil || addressRows != 1 {
		t.Fatalf("disabled address writes changed count=%d err=%v", addressRows, err)
	}
	persistedLine, err := fulfillments.FindByID(ctx, line.ID)
	if err != nil || persistedLine.CustomerAddressID != nil || persistedLine.AddressState != string(domain.AddressStateMissing) {
		t.Fatalf("disabled address binding changed fulfillment: line=%+v err=%v", persistedLine, err)
	}
}

func TestCandidateMergeAndSplitGatesKeepReadSurfacesAvailable(t *testing.T) {
	db, _ := newFeaturePolicyTestDB(t)
	defer closeFeaturePolicyTestDB(t, db)
	ctx := context.Background()
	profiles := infra.NewProfileRepository(db)
	source := &domain.CustomerProfile{DisplayName: "Source", ProfileType: "member", Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	target := &domain.CustomerProfile{DisplayName: "Target", ProfileType: "member", Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	if err := profiles.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	if err := profiles.Create(ctx, target); err != nil {
		t.Fatal(err)
	}
	disableAllFeaturePolicy(t, db)

	governance := NewMergeGovernanceUseCase(infra.NewMergeGovernanceRepository(db), profiles,
		infra.NewAddressRepository(db), infra.NewCustomerProfileOriginRepository(db))
	if candidates, err := governance.ListCandidates(ctx, ""); err != nil || len(candidates) != 0 {
		t.Fatalf("candidate history read blocked: rows=%+v err=%v", candidates, err)
	}
	_, err := governance.ScanMergeCandidates(ctx)
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeCandidateScanDisabled)
	assertFeaturePolicyCode(t, governance.DismissCandidate(ctx, dto.DismissMergeCandidateInput{}), domain.FeaturePolicyCodeCandidateScanDisabled)

	mergeStore := infra.NewMergeExecutionStore(db)
	mergeExecutor := NewCustomerMergeExecutor(mergeStore)
	if _, err := mergeExecutor.PreviewMerge(ctx, dto.CustomerMergePreviewInput{SourceProfileID: source.ID, TargetProfileID: target.ID}); err != nil {
		t.Fatalf("merge preview was blocked by execute kill-switch: %v", err)
	}
	if page, err := NewCustomerMergeHistoryUseCase(mergeStore).ListMergeHistory(ctx, dto.CustomerMergeHistoryQuery{Limit: 10}); err != nil || len(page.Items) != 0 {
		t.Fatalf("merge history read blocked: page=%+v err=%v", page, err)
	}
	_, err = mergeExecutor.ExecuteMerge(ctx, dto.ExecuteCustomerMergeInput{})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeMergeExecutionDisabled)
	undo := NewCustomerMergeUndoService(mergeStore)
	if _, err := undo.DryRunUndo(ctx, dto.CustomerMergeUndoDryRunInput{MergeID: 999}); featurePolicyErrorCode(err) == domain.FeaturePolicyCodeMergeExecutionDisabled {
		t.Fatalf("undo dry-run was incorrectly gated: %v", err)
	}
	_, err = undo.ExecuteUndo(ctx, dto.ExecuteCustomerMergeUndoInput{})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeMergeExecutionDisabled)

	splitStore := infra.NewSplitExecutionStore(db)
	splitExecutor := NewCustomerSplitExecutor(splitStore)
	if _, err := splitExecutor.PreviewSplit(ctx, dto.CustomerSplitPreviewInput{SourceProfileID: source.ID}); err != nil {
		t.Fatalf("split preview was blocked by execute kill-switch: %v", err)
	}
	if page, err := NewCustomerSplitHistoryUseCase(splitStore).ListSplitHistory(ctx, dto.CustomerSplitHistoryQuery{Limit: 10}); err != nil || len(page.Items) != 0 {
		t.Fatalf("split history read blocked: page=%+v err=%v", page, err)
	}
	_, err = splitExecutor.ExecuteSplit(ctx, dto.ExecuteCustomerSplitInput{})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeSplitExecutionDisabled)
}

func TestImportEvidenceDisabledContinuesBusinessImportAndMarksResult(t *testing.T) {
	db, _ := newFeaturePolicyTestDB(t)
	defer closeFeaturePolicyTestDB(t, db)
	disableFeaturePolicy(t, db, domain.CustomerResolutionFeatureImportEvidence)
	shipmentRepo, supplierRepo, fulfillRepo := buildImportFixture()
	uc := NewShipmentImportUseCase(shipmentRepo, supplierRepo, fulfillRepo, nil)
	uc = WithShipmentImportEvidence(uc, NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(db)))
	result, err := uc.ImportShipments(context.Background(), dto.ImportShipmentInput{WaveID: 1, ImportMode: "skip_invalid", Entries: threeGroupEntries()})
	if err != nil {
		t.Fatalf("business import was blocked when only evidence was disabled: %v", err)
	}
	if !result.EvidenceDisabled || result.ImportRunID != 0 || result.SuccessCount != 2 {
		t.Fatalf("disabled evidence result is misleading: %+v", result)
	}
	var runs, records int64
	_ = db.Model(&persistence.ImportRun{}).Count(&runs).Error
	_ = db.Model(&persistence.ImportRawRecord{}).Count(&records).Error
	if runs != 0 || records != 0 {
		t.Fatalf("disabled evidence leaked RAW writes runs=%d records=%d", runs, records)
	}
}

func TestCarrierRegistryWritesDisabledKeepsExistingMappingsReadOnly(t *testing.T) {
	db, _ := newFeaturePolicyTestDB(t)
	defer closeFeaturePolicyTestDB(t, db)
	ctx := context.Background()
	mappingRepo := infra.NewCarrierMappingRepository(db)
	mapping := &domain.CarrierMapping{IntegrationProfileID: 9, InternalCarrierCode: "SF", ExternalCarrierCode: "SF-EXT", ExternalCarrierName: "Existing", Aliases: "[]"}
	if err := mappingRepo.Create(ctx, mapping); err != nil {
		t.Fatal(err)
	}
	externalRepo := infra.NewExternalCarrierRepository(db)
	external := &domain.ExternalCarrier{IntegrationProfileID: 9, CanonicalKey: "code:sf-ext", ExternalCarrierCode: "SF-EXT", ExternalCarrierName: "Existing", Status: "provisional"}
	if err := externalRepo.Create(ctx, external); err != nil {
		t.Fatal(err)
	}
	disableFeaturePolicy(t, db, domain.CustomerResolutionFeatureCarrierRegistry)
	mappingUC := NewCarrierMappingUseCase(mappingRepo, infra.NewIntegrationProfileRepository(db))
	if rows, err := mappingUC.ListMappingsByProfile(ctx, 9); err != nil || len(rows) != 1 {
		t.Fatalf("existing carrier mapping read blocked: rows=%+v err=%v", rows, err)
	}
	if code, _, err := mappingUC.ResolveCarrier(ctx, 9, "SF"); err != nil || code != "SF-EXT" {
		t.Fatalf("existing carrier resolution blocked: code=%q err=%v", code, err)
	}
	_, err := mappingUC.CreateMapping(ctx, dto.CreateCarrierMappingInput{})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeCarrierRegistryDisabled)
	assertFeaturePolicyCode(t, mappingUC.DeleteMapping(ctx, mapping.ID), domain.FeaturePolicyCodeCarrierRegistryDisabled)
	externalUC := NewExternalCarrierUseCase(externalRepo)
	if rows, err := externalUC.ListByProfile(ctx, 9); err != nil || len(rows) != 1 {
		t.Fatalf("external carrier read blocked: rows=%+v err=%v", rows, err)
	}
	_, err = externalUC.RegisterExternalCarrier(ctx, dto.RegisterExternalCarrierInput{IntegrationProfileID: 9, ExternalCarrierCode: "NEW"})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeCarrierRegistryDisabled)
	_, err = externalUC.BindInternalCarrier(ctx, dto.BindInternalCarrierInput{ExternalCarrierID: external.ID, InternalCarrierCode: "SF"})
	assertFeaturePolicyCode(t, err, domain.FeaturePolicyCodeCarrierRegistryDisabled)
}

func disableFeaturePolicy(t *testing.T, db *gorm.DB, feature string) {
	t.Helper()
	repo := infra.NewCustomerResolutionFeaturePolicyRepository(db)
	current, err := repo.GetFeaturePolicy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	next := *current
	switch feature {
	case domain.CustomerResolutionFeatureWrites:
		next.CustomerResolutionWritesEnabled = false
	case domain.CustomerResolutionFeatureCandidateScan:
		next.CandidateScanEnabled = false
	case domain.CustomerResolutionFeatureMergeExecution:
		next.MergeExecutionEnabled = false
	case domain.CustomerResolutionFeatureSplitExecution:
		next.SplitExecutionEnabled = false
	case domain.CustomerResolutionFeatureImportEvidence:
		next.ImportEvidenceEnabled = false
	case domain.CustomerResolutionFeatureCarrierRegistry:
		next.CarrierRegistryWritesEnabled = false
	default:
		t.Fatalf("unknown test feature %q", feature)
	}
	if _, applied, err := repo.UpdateFeaturePolicyCAS(context.Background(), current.Revision, &next); err != nil || !applied {
		t.Fatalf("disable feature %s: applied=%v err=%v", feature, applied, err)
	}
}

func disableAllFeaturePolicy(t *testing.T, db *gorm.DB) {
	t.Helper()
	repo := infra.NewCustomerResolutionFeaturePolicyRepository(db)
	current, err := repo.GetFeaturePolicy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	next := *current
	next.CustomerResolutionWritesEnabled = false
	next.CandidateScanEnabled = false
	next.MergeExecutionEnabled = false
	next.SplitExecutionEnabled = false
	next.ImportEvidenceEnabled = false
	next.CarrierRegistryWritesEnabled = false
	if _, applied, err := repo.UpdateFeaturePolicyCAS(context.Background(), current.Revision, &next); err != nil || !applied {
		t.Fatalf("disable all features: applied=%v err=%v", applied, err)
	}
}

func assertFeaturePolicyCode(t *testing.T, err error, want string) {
	t.Helper()
	if got := featurePolicyErrorCode(err); got != want {
		t.Fatalf("feature policy error=%v code=%q want=%q", err, got, want)
	}
}

func featurePolicyErrorCode(err error) string {
	var policyErr *domain.FeaturePolicyError
	if errors.As(err, &policyErr) {
		return policyErr.Code
	}
	if err == nil {
		return ""
	}
	for _, code := range []string{domain.FeaturePolicyCodeWritesDisabled, domain.FeaturePolicyCodeCandidateScanDisabled,
		domain.FeaturePolicyCodeMergeExecutionDisabled, domain.FeaturePolicyCodeSplitExecutionDisabled,
		domain.FeaturePolicyCodeImportEvidenceDisabled, domain.FeaturePolicyCodeCarrierRegistryDisabled,
		domain.FeaturePolicyCodeRevisionConflict, domain.FeaturePolicyCodeUnavailable} {
		if strings.Contains(err.Error(), code) {
			return code
		}
	}
	return ""
}

func newFeaturePolicyTestDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "feature-policy.db")
	db, err := database.InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	return db, path
}

func closeFeaturePolicyTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
}
