package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type mergeExecutionStore struct{ db *gorm.DB }

var _ domain.MergeExecutionStore = (*mergeExecutionStore)(nil)

func NewMergeExecutionStore(db *gorm.DB) domain.MergeExecutionStore {
	return &mergeExecutionStore{db: db}
}

func (r *mergeExecutionStore) FindProfileForMerge(ctx context.Context, id uint, includeDeleted bool) (*domain.CustomerProfile, error) {
	query := r.db.WithContext(ctx)
	if includeDeleted {
		query = query.Unscoped()
	}
	var row persistence.CustomerProfile
	if err := query.First(&row, id).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerProfileToDomain(&row), nil
}

func (r *mergeExecutionStore) ListIdentitiesForMerge(ctx context.Context, profileID uint) ([]domain.CustomerIdentity, error) {
	var rows []persistence.CustomerIdentity
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CustomerIdentity, len(rows))
	for i := range rows {
		out[i] = *persistence.CustomerIdentityToDomain(&rows[i])
	}
	return out, nil
}

func (r *mergeExecutionStore) ListAddressesForMerge(ctx context.Context, profileID uint) ([]domain.CustomerAddress, error) {
	var rows []persistence.CustomerAddress
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CustomerAddress, len(rows))
	for i := range rows {
		out[i] = *persistence.CustomerAddressToDomain(&rows[i])
	}
	return out, nil
}

func (r *mergeExecutionStore) ListDemandForMerge(ctx context.Context, profileID uint) ([]domain.DemandDocument, error) {
	var rows []persistence.DemandDocument
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.DemandDocument, len(rows))
	for i := range rows {
		out[i] = *persistence.DemandDocumentToDomain(&rows[i])
	}
	return out, nil
}

func (r *mergeExecutionStore) ListNameObservationsForMerge(ctx context.Context, profileID uint) ([]domain.CustomerNameObservation, error) {
	var rows []persistence.CustomerNameObservation
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CustomerNameObservation, len(rows))
	for i := range rows {
		out[i] = *persistence.CustomerNameObservationToDomain(&rows[i])
	}
	return out, nil
}

func (r *mergeExecutionStore) ListNameEventsForMerge(ctx context.Context, profileID uint) ([]domain.CustomerNameEvent, error) {
	var rows []persistence.CustomerNameEvent
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CustomerNameEvent, len(rows))
	for i := range rows {
		out[i] = *persistence.CustomerNameEventToDomain(&rows[i])
	}
	return out, nil
}

func (r *mergeExecutionStore) ListOriginsForMerge(ctx context.Context, profileID uint) ([]domain.CustomerProfileOrigin, error) {
	var rows []persistence.CustomerProfileOrigin
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CustomerProfileOrigin, len(rows))
	for i := range rows {
		out[i] = *persistence.CustomerProfileOriginToDomain(&rows[i])
	}
	return out, nil
}

func (r *mergeExecutionStore) FindCandidateExecutionContext(ctx context.Context, id uint) (*domain.MergeCandidate, []domain.MergeEvidence, *domain.MergePolicyRevision, error) {
	var candidate persistence.MergeCandidate
	if err := r.db.WithContext(ctx).First(&candidate, id).Error; err != nil {
		return nil, nil, nil, err
	}
	var evidenceRows []persistence.MergeEvidence
	if err := r.db.WithContext(ctx).Where("merge_candidate_id = ?", id).Order("id").Find(&evidenceRows).Error; err != nil {
		return nil, nil, nil, err
	}
	evidence := make([]domain.MergeEvidence, len(evidenceRows))
	for i := range evidenceRows {
		evidence[i] = *persistence.MergeEvidenceToDomain(&evidenceRows[i])
	}
	if candidate.MergePolicyRevisionID == nil {
		return persistence.MergeCandidateToDomain(&candidate), evidence, nil, nil
	}
	var revision persistence.MergePolicyRevision
	if err := r.db.WithContext(ctx).First(&revision, *candidate.MergePolicyRevisionID).Error; err != nil {
		return nil, nil, nil, err
	}
	return persistence.MergeCandidateToDomain(&candidate), evidence, persistence.MergePolicyRevisionToDomain(&revision), nil
}

