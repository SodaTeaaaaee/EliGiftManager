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

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type customerMergePlan struct {
	Source                               *domain.CustomerProfile
	Target                               *domain.CustomerProfile
	Candidate                            *domain.MergeCandidate
	Evidence                             []domain.MergeEvidence
	Revision                             *domain.MergePolicyRevision
	Dependency                           *uint
	Moved                                []domain.MergeMovedEntity
	FrozenDemand                         []uint
	Counts                               dto.MergeEntityCounts
	Blockers                             []dto.MergeBlocker
	Hash                                 string
	Token                                string
	GeneratedAt                          time.Time
	PrimaryIdentityOptions               []dto.PrimaryIdentityOption
	RecommendedPrimaryIdentitySelections []dto.PrimaryIdentitySelection
	DefaultAddressOptions                []dto.DefaultAddressOption
	RecommendedDefaultAddressID          *uint
	DisplayNameOptions                   []dto.DisplayNameOption
	RecommendedDisplayNameResolution     string
}

type MergePlanBuilder struct {
	store domain.MergeExecutionStore
	graph *MergeGraphService
	now   func() time.Time
}

func NewMergePlanBuilder(store domain.MergeExecutionStore) *MergePlanBuilder {
	return &MergePlanBuilder{store: store, graph: NewMergeGraphService(store), now: func() time.Time { return time.Now().UTC() }}
}

