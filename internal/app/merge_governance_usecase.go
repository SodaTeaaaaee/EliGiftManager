package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

var (
	ErrMergePolicyRevisionConflict = errors.New("merge policy revision conflict")
	ErrMergeCandidateChanged       = errors.New("merge candidate evidence or policy version changed")
)

type LegacyMergeSettings struct {
	AutoMergeCrossPlatform bool
	AutoMergeByEmail       bool
	AutoMergeByPhone       bool
}

type MergeGovernanceUseCase struct {
	repo        domain.MergeGovernanceRepository
	profileRepo domain.CustomerProfileRepository
	addressRepo domain.CustomerAddressRepository
	originRepo  domain.CustomerProfileOriginRepository
	now         func() time.Time
}

func NewMergeGovernanceUseCase(
	repo domain.MergeGovernanceRepository,
	profileRepo domain.CustomerProfileRepository,
	addressRepo domain.CustomerAddressRepository,
	originRepo domain.CustomerProfileOriginRepository,
) *MergeGovernanceUseCase {
	return &MergeGovernanceUseCase{repo: repo, profileRepo: profileRepo, addressRepo: addressRepo, originRepo: originRepo,
		now: func() time.Time { return time.Now().UTC() }}
}

func (uc *MergeGovernanceUseCase) MigrateLegacyPolicy(ctx context.Context, legacy LegacyMergeSettings) (bool, error) {
	rules := domain.MergePolicyRules{
		SchemaVersion: 1, CandidateDetectionEnabled: legacy.AutoMergeCrossPlatform,
		EmailEvidenceMode: boolMode(legacy.AutoMergeByEmail, domain.MergeEvidenceModeLegacyRawExact),
		PhoneEvidenceMode: boolMode(legacy.AutoMergeByPhone, domain.MergeEvidenceModeLegacyRawExact),
		ExecutionMode:     domain.MergePolicyActionSuggestOnly,
	}
	rulesJSON, checksum, err := encodePolicyRules(rules)
	if err != nil {
		return false, err
	}
	now := uc.now()
	policy := &domain.MergePolicy{PolicyKey: domain.MergePolicyKeyDefault, Name: "Customer merge review",
		IsActive: true, DefaultAction: domain.MergePolicyActionSuggestOnly, RowVersion: 1, NeedsScan: true,
		ExtraData: `{"migrationSource":"legacy_settings_json"}`, CreatedAt: now, UpdatedAt: now}
	revision := &domain.MergePolicyRevision{Revision: 1, Action: domain.MergePolicyActionSuggestOnly,
		Rules: rulesJSON, Checksum: checksum, CreatedBy: "legacy_settings_migration", SchemaVersion: 1, CreatedAt: now}
	return uc.repo.EnsurePolicy(ctx, policy, revision)
}

func (uc *MergeGovernanceUseCase) GetPolicy(ctx context.Context) (*dto.MergePolicyDTO, error) {
	policy, revision, err := uc.repo.FindPolicyByKey(ctx, domain.MergePolicyKeyDefault)
	if err != nil {
		return nil, err
	}
	return policyDTO(policy, revision)
}

func (uc *MergeGovernanceUseCase) UpdatePolicy(ctx context.Context, input dto.UpdateMergePolicyInput) (*dto.MergePolicyDTO, error) {
	rules, err := rulesFromDTO(input.Rules)
	if err != nil {
		return nil, err
	}
	rulesJSON, checksum, err := encodePolicyRules(rules)
	if err != nil {
		return nil, err
	}
	actor := strings.TrimSpace(input.ActorRef)
	if actor == "" {
		actor = "local_user"
	}
	revision := &domain.MergePolicyRevision{Action: domain.MergePolicyActionSuggestOnly,
		Rules: rulesJSON, Checksum: checksum, CreatedBy: actor, SchemaVersion: 1, CreatedAt: uc.now()}
	policy, swapped, err := uc.repo.UpdatePolicyCAS(ctx, domain.MergePolicyKeyDefault, input.ExpectedRevision, revision)
	if err != nil {
		return nil, err
	}
	if !swapped {
		return nil, ErrMergePolicyRevisionConflict
	}
	return policyDTO(policy, revision)
}