func (r *mergeExecutionStore) FindMergeRecord(ctx context.Context, id uint) (*domain.CustomerMergeRecord, error) {
	var row persistence.CustomerMergeRecord
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return mergeRecordToDomain(&row), nil
}

func (r *mergeExecutionStore) FindMergeRecordByOperationKey(ctx context.Context, operationKey string) (*domain.CustomerMergeRecord, error) {
	var row persistence.CustomerMergeRecord
	if err := r.db.WithContext(ctx).Where("operation_key = ?", operationKey).First(&row).Error; err != nil {
		return nil, err
	}
	return mergeRecordToDomain(&row), nil
}

func (r *mergeExecutionStore) FindMergeRecordByUndoOperationKey(ctx context.Context, operationKey string) (*domain.CustomerMergeRecord, error) {
	var row persistence.CustomerMergeRecord
	if err := r.db.WithContext(ctx).Where("undo_operation_key = ?", operationKey).First(&row).Error; err != nil {
		return nil, err
	}
	return mergeRecordToDomain(&row), nil
}

func (r *mergeExecutionStore) ListActiveMergeRecords(ctx context.Context, profileIDs []uint) ([]domain.CustomerMergeRecord, error) {
	if len(profileIDs) == 0 {
		return []domain.CustomerMergeRecord{}, nil
	}
	var rows []persistence.CustomerMergeRecord
	if err := r.db.WithContext(ctx).
		Where("undone_at IS NULL AND status = ?", domain.MergeRecordStatusCompleted).
		Where("source_profile_id IN ? OR target_profile_id IN ?", profileIDs, profileIDs).
		Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CustomerMergeRecord, len(rows))
	for i := range rows {
		out[i] = *mergeRecordToDomain(&rows[i])
	}
	return out, nil
}

