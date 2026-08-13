package controller

import (
	"errors"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
)

func TestFeaturePolicyMasterSwitchCascadesThroughDirectWailsControllers(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	profile := &domain.IntegrationProfile{
		ProfileKey: "master-gated-retail", SourceChannel: "bilibili",
		DemandKind: string(domain.DemandKindRetailOrder), IdentityStrategy: app.IdentityStrategyOrderScopedProvisional,
	}
	if err := infra.NewIntegrationProfileRepository(gdb).Create(appContext, profile); err != nil {
		t.Fatal(err)
	}

	policyRepo := infra.NewCustomerResolutionFeaturePolicyRepository(gdb)
	current, err := policyRepo.GetFeaturePolicy(appContext)
	if err != nil {
		t.Fatal(err)
	}
	next := *current
	next.CustomerResolutionWritesEnabled = false
	// Children deliberately remain stored as enabled to prove this is the
	// backend master cascade, independent of any frontend control state.
	if _, applied, err := policyRepo.UpdateFeaturePolicyCAS(appContext, current.Revision, &next); err != nil || !applied {
		t.Fatalf("disable master policy: applied=%v err=%v", applied, err)
	}

	demand := newDemandCSVImportTestController(gdb)
	profileID := profile.ID
	_, err = demand.ImportDemandDocument(dto.CreateDemandInput{
		IntegrationProfileID: &profileID, CaptureMode: "manual", SourceDocumentNo: "GATED-ORDER",
		SourceCustomerRef: "nickname",
	})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	missingCustomerProfileID := uint(999)
	_, err = demand.ImportDemandDocument(dto.CreateDemandInput{CustomerProfileID: &missingCustomerProfileID})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	_, err = demand.ImportDemandFromCSV(dto.ImportDemandTemplateInput{IntegrationProfileID: 999})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	_, err = demand.ImportDemandCSV(dto.ImportDemandCSVInput{ImportMode: "invalid", FilePath: "must-not-be-read.csv"})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	var demandDocuments, demandLines, customerProfiles, rawRuns, rawRecords int64
	_ = gdb.Model(&persistence.DemandDocument{}).Count(&demandDocuments).Error
	_ = gdb.Model(&persistence.DemandLine{}).Count(&demandLines).Error
	_ = gdb.Model(&persistence.CustomerProfile{}).Count(&customerProfiles).Error
	_ = gdb.Model(&persistence.ImportRun{}).Count(&rawRuns).Error
	_ = gdb.Model(&persistence.ImportRawRecord{}).Count(&rawRecords).Error
	if demandDocuments != 0 || demandLines != 0 || customerProfiles != 0 || rawRuns != 0 || rawRecords != 0 {
		t.Fatalf("master-off Demand entry leaked writes: docs=%d lines=%d profiles=%d runs=%d records=%d",
			demandDocuments, demandLines, customerProfiles, rawRuns, rawRecords)
	}

	governance := &MergeGovernanceController{uc: app.NewMergeGovernanceUseCase(
		infra.NewMergeGovernanceRepository(gdb),
		infra.NewProfileRepository(gdb),
		infra.NewAddressRepository(gdb),
		infra.NewCustomerProfileOriginRepository(gdb),
	)}
	_, err = governance.ScanMergeCandidates()
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeCandidateScanDisabled)
	assertRootFeaturePolicyCode(t, governance.DismissMergeCandidate(dto.DismissMergeCandidateInput{}), domain.FeaturePolicyCodeCandidateScanDisabled)

	merge := &MergeController{gdb: gdb}
	_, err = merge.MergeProfiles(dto.MergeProfilesInput{})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeMergeExecutionDisabled)
	_, err = merge.ExecuteCustomerMerge(dto.ExecuteCustomerMergeInput{})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeMergeExecutionDisabled)
	undo := &MergeUndoController{gdb: gdb}
	_, err = undo.UndoCustomerMerge(dto.UndoCustomerMergeInput{MergeID: 999})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeMergeExecutionDisabled)
	_, err = undo.ExecuteCustomerMergeUndo(dto.ExecuteCustomerMergeUndoInput{})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeMergeExecutionDisabled)

	split := &SplitController{gdb: gdb}
	_, err = split.ExecuteCustomerSplit(dto.ExecuteCustomerSplitInput{})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeSplitExecutionDisabled)

	evidence := &ImportEvidenceController{uc: app.NewImportEvidenceUseCase(infra.NewImportEvidenceRepository(gdb))}
	_, err = evidence.SetImportEvidenceRetention(dto.SetImportEvidenceRetentionInput{})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeImportEvidenceDisabled)
	_, err = evidence.PruneExpiredImportEvidence()
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeImportEvidenceDisabled)

	channel := &ChannelSyncController{
		carrierMappingUC: app.NewCarrierMappingUseCase(
			infra.NewCarrierMappingRepository(gdb),
			infra.NewIntegrationProfileRepository(gdb),
		),
		externalCarrierUC: app.NewExternalCarrierUseCase(infra.NewExternalCarrierRepository(gdb)),
	}
	_, err = channel.CreateCarrierMapping(dto.CreateCarrierMappingInput{})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeCarrierRegistryDisabled)
	assertRootFeaturePolicyCode(t, channel.DeleteCarrierMapping(1), domain.FeaturePolicyCodeCarrierRegistryDisabled)
	_, err = channel.RegisterExternalCarrier(dto.RegisterExternalCarrierInput{})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeCarrierRegistryDisabled)
	_, err = channel.BindInternalCarrier(dto.BindInternalCarrierInput{})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeCarrierRegistryDisabled)
}