func (uc *MergeGovernanceUseCase) ScanMergeCandidates(ctx context.Context) (_ *dto.MergeScanRunDTO, returnErr error) {
	if err := requireCustomerResolutionFeature(ctx, uc.repo, domain.CustomerResolutionFeatureCandidateScan); err != nil {
		return nil, err
	}
	policy, revision, err := uc.repo.FindPolicyByKey(ctx, domain.MergePolicyKeyDefault)
	if err != nil {
		return nil, err
	}
	rules, err := decodePolicyRules(revision.Rules)
	if err != nil {
		return nil, err
	}
	now := uc.now()
	run := &domain.MergeScanRun{MergePolicyID: policy.ID, PolicyRevisionID: revision.ID, PolicyVersion: revision.Revision,
		Status: domain.MergeScanStatusRunning, StartedAt: now, CreatedAt: now, UpdatedAt: now}
	if err := uc.repo.CreateScanRun(ctx, run); err != nil {
		return nil, err
	}
	defer func() {
		if returnErr == nil {
			return
		}
		finished := uc.now()
		run.Status = domain.MergeScanStatusFailed
		run.CompletedAt = &finished
		run.ErrorMessage = returnErr.Error()
		run.UpdatedAt = finished
		_ = uc.repo.UpdateScanRun(context.Background(), run)
	}()

	profiles, err := uc.loadScanProfiles(ctx)
	if err != nil {
		return nil, err
	}
	run.ProfilesScanned = uint(len(profiles))
	if rules.CandidateDetectionEnabled {
		for i := 0; i < len(profiles); i++ {
			for j := i + 1; j < len(profiles); j++ {
				run.PairsEvaluated++
				candidate, evidence := evaluateProfilePair(profiles[i], profiles[j], rules, revision, run.ID, uc.now())
				if candidate == nil {
					continue
				}
				created, err := uc.repo.UpsertCandidateEvaluation(ctx, candidate, evidence)
				if err != nil {
					return nil, err
				}
				if created {
					run.CandidatesCreated++
				} else {
					run.CandidatesUpdated++
				}
				if candidate.Status == domain.MergeCandidateStatusBlocked {
					run.CandidatesBlocked++
				}
			}
		}
	}
	if err := uc.repo.MarkUnseenCandidatesStale(ctx, revision.Revision, run.ID); err != nil {
		return nil, err
	}
	finished := uc.now()
	run.Status = domain.MergeScanStatusCompleted
	run.CompletedAt = &finished
	run.UpdatedAt = finished
	if err := uc.repo.UpdateScanRun(ctx, run); err != nil {
		return nil, err
	}
	if err := uc.repo.CompletePolicyScan(ctx, policy.ID, revision.ID, finished); err != nil {
		return nil, err
	}
	result := scanRunDTO(run)
	return &result, nil
}

func (uc *MergeGovernanceUseCase) GetScanRun(ctx context.Context, id uint) (*dto.MergeScanRunDTO, error) {
	run, err := uc.repo.FindScanRun(ctx, id)
	if err != nil {
		return nil, err
	}
	result := scanRunDTO(run)
	return &result, nil
}

func (uc *MergeGovernanceUseCase) ListCandidates(ctx context.Context, status string) ([]dto.MergeCandidateDTO, error) {
	if status != "" && !validCandidateStatus(status) {
		return nil, fmt.Errorf("invalid merge candidate status %q", status)
	}
	candidates, err := uc.repo.ListCandidates(ctx, status)
	if err != nil {
		return nil, err
	}
	result := make([]dto.MergeCandidateDTO, 0, len(candidates))
	for i := range candidates {
		result = append(result, candidateDTO(&candidates[i], nil))
	}
	return result, nil
}

func (uc *MergeGovernanceUseCase) GetCandidate(ctx context.Context, id uint) (*dto.MergeCandidateDTO, error) {
	candidate, evidence, err := uc.repo.FindCandidateWithEvidence(ctx, id)
	if err != nil {
		return nil, err
	}
	result := candidateDTO(candidate, evidence)
	return &result, nil
}