func (r *mergeExecutionStore) ListMergeRecords(ctx context.Context, filter domain.MergeHistoryFilter) ([]domain.CustomerMergeRecord, error) {
	query := r.db.WithContext(ctx).Model(&persistence.CustomerMergeRecord{})
	if filter.ProfileID != 0 {
		query = query.Where("source_profile_id = ? OR target_profile_id = ?", filter.ProfileID, filter.ProfileID)
	}
	if filter.CandidateID != 0 {
		query = query.Where("merge_candidate_id = ?", filter.CandidateID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.BeforeCreatedAt != nil {
		query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", *filter.BeforeCreatedAt, *filter.BeforeCreatedAt, filter.BeforeID)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows []persistence.CustomerMergeRecord
	if err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CustomerMergeRecord, len(rows))
	for i := range rows {
		out[i] = *mergeRecordToDomain(&rows[i])
	}
	return out, nil
}

func (r *mergeExecutionStore) CreateMergeRecord(ctx context.Context, record *domain.CustomerMergeRecord) error {
	repo := customerMergeRecordRepository{db: r.db}
	return repo.Create(ctx, record)
}

func (r *mergeExecutionStore) CompleteMergeRecord(ctx context.Context, record *domain.CustomerMergeRecord) (bool, error) {
	result := r.db.WithContext(ctx).Model(&persistence.CustomerMergeRecord{}).
		Where("id = ? AND row_version = ? AND status = ?", record.ID, record.RowVersion, domain.MergeRecordStatusExecuting).
		Updates(map[string]any{"status": domain.MergeRecordStatusCompleted, "completed_at": record.CompletedAt,
			"source_row_version_after": record.SourceRowVersionAfter, "target_row_version_after": record.TargetRowVersionAfter,
			"payload": record.Payload, "row_version": gorm.Expr("row_version + 1")})
	if result.Error == nil && result.RowsAffected == 1 {
		record.RowVersion++
	}
	return result.RowsAffected == 1, result.Error
}

func (r *mergeExecutionStore) CreateMovedEntities(ctx context.Context, moved []domain.MergeMovedEntity) error {
	if len(moved) == 0 {
		return nil
	}
	rows := make([]persistence.MergeMovedEntity, len(moved))
	for i := range moved {
		rows[i] = *persistence.MergeMovedEntityFromDomain(&moved[i])
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		moved[i].ID = rows[i].ID
	}
	return nil
}

func (r *mergeExecutionStore) ListMovedEntities(ctx context.Context, mergeRecordID uint) ([]domain.MergeMovedEntity, error) {
	var rows []persistence.MergeMovedEntity
	if err := r.db.WithContext(ctx).Where("merge_record_id = ?", mergeRecordID).Order("move_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.MergeMovedEntity, len(rows))
	for i := range rows {
		out[i] = *persistence.MergeMovedEntityToDomain(&rows[i])
	}
	return out, nil
}

func (r *mergeExecutionStore) CurrentEntityState(ctx context.Context, moved domain.MergeMovedEntity) (domain.MergeEntityState, error) {
	switch moved.EntityType {
	case domain.MergeEntityIdentity:
		var row persistence.CustomerIdentity
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.MergeEntityState{}, err
		}
		return domain.MergeEntityState{ProfileID: uintPointer(row.CustomerProfileID), IsPrimary: boolPointer(row.IsPrimary)}, nil
	case domain.MergeEntityAddress:
		var row persistence.CustomerAddress
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.MergeEntityState{}, err
		}
		return domain.MergeEntityState{ProfileID: uintPointer(row.CustomerProfileID), IsDefault: boolPointer(row.IsDefault)}, nil
	case domain.MergeEntityDemandDocument:
		var row persistence.DemandDocument
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.MergeEntityState{}, err
		}
		return domain.MergeEntityState{ProfileID: row.CustomerProfileID}, nil
	case domain.MergeEntityNameObservation:
		var row persistence.CustomerNameObservation
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.MergeEntityState{}, err
		}
		return domain.MergeEntityState{ProfileID: uintPointer(row.CustomerProfileID)}, nil
	case domain.MergeEntityNameEvent:
		var row persistence.CustomerNameEvent
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.MergeEntityState{}, err
		}
		return domain.MergeEntityState{ProfileID: uintPointer(row.CustomerProfileID)}, nil
	case domain.MergeEntityOrigin:
		var row persistence.CustomerProfileOrigin
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.MergeEntityState{}, err
		}
		return domain.MergeEntityState{ProfileID: uintPointer(row.CustomerProfileID)}, nil
	case domain.MergeEntityProfile:
		var row persistence.CustomerProfile
		if err := r.db.WithContext(ctx).Unscoped().First(&row, moved.EntityID).Error; err != nil {
			return domain.MergeEntityState{}, err
		}
		deleted := row.DeletedAt.Valid
		state := domain.MergeEntityState{ProfileID: uintPointer(row.ID), RowVersion: row.RowVersion,
			DisplayName: row.DisplayName, DisplayNameMode: row.DisplayNameMode,
			DisplayNameObservationID: row.DisplayNameObservationID, SoftDeleted: &deleted}
		if moved.MutationKind == domain.MergeMutationProfileState {
			state.Status = row.Status
			state.MergedIntoProfileID = row.MergedIntoProfileID
		}
		return state, nil
	default:
		return domain.MergeEntityState{}, fmt.Errorf("unsupported merge entity type %q", moved.EntityType)
	}
}

