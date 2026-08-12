package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type customerSplitPlan struct {
	Source           *domain.CustomerProfile
	SourceBefore     domain.SplitEntityState
	TargetDraft      *domain.CustomerProfile
	TargetStrategy   string
	Moves            []splitPlannedMove
	Counts           dto.MergeEntityCounts
	Blockers         []dto.MergeBlocker
	ImmutableHistory domain.SplitImmutableHistoryRefs
	Hash             string
	Token            string
	GeneratedAt      time.Time
	RestoreHint      string
}

type splitPlannedMove struct {
	Moved  domain.SplitMovedEntity
	Before domain.SplitEntityState
	After  domain.SplitEntityState
}

type customerSplitPlanBuilder struct {
	store domain.SplitExecutionStore
	now   func() time.Time
}

func newCustomerSplitPlanBuilder(store domain.SplitExecutionStore) *customerSplitPlanBuilder {
	return &customerSplitPlanBuilder{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (b *customerSplitPlanBuilder) Build(ctx context.Context, input dto.CustomerSplitPreviewInput) (*customerSplitPlan, error) {
	input = normalizeSplitPreviewInput(input)
	plan := &customerSplitPlan{TargetStrategy: input.TargetStrategy, GeneratedAt: b.now(),
		RestoreHint: "restore_merged is not implemented; use the audited merge history to undo the original merge when eligible, or create a new split and use an explicit reviewed merge as the reverse operation"}
	if input.SourceProfileID == 0 {
		plan.Blockers = append(plan.Blockers, mergeBlocker("split_source_required", domain.MergeEntityProfile, 0, ""))
		return b.finish(plan, input), nil
	}
	source, err := b.store.FindProfileForSplit(ctx, input.SourceProfileID)
	if err != nil {
		return nil, fmt.Errorf("find split source profile: %w", err)
	}
	plan.Source = source
	plan.SourceBefore = domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(source.ID), Status: source.Status,
		MergedIntoProfileID: source.MergedIntoProfileID, RowVersion: source.RowVersion, DisplayName: source.DisplayName,
		DisplayNameMode: source.DisplayNameMode, DisplayNameObservationID: source.DisplayNameObservationID}
	if input.TargetStrategy != domain.SplitTargetStrategyCreateNew {
		code := "split_target_strategy_invalid"
		if input.TargetStrategy == domain.SplitTargetStrategyRestoreMerged {
			code = "split_restore_merged_not_supported"
		}
		plan.Blockers = append(plan.Blockers, mergeBlocker(code, domain.MergeEntityProfile, source.ID, plan.RestoreHint))
	}
	if source.Status != domain.CustomerProfileStatusActive || source.MergedIntoProfileID != nil {
		plan.Blockers = append(plan.Blockers, mergeBlocker("split_source_not_active_root", domain.MergeEntityProfile, source.ID, source.Status))
	}
	activeMerges, err := b.store.ListActiveMergeRecordsForSplit(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("inspect split merge graph: %w", err)
	}
	if len(activeMerges) > 0 {
		plan.Blockers = append(plan.Blockers, mergeBlocker("split_merge_graph_active", domain.MergeEntityProfile, source.ID,
			fmt.Sprintf("active merge records=%v", splitMergeRecordIDs(activeMerges))))
	}

	identities, err := b.store.ListIdentitiesForSplit(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list split identities: %w", err)
	}
	addresses, err := b.store.ListAddressesForSplit(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list split addresses: %w", err)
	}
	documents, err := b.store.ListDemandForSplit(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list split demand documents: %w", err)
	}
	observations, err := b.store.ListNameObservationsForSplit(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list split name observations: %w", err)
	}
	events, err := b.store.ListNameEventsForSplit(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list split name events: %w", err)
	}
	origins, err := b.store.ListOriginsForSplit(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list split origins: %w", err)
	}
	immutable, err := b.store.ListImmutableHistoryRefsForSplit(ctx, source.ID)
	if err != nil {
		return nil, fmt.Errorf("list immutable split history: %w", err)
	}
	plan.ImmutableHistory = immutable

	identitySelection := splitIDSet(input.Selection.IdentityIDs)
	addressSelection := splitIDSet(input.Selection.AddressIDs)
	demandSelection := splitIDSet(input.Selection.DemandDocumentIDs)
	observationSelection := splitIDSet(input.Selection.NameObservationIDs)
	originSelection := splitIDSet(input.Selection.OriginIDs)
	if len(identitySelection)+len(addressSelection)+len(demandSelection)+len(observationSelection)+len(originSelection) == 0 {
		plan.Blockers = append(plan.Blockers, mergeBlocker("split_selection_required", domain.MergeEntityProfile, source.ID, ""))
	}

	selectedIdentities, remainingIdentities := splitIdentitiesBySelection(identities, identitySelection, &plan.Blockers, source.ID)
	splitAddressesBySelection(addresses, addressSelection, &plan.Blockers, source.ID)
	selectedDocuments, remainingDocuments := splitDemandBySelection(documents, demandSelection, &plan.Blockers, source.ID)
	selectedObservations, remainingObservations := splitObservationsBySelection(observations, observationSelection, &plan.Blockers, source.ID)
	selectedOrigins, remainingOrigins := splitOriginsBySelection(origins, originSelection, &plan.Blockers, source.ID)

	b.validateIdentityAnchors(ctx, plan, selectedIdentities, remainingIdentities, selectedOrigins, remainingOrigins, selectedDocuments, remainingDocuments)
	b.planIdentityMoves(plan, identities, identitySelection, splitIDSet(input.TargetPrimaryIdentityIDs))
	b.planAddressMoves(plan, addresses, addressSelection, input.TargetDefaultAddressID)
	b.planDemandMoves(ctx, plan, selectedDocuments)
	b.planOriginMoves(plan, selectedOrigins)
	b.planNameMoves(plan, source, selectedObservations, remainingObservations, events, observationSelection, input)

	profileType := strings.TrimSpace(input.NewProfileType)
	if profileType == "" {
		profileType = source.ProfileType
	}
	if plan.TargetDraft == nil {
		plan.TargetDraft = &domain.CustomerProfile{DisplayName: source.DisplayName, ProfileType: profileType,
			Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	}
	plan.TargetDraft.ProfileType = profileType
	if strings.TrimSpace(input.NewProfileDisplayName) != "" {
		plan.TargetDraft.DisplayName = strings.TrimSpace(input.NewProfileDisplayName)
		plan.TargetDraft.DisplayNameMode = domain.DisplayNameModePinned
		if plan.TargetDraft.DisplayNameObservationID != nil {
			for i := range selectedObservations {
				if selectedObservations[i].ID == *plan.TargetDraft.DisplayNameObservationID && selectedObservations[i].Name != plan.TargetDraft.DisplayName {
					plan.Blockers = append(plan.Blockers, mergeBlocker("split_target_display_observation_mismatch",
						domain.MergeEntityNameObservation, selectedObservations[i].ID, ""))
					plan.TargetDraft.DisplayNameObservationID = nil
				}
			}
		}
	}
	if strings.TrimSpace(plan.TargetDraft.DisplayName) == "" {
		plan.Blockers = append(plan.Blockers, mergeBlocker("split_target_display_name_required", domain.MergeEntityProfile, 0, ""))
	}
	b.addProfileMoves(plan)
	return b.finish(plan, input), nil
}

func (b *customerSplitPlanBuilder) validateIdentityAnchors(
	ctx context.Context,
	plan *customerSplitPlan,
	selectedIdentities, remainingIdentities []domain.CustomerIdentity,
	selectedOrigins, remainingOrigins []domain.CustomerProfileOrigin,
	selectedDocuments, remainingDocuments []domain.DemandDocument,
) {
	selectedAnchor := len(selectedOrigins) > 0 || len(selectedDocuments) > 0
	remainingAnchor := len(remainingOrigins) > 0 || len(remainingDocuments) > 0
	groupValues := map[string]map[string]struct{}{}
	seenStrong := map[string]uint{}
	for _, identity := range selectedIdentities {
		if !isSplitOwnershipIdentity(identity) {
			continue
		}
		selectedAnchor = true
		key := splitStrongIdentityKey(identity)
		group := identityResolutionKey(identity)
		if groupValues[group] == nil {
			groupValues[group] = map[string]struct{}{}
		}
		groupValues[group][key.NormalizedValue] = struct{}{}
		canonical := group + "\x00" + key.NormalizedValue
		if previous := seenStrong[canonical]; previous != 0 && previous != identity.ID {
			plan.Blockers = append(plan.Blockers, mergeBlocker("split_duplicate_strong_identity", domain.MergeEntityIdentity, identity.ID, canonical))
		}
		seenStrong[canonical] = identity.ID
		owners, err := b.store.ListStrongIdentityOwnerIDs(ctx, key, plan.Source.ID)
		if err != nil {
			plan.Blockers = append(plan.Blockers, mergeBlocker("split_strong_identity_lookup_failed", domain.MergeEntityIdentity, identity.ID, err.Error()))
		} else if len(owners) > 0 {
			plan.Blockers = append(plan.Blockers, mergeBlocker("split_strong_identity_ambiguous", domain.MergeEntityIdentity, identity.ID, fmt.Sprint(owners)))
		}
	}
	for group, values := range groupValues {
		if len(values) > 1 {
			plan.Blockers = append(plan.Blockers, mergeBlocker("split_strong_identity_group_conflict", domain.MergeEntityIdentity, 0, group))
		}
	}
	for _, identity := range remainingIdentities {
		if isSplitOwnershipIdentity(identity) {
			remainingAnchor = true
			break
		}
	}
	if !selectedAnchor {
		plan.Blockers = append(plan.Blockers, mergeBlocker("split_target_ownership_anchor_required", domain.MergeEntityProfile, 0, ""))
	}
	if !remainingAnchor {
		plan.Blockers = append(plan.Blockers, mergeBlocker("split_source_ownership_anchor_required", domain.MergeEntityProfile, plan.Source.ID, ""))
	}
}

func (b *customerSplitPlanBuilder) planIdentityMoves(plan *customerSplitPlan, identities []domain.CustomerIdentity, selected, requestedPrimary map[uint]struct{}) {
	byGroup := map[string][]domain.CustomerIdentity{}
	for _, identity := range identities {
		byGroup[identityResolutionKey(identity)] = append(byGroup[identityResolutionKey(identity)], identity)
	}
	for requested := range requestedPrimary {
		if _, ok := selected[requested]; !ok {
			plan.Blockers = append(plan.Blockers, mergeBlocker("split_invalid_target_primary_identity", domain.MergeEntityIdentity, requested, "not selected"))
		}
	}
	for group, rows := range byGroup {
		sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
		selectedRows, remainingRows := make([]domain.CustomerIdentity, 0), make([]domain.CustomerIdentity, 0)
		for _, row := range rows {
			if _, ok := selected[row.ID]; ok {
				selectedRows = append(selectedRows, row)
			} else {
				remainingRows = append(remainingRows, row)
			}
		}
		if len(selectedRows) == 0 {
			continue
		}
		targetWinner, targetBlockers := splitIdentityPrimaryWinner(selectedRows, requestedPrimary, group, true)
		plan.Blockers = append(plan.Blockers, targetBlockers...)
		sourceWinner, sourceBlockers := splitIdentityPrimaryWinner(remainingRows, nil, group, false)
		plan.Blockers = append(plan.Blockers, sourceBlockers...)
		for _, row := range selectedRows {
			before := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(plan.Source.ID), IsPrimary: splitPlanBool(row.IsPrimary)}
			after := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(0), IsPrimary: splitPlanBool(row.ID == targetWinner)}
			plan.addMove(domain.MergeEntityIdentity, row.ID, plan.Source.ID, 0, domain.SplitMutationReassign, before, after)
			plan.Counts.Identities++
		}
		for _, row := range remainingRows {
			if row.IsPrimary == (row.ID == sourceWinner) {
				continue
			}
			before := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(plan.Source.ID), IsPrimary: splitPlanBool(row.IsPrimary)}
			after := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(plan.Source.ID), IsPrimary: splitPlanBool(row.ID == sourceWinner)}
			plan.addMove(domain.MergeEntityIdentity, row.ID, plan.Source.ID, plan.Source.ID, domain.SplitMutationPrimaryProjection, before, after)
		}
	}
}