func (uc *MergeGovernanceUseCase) DismissCandidate(ctx context.Context, input dto.DismissMergeCandidateInput) error {
	if err := requireCustomerResolutionFeature(ctx, uc.repo, domain.CustomerResolutionFeatureCandidateScan); err != nil {
		return err
	}
	if input.ID == 0 || input.EvidenceHash == "" || input.PolicyVersion == 0 {
		return errors.New("candidate id, evidenceHash, and policyVersion are required")
	}
	dismissed, err := uc.repo.DismissCandidate(ctx, input.ID, input.EvidenceHash, input.PolicyVersion)
	if err != nil {
		return err
	}
	if !dismissed {
		return ErrMergeCandidateChanged
	}
	return nil
}

type mergeScanProfile struct {
	profile    domain.CustomerProfile
	identities []domain.CustomerIdentity
	addresses  []domain.CustomerAddress
	origins    []domain.CustomerProfileOrigin
}

func (uc *MergeGovernanceUseCase) loadScanProfiles(ctx context.Context) ([]mergeScanProfile, error) {
	profiles, err := uc.profileRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]mergeScanProfile, 0, len(profiles))
	for i := range profiles {
		if profiles[i].Status != "" && profiles[i].Status != domain.CustomerProfileStatusActive {
			continue
		}
		identities, err := uc.profileRepo.ListIdentitiesByProfile(ctx, profiles[i].ID)
		if err != nil {
			return nil, err
		}
		addresses, err := uc.addressRepo.ListByProfile(ctx, profiles[i].ID)
		if err != nil {
			return nil, err
		}
		origins, err := uc.originRepo.ListByProfile(ctx, profiles[i].ID)
		if err != nil {
			return nil, err
		}
		result = append(result, mergeScanProfile{profile: profiles[i], identities: identities, addresses: addresses, origins: origins})
	}
	return result, nil
}

func evaluateProfilePair(a, b mergeScanProfile, rules domain.MergePolicyRules, revision *domain.MergePolicyRevision, scanRunID uint, now time.Time) (*domain.MergeCandidate, []domain.MergeEvidence) {
	evidence := make([]domain.MergeEvidence, 0)
	evidence = append(evidence, stableIdentityEvidence(a, b, now)...)
	if rules.EmailEvidenceMode != domain.MergeEvidenceModeOff {
		evidence = append(evidence, emailEvidence(a, b, rules.EmailEvidenceMode, now)...)
	}
	if rules.PhoneEvidenceMode != domain.MergeEvidenceModeOff {
		evidence = append(evidence, phoneAndAddressEvidence(a, b, rules.PhoneEvidenceMode, now)...)
	}
	positives, blockers := 0, make([]string, 0)
	for i := range evidence {
		if evidence[i].Polarity == "blocker" {
			blockers = append(blockers, evidence[i].ExplanationCode)
		} else {
			positives++
		}
	}
	if positives == 0 {
		return nil, nil
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].EvidenceKey < evidence[j].EvidenceKey })
	evidence = compactEvidence(evidence)
	keys := make([]string, len(evidence))
	for i := range evidence {
		keys[i] = evidence[i].EvidenceKey
	}
	evidenceHash := digest(strings.Join(keys, "\n"))
	sort.Strings(blockers)
	blockers = compactStrings(blockers)
	blockersJSON, _ := json.Marshal(blockers)
	source, target := selectMergeDirection(a, b)
	status := domain.MergeCandidateStatusPending
	explanation := firstPositiveExplanation(evidence)
	confidence := evidenceConfidence(evidence)
	if len(blockers) > 0 {
		status = domain.MergeCandidateStatusBlocked
		explanation = "stable_identity_conflict"
		confidence = 0
	}
	pairKey := canonicalPairKey(a.profile.ID, b.profile.ID)
	revisionID, runID := revision.ID, scanRunID
	candidate := &domain.MergeCandidate{SourceProfileID: source.profile.ID, TargetProfileID: target.profile.ID,
		Status: status, Score: confidence, Confidence: confidence, MergePolicyRevisionID: &revisionID,
		Reason: explanation, ExplanationCode: explanation, CanonicalPairKey: pairKey, EvidenceHash: evidenceHash,
		PolicyVersion: revision.Revision, Blockers: string(blockersJSON), LastEvaluatedAt: &now, ScanRunID: &runID,
		CreatedAt: now, UpdatedAt: now}
	return candidate, evidence
}