func (b *MergePlanBuilder) Build(ctx context.Context, input dto.CustomerMergePreviewInput) (*customerMergePlan, error) {
	if input.SourceProfileID == 0 || input.TargetProfileID == 0 {
		return nil, errors.New("both source and target profile IDs are required")
	}
	if input.SourceProfileID == input.TargetProfileID {
		return nil, errors.New("cannot merge a profile into itself")
	}
	source, err := b.store.FindProfileForMerge(ctx, input.SourceProfileID, false)
	if err != nil {
		return nil, fmt.Errorf("source profile %d not found: %w", input.SourceProfileID, err)
	}
	target, err := b.store.FindProfileForMerge(ctx, input.TargetProfileID, false)
	if err != nil {
		return nil, fmt.Errorf("target profile %d not found: %w", input.TargetProfileID, err)
	}
	plan := &customerMergePlan{Source: source, Target: target, GeneratedAt: b.now()}
	dependency, graphBlockers, err := b.graph.ValidateExecution(ctx, source, target)
	if err != nil {
		return nil, err
	}
	plan.Dependency = dependency
	for _, code := range graphBlockers {
		plan.Blockers = append(plan.Blockers, mergeBlocker(code, domain.MergeEntityProfile, 0, ""))
	}
	if input.CandidateID != nil {
		candidate, evidence, revision, err := b.store.FindCandidateExecutionContext(ctx, *input.CandidateID)
		if err != nil {
			return nil, fmt.Errorf("load merge candidate %d: %w", *input.CandidateID, err)
		}
		plan.Candidate, plan.Evidence, plan.Revision = candidate, evidence, revision
		if candidate.SourceProfileID != source.ID || candidate.TargetProfileID != target.ID {
			plan.Blockers = append(plan.Blockers, mergeBlocker("candidate_pair_changed", "candidate", candidate.ID, candidate.CanonicalPairKey))
		}
		if candidate.Status != domain.MergeCandidateStatusPending {
			plan.Blockers = append(plan.Blockers, mergeBlocker("candidate_not_pending", "candidate", candidate.ID, candidate.Status))
		}
		if candidate.ExpiresAt != nil && !candidate.ExpiresAt.After(plan.GeneratedAt) {
			plan.Blockers = append(plan.Blockers, mergeBlocker("candidate_expired", "candidate", candidate.ID, ""))
		}
		var candidateBlockers []string
		if json.Unmarshal([]byte(candidate.Blockers), &candidateBlockers) == nil {
			for _, code := range candidateBlockers {
				plan.Blockers = append(plan.Blockers, mergeBlocker(code, "candidate", candidate.ID, ""))
			}
		}
		for _, item := range evidence {
			if item.Polarity == "blocker" {
				plan.Blockers = append(plan.Blockers, mergeBlocker(item.ExplanationCode, item.SourceEntityType, item.SourceEntityID, ""))
			}
		}
		if revision == nil || revision.Action != domain.MergePolicyActionSuggestOnly {
			plan.Blockers = append(plan.Blockers, mergeBlocker("policy_not_suggest_only", "policy", 0, ""))
		} else {
			var rules domain.MergePolicyRules
			if json.Unmarshal([]byte(revision.Rules), &rules) != nil || rules.ExecutionMode != domain.MergePolicyActionSuggestOnly {
				plan.Blockers = append(plan.Blockers, mergeBlocker("policy_execution_mode_invalid", "policy", revision.ID, ""))
			}
		}
	}

	sourceIdentities, err := b.store.ListIdentitiesForMerge(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list source identities: %w", err)
	}
	targetIdentities, err := b.store.ListIdentitiesForMerge(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("list target identities: %w", err)
	}
	primary, blockers := chooseIdentityPrimary(sourceIdentities, targetIdentities, input.PrimaryIdentitySelections)
	plan.PrimaryIdentityOptions, plan.RecommendedPrimaryIdentitySelections = buildPrimaryIdentityOptions(sourceIdentities, targetIdentities, primary)
	plan.Blockers = append(plan.Blockers, blockers...)
	plan.Blockers = append(plan.Blockers, stableIdentityBlockers(sourceIdentities, targetIdentities)...)
	for _, identity := range sourceIdentities {
		before := domain.MergeEntityState{ProfileID: mergeUintPtr(source.ID), IsPrimary: mergeBoolPtr(identity.IsPrimary)}
		after := domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID), IsPrimary: mergeBoolPtr(primary[identity.ID])}
		plan.addMoved(domain.MergeEntityIdentity, identity.ID, source.ID, target.ID, domain.MergeMutationReassign, domain.MergeRestoreOwnerOnly, before, after)
	}
	for _, identity := range targetIdentities {
		if identity.IsPrimary == primary[identity.ID] {
			continue
		}
		before := domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID), IsPrimary: mergeBoolPtr(identity.IsPrimary)}
		after := domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID), IsPrimary: mergeBoolPtr(primary[identity.ID])}
		plan.addMoved(domain.MergeEntityIdentity, identity.ID, target.ID, target.ID, "primary", domain.MergeRestoreExact, before, after)
	}
	plan.Counts.Identities = len(sourceIdentities)

	sourceAddresses, err := b.store.ListAddressesForMerge(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list source addresses: %w", err)
	}
	targetAddresses, err := b.store.ListAddressesForMerge(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("list target addresses: %w", err)
	}
	defaults, addressBlockers := chooseDefaultAddress(sourceAddresses, targetAddresses, input.DefaultAddressID)
	plan.DefaultAddressOptions, plan.RecommendedDefaultAddressID = buildDefaultAddressOptions(sourceAddresses, targetAddresses, defaults)
	plan.Blockers = append(plan.Blockers, addressBlockers...)
	for _, address := range sourceAddresses {
		before := domain.MergeEntityState{ProfileID: mergeUintPtr(source.ID), IsDefault: mergeBoolPtr(address.IsDefault)}
		after := domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID), IsDefault: mergeBoolPtr(defaults[address.ID])}
		plan.addMoved(domain.MergeEntityAddress, address.ID, source.ID, target.ID, domain.MergeMutationReassign, domain.MergeRestoreOwnerOnly, before, after)
	}
	for _, address := range targetAddresses {
		if address.IsDefault == defaults[address.ID] {
			continue
		}
		before := domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID), IsDefault: mergeBoolPtr(address.IsDefault)}
		after := domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID), IsDefault: mergeBoolPtr(defaults[address.ID])}
		plan.addMoved(domain.MergeEntityAddress, address.ID, target.ID, target.ID, "default", domain.MergeRestoreExact, before, after)
	}
	plan.Counts.Addresses = len(sourceAddresses)

	documents, err := b.store.ListDemandForMerge(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list source demand documents: %w", err)
	}
	for _, document := range documents {
		assigned, err := b.store.IsDemandDocumentAssigned(ctx, document.ID)
		if err != nil {
			return nil, fmt.Errorf("check demand document %d assignment: %w", document.ID, err)
		}
		if assigned {
			plan.FrozenDemand = append(plan.FrozenDemand, document.ID)
			continue
		}
		before := domain.MergeEntityState{ProfileID: mergeUintPtr(source.ID)}
		after := domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID)}
		plan.addMoved(domain.MergeEntityDemandDocument, document.ID, source.ID, target.ID, domain.MergeMutationReassign, domain.MergeRestoreOwnerOnly, before, after)
		plan.Counts.DemandDocuments++
	}

	sourceObservations, err := b.store.ListNameObservationsForMerge(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list source name observations: %w", err)
	}
	targetObservations, err := b.store.ListNameObservationsForMerge(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("list target name observations: %w", err)
	}
	targetEpisodes := map[string]struct{}{}
	for _, observation := range targetObservations {
		if observation.EpisodeKey != "" {
			targetEpisodes[observation.EpisodeKey] = struct{}{}
		}
	}
	for _, observation := range sourceObservations {
		if _, collision := targetEpisodes[observation.EpisodeKey]; collision && observation.EpisodeKey != "" {
			plan.Blockers = append(plan.Blockers, mergeBlocker("name_episode_collision", domain.MergeEntityNameObservation, observation.ID, observation.EpisodeKey))
		}
		plan.addMoved(domain.MergeEntityNameObservation, observation.ID, source.ID, target.ID, domain.MergeMutationReassign, domain.MergeRestoreOwnerOnly,
			domain.MergeEntityState{ProfileID: mergeUintPtr(source.ID)}, domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID)})
	}
	plan.Counts.NameObservations = len(sourceObservations)
	sourceEvents, err := b.store.ListNameEventsForMerge(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list source name events: %w", err)
	}
	for _, event := range sourceEvents {
		plan.addMoved(domain.MergeEntityNameEvent, event.ID, source.ID, target.ID, domain.MergeMutationReassign, domain.MergeRestoreOwnerOnly,
			domain.MergeEntityState{ProfileID: mergeUintPtr(source.ID)}, domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID)})
	}
	plan.Counts.NameEvents = len(sourceEvents)

	sourceOrigins, err := b.store.ListOriginsForMerge(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list source origins: %w", err)
	}
	targetOrigins, err := b.store.ListOriginsForMerge(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("list target origins: %w", err)
	}
	targetOriginKeys := map[string]struct{}{}
	for _, origin := range targetOrigins {
		targetOriginKeys[originMergeKey(origin)] = struct{}{}
	}
	for _, origin := range sourceOrigins {
		if _, collision := targetOriginKeys[originMergeKey(origin)]; collision {
			plan.Blockers = append(plan.Blockers, mergeBlocker("origin_collision", domain.MergeEntityOrigin, origin.ID, originMergeKey(origin)))
		}
		plan.addMoved(domain.MergeEntityOrigin, origin.ID, source.ID, target.ID, domain.MergeMutationReassign, domain.MergeRestoreOwnerOnly,
			domain.MergeEntityState{ProfileID: mergeUintPtr(source.ID)}, domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID)})
	}
	plan.Counts.Origins = len(sourceOrigins)

	displayName, mode, observationID, displayBlockers := chooseMergedDisplayName(source, target, sourceObservations, targetObservations, input.DisplayNameResolution)
	plan.DisplayNameOptions, plan.RecommendedDisplayNameResolution = buildDisplayNameOptions(source, target)
	plan.Blockers = append(plan.Blockers, displayBlockers...)
	targetBefore := domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID), RowVersion: target.RowVersion, DisplayName: target.DisplayName,
		DisplayNameMode: target.DisplayNameMode, DisplayNameObservationID: target.DisplayNameObservationID, SoftDeleted: mergeBoolPtr(false)}
	targetAfter := domain.MergeEntityState{ProfileID: mergeUintPtr(target.ID), RowVersion: target.RowVersion + 1, DisplayName: displayName,
		DisplayNameMode: mode, DisplayNameObservationID: observationID, SoftDeleted: mergeBoolPtr(false)}
	plan.addMoved(domain.MergeEntityProfile, target.ID, target.ID, target.ID, domain.MergeMutationDisplayProjection, domain.MergeRestoreExact, targetBefore, targetAfter)
	sourceBefore := domain.MergeEntityState{ProfileID: mergeUintPtr(source.ID), Status: normalizedProfileStatus(source.Status), MergedIntoProfileID: source.MergedIntoProfileID,
		RowVersion: source.RowVersion, DisplayName: source.DisplayName, DisplayNameMode: source.DisplayNameMode,
		DisplayNameObservationID: source.DisplayNameObservationID, SoftDeleted: mergeBoolPtr(false)}
	sourceAfter := sourceBefore
	sourceAfter.Status = domain.CustomerProfileStatusMerged
	sourceAfter.MergedIntoProfileID = mergeUintPtr(target.ID)
	sourceAfter.RowVersion = source.RowVersion + 1
	sourceAfter.SoftDeleted = mergeBoolPtr(true)
	plan.addMoved(domain.MergeEntityProfile, source.ID, source.ID, target.ID, domain.MergeMutationProfileState, domain.MergeRestoreExact, sourceBefore, sourceAfter)
	plan.Counts.ProfileMutations = 2

	sort.Slice(plan.FrozenDemand, func(i, j int) bool { return plan.FrozenDemand[i] < plan.FrozenDemand[j] })
	plan.Blockers = compactMergeBlockers(plan.Blockers)
	plan.Hash, err = hashMergePlan(plan)
	if err != nil {
		return nil, err
	}
	plan.Token = "v1:" + plan.Hash
	return plan, nil
}