func splitIdentityPrimaryWinner(rows []domain.CustomerIdentity, requested map[uint]struct{}, group string, target bool) (uint, []dto.MergeBlocker) {
	if len(rows) == 0 {
		return 0, nil
	}
	winner := uint(0)
	blockers := make([]dto.MergeBlocker, 0)
	if target && requested != nil {
		for _, row := range rows {
			if _, ok := requested[row.ID]; !ok {
				continue
			}
			if winner != 0 {
				blockers = append(blockers, mergeBlocker("split_multiple_target_primary_identities", domain.MergeEntityIdentity, row.ID, group))
			}
			winner = row.ID
		}
	}
	if winner == 0 {
		for _, row := range rows {
			if !row.IsPrimary {
				continue
			}
			if winner != 0 {
				blockers = append(blockers, mergeBlocker("split_invalid_multiple_primary_identities", domain.MergeEntityIdentity, row.ID, group))
			}
			winner = row.ID
		}
	}
	if winner == 0 {
		winner = rows[0].ID
	}
	return winner, blockers
}

func (b *customerSplitPlanBuilder) planAddressMoves(plan *customerSplitPlan, addresses []domain.CustomerAddress, selected map[uint]struct{}, requestedDefault *uint) {
	selectedRows, remainingRows := make([]domain.CustomerAddress, 0), make([]domain.CustomerAddress, 0)
	for _, row := range addresses {
		if _, ok := selected[row.ID]; ok {
			selectedRows = append(selectedRows, row)
		} else {
			remainingRows = append(remainingRows, row)
		}
	}
	if len(selectedRows) == 0 {
		if requestedDefault != nil {
			plan.Blockers = append(plan.Blockers, mergeBlocker("split_invalid_target_default_address", domain.MergeEntityAddress, *requestedDefault, "not selected"))
		}
		return
	}
	sort.Slice(selectedRows, func(i, j int) bool { return selectedRows[i].ID < selectedRows[j].ID })
	sort.Slice(remainingRows, func(i, j int) bool { return remainingRows[i].ID < remainingRows[j].ID })
	targetWinner, targetBlockers := splitAddressDefaultWinner(selectedRows, requestedDefault, true)
	sourceWinner, sourceBlockers := splitAddressDefaultWinner(remainingRows, nil, false)
	plan.Blockers = append(plan.Blockers, targetBlockers...)
	plan.Blockers = append(plan.Blockers, sourceBlockers...)
	for _, row := range selectedRows {
		before := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(plan.Source.ID), IsDefault: splitPlanBool(row.IsDefault)}
		after := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(0), IsDefault: splitPlanBool(row.ID == targetWinner)}
		plan.addMove(domain.MergeEntityAddress, row.ID, plan.Source.ID, 0, domain.SplitMutationReassign, before, after)
		plan.Counts.Addresses++
	}
	for _, row := range remainingRows {
		if row.IsDefault == (row.ID == sourceWinner) {
			continue
		}
		before := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(plan.Source.ID), IsDefault: splitPlanBool(row.IsDefault)}
		after := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(plan.Source.ID), IsDefault: splitPlanBool(row.ID == sourceWinner)}
		plan.addMove(domain.MergeEntityAddress, row.ID, plan.Source.ID, plan.Source.ID, domain.SplitMutationDefaultProjection, before, after)
	}
}