func stableIdentityEvidence(a, b mergeScanProfile, now time.Time) []domain.MergeEvidence {
	type fact struct {
		value string
		id    uint
	}
	left, right := map[string][]fact{}, map[string][]fact{}
	collect := func(identities []domain.CustomerIdentity, into map[string][]fact) {
		for i := range identities {
			identityType := strings.ToLower(strings.TrimSpace(identities[i].IdentityType))
			if identityType != string(domain.IdentityTypePlatformUID) && identityType != string(domain.IdentityTypeExternalBuyerID) {
				continue
			}
			namespace := strings.ToLower(strings.TrimSpace(identities[i].Namespace))
			if namespace == "" {
				namespace = strings.ToLower(strings.TrimSpace(identities[i].IdentityPlatform))
			}
			value := strings.TrimSpace(identities[i].NormalizedValue)
			if value == "" {
				value = strings.TrimSpace(identities[i].IdentityValue)
			}
			if namespace != "" && value != "" {
				into[namespace+"\x00"+identityType] = append(into[namespace+"\x00"+identityType], fact{value: value, id: identities[i].ID})
			}
		}
	}
	collect(a.identities, left)
	collect(b.identities, right)
	result := []domain.MergeEvidence{}
	for key, leftFacts := range left {
		rightFacts := right[key]
		if len(rightFacts) == 0 {
			continue
		}
		matched := false
		for _, l := range leftFacts {
			for _, r := range rightFacts {
				if l.value != r.value {
					continue
				}
				matched = true
				valueHash := digest(l.value)
				result = append(result, newEvidence("stable_identity", "positive", "stable_identity_match", valueHash,
					maskOpaque(l.value), "customer_identity", minUint(l.id, r.id), 1, now, key))
			}
		}
		if !matched {
			conflictHash := digest(leftFacts[0].value + "\x00" + rightFacts[0].value)
			result = append(result, newEvidence("stable_identity", "blocker", "stable_identity_conflict", conflictHash,
				"conflicting stable identifiers", "customer_identity", minUint(leftFacts[0].id, rightFacts[0].id), 1, now, key))
		}
	}
	return result
}

func emailEvidence(a, b mergeScanProfile, mode string, now time.Time) []domain.MergeEvidence {
	type fact struct {
		value string
		id    uint
	}
	collect := func(profile mergeScanProfile) map[string]fact {
		result := map[string]fact{}
		for _, identity := range profile.identities {
			if !strings.EqualFold(identity.IdentityType, string(domain.IdentityTypeEmail)) {
				continue
			}
			value := identity.IdentityValue
			if mode == domain.MergeEvidenceModeNormalizedVerified {
				if !strings.EqualFold(identity.VerificationStatus, "verified") {
					continue
				}
				value = identity.NormalizedValue
				if value == "" {
					value = strings.ToLower(strings.TrimSpace(identity.IdentityValue))
				}
			}
			if value != "" {
				result[value] = fact{value: value, id: identity.ID}
			}
		}
		return result
	}
	left, right := collect(a), collect(b)
	result := []domain.MergeEvidence{}
	for value, l := range left {
		if r, ok := right[value]; ok {
			code, confidence := "legacy_email_exact", 0.7
			if mode == domain.MergeEvidenceModeNormalizedVerified {
				code, confidence = "verified_email_match", 0.9
			}
			result = append(result, newEvidence("email", "positive", code, digest(value), maskEmail(value),
				"customer_identity", minUint(l.id, r.id), confidence, now, mode))
		}
	}
	return result
}

