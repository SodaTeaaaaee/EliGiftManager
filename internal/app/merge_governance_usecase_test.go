package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	database "github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateLegacyMergePolicyPreservesAllBooleanCombinations(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		mask := mask
		t.Run(fmt.Sprintf("mask_%d", mask), func(t *testing.T) {
			gdb, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:merge-policy-%d?mode=memory&cache=shared", mask)), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			if err := gdb.AutoMigrate(&persistence.MergePolicy{}, &persistence.MergePolicyRevision{}); err != nil {
				t.Fatal(err)
			}
			uc := NewMergeGovernanceUseCase(infra.NewMergeGovernanceRepository(gdb), nil, nil, nil)
			legacy := LegacyMergeSettings{AutoMergeCrossPlatform: mask&1 != 0, AutoMergeByEmail: mask&2 != 0, AutoMergeByPhone: mask&4 != 0}
			created, err := uc.MigrateLegacyPolicy(context.Background(), legacy)
			if err != nil || !created {
				t.Fatalf("first migration: created=%v err=%v", created, err)
			}
			got, err := uc.GetPolicy(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if got.Rules.CandidateDetectionEnabled != legacy.AutoMergeCrossPlatform {
				t.Fatalf("master flag changed: %+v", got.Rules)
			}
			if (got.Rules.EmailEvidenceMode == domain.MergeEvidenceModeLegacyRawExact) != legacy.AutoMergeByEmail {
				t.Fatalf("email raw-exact mode changed: %+v", got.Rules)
			}
			if (got.Rules.PhoneEvidenceMode == domain.MergeEvidenceModeLegacyRawExact) != legacy.AutoMergeByPhone {
				t.Fatalf("phone raw-exact mode changed: %+v", got.Rules)
			}
			if got.Rules.ExecutionMode != domain.MergePolicyActionSuggestOnly {
				t.Fatalf("unsafe execution mode: %+v", got.Rules)
			}
			created, err = uc.MigrateLegacyPolicy(context.Background(), LegacyMergeSettings{})
			if err != nil || created {
				t.Fatalf("migration was not idempotent: created=%v err=%v", created, err)
			}
			sqlDB, _ := gdb.DB()
			_ = sqlDB.Close()
		})
	}
}