func (p *customerMergePlan) addMoved(entityType string, entityID, from, to uint, mutation, restore string, before, after domain.MergeEntityState) {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	hash := sha256.Sum256(afterJSON)
	p.Moved = append(p.Moved, domain.MergeMovedEntity{EntityType: entityType, EntityID: entityID,
		FromProfileID: from, ToProfileID: to, MoveOrder: uint(len(p.Moved) + 1), BeforeSnapshot: string(beforeJSON),
		MutationKind: mutation, RestoreMode: restore, SnapshotVersion: 1, AfterSnapshot: string(afterJSON),
		AfterStateHash: hex.EncodeToString(hash[:]), CreatedAt: p.GeneratedAt})
}

func hashMergePlan(plan *customerMergePlan) (string, error) {
	type canonicalMoved struct {
		EntityType, MutationKind, RestoreMode           string
		EntityID, FromProfileID, ToProfileID, MoveOrder uint
		SnapshotVersion                                 uint
		BeforeSnapshot, AfterSnapshot, AfterStateHash   string
	}
	moved := make([]canonicalMoved, len(plan.Moved))
	for i := range plan.Moved {
		moved[i] = canonicalMoved{EntityType: plan.Moved[i].EntityType, MutationKind: plan.Moved[i].MutationKind,
			RestoreMode: plan.Moved[i].RestoreMode, EntityID: plan.Moved[i].EntityID,
			FromProfileID: plan.Moved[i].FromProfileID, ToProfileID: plan.Moved[i].ToProfileID,
			MoveOrder: plan.Moved[i].MoveOrder, SnapshotVersion: plan.Moved[i].SnapshotVersion,
			BeforeSnapshot: plan.Moved[i].BeforeSnapshot, AfterSnapshot: plan.Moved[i].AfterSnapshot,
			AfterStateHash: plan.Moved[i].AfterStateHash}
	}
	value := struct {
		SourceID, TargetID           uint
		SourceVersion, TargetVersion uint64
		CandidateID                  *uint
		CandidateVersion             uint64
		EvidenceHash                 string
		PolicyVersion                uint
		RevisionID                   *uint
		Dependency                   *uint
		Moved                        []canonicalMoved
		Frozen                       []uint
		Blockers                     []dto.MergeBlocker
	}{SourceID: plan.Source.ID, TargetID: plan.Target.ID, SourceVersion: plan.Source.RowVersion, TargetVersion: plan.Target.RowVersion,
		Dependency: plan.Dependency, Moved: moved, Frozen: plan.FrozenDemand, Blockers: plan.Blockers}
	if plan.Candidate != nil {
		value.CandidateID = &plan.Candidate.ID
		value.CandidateVersion = plan.Candidate.RowVersion
		value.EvidenceHash = plan.Candidate.EvidenceHash
		value.PolicyVersion = plan.Candidate.PolicyVersion
		value.RevisionID = plan.Candidate.MergePolicyRevisionID
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode merge plan: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func chooseMergedDisplayName(source, target *domain.CustomerProfile, sourceObs, targetObs []domain.CustomerNameObservation, resolution string) (string, string, *uint, []dto.MergeBlocker) {
	resolution = strings.TrimSpace(resolution)
	blockers := make([]dto.MergeBlocker, 0)
	if source.DisplayNameMode == domain.DisplayNameModePinned && target.DisplayNameMode == domain.DisplayNameModePinned && source.DisplayName != target.DisplayName && resolution == "" {
		blockers = append(blockers, mergeBlocker("double_pinned_display_name", domain.MergeEntityProfile, target.ID, "explicit keep_target or keep_source is required"))
	}
	if resolution != "" && resolution != "keep_target" && resolution != "keep_source" {
		blockers = append(blockers, mergeBlocker("invalid_display_name_resolution", domain.MergeEntityProfile, target.ID, resolution))
	}
	if resolution == "keep_target" || (resolution == "" && target.DisplayNameMode == domain.DisplayNameModePinned) {
		return target.DisplayName, target.DisplayNameMode, target.DisplayNameObservationID, blockers
	}
	if resolution == "keep_source" || (resolution == "" && source.DisplayNameMode == domain.DisplayNameModePinned) {
		return source.DisplayName, source.DisplayNameMode, source.DisplayNameObservationID, blockers
	}
	preferred := preferredObservation(append(append([]domain.CustomerNameObservation{}, targetObs...), sourceObs...))
	if preferred != nil {
		return preferred.Name, domain.DisplayNameModeAuto, &preferred.ID, blockers
	}
	return target.DisplayName, domain.DisplayNameModeAuto, target.DisplayNameObservationID, blockers
}

func compactMergeBlockers(values []dto.MergeBlocker) []dto.MergeBlocker {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		if values[i].EntityType != values[j].EntityType {
			return values[i].EntityType < values[j].EntityType
		}
		return values[i].EntityID < values[j].EntityID
	})
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		last := out[len(out)-1]
		if value.Code != last.Code || value.EntityType != last.EntityType || value.EntityID != last.EntityID {
			out = append(out, value)
		}
	}
	return out
}

func normalizedProfileStatus(status string) string {
	if status == "" {
		return domain.CustomerProfileStatusActive
	}
	return status
}

func mergeUintPtr(value uint) *uint { return &value }
func mergeBoolPtr(value bool) *bool { return &value }
