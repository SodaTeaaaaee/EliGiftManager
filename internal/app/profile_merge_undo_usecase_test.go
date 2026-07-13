package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProfileMergeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(
		&persistence.CustomerProfile{},
		&persistence.CustomerMergeRecord{},
		&persistence.CustomerIdentity{},
		&persistence.CustomerAddress{},
		&persistence.DemandDocument{},
		&persistence.Wave{},
		&persistence.WaveDemandAssignment{},
		&persistence.WaveParticipantSnapshot{},
		&persistence.FulfillmentLine{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return db
}

type profileMergeFixture struct {
	source        persistence.CustomerProfile
	target        persistence.CustomerProfile
	identity      persistence.CustomerIdentity
	address       persistence.CustomerAddress
	unassignedDoc persistence.DemandDocument
	assignedDoc   persistence.DemandDocument
	participant   persistence.WaveParticipantSnapshot
	fulfillment   persistence.FulfillmentLine
}

func createProfileMergeFixture(t *testing.T, db *gorm.DB) profileMergeFixture {
	t.Helper()
	f := profileMergeFixture{
		source: persistence.CustomerProfile{DisplayName: "Source", ProfileType: persistence.ProfileType("member")},
		target: persistence.CustomerProfile{DisplayName: "Target", ProfileType: persistence.ProfileType("member")},
	}
	for _, profile := range []*persistence.CustomerProfile{&f.source, &f.target} {
		if err := db.Create(profile).Error; err != nil {
			t.Fatalf("create profile: %v", err)
		}
	}
	f.identity = persistence.CustomerIdentity{
		CustomerProfileID: f.source.ID,
		IdentityPlatform:  "email",
		IdentityValue:     "source@example.com",
		IdentityType:      persistence.IdentityType("email"),
	}
	if err := db.Create(&f.identity).Error; err != nil {
		t.Fatalf("create identity: %v", err)
	}
	f.address = persistence.CustomerAddress{CustomerProfileID: f.source.ID, Label: "Original"}
	if err := db.Create(&f.address).Error; err != nil {
		t.Fatalf("create address: %v", err)
	}
	sourceID := f.source.ID
	f.unassignedDoc = persistence.DemandDocument{
		Kind: persistence.DemandKind("retail_order"), CaptureMode: persistence.CaptureMode("document_import"), CustomerProfileID: &sourceID,
	}
	f.assignedDoc = persistence.DemandDocument{
		Kind: persistence.DemandKind("retail_order"), CaptureMode: persistence.CaptureMode("document_import"), CustomerProfileID: &sourceID,
	}
	for _, document := range []*persistence.DemandDocument{&f.unassignedDoc, &f.assignedDoc} {
		if err := db.Create(document).Error; err != nil {
			t.Fatalf("create demand document: %v", err)
		}
	}
	wave := persistence.Wave{WaveNo: "MERGE-1", WaveType: persistence.WaveType("mixed")}
	if err := db.Create(&wave).Error; err != nil {
		t.Fatalf("create wave: %v", err)
	}
	if err := db.Create(&persistence.WaveDemandAssignment{WaveID: wave.ID, DemandDocumentID: f.assignedDoc.ID}).Error; err != nil {
		t.Fatalf("assign demand document: %v", err)
	}
	f.participant = persistence.WaveParticipantSnapshot{WaveID: wave.ID, CustomerProfileID: f.source.ID}
	if err := db.Create(&f.participant).Error; err != nil {
		t.Fatalf("create participant: %v", err)
	}
	f.fulfillment = persistence.FulfillmentLine{
		WaveID: wave.ID, CustomerProfileID: &sourceID, Quantity: 1, LineReason: persistence.FulfillmentLineReason("entitlement"),
	}
	if err := db.Create(&f.fulfillment).Error; err != nil {
		t.Fatalf("create fulfillment line: %v", err)
	}
	return f
}

func mergeProfilesInTransaction(t *testing.T, db *gorm.DB, sourceID, targetID uint) *dto.MergeProfilesResult {
	t.Helper()
	var result *dto.MergeProfilesResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		uc := NewProfileMergeUseCase(repos.CustomerProfile, repos.Address, repos.DemandRepo, repos.CustomerMerge)
		var err error
		result, err = uc.MergeProfiles(context.Background(), dto.MergeProfilesInput{SourceProfileID: sourceID, TargetProfileID: targetID})
		return err
	}); err != nil {
		t.Fatalf("merge profiles: %v", err)
	}
	return result
}

func undoMergeInTransaction(db *gorm.DB, mergeID uint) (*dto.UndoCustomerMergeResult, error) {
	var result *dto.UndoCustomerMergeResult
	err := db.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		uc := NewProfileMergeUndoUseCase(repos.CustomerProfile, repos.Address, repos.DemandRepo, repos.CustomerMerge)
		var err error
		result, err = uc.UndoCustomerMerge(context.Background(), dto.UndoCustomerMergeInput{MergeID: mergeID})
		return err
	})
	return result, err
}