func splitAddressDefaultWinner(rows []domain.CustomerAddress, requested *uint, target bool) (uint, []dto.MergeBlocker) {
	if len(rows) == 0 {
		return 0, nil
	}
	winner := uint(0)
	blockers := make([]dto.MergeBlocker, 0)
	if target && requested != nil {
		for _, row := range rows {
			if row.ID == *requested {
				winner = row.ID
			}
		}
		if winner == 0 {
			blockers = append(blockers, mergeBlocker("split_invalid_target_default_address", domain.MergeEntityAddress, *requested, "not selected"))
		}
	}
	if winner == 0 {
		for _, row := range rows {
			if !row.IsDefault {
				continue
			}
			if winner != 0 {
				blockers = append(blockers, mergeBlocker("split_invalid_multiple_default_addresses", domain.MergeEntityAddress, row.ID, ""))
			}
			winner = row.ID
		}
	}
	if winner == 0 {
		winner = rows[0].ID
	}
	return winner, blockers
}

func (b *customerSplitPlanBuilder) planDemandMoves(ctx context.Context, plan *customerSplitPlan, documents []domain.DemandDocument) {
	for _, document := range documents {
		frozen, err := b.store.IsDemandDocumentFrozenForSplit(ctx, document.ID)
		if err != nil {
			plan.Blockers = append(plan.Blockers, mergeBlocker("split_demand_freeze_check_failed", domain.MergeEntityDemandDocument, document.ID, err.Error()))
		} else if frozen {
			plan.Blockers = append(plan.Blockers, mergeBlocker("split_demand_document_assigned", domain.MergeEntityDemandDocument, document.ID, "wave/fulfillment history is immutable"))
		}
		before := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(plan.Source.ID)}
		after := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(0)}
		plan.addMove(domain.MergeEntityDemandDocument, document.ID, plan.Source.ID, 0, domain.SplitMutationReassign, before, after)
		plan.Counts.DemandDocuments++
	}
}