func phoneAndAddressEvidence(a, b mergeScanProfile, mode string, now time.Time) []domain.MergeEvidence {
	type fact struct {
		value string
		id    uint
	}
	phones := func(profile mergeScanProfile) map[string]fact {
		result := map[string]fact{}
		for _, address := range profile.addresses {
			if invalidMergeAddress(address) {
				continue
			}
			value := address.Phone
			if mode == domain.MergeEvidenceModeNormalized {
				value = address.NormalizedPhone
				if value == "" {
					value = normalizePhone(address.Phone)
				}
			}
			if value != "" {
				result[value] = fact{value: value, id: address.ID}
			}
		}
		return result
	}
	fingerprints := func(profile mergeScanProfile) map[string]fact {
		result := map[string]fact{}
		for _, address := range profile.addresses {
			if !invalidMergeAddress(address) && address.AddressFingerprint != "" {
				result[address.AddressFingerprint] = fact{value: address.AddressFingerprint, id: address.ID}
			}
		}
		return result
	}
	result := []domain.MergeEvidence{}
	for value, l := range phones(a) {
		if r, ok := phones(b)[value]; ok {
			code := "legacy_phone_exact"
			if mode == domain.MergeEvidenceModeNormalized {
				code = "normalized_phone_match"
			}
			result = append(result, newEvidence("phone", "positive", code, digest(value), maskPhone(value),
				"customer_address", minUint(l.id, r.id), 0.55, now, mode))
		}
	}
	for value, l := range fingerprints(a) {
		if r, ok := fingerprints(b)[value]; ok {
			result = append(result, newEvidence("address", "positive", "address_fingerprint_match", digest(value),
				maskOpaque(value), "customer_address", minUint(l.id, r.id), 0.6, now, "fingerprint"))
		}
	}
	return result
}

func newEvidence(kind, polarity, code, valueHash, masked, entityType string, entityID uint, confidence float64, now time.Time, scope string) domain.MergeEvidence {
	key := digest(strings.Join([]string{kind, polarity, code, valueHash, scope}, "\x00"))
	return domain.MergeEvidence{EvidenceKind: kind, EvidenceKey: key, Polarity: polarity, ExplanationCode: code,
		SourceRef: valueHash, ValueHash: valueHash, MaskedValue: masked, SourceEntityType: entityType,
		SourceEntityID: entityID, Weight: confidence, Confidence: confidence, Payload: "{}", ObservedAt: &now, CreatedAt: now}
}

func selectMergeDirection(a, b mergeScanProfile) (mergeScanProfile, mergeScanProfile) {
	if targetPreferred(a, b) {
		return b, a
	}
	return a, b
}

func targetPreferred(a, b mergeScanProfile) bool {
	aActive, bActive := isActiveProfile(a.profile), isActiveProfile(b.profile)
	if aActive != bActive {
		return aActive
	}
	aProvisional, bProvisional := isProvisionalProfile(a), isProvisionalProfile(b)
	if aProvisional != bProvisional {
		return !aProvisional
	}
	aStrong, bStrong := hasStrongIdentity(a), hasStrongIdentity(b)
	if aStrong != bStrong {
		return aStrong
	}
	aComplete, bComplete := profileCompleteness(a), profileCompleteness(b)
	if aComplete != bComplete {
		return aComplete > bComplete
	}
	if !a.profile.CreatedAt.Equal(b.profile.CreatedAt) {
		if a.profile.CreatedAt.IsZero() {
			return false
		}
		if b.profile.CreatedAt.IsZero() {
			return true
		}
		return a.profile.CreatedAt.Before(b.profile.CreatedAt)
	}
	return a.profile.ID < b.profile.ID
}

func isActiveProfile(profile domain.CustomerProfile) bool {
	return profile.Status == "" || profile.Status == domain.CustomerProfileStatusActive
}

func isProvisionalProfile(profile mergeScanProfile) bool {
	for i := range profile.origins {
		if profile.origins[i].IsProvisional {
			return true
		}
	}
	return false
}

func hasStrongIdentity(profile mergeScanProfile) bool {
	for i := range profile.identities {
		if profile.identities[i].IdentityType == string(domain.IdentityTypePlatformUID) ||
			profile.identities[i].IdentityType == string(domain.IdentityTypeExternalBuyerID) {
			return true
		}
	}
	return false
}