func TestProfileMergeRecordsExactRowsAndUndoRestoresOnlyThoseRows(t *testing.T) {
	db := setupProfileMergeTestDB(t)
	f := createProfileMergeFixture(t, db)
	targetNativeIdentity := persistence.CustomerIdentity{CustomerProfileID: f.target.ID, IdentityPlatform: "email", IdentityValue: "target@example.com", IdentityType: persistence.IdentityType("email")}
	if err := db.Create(&targetNativeIdentity).Error; err != nil {
		t.Fatal(err)
	}
	result := mergeProfilesInTransaction(t, db, f.source.ID, f.target.ID)

	if result.MergeID == 0 || !result.UndoAvailable {
		t.Fatalf("expected undoable merge ID, got %+v", result)
	}
	if result.UpdatedDemandDocs != 1 || result.UpdatedParticipants != 0 || result.UpdatedFulfillmentLines != 0 {
		t.Fatalf("unexpected migration counts: %+v", result)
	}

	var record persistence.CustomerMergeRecord
	if err := db.First(&record, result.MergeID).Error; err != nil {
		t.Fatalf("load merge record: %v", err)
	}
	var payload domain.CustomerMergePayload
	if err := json.Unmarshal([]byte(record.Payload), &payload); err != nil {
		t.Fatalf("decode merge payload: %v", err)
	}
	if len(payload.IdentityIDs) != 1 || payload.IdentityIDs[0] != f.identity.ID ||
		len(payload.AddressIDs) != 1 || payload.AddressIDs[0] != f.address.ID ||
		len(payload.DemandDocumentIDs) != 1 || payload.DemandDocumentIDs[0] != f.unassignedDoc.ID {
		t.Fatalf("merge payload did not record exact moved IDs: %+v", payload)
	}

	var assigned, unassigned persistence.DemandDocument
	if err := db.First(&assigned, f.assignedDoc.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&unassigned, f.unassignedDoc.ID).Error; err != nil {
		t.Fatal(err)
	}
	if assigned.CustomerProfileID == nil || *assigned.CustomerProfileID != f.source.ID {
		t.Fatalf("assigned document moved from historical source: %+v", assigned)
	}
	if unassigned.CustomerProfileID == nil || *unassigned.CustomerProfileID != f.target.ID {
		t.Fatalf("unassigned document did not move to target: %+v", unassigned)
	}
	var participant persistence.WaveParticipantSnapshot
	if err := db.First(&participant, f.participant.ID).Error; err != nil || participant.CustomerProfileID != f.source.ID {
		t.Fatalf("participant snapshot was rewritten: %+v, %v", participant, err)
	}
	var fulfillment persistence.FulfillmentLine
	if err := db.First(&fulfillment, f.fulfillment.ID).Error; err != nil || fulfillment.CustomerProfileID == nil || *fulfillment.CustomerProfileID != f.source.ID {
		t.Fatalf("fulfillment line was rewritten: %+v, %v", fulfillment, err)
	}

	postMergeAddress := persistence.CustomerAddress{CustomerProfileID: f.target.ID, Label: "Post merge"}
	if err := db.Create(&postMergeAddress).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&persistence.CustomerAddress{}).Where("id = ?", f.address.ID).Update("label", "Edited after merge").Error; err != nil {
		t.Fatal(err)
	}

	undoResult, err := undoMergeInTransaction(db, result.MergeID)
	if err != nil {
		t.Fatalf("undo merge: %v", err)
	}
	if undoResult.RestoredIdentityCount != 1 || undoResult.RestoredAddressCount != 1 || undoResult.RestoredDemandDocumentCount != 1 {
		t.Fatalf("unexpected undo counts: %+v", undoResult)
	}
	var restoredAddress, untouchedAddress persistence.CustomerAddress
	if err := db.First(&restoredAddress, f.address.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&untouchedAddress, postMergeAddress.ID).Error; err != nil {
		t.Fatal(err)
	}
	if restoredAddress.CustomerProfileID != f.source.ID || restoredAddress.Label != "Edited after merge" {
		t.Fatalf("moved address did not follow back with edits: %+v", restoredAddress)
	}
	if untouchedAddress.CustomerProfileID != f.target.ID {
		t.Fatalf("post-merge target row moved during undo: %+v", untouchedAddress)
	}
	var untouchedIdentity persistence.CustomerIdentity
	if err := db.First(&untouchedIdentity, targetNativeIdentity.ID).Error; err != nil || untouchedIdentity.CustomerProfileID != f.target.ID {
		t.Fatalf("target-native identity moved during undo: %+v, %v", untouchedIdentity, err)
	}
	if err := db.First(&record, result.MergeID).Error; err != nil || record.UndoneAt == nil {
		t.Fatalf("merge record not marked undone: %+v, %v", record, err)
	}
	if _, err := undoMergeInTransaction(db, result.MergeID); err == nil || !strings.Contains(err.Error(), "already been undone") {
		t.Fatalf("expected already-undone error, got %v", err)
	}
}