func (b *customerSplitPlanBuilder) planOriginMoves(plan *customerSplitPlan, origins []domain.CustomerProfileOrigin) {
	seen := map[string]uint{}
	for _, origin := range origins {
		key := originMergeKey(origin)
		if previous := seen[key]; previous != 0 {
			plan.Blockers = append(plan.Blockers, mergeBlocker("split_origin_collision", domain.MergeEntityOrigin, origin.ID,
				fmt.Sprintf("duplicates origin %d", previous)))
		}
		seen[key] = origin.ID
		before := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(plan.Source.ID)}
		after := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(0)}
		plan.addMove(domain.MergeEntityOrigin, origin.ID, plan.Source.ID, 0, domain.SplitMutationReassign, before, after)
		plan.Counts.Origins++
	}
}

func (b *customerSplitPlanBuilder) planNameMoves(
	plan *customerSplitPlan,
	source *domain.CustomerProfile,
	selected, remaining []domain.CustomerNameObservation,
	events []domain.CustomerNameEvent,
	selectedIDs map[uint]struct{},
	input dto.CustomerSplitPreviewInput,
) {
	resolution := strings.TrimSpace(input.SourceDisplayNameResolution)
	if resolution != "" && resolution != "keep_current" && resolution != "auto_remaining" {
		plan.Blockers = append(plan.Blockers, mergeBlocker("split_source_display_resolution_invalid", domain.MergeEntityProfile, source.ID, resolution))
	}
	pinnedCount := 0
	for _, observation := range selected {
		if observation.IsPinned {
			pinnedCount++
		}
		before := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(source.ID)}
		after := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(0)}
		plan.addMove(domain.MergeEntityNameObservation, observation.ID, source.ID, 0, domain.SplitMutationReassign, before, after)
		plan.Counts.NameObservations++
	}
	if pinnedCount > 1 {
		plan.Blockers = append(plan.Blockers, mergeBlocker("split_multiple_pinned_name_observations", domain.MergeEntityNameObservation, 0, ""))
	}
	for _, event := range events {
		if event.ObservationID == nil {
			continue
		}
		if _, ok := selectedIDs[*event.ObservationID]; !ok {
			continue
		}
		before := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(source.ID)}
		after := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(0)}
		plan.addMove(domain.MergeEntityNameEvent, event.ID, source.ID, 0, domain.SplitMutationReassign, before, after)
		plan.Counts.NameEvents++
	}

	var targetObservation *domain.CustomerNameObservation
	if input.TargetDisplayNameObservationID != nil {
		for i := range selected {
			if selected[i].ID == *input.TargetDisplayNameObservationID {
				targetObservation = &selected[i]
				break
			}
		}
		if targetObservation == nil {
			plan.Blockers = append(plan.Blockers, mergeBlocker("split_target_display_observation_not_selected",
				domain.MergeEntityNameObservation, *input.TargetDisplayNameObservationID, ""))
		}
	} else if preferred := preferredObservation(selected); preferred != nil {
		targetObservation = preferred
	}
	target := &domain.CustomerProfile{DisplayName: source.DisplayName, ProfileType: source.ProfileType,
		Status: domain.CustomerProfileStatusActive, RowVersion: 1, DisplayNameMode: domain.DisplayNameModeAuto}
	if targetObservation != nil {
		target.DisplayName = targetObservation.Name
		target.DisplayNameObservationID = splitPlanUint(targetObservation.ID)
		if targetObservation.IsPinned {
			target.DisplayNameMode = domain.DisplayNameModePinned
		}
	}
	plan.TargetDraft = target

	selectedCurrent := source.DisplayNameObservationID != nil
	if selectedCurrent {
		_, selectedCurrent = selectedIDs[*source.DisplayNameObservationID]
	}
	if selectedCurrent && resolution != "auto_remaining" {
		plan.Blockers = append(plan.Blockers, mergeBlocker("split_source_display_projection_required", domain.MergeEntityProfile,
			source.ID, "select auto_remaining because the current display-name episode is moving"))
	}
	if selectedCurrent && resolution == "auto_remaining" {
		preferred := preferredObservation(remaining)
		if preferred == nil {
			source.DisplayNameObservationID = nil
			source.DisplayNameMode = domain.DisplayNameModeAuto
		} else {
			source.DisplayName = preferred.Name
			source.DisplayNameObservationID = splitPlanUint(preferred.ID)
			source.DisplayNameMode = domain.DisplayNameModeAuto
		}
	}
}