func TestMergeGovernancePolicyCASExplicitScanAndCandidateIdempotency(t *testing.T) {
	gdb, uc, profileRepo, originRepo := newMergeGovernanceFixture(t)
	ctx := context.Background()
	if _, err := uc.MigrateLegacyPolicy(ctx, LegacyMergeSettings{AutoMergeCrossPlatform: true, AutoMergeByEmail: true}); err != nil {
		t.Fatal(err)
	}

	p1 := createMergeTestProfile(t, profileRepo, "provisional")
	p2 := createMergeTestProfile(t, profileRepo, "durable")
	p3 := createMergeTestProfile(t, profileRepo, "newer canonical")
	p4 := createMergeTestProfile(t, profileRepo, "older canonical")
	createMergeTestIdentity(t, profileRepo, p1.ID, "shop", "uid-one", "platform_uid", "observed")
	createMergeTestIdentity(t, profileRepo, p2.ID, "shop", "uid-two", "platform_uid", "observed")
	createMergeTestIdentity(t, profileRepo, p1.ID, "email", "Same@Example.test", "email", "unverified")
	createMergeTestIdentity(t, profileRepo, p2.ID, "email", "Same@Example.test", "email", "unverified")
	createMergeTestIdentity(t, profileRepo, p3.ID, "membership", "shared-stable-secret", "platform_uid", "observed")
	createMergeTestIdentity(t, profileRepo, p4.ID, "membership", "shared-stable-secret", "platform_uid", "observed")
	if err := originRepo.Create(ctx, &domain.CustomerProfileOrigin{CustomerProfileID: p1.ID, OriginKind: domain.CustomerOriginKindRetailOrder,
		ExternalRef: "order-1", IsProvisional: true}); err != nil {
		t.Fatal(err)
	}
	older, newer := time.Now().UTC().Add(-48*time.Hour), time.Now().UTC()
	if err := gdb.Model(&persistence.CustomerProfile{}).Where("id = ?", p3.ID).Update("created_at", newer).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Model(&persistence.CustomerProfile{}).Where("id = ?", p4.ID).Update("created_at", older).Error; err != nil {
		t.Fatal(err)
	}

	var runCount int64
	if err := gdb.Model(&persistence.MergeScanRun{}).Count(&runCount).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := uc.GetPolicy(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.ListCandidates(ctx, ""); err != nil {
		t.Fatal(err)
	}
	var afterRead int64
	_ = gdb.Model(&persistence.MergeScanRun{}).Count(&afterRead).Error
	if afterRead != runCount {
		t.Fatalf("read implicitly scanned: before=%d after=%d", runCount, afterRead)
	}

	run, err := uc.ScanMergeCandidates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != domain.MergeScanStatusCompleted || run.CandidatesCreated != 2 || run.CandidatesBlocked != 1 {
		t.Fatalf("unexpected first scan: %+v", run)
	}
	candidates, err := uc.ListCandidates(ctx, "")
	if err != nil || len(candidates) != 2 {
		t.Fatalf("list candidates: len=%d err=%v", len(candidates), err)
	}
	var blocked, stable dto.MergeCandidateDTO
	for _, candidate := range candidates {
		switch candidate.ExplanationCode {
		case "stable_identity_conflict":
			blocked = candidate
		case "stable_identity_match":
			stable = candidate
		}
	}
	if blocked.Status != domain.MergeCandidateStatusBlocked || blocked.TargetProfileID != p2.ID {
		t.Fatalf("blocker or active/provisional target selection failed: %+v", blocked)
	}
	if stable.Status != domain.MergeCandidateStatusPending || stable.TargetProfileID != p4.ID || p4.ID < p3.ID {
		t.Fatalf("created-at target selection fell back to min id: p3=%d p4=%d candidate=%+v", p3.ID, p4.ID, stable)
	}
	detail, err := uc.GetCandidate(ctx, stable.ID)
	if err != nil || len(detail.Evidence) == 0 {
		t.Fatalf("candidate detail: %+v err=%v", detail, err)
	}
	for _, item := range detail.Evidence {
		if strings.Contains(item.MaskedValue, "shared-stable-secret") || item.ValueHash == "" {
			t.Fatalf("sensitive evidence was not masked/hashed: %+v", item)
		}
	}

	second, err := uc.ScanMergeCandidates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if second.CandidatesCreated != 0 || second.CandidatesUpdated != 2 {
		t.Fatalf("scan was not idempotent: %+v", second)
	}
	if err := uc.DismissCandidate(ctx, dto.DismissMergeCandidateInput{ID: stable.ID, EvidenceHash: "wrong", PolicyVersion: stable.PolicyVersion}); !errors.Is(err, ErrMergeCandidateChanged) {
		t.Fatalf("stale dismiss should fail, got %v", err)
	}
	if err := uc.DismissCandidate(ctx, dto.DismissMergeCandidateInput{ID: stable.ID, EvidenceHash: stable.EvidenceHash, PolicyVersion: stable.PolicyVersion}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.ScanMergeCandidates(ctx); err != nil {
		t.Fatal(err)
	}
	detail, err = uc.GetCandidate(ctx, stable.ID)
	if err != nil || detail.Status != domain.MergeCandidateStatusDismissed {
		t.Fatalf("exact evidence dismissal did not survive rescan: %+v err=%v", detail, err)
	}

	policy, err := uc.UpdatePolicy(ctx, dto.UpdateMergePolicyInput{ExpectedRevision: 1,
		Rules: dto.MergePolicyRulesDTO{CandidateDetectionEnabled: true, EmailEvidenceMode: domain.MergeEvidenceModeNormalizedVerified,
			PhoneEvidenceMode: domain.MergeEvidenceModeNormalized, ExecutionMode: domain.MergePolicyActionSuggestOnly}})
	if err != nil || policy.Revision != 2 || !policy.NeedsScan {
		t.Fatalf("update policy: %+v err=%v", policy, err)
	}
	if _, err := uc.UpdatePolicy(ctx, dto.UpdateMergePolicyInput{ExpectedRevision: 1, Rules: policy.Rules}); !errors.Is(err, ErrMergePolicyRevisionConflict) {
		t.Fatalf("stale CAS should conflict, got %v", err)
	}
	var runsAfterUpdate int64
	_ = gdb.Model(&persistence.MergeScanRun{}).Count(&runsAfterUpdate).Error
	if runsAfterUpdate != 3 {
		t.Fatalf("policy save implicitly scanned: runs=%d", runsAfterUpdate)
	}
}

func TestMergeEvidenceRulesIgnoreNamesAndInvalidAddresses(t *testing.T) {
	now := time.Now().UTC()
	rules := domain.MergePolicyRules{CandidateDetectionEnabled: true, EmailEvidenceMode: domain.MergeEvidenceModeOff,
		PhoneEvidenceMode: domain.MergeEvidenceModeNormalized, ExecutionMode: domain.MergePolicyActionSuggestOnly}
	revision := &domain.MergePolicyRevision{ID: 1, Revision: 1}
	a := mergeScanProfile{profile: domain.CustomerProfile{ID: 1, DisplayName: "same"}, addresses: []domain.CustomerAddress{{
		ID: 1, RecipientName: "same recipient", Phone: "123-456", NormalizedPhone: "123456", IsTest: true,
	}}}
	b := mergeScanProfile{profile: domain.CustomerProfile{ID: 2, DisplayName: "same"}, addresses: []domain.CustomerAddress{{
		ID: 2, RecipientName: "same recipient", Phone: "123456", NormalizedPhone: "123456", ValidationStatus: "invalid",
	}}}
	if candidate, _ := evaluateProfilePair(a, b, rules, revision, 1, now); candidate != nil {
		t.Fatalf("name/recipient/test/invalid address created candidate: %+v", candidate)
	}
	a.addresses[0].IsTest = false
	a.addresses[0].ValidationStatus = "valid"
	b.addresses[0].ValidationStatus = "valid"
	if candidate, evidence := evaluateProfilePair(a, b, rules, revision, 1, now); candidate == nil || candidate.Status != domain.MergeCandidateStatusPending || len(evidence) == 0 {
		t.Fatalf("normalized phone should create review-only candidate: candidate=%+v evidence=%+v", candidate, evidence)
	}

	legacyEmailRules := rules
	legacyEmailRules.PhoneEvidenceMode = domain.MergeEvidenceModeOff
	legacyEmailRules.EmailEvidenceMode = domain.MergeEvidenceModeLegacyRawExact
	a = mergeScanProfile{profile: domain.CustomerProfile{ID: 1}, identities: []domain.CustomerIdentity{{
		ID: 1, IdentityType: "email", IdentityValue: "Case@Example.test", NormalizedValue: "case@example.test", VerificationStatus: "verified",
	}, {ID: 3, IdentityType: "username", IdentityValue: "same nickname"}}}
	b = mergeScanProfile{profile: domain.CustomerProfile{ID: 2}, identities: []domain.CustomerIdentity{{
		ID: 2, IdentityType: "email", IdentityValue: "case@example.test", NormalizedValue: "case@example.test", VerificationStatus: "verified",
	}, {ID: 4, IdentityType: "username", IdentityValue: "same nickname"}}}
	if candidate, _ := evaluateProfilePair(a, b, legacyEmailRules, revision, 1, now); candidate != nil {
		t.Fatalf("legacy raw-exact email mode normalized values or used nickname: %+v", candidate)
	}
	verifiedRules := legacyEmailRules
	verifiedRules.EmailEvidenceMode = domain.MergeEvidenceModeNormalizedVerified
	if candidate, evidence := evaluateProfilePair(a, b, verifiedRules, revision, 1, now); candidate == nil || candidate.ExplanationCode != "verified_email_match" || len(evidence) != 1 {
		t.Fatalf("verified normalized email did not match: candidate=%+v evidence=%+v", candidate, evidence)
	}
	b.identities[0].VerificationStatus = "observed"
	if candidate, _ := evaluateProfilePair(a, b, verifiedRules, revision, 1, now); candidate != nil {
		t.Fatalf("unverified normalized email created candidate: %+v", candidate)
	}

	a = mergeScanProfile{profile: domain.CustomerProfile{ID: 1}, addresses: []domain.CustomerAddress{{ID: 1, AddressFingerprint: "fingerprint-secret", ValidationStatus: "valid"}}}
	b = mergeScanProfile{profile: domain.CustomerProfile{ID: 2}, addresses: []domain.CustomerAddress{{ID: 2, AddressFingerprint: "fingerprint-secret", ValidationStatus: "valid"}}}
	if candidate, evidence := evaluateProfilePair(a, b, rules, revision, 1, now); candidate == nil || candidate.Status != domain.MergeCandidateStatusPending || len(evidence) != 1 || evidence[0].ExplanationCode != "address_fingerprint_match" {
		t.Fatalf("address fingerprint should be review-only evidence: candidate=%+v evidence=%+v", candidate, evidence)
	}
}

func newMergeGovernanceFixture(t *testing.T) (*gorm.DB, *MergeGovernanceUseCase, domain.CustomerProfileRepository, domain.CustomerProfileOriginRepository) {
	t.Helper()
	gdb, err := database.InitDB(filepath.Join(t.TempDir(), "merge-governance.db"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := gdb.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	profileRepo := infra.NewProfileRepository(gdb)
	originRepo := infra.NewCustomerProfileOriginRepository(gdb)
	uc := NewMergeGovernanceUseCase(infra.NewMergeGovernanceRepository(gdb), profileRepo,
		infra.NewAddressRepository(gdb), originRepo)
	return gdb, uc, profileRepo, originRepo
}

func createMergeTestProfile(t *testing.T, repo domain.CustomerProfileRepository, name string) domain.CustomerProfile {
	t.Helper()
	profile := domain.CustomerProfile{DisplayName: name, ProfileType: string(domain.ProfileTypeMember),
		Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	if err := repo.Create(context.Background(), &profile); err != nil {
		t.Fatal(err)
	}
	return profile
}

func createMergeTestIdentity(t *testing.T, repo domain.CustomerProfileRepository, profileID uint, namespace, value, identityType, verification string) {
	t.Helper()
	identity := domain.CustomerIdentity{CustomerProfileID: profileID, IdentityPlatform: namespace,
		IdentityValue: value, IdentityType: identityType, Namespace: namespace, NormalizedValue: strings.ToLower(strings.TrimSpace(value)),
		VerificationStatus: verification}
	if err := repo.CreateIdentity(context.Background(), &identity); err != nil {
		t.Fatal(err)
	}
}