func profileCompleteness(profile mergeScanProfile) int {
	score := 0
	if strings.TrimSpace(profile.profile.DisplayName) != "" {
		score++
	}
	if strings.TrimSpace(profile.profile.ProfileType) != "" {
		score++
	}
	if len(profile.identities) > 0 {
		score++
	}
	for i := range profile.addresses {
		if !invalidMergeAddress(profile.addresses[i]) {
			score++
			break
		}
	}
	return score
}

func invalidMergeAddress(address domain.CustomerAddress) bool {
	return address.IsTest || strings.EqualFold(address.ValidationStatus, string(domain.AddressValidationStatusInvalid)) ||
		strings.EqualFold(address.QualityStatus, "invalid")
}

func firstPositiveExplanation(evidence []domain.MergeEvidence) string {
	priority := []string{"stable_identity_match", "verified_email_match", "legacy_email_exact", "address_fingerprint_match", "normalized_phone_match", "legacy_phone_exact"}
	for _, code := range priority {
		for i := range evidence {
			if evidence[i].Polarity != "blocker" && evidence[i].ExplanationCode == code {
				return code
			}
		}
	}
	return "review_evidence"
}

func evidenceConfidence(evidence []domain.MergeEvidence) float64 {
	confidence := 0.0
	for i := range evidence {
		if evidence[i].Polarity != "blocker" && evidence[i].Confidence > confidence {
			confidence = evidence[i].Confidence
		}
	}
	return confidence
}

func rulesFromDTO(input dto.MergePolicyRulesDTO) (domain.MergePolicyRules, error) {
	rules := domain.MergePolicyRules{SchemaVersion: 1, CandidateDetectionEnabled: input.CandidateDetectionEnabled,
		EmailEvidenceMode: input.EmailEvidenceMode, PhoneEvidenceMode: input.PhoneEvidenceMode,
		ExecutionMode: domain.MergePolicyActionSuggestOnly}
	if input.ExecutionMode != "" && input.ExecutionMode != domain.MergePolicyActionSuggestOnly {
		return rules, errors.New("merge execution mode is fixed to suggest_only")
	}
	if !oneOf(rules.EmailEvidenceMode, domain.MergeEvidenceModeOff, domain.MergeEvidenceModeLegacyRawExact, domain.MergeEvidenceModeNormalizedVerified) {
		return rules, fmt.Errorf("invalid email evidence mode %q", rules.EmailEvidenceMode)
	}
	if !oneOf(rules.PhoneEvidenceMode, domain.MergeEvidenceModeOff, domain.MergeEvidenceModeLegacyRawExact, domain.MergeEvidenceModeNormalized) {
		return rules, fmt.Errorf("invalid phone evidence mode %q", rules.PhoneEvidenceMode)
	}
	return rules, nil
}

func encodePolicyRules(rules domain.MergePolicyRules) (string, string, error) {
	rules.ExecutionMode = domain.MergePolicyActionSuggestOnly
	data, err := json.Marshal(rules)
	if err != nil {
		return "", "", err
	}
	return string(data), digest(string(data)), nil
}

func decodePolicyRules(raw string) (domain.MergePolicyRules, error) {
	var rules domain.MergePolicyRules
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return rules, fmt.Errorf("decode merge policy rules: %w", err)
	}
	if rules.ExecutionMode != domain.MergePolicyActionSuggestOnly {
		return rules, errors.New("stored merge policy execution mode is not suggest_only")
	}
	return rules, nil
}

func policyDTO(policy *domain.MergePolicy, revision *domain.MergePolicyRevision) (*dto.MergePolicyDTO, error) {
	rules, err := decodePolicyRules(revision.Rules)
	if err != nil {
		return nil, err
	}
	return &dto.MergePolicyDTO{ID: policy.ID, PolicyKey: policy.PolicyKey, Revision: revision.Revision,
		Rules: dto.MergePolicyRulesDTO{CandidateDetectionEnabled: rules.CandidateDetectionEnabled,
			EmailEvidenceMode: rules.EmailEvidenceMode, PhoneEvidenceMode: rules.PhoneEvidenceMode,
			ExecutionMode: domain.MergePolicyActionSuggestOnly},
		NeedsScan: policy.NeedsScan, LastScanAt: policy.LastScanAt, RevisionTime: revision.CreatedAt}, nil
}