func (b *customerSplitPlanBuilder) addProfileMoves(plan *customerSplitPlan) {
	sourceBefore := plan.SourceBefore
	sourceAfter := sourceBefore
	sourceAfter.RowVersion++
	sourceAfter.DisplayName = plan.Source.DisplayName
	sourceAfter.DisplayNameMode = plan.Source.DisplayNameMode
	sourceAfter.DisplayNameObservationID = plan.Source.DisplayNameObservationID
	plan.addMove(domain.MergeEntityProfile, plan.Source.ID, plan.Source.ID, plan.Source.ID, domain.SplitMutationSourceProjection, sourceBefore, sourceAfter)

	targetAfter := domain.SplitEntityState{Exists: true, ProfileID: splitPlanUint(0), Status: domain.CustomerProfileStatusActive,
		RowVersion: 1, DisplayName: plan.TargetDraft.DisplayName, DisplayNameMode: plan.TargetDraft.DisplayNameMode,
		DisplayNameObservationID: plan.TargetDraft.DisplayNameObservationID}
	plan.addMove(domain.MergeEntityProfile, 0, plan.Source.ID, 0, domain.SplitMutationTargetCreated,
		domain.SplitEntityState{Exists: false}, targetAfter)
	plan.Counts.ProfileMutations = 2
}