func (r *mergeExecutionStore) ApplyEntityState(ctx context.Context, moved domain.MergeMovedEntity, state domain.MergeEntityState) error {
	now := time.Now().UTC()
	updates := map[string]any{"updated_at": now}
	switch moved.EntityType {
	case domain.MergeEntityIdentity:
		if state.ProfileID != nil {
			updates["customer_profile_id"] = *state.ProfileID
		}
		if state.IsPrimary != nil {
			updates["is_primary"] = *state.IsPrimary
		}
		return exactUpdate(r.db.WithContext(ctx).Model(&persistence.CustomerIdentity{}).Where("id = ?", moved.EntityID), updates, moved)
	case domain.MergeEntityAddress:
		if state.ProfileID != nil {
			updates["customer_profile_id"] = *state.ProfileID
		}
		if state.IsDefault != nil {
			updates["is_default"] = *state.IsDefault
		}
		return exactUpdate(r.db.WithContext(ctx).Model(&persistence.CustomerAddress{}).Where("id = ?", moved.EntityID), updates, moved)
	case domain.MergeEntityDemandDocument:
		if state.ProfileID != nil {
			updates["customer_profile_id"] = *state.ProfileID
		}
		return exactUpdate(r.db.WithContext(ctx).Model(&persistence.DemandDocument{}).Where("id = ?", moved.EntityID), updates, moved)
	case domain.MergeEntityNameObservation:
		if state.ProfileID != nil {
			updates["customer_profile_id"] = *state.ProfileID
		}
		return exactUpdate(r.db.WithContext(ctx).Model(&persistence.CustomerNameObservation{}).Where("id = ?", moved.EntityID), updates, moved)
	case domain.MergeEntityNameEvent:
		delete(updates, "updated_at")
		if state.ProfileID != nil {
			updates["customer_profile_id"] = *state.ProfileID
		}
		return exactUpdate(r.db.WithContext(ctx).Model(&persistence.CustomerNameEvent{}).Where("id = ?", moved.EntityID), updates, moved)
	case domain.MergeEntityOrigin:
		if state.ProfileID != nil {
			updates["customer_profile_id"] = *state.ProfileID
		}
		return exactUpdate(r.db.WithContext(ctx).Model(&persistence.CustomerProfileOrigin{}).Where("id = ?", moved.EntityID), updates, moved)
	case domain.MergeEntityProfile:
		query := r.db.WithContext(ctx).Unscoped().Model(&persistence.CustomerProfile{}).
			Where("id = ?", moved.EntityID)
		current, err := r.CurrentEntityState(ctx, moved)
		if err != nil {
			return err
		}
		query = query.Where("row_version = ?", current.RowVersion)
		updates["row_version"] = gorm.Expr("row_version + 1")
		if moved.MutationKind == domain.MergeMutationProfileState {
			updates["status"] = state.Status
			updates["merged_into_profile_id"] = state.MergedIntoProfileID
		}
		updates["display_name"] = state.DisplayName
		updates["display_name_mode"] = state.DisplayNameMode
		updates["display_name_observation_id"] = state.DisplayNameObservationID
		if state.SoftDeleted != nil {
			if *state.SoftDeleted {
				updates["deleted_at"] = now
			} else {
				updates["deleted_at"] = nil
			}
		}
		return exactUpdate(query, updates, moved)
	default:
		return fmt.Errorf("unsupported merge entity type %q", moved.EntityType)
	}
}

func exactUpdate(query *gorm.DB, updates map[string]any, moved domain.MergeMovedEntity) error {
	result := query.Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("merge entity %s/%d was changed concurrently", moved.EntityType, moved.EntityID)
	}
	return nil
}

func (r *mergeExecutionStore) MarkMovedEntitiesReverted(ctx context.Context, mergeRecordID uint, operationKey string, revertedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&persistence.MergeMovedEntity{}).
		Where("merge_record_id = ? AND reverted_at IS NULL", mergeRecordID).
		Updates(map[string]any{"reverted_at": revertedAt, "revert_operation_key": operationKey, "undo_state": "restored"})
	return result.Error
}

func (r *mergeExecutionStore) MarkCandidateExecuted(ctx context.Context, candidateID uint, expectedRowVersion uint64, evidenceHash string, policyVersion, policyRevisionID, mergeRecordID uint, at time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&persistence.MergeCandidate{}).
		Where("id = ? AND row_version = ? AND status = ? AND evidence_hash = ? AND policy_version = ? AND merge_policy_revision_id = ?",
			candidateID, expectedRowVersion, domain.MergeCandidateStatusPending, evidenceHash, policyVersion, policyRevisionID).
		Updates(map[string]any{"status": domain.MergeCandidateStatusExecuted, "executed_merge_record_id": mergeRecordID,
			"executed_at": at, "row_version": gorm.Expr("row_version + 1")})
	return result.RowsAffected == 1, result.Error
}

func (r *mergeExecutionStore) InvalidateCandidatesAfterMerge(ctx context.Context, sourceProfileID, targetProfileID, executedCandidateID uint) error {
	reviewable := []string{domain.MergeCandidateStatusPending, domain.MergeCandidateStatusBlocked}
	if err := r.db.WithContext(ctx).Model(&persistence.MergeCandidate{}).
		Where("id <> ? AND status IN ? AND (source_profile_id = ? OR target_profile_id = ?)", executedCandidateID, reviewable, sourceProfileID, sourceProfileID).
		Updates(map[string]any{"status": domain.MergeCandidateStatusSuperseded, "row_version": gorm.Expr("row_version + 1")}).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&persistence.MergeCandidate{}).
		Where("id <> ? AND status IN ? AND (source_profile_id = ? OR target_profile_id = ?)", executedCandidateID, reviewable, targetProfileID, targetProfileID).
		Updates(map[string]any{"status": domain.MergeCandidateStatusStale, "row_version": gorm.Expr("row_version + 1")}).Error
}