func candidateDTO(candidate *domain.MergeCandidate, evidence []domain.MergeEvidence) dto.MergeCandidateDTO {
	blockers := []string{}
	_ = json.Unmarshal([]byte(candidate.Blockers), &blockers)
	evidenceDTO := make([]dto.MergeEvidenceDTO, len(evidence))
	for i := range evidence {
		evidenceDTO[i] = dto.MergeEvidenceDTO{ID: evidence[i].ID, EvidenceKind: evidence[i].EvidenceKind,
			Polarity: evidence[i].Polarity, ExplanationCode: evidence[i].ExplanationCode,
			Confidence: evidence[i].Confidence, ValueHash: evidence[i].ValueHash,
			MaskedValue: evidence[i].MaskedValue, ObservedAt: evidence[i].ObservedAt}
	}
	return dto.MergeCandidateDTO{ID: candidate.ID, SourceProfileID: candidate.SourceProfileID,
		TargetProfileID: candidate.TargetProfileID, Status: candidate.Status, Confidence: candidate.Confidence,
		ExplanationCode: candidate.ExplanationCode, EvidenceHash: candidate.EvidenceHash,
		PolicyVersion: candidate.PolicyVersion, PolicyRevisionID: candidate.MergePolicyRevisionID,
		BlockerCodes: blockers, LastEvaluatedAt: candidate.LastEvaluatedAt, ExpiresAt: candidate.ExpiresAt,
		Evidence: evidenceDTO, CreatedAt: candidate.CreatedAt, UpdatedAt: candidate.UpdatedAt,
		RowVersion: candidate.RowVersion}
}

func scanRunDTO(run *domain.MergeScanRun) dto.MergeScanRunDTO {
	return dto.MergeScanRunDTO{ID: run.ID, PolicyVersion: run.PolicyVersion, Status: run.Status,
		StartedAt: run.StartedAt, CompletedAt: run.CompletedAt, ProfilesScanned: run.ProfilesScanned,
		PairsEvaluated: run.PairsEvaluated, CandidatesCreated: run.CandidatesCreated,
		CandidatesUpdated: run.CandidatesUpdated, CandidatesBlocked: run.CandidatesBlocked,
		ErrorMessage: run.ErrorMessage}
}

func boolMode(enabled bool, mode string) string {
	if !enabled {
		return domain.MergeEvidenceModeOff
	}
	return mode
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func validCandidateStatus(status string) bool {
	return oneOf(status, domain.MergeCandidateStatusPending, domain.MergeCandidateStatusBlocked,
		domain.MergeCandidateStatusDismissed, domain.MergeCandidateStatusStale,
		domain.MergeCandidateStatusSuperseded, domain.MergeCandidateStatusExpired, domain.MergeCandidateStatusFailed,
		domain.MergeCandidateStatusExecuted)
}

func canonicalPairKey(a, b uint) string {
	if a > b {
		a, b = b, a
	}
	return fmt.Sprintf("%d:%d", a, b)
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func maskEmail(value string) string {
	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "***"
	}
	return string([]rune(parts[0])[:1]) + "***@" + parts[1]
}

func maskPhone(value string) string {
	runes := []rune(value)
	if len(runes) <= 4 {
		return "***"
	}
	return "***" + string(runes[len(runes)-4:])
}

func maskOpaque(value string) string {
	runes := []rune(value)
	if len(runes) <= 4 {
		return "***"
	}
	return "***" + string(runes[len(runes)-4:])
}

func normalizePhone(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || (r == '+' && len(value) > 0) {
			return r
		}
		return -1
	}, strings.TrimSpace(value))
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func compactEvidence(values []domain.MergeEvidence) []domain.MergeEvidence {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value.EvidenceKey != result[len(result)-1].EvidenceKey {
			result = append(result, value)
		}
	}
	return result
}

func minUint(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}