func (b *customerSplitPlanBuilder) finish(plan *customerSplitPlan, input dto.CustomerSplitPreviewInput) *customerSplitPlan {
	plan.Blockers = compactMergeBlockers(plan.Blockers)
	hashInput := struct {
		SourceID         uint
		SourceVersion    uint64
		TargetStrategy   string
		TargetDisplay    string
		TargetType       string
		Input            dto.CustomerSplitPreviewInput
		Moves            []domain.SplitMovedEntity
		Blockers         []dto.MergeBlocker
		ImmutableHistory domain.SplitImmutableHistoryRefs
	}{TargetStrategy: plan.TargetStrategy, Input: input, Blockers: plan.Blockers, ImmutableHistory: plan.ImmutableHistory}
	if plan.Source != nil {
		hashInput.SourceID = plan.Source.ID
		hashInput.SourceVersion = plan.Source.RowVersion
	}
	if plan.TargetDraft != nil {
		hashInput.TargetDisplay = plan.TargetDraft.DisplayName
		hashInput.TargetType = plan.TargetDraft.ProfileType
	}
	for _, move := range plan.Moves {
		hashInput.Moves = append(hashInput.Moves, move.Moved)
	}
	encoded, _ := json.Marshal(hashInput)
	sum := sha256.Sum256(encoded)
	plan.Hash = hex.EncodeToString(sum[:])
	plan.Token = "split-v1:" + plan.Hash
	return plan
}