func TestCustomerProfileControllerMasterSwitchBlocksEveryResolutionWriteAndPreservesIdentity(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	if err := gdb.AutoMigrate(&persistence.MergeSuggestion{}, &persistence.Wave{}, &persistence.FulfillmentLine{}); err != nil {
		t.Fatal(err)
	}
	profileRepo := infra.NewProfileRepository(gdb)
	profile := &domain.CustomerProfile{
		DisplayName: "Before", ProfileType: "member", Status: domain.CustomerProfileStatusActive,
		RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto,
	}
	if err := profileRepo.Create(appContext, profile); err != nil {
		t.Fatal(err)
	}
	identity := &domain.CustomerIdentity{
		CustomerProfileID: profile.ID, IdentityPlatform: "bilibili", IdentityValue: "uid-preserved",
		IdentityType: string(domain.IdentityTypePlatformUID), IsPrimary: true,
	}
	if err := profileRepo.CreateIdentity(appContext, identity); err != nil {
		t.Fatal(err)
	}
	addressRepo := infra.NewAddressRepository(gdb)
	address := &domain.CustomerAddress{CustomerProfileID: profile.ID, Label: "Before", RecipientName: "Before",
		IsDefault: true, ValidationStatus: "valid"}
	if err := addressRepo.Create(appContext, address); err != nil {
		t.Fatal(err)
	}
	wave := &domain.Wave{WaveNo: "feature-gate-address", Name: "feature-gate-address", WaveType: "manual", LifecycleStage: "draft"}
	if err := infra.NewWaveRepository(gdb).Create(appContext, wave); err != nil {
		t.Fatal(err)
	}
	profileID := profile.ID
	line := &domain.FulfillmentLine{WaveID: wave.ID, CustomerProfileID: &profileID, Quantity: 1,
		AddressState: string(domain.AddressStateMissing), LineReason: string(domain.LineReasonEntitlement)}
	fulfillmentRepo := infra.NewFulfillmentRepository(gdb)
	if err := fulfillmentRepo.Create(appContext, line); err != nil {
		t.Fatal(err)
	}
	suggestion := &domain.MergeSuggestion{SourceProfileID: profile.ID, TargetProfileID: profile.ID + 1, Status: "pending"}
	suggestionRepo := infra.NewMergeSuggestionRepository(gdb)
	if err := suggestionRepo.Create(appContext, suggestion); err != nil {
		t.Fatal(err)
	}

	policyRepo := infra.NewCustomerResolutionFeaturePolicyRepository(gdb)
	current, err := policyRepo.GetFeaturePolicy(appContext)
	if err != nil {
		t.Fatal(err)
	}
	next := *current
	next.CustomerResolutionWritesEnabled = false
	if _, applied, err := policyRepo.UpdateFeaturePolicyCAS(appContext, current.Revision, &next); err != nil || !applied {
		t.Fatalf("disable master policy: applied=%v err=%v", applied, err)
	}

	controller := &CustomerProfileController{uc: newCustomerProfileUseCase(gdb), db: gdb}
	_, err = controller.CreateCustomerProfile(dto.CreateCustomerProfileInput{DisplayName: "Blocked", ProfileType: "buyer"})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	_, err = controller.UpdateCustomerProfile(dto.UpdateCustomerProfileInput{
		ID: profile.ID, DisplayName: profile.DisplayName, ProfileType: "buyer", ExpectedRowVersion: profile.RowVersion,
	})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	_, err = controller.PinCustomerDisplayName(dto.PinCustomerDisplayNameInput{ProfileID: profile.ID})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	_, err = controller.UnpinCustomerDisplayName(dto.UnpinCustomerDisplayNameInput{ProfileID: profile.ID})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	assertRootFeaturePolicyCode(t, controller.DeleteCustomerProfile(profile.ID), domain.FeaturePolicyCodeWritesDisabled)
	_, err = controller.AddCustomerIdentity(dto.CreateCustomerIdentityInput{
		CustomerProfileID: profile.ID, IdentityPlatform: "other", IdentityValue: "blocked", IdentityType: "platform_uid",
	})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	assertRootFeaturePolicyCode(t, controller.DeleteCustomerIdentity(identity.ID), domain.FeaturePolicyCodeWritesDisabled)
	assertRootFeaturePolicyCode(t, controller.DismissMergeSuggestion(suggestion.ID), domain.FeaturePolicyCodeCandidateScanDisabled)

	addressController := &AddressController{
		uc:      app.NewAddressManagementUseCase(addressRepo, fulfillmentRepo),
		batchUC: app.NewAddressBatchUseCase(addressRepo, fulfillmentRepo),
	}
	_, err = addressController.CreateAddress(dto.CreateAddressInput{CustomerProfileID: profile.ID, IsDefault: true})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	_, err = addressController.UpdateAddress(dto.UpdateAddressInput{ID: address.ID, CustomerProfileID: profile.ID,
		Label: "Blocked", IsDefault: true})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	assertRootFeaturePolicyCode(t, addressController.DeleteAddress(address.ID), domain.FeaturePolicyCodeWritesDisabled)
	_, err = addressController.BindAddressToLine(dto.BindAddressInput{FulfillmentLineID: line.ID, CustomerAddressID: address.ID})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	assertRootFeaturePolicyCode(t, addressController.UnbindAddressFromLine(line.ID), domain.FeaturePolicyCodeWritesDisabled)
	_, err = addressController.BatchBindAddressToLines([]dto.BindAddressEntry{{FulfillmentLineID: line.ID, CustomerAddressID: address.ID}})
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	_, err = addressController.BindDefaultAddressesForWave(wave.ID)
	assertRootFeaturePolicyCode(t, err, domain.FeaturePolicyCodeWritesDisabled)
	if readAddress, readErr := addressController.GetAddress(address.ID); readErr != nil || readAddress.Label != "Before" {
		t.Fatalf("master-off address read blocked or changed: address=%+v err=%v", readAddress, readErr)
	}
	if rows, readErr := addressController.ListAddressesByProfile(profile.ID); readErr != nil || len(rows) != 1 {
		t.Fatalf("master-off address list blocked: rows=%+v err=%v", rows, readErr)
	}

	readBack, err := controller.GetCustomerProfile(profile.ID)
	if err != nil || readBack.DisplayName != "Before" || readBack.ProfileType != "member" || len(readBack.Identities) != 1 {
		t.Fatalf("master-off read or preserved state: profile=%+v err=%v", readBack, err)
	}
	var identityRow persistence.CustomerIdentity
	if err := gdb.First(&identityRow, identity.ID).Error; err != nil || identityRow.IdentityValue != "uid-preserved" {
		t.Fatalf("blocked DeleteCustomerIdentity changed row: identity=%+v err=%v", identityRow, err)
	}
	var addressRow persistence.CustomerAddress
	if err := gdb.First(&addressRow, address.ID).Error; err != nil || addressRow.Label != "Before" || !addressRow.IsDefault {
		t.Fatalf("blocked address write changed row: address=%+v err=%v", addressRow, err)
	}
	boundLine, err := fulfillmentRepo.FindByID(appContext, line.ID)
	if err != nil || boundLine.CustomerAddressID != nil || boundLine.AddressState != string(domain.AddressStateMissing) {
		t.Fatalf("blocked address binding changed fulfillment: line=%+v err=%v", boundLine, err)
	}
	var profileCount, identityCount, observationCount, eventCount int64
	_ = gdb.Model(&persistence.CustomerProfile{}).Count(&profileCount).Error
	_ = gdb.Model(&persistence.CustomerIdentity{}).Count(&identityCount).Error
	_ = gdb.Model(&persistence.CustomerNameObservation{}).Count(&observationCount).Error
	_ = gdb.Model(&persistence.CustomerNameEvent{}).Count(&eventCount).Error
	if profileCount != 1 || identityCount != 1 || observationCount != 0 || eventCount != 0 {
		t.Fatalf("master-off CustomerProfileController leaked writes: profiles=%d identities=%d observations=%d events=%d",
			profileCount, identityCount, observationCount, eventCount)
	}
	var suggestionRow persistence.MergeSuggestion
	if err := gdb.First(&suggestionRow, suggestion.ID).Error; err != nil || suggestionRow.Status != "pending" {
		t.Fatalf("blocked DismissMergeSuggestion changed row: suggestion=%+v err=%v", suggestionRow, err)
	}
}

func assertRootFeaturePolicyCode(t *testing.T, err error, want string) {
	t.Helper()
	var policyErr *domain.FeaturePolicyError
	if !errors.As(err, &policyErr) || policyErr.Code != want {
		t.Fatalf("feature policy error=%v code=%q want=%q", err, rootFeaturePolicyCode(policyErr), want)
	}
}

func rootFeaturePolicyCode(err *domain.FeaturePolicyError) string {
	if err == nil {
		return ""
	}
	return err.Code
}