type failingCreateMergeRecordRepo struct {
	domain.CustomerMergeRecordRepository
}

func (r failingCreateMergeRecordRepo) Create(context.Context, *domain.CustomerMergeRecord) error {
	return errors.New("forced merge record failure")
}

func TestProfileMergeRollsBackWhenRecordCreationFails(t *testing.T) {
	db := setupProfileMergeTestDB(t)
	f := createProfileMergeFixture(t, db)
	err := db.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		uc := NewProfileMergeUseCase(
			repos.CustomerProfile,
			repos.Address,
			repos.DemandRepo,
			failingCreateMergeRecordRepo{CustomerMergeRecordRepository: repos.CustomerMerge},
		)
		_, err := uc.MergeProfiles(context.Background(), dto.MergeProfilesInput{SourceProfileID: f.source.ID, TargetProfileID: f.target.ID})
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "forced merge record failure") {
		t.Fatalf("expected record creation failure, got %v", err)
	}
	var source persistence.CustomerProfile
	if err := db.First(&source, f.source.ID).Error; err != nil {
		t.Fatalf("source deletion was not rolled back: %v", err)
	}
	var identity persistence.CustomerIdentity
	if err := db.First(&identity, f.identity.ID).Error; err != nil || identity.CustomerProfileID != f.source.ID {
		t.Fatalf("identity move was not rolled back: %+v, %v", identity, err)
	}
	var count int64
	if err := db.Model(&persistence.CustomerMergeRecord{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("unexpected merge records after rollback: count=%d err=%v", count, err)
	}
}

func TestProfileMergeUndoIntegrityFailuresDoNotPartiallyRestore(t *testing.T) {
	tests := []struct {
		name      string
		breakData func(*gorm.DB, profileMergeFixture, uint)
		wantError string
	}{
		{
			name: "deleted recorded row",
			breakData: func(db *gorm.DB, f profileMergeFixture, _ uint) {
				if err := db.Delete(&persistence.CustomerAddress{}, f.address.ID).Error; err != nil {
					panic(err)
				}
			},
			wantError: "merged address",
		},
		{
			name: "missing target",
			breakData: func(db *gorm.DB, f profileMergeFixture, _ uint) {
				if err := db.Delete(&persistence.CustomerProfile{}, f.target.ID).Error; err != nil {
					panic(err)
				}
			},
			wantError: "target profile",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupProfileMergeTestDB(t)
			f := createProfileMergeFixture(t, db)
			mergeResult := mergeProfilesInTransaction(t, db, f.source.ID, f.target.ID)
			test.breakData(db, f, mergeResult.MergeID)

			if _, err := undoMergeInTransaction(db, mergeResult.MergeID); err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected %q integrity error, got %v", test.wantError, err)
			}
			var source persistence.CustomerProfile
			if err := db.Unscoped().First(&source, f.source.ID).Error; err != nil || !source.DeletedAt.Valid {
				t.Fatalf("source was partially restored: %+v, %v", source, err)
			}
			var identity persistence.CustomerIdentity
			if err := db.First(&identity, f.identity.ID).Error; err != nil || identity.CustomerProfileID != f.target.ID {
				t.Fatalf("identity was partially restored: %+v, %v", identity, err)
			}
			var record persistence.CustomerMergeRecord
			if err := db.First(&record, mergeResult.MergeID).Error; err != nil || record.UndoneAt != nil {
				t.Fatalf("record was partially marked undone: %+v, %v", record, err)
			}
		})
	}

	db := setupProfileMergeTestDB(t)
	if _, err := undoMergeInTransaction(db, 999); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing-record error, got %v", err)
	}
}

func TestProfileMergeUndoHasNoTimeWindow(t *testing.T) {
	db := setupProfileMergeTestDB(t)
	f := createProfileMergeFixture(t, db)
	result := mergeProfilesInTransaction(t, db, f.source.ID, f.target.ID)
	old := time.Now().AddDate(-10, 0, 0)
	if err := db.Model(&persistence.CustomerMergeRecord{}).Where("id = ?", result.MergeID).Update("created_at", old).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := undoMergeInTransaction(db, result.MergeID); err != nil {
		t.Fatalf("old but intact merge should be undoable: %v", err)
	}
}