func (plan *customerSplitPlan) addMove(entityType string, entityID, fromProfileID, toProfileID uint, mutationKind string, before, after domain.SplitEntityState) {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	afterSum := sha256.Sum256(afterJSON)
	plan.Moves = append(plan.Moves, splitPlannedMove{Moved: domain.SplitMovedEntity{
		EntityType: entityType, EntityID: entityID, FromProfileID: fromProfileID, ToProfileID: toProfileID,
		MoveOrder: uint(len(plan.Moves) + 1), BeforeSnapshot: string(beforeJSON), AfterSnapshot: string(afterJSON),
		AfterStateHash: hex.EncodeToString(afterSum[:]), MutationKind: mutationKind, SnapshotVersion: 1,
	}, Before: before, After: after})
}

func (plan *customerSplitPlan) bindTarget(targetID uint) {
	for i := range plan.Moves {
		move := &plan.Moves[i]
		if move.Moved.MutationKind == domain.SplitMutationTargetCreated {
			move.Moved.EntityID = targetID
			move.Moved.ToProfileID = targetID
			move.After.ProfileID = splitPlanUint(targetID)
		} else if move.Moved.ToProfileID == 0 {
			move.Moved.ToProfileID = targetID
			if move.After.ProfileID != nil && *move.After.ProfileID == 0 {
				move.After.ProfileID = splitPlanUint(targetID)
			}
		}
		afterJSON, _ := json.Marshal(move.After)
		move.Moved.AfterSnapshot = string(afterJSON)
		sum := sha256.Sum256(afterJSON)
		move.Moved.AfterStateHash = hex.EncodeToString(sum[:])
	}
	plan.TargetDraft.ID = targetID
}

func normalizeSplitPreviewInput(input dto.CustomerSplitPreviewInput) dto.CustomerSplitPreviewInput {
	input.TargetStrategy = strings.TrimSpace(input.TargetStrategy)
	if input.TargetStrategy == "" {
		input.TargetStrategy = domain.SplitTargetStrategyCreateNew
	}
	input.NewProfileDisplayName = strings.TrimSpace(input.NewProfileDisplayName)
	input.NewProfileType = strings.TrimSpace(input.NewProfileType)
	input.SourceDisplayNameResolution = strings.TrimSpace(input.SourceDisplayNameResolution)
	input.Selection.IdentityIDs = sortedUniqueUint(input.Selection.IdentityIDs)
	input.Selection.AddressIDs = sortedUniqueUint(input.Selection.AddressIDs)
	input.Selection.DemandDocumentIDs = sortedUniqueUint(input.Selection.DemandDocumentIDs)
	input.Selection.NameObservationIDs = sortedUniqueUint(input.Selection.NameObservationIDs)
	input.Selection.OriginIDs = sortedUniqueUint(input.Selection.OriginIDs)
	input.TargetPrimaryIdentityIDs = sortedUniqueUint(input.TargetPrimaryIdentityIDs)
	return input
}