func (r *mergeExecutionStore) MarkCandidateStaleAfterUndo(ctx context.Context, candidateID uint, mergeRecordID uint) error {
	if candidateID == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).Model(&persistence.MergeCandidate{}).
		Where("id = ? AND status = ? AND executed_merge_record_id = ?", candidateID, domain.MergeCandidateStatusExecuted, mergeRecordID).
		Updates(map[string]any{"status": domain.MergeCandidateStatusStale, "executed_merge_record_id": nil,
			"executed_at": nil, "row_version": gorm.Expr("row_version + 1")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("merge candidate %d is no longer the executed candidate", candidateID)
	}
	return nil
}

func (r *mergeExecutionStore) MarkPolicyNeedsScan(ctx context.Context, policyRevisionID *uint) error {
	if policyRevisionID == nil {
		return nil
	}
	var revision persistence.MergePolicyRevision
	if err := r.db.WithContext(ctx).First(&revision, *policyRevisionID).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&persistence.MergePolicy{}).Where("id = ?", revision.MergePolicyID).
		Updates(map[string]any{"needs_scan": true, "row_version": gorm.Expr("row_version + 1")}).Error
}

func (r *mergeExecutionStore) IsDemandDocumentAssigned(ctx context.Context, documentID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&persistence.WaveDemandAssignment{}).Where("demand_document_id = ?", documentID).Count(&count).Error
	return count > 0, err
}

func (r *mergeExecutionStore) MarkMergeUndone(ctx context.Context, recordID uint, expectedRowVersion uint64, operationKey, undoPlanHash, actorRef, reason string, sourceVersion, targetVersion uint64, at time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&persistence.CustomerMergeRecord{}).
		Where("id = ? AND row_version = ? AND status = ? AND undone_at IS NULL", recordID, expectedRowVersion, domain.MergeRecordStatusCompleted).
		Updates(map[string]any{"status": domain.MergeRecordStatusUndone, "undone_at": at, "undo_operation_key": operationKey,
			"last_undo_plan_hash": undoPlanHash, "last_undo_checked_at": at, "undone_by": actorRef, "undo_reason": reason,
			"undone_source_row_version": sourceVersion, "undone_target_row_version": targetVersion,
			"row_version": gorm.Expr("row_version + 1")})
	return result.RowsAffected == 1, result.Error
}

func (r *mergeExecutionStore) CreateMergeOperationEvent(ctx context.Context, event *domain.CustomerMergeOperationEvent) error {
	row := persistence.CustomerMergeOperationEvent{MergeRecordID: event.MergeRecordID, EventKey: event.EventKey,
		OperationKey: event.OperationKey, EventType: event.EventType, Status: event.Status, ActorRef: event.ActorRef,
		ReasonCode: event.ReasonCode, Payload: event.Payload, CreatedAt: event.CreatedAt}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	event.ID = row.ID
	return nil
}

func (r *mergeExecutionStore) ListMergeOperationEvents(ctx context.Context, mergeRecordID uint) ([]domain.CustomerMergeOperationEvent, error) {
	var rows []persistence.CustomerMergeOperationEvent
	if err := r.db.WithContext(ctx).Where("merge_record_id = ?", mergeRecordID).Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CustomerMergeOperationEvent, len(rows))
	for i := range rows {
		out[i] = domain.CustomerMergeOperationEvent{ID: rows[i].ID, MergeRecordID: rows[i].MergeRecordID,
			EventKey: rows[i].EventKey, OperationKey: rows[i].OperationKey, EventType: rows[i].EventType,
			Status: rows[i].Status, ActorRef: rows[i].ActorRef, ReasonCode: rows[i].ReasonCode,
			Payload: rows[i].Payload, CreatedAt: rows[i].CreatedAt}
	}
	return out, nil
}

func uintPointer(value uint) *uint { return &value }
func boolPointer(value bool) *bool { return &value }