func sortedUniqueUint(values []uint) []uint {
	set := splitIDSet(values)
	result := make([]uint, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func splitIDSet(values []uint) map[uint]struct{} {
	result := make(map[uint]struct{}, len(values))
	for _, value := range values {
		if value != 0 {
			result[value] = struct{}{}
		}
	}
	return result
}

func splitIdentitiesBySelection(rows []domain.CustomerIdentity, selected map[uint]struct{}, blockers *[]dto.MergeBlocker, profileID uint) ([]domain.CustomerIdentity, []domain.CustomerIdentity) {
	chosen, remaining := make([]domain.CustomerIdentity, 0), make([]domain.CustomerIdentity, 0)
	found := map[uint]struct{}{}
	for _, row := range rows {
		if _, ok := selected[row.ID]; ok {
			chosen = append(chosen, row)
			found[row.ID] = struct{}{}
		} else {
			remaining = append(remaining, row)
		}
	}
	appendMissingSplitSelections(selected, found, domain.MergeEntityIdentity, profileID, blockers)
	return chosen, remaining
}

func splitAddressesBySelection(rows []domain.CustomerAddress, selected map[uint]struct{}, blockers *[]dto.MergeBlocker, profileID uint) ([]domain.CustomerAddress, []domain.CustomerAddress) {
	chosen, remaining := make([]domain.CustomerAddress, 0), make([]domain.CustomerAddress, 0)
	found := map[uint]struct{}{}
	for _, row := range rows {
		if _, ok := selected[row.ID]; ok {
			chosen = append(chosen, row)
			found[row.ID] = struct{}{}
		} else {
			remaining = append(remaining, row)
		}
	}
	appendMissingSplitSelections(selected, found, domain.MergeEntityAddress, profileID, blockers)
	return chosen, remaining
}

func splitDemandBySelection(rows []domain.DemandDocument, selected map[uint]struct{}, blockers *[]dto.MergeBlocker, profileID uint) ([]domain.DemandDocument, []domain.DemandDocument) {
	chosen, remaining := make([]domain.DemandDocument, 0), make([]domain.DemandDocument, 0)
	found := map[uint]struct{}{}
	for _, row := range rows {
		if _, ok := selected[row.ID]; ok {
			chosen = append(chosen, row)
			found[row.ID] = struct{}{}
		} else {
			remaining = append(remaining, row)
		}
	}
	appendMissingSplitSelections(selected, found, domain.MergeEntityDemandDocument, profileID, blockers)
	return chosen, remaining
}

func splitObservationsBySelection(rows []domain.CustomerNameObservation, selected map[uint]struct{}, blockers *[]dto.MergeBlocker, profileID uint) ([]domain.CustomerNameObservation, []domain.CustomerNameObservation) {
	chosen, remaining := make([]domain.CustomerNameObservation, 0), make([]domain.CustomerNameObservation, 0)
	found := map[uint]struct{}{}
	for _, row := range rows {
		if _, ok := selected[row.ID]; ok {
			chosen = append(chosen, row)
			found[row.ID] = struct{}{}
		} else {
			remaining = append(remaining, row)
		}
	}
	appendMissingSplitSelections(selected, found, domain.MergeEntityNameObservation, profileID, blockers)
	return chosen, remaining
}

func splitOriginsBySelection(rows []domain.CustomerProfileOrigin, selected map[uint]struct{}, blockers *[]dto.MergeBlocker, profileID uint) ([]domain.CustomerProfileOrigin, []domain.CustomerProfileOrigin) {
	chosen, remaining := make([]domain.CustomerProfileOrigin, 0), make([]domain.CustomerProfileOrigin, 0)
	found := map[uint]struct{}{}
	for _, row := range rows {
		if _, ok := selected[row.ID]; ok {
			chosen = append(chosen, row)
			found[row.ID] = struct{}{}
		} else {
			remaining = append(remaining, row)
		}
	}
	appendMissingSplitSelections(selected, found, domain.MergeEntityOrigin, profileID, blockers)
	return chosen, remaining
}

func appendMissingSplitSelections(selected, found map[uint]struct{}, entityType string, profileID uint, blockers *[]dto.MergeBlocker) {
	for id := range selected {
		if _, ok := found[id]; !ok {
			*blockers = append(*blockers, mergeBlocker("split_entity_not_owned", entityType, id, fmt.Sprintf("source profile %d", profileID)))
		}
	}
}

func isSplitOwnershipIdentity(identity domain.CustomerIdentity) bool {
	if identityResolutionValue(identity) == "" {
		return false
	}
	if isStrongMergeIdentity(identity) {
		return true
	}
	return strings.EqualFold(identity.IdentityType, string(domain.IdentityTypeEmail)) &&
		strings.EqualFold(identity.VerificationStatus, "verified")
}

func splitStrongIdentityKey(identity domain.CustomerIdentity) domain.SplitIdentityKey {
	return domain.SplitIdentityKey{Namespace: strings.SplitN(identityResolutionKey(identity), "\x00", 2)[0],
		IdentityType:    strings.ToLower(strings.TrimSpace(identity.IdentityType)),
		NormalizedValue: identityResolutionValue(identity)}
}

func splitMergeRecordIDs(records []domain.CustomerMergeRecord) []uint {
	result := make([]uint, len(records))
	for i := range records {
		result[i] = records[i].ID
	}
	return result
}

func splitPlanUint(value uint) *uint { return &value }
func splitPlanBool(value bool) *bool { return &value }
