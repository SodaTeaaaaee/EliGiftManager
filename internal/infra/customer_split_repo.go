package infra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type splitExecutionStore struct {
	db    *gorm.DB
	merge *mergeExecutionStore
}

var _ domain.SplitExecutionStore = (*splitExecutionStore)(nil)

func NewSplitExecutionStore(db *gorm.DB) domain.SplitExecutionStore {
	return &splitExecutionStore{db: db, merge: &mergeExecutionStore{db: db}}
}

func (r *splitExecutionStore) FindProfileForSplit(ctx context.Context, id uint) (*domain.CustomerProfile, error) {
	return r.merge.FindProfileForMerge(ctx, id, false)
}

func (r *splitExecutionStore) CreateSplitTargetProfile(ctx context.Context, profile *domain.CustomerProfile) error {
	row := persistence.CustomerProfileFromDomain(profile)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	created := persistence.CustomerProfileToDomain(row)
	*profile = *created
	return nil
}

func (r *splitExecutionStore) ListIdentitiesForSplit(ctx context.Context, profileID uint) ([]domain.CustomerIdentity, error) {
	return r.merge.ListIdentitiesForMerge(ctx, profileID)
}

func (r *splitExecutionStore) ListAddressesForSplit(ctx context.Context, profileID uint) ([]domain.CustomerAddress, error) {
	return r.merge.ListAddressesForMerge(ctx, profileID)
}

func (r *splitExecutionStore) ListDemandForSplit(ctx context.Context, profileID uint) ([]domain.DemandDocument, error) {
	return r.merge.ListDemandForMerge(ctx, profileID)
}

func (r *splitExecutionStore) ListNameObservationsForSplit(ctx context.Context, profileID uint) ([]domain.CustomerNameObservation, error) {
	return r.merge.ListNameObservationsForMerge(ctx, profileID)
}

func (r *splitExecutionStore) ListNameEventsForSplit(ctx context.Context, profileID uint) ([]domain.CustomerNameEvent, error) {
	return r.merge.ListNameEventsForMerge(ctx, profileID)
}

func (r *splitExecutionStore) ListOriginsForSplit(ctx context.Context, profileID uint) ([]domain.CustomerProfileOrigin, error) {
	return r.merge.ListOriginsForMerge(ctx, profileID)
}

func (r *splitExecutionStore) ListActiveMergeRecordsForSplit(ctx context.Context, profileID uint) ([]domain.CustomerMergeRecord, error) {
	return r.merge.ListActiveMergeRecords(ctx, []uint{profileID})
}

func (r *splitExecutionStore) ListImmutableHistoryRefsForSplit(ctx context.Context, profileID uint) (domain.SplitImmutableHistoryRefs, error) {
	result := domain.SplitImmutableHistoryRefs{}
	if err := r.db.WithContext(ctx).Model(&persistence.WaveParticipantSnapshot{}).
		Where("customer_profile_id = ?", profileID).Order("id").Pluck("id", &result.WaveParticipantSnapshotIDs).Error; err != nil {
		return result, err
	}
	if err := r.db.WithContext(ctx).Model(&persistence.FulfillmentLine{}).
		Where("deleted_at IS NULL").
		Where("customer_profile_id = ? OR demand_document_id IN (?)", profileID,
			r.db.Model(&persistence.DemandDocument{}).Select("id").Where("customer_profile_id = ?", profileID)).
		Order("id").Pluck("id", &result.FulfillmentLineIDs).Error; err != nil {
		return result, err
	}
	return result, nil
}

func (r *splitExecutionStore) ListStrongIdentityOwnerIDs(ctx context.Context, key domain.SplitIdentityKey, excludingProfileID uint) ([]uint, error) {
	namespace := strings.ToLower(strings.TrimSpace(key.Namespace))
	identityType := strings.ToLower(strings.TrimSpace(key.IdentityType))
	value := strings.TrimSpace(key.NormalizedValue)
	var ids []uint
	err := r.db.WithContext(ctx).Model(&persistence.CustomerIdentity{}).
		Distinct("customer_profile_id").
		Where("deleted_at IS NULL AND customer_profile_id <> ?", excludingProfileID).
		Where("LOWER(CASE WHEN namespace = '' THEN identity_platform ELSE namespace END) = ?", namespace).
		Where("LOWER(identity_type) = ?", identityType).
		Where("CASE WHEN normalized_value = '' THEN identity_value ELSE normalized_value END = ?", value).
		Order("customer_profile_id").Pluck("customer_profile_id", &ids).Error
	return ids, err
}

func (r *splitExecutionStore) IsDemandDocumentFrozenForSplit(ctx context.Context, documentID uint) (bool, error) {
	var assignmentCount int64
	if err := r.db.WithContext(ctx).Model(&persistence.WaveDemandAssignment{}).
		Where("demand_document_id = ?", documentID).Count(&assignmentCount).Error; err != nil {
		return false, err
	}
	if assignmentCount > 0 {
		return true, nil
	}
	var fulfillmentCount int64
	if err := r.db.WithContext(ctx).Model(&persistence.FulfillmentLine{}).
		Where("deleted_at IS NULL AND demand_document_id = ?", documentID).Count(&fulfillmentCount).Error; err != nil {
		return false, err
	}
	return fulfillmentCount > 0, nil
}

func (r *splitExecutionStore) CreateSplitRecord(ctx context.Context, record *domain.CustomerSplitRecord) error {
	row := persistence.CustomerSplitRecordFromDomain(record)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return err
	}
	record.ID = row.ID
	return nil
}

func (r *splitExecutionStore) FindSplitRecord(ctx context.Context, id uint) (*domain.CustomerSplitRecord, error) {
	var row persistence.CustomerSplitRecord
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerSplitRecordToDomain(&row), nil
}

func (r *splitExecutionStore) FindSplitRecordByOperationKey(ctx context.Context, operationKey string) (*domain.CustomerSplitRecord, error) {
	var row persistence.CustomerSplitRecord
	if err := r.db.WithContext(ctx).Where("operation_key = ?", operationKey).First(&row).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerSplitRecordToDomain(&row), nil
}

func (r *splitExecutionStore) CompleteSplitRecord(ctx context.Context, record *domain.CustomerSplitRecord) (bool, error) {
	result := r.db.WithContext(ctx).Model(&persistence.CustomerSplitRecord{}).
		Where("id = ? AND row_version = ? AND status = ?", record.ID, record.RowVersion, domain.SplitRecordStatusExecuting).
		Updates(map[string]any{
			"status":                   domain.SplitRecordStatusCompleted,
			"completed_at":             record.CompletedAt,
			"source_row_version_after": record.SourceRowVersionAfter,
			"target_row_version_after": record.TargetRowVersionAfter,
			"target_profile_snapshot":  record.TargetProfileSnapshot,
			"payload":                  record.Payload,
			"row_version":              gorm.Expr("row_version + 1"),
		})
	if result.Error == nil && result.RowsAffected == 1 {
		record.RowVersion++
	}
	return result.RowsAffected == 1, result.Error
}

func (r *splitExecutionStore) ListSplitRecords(ctx context.Context, filter domain.SplitHistoryFilter) ([]domain.CustomerSplitRecord, error) {
	query := r.db.WithContext(ctx).Model(&persistence.CustomerSplitRecord{})
	if filter.ProfileID != 0 {
		query = query.Where("source_profile_id = ? OR target_profile_id = ?", filter.ProfileID, filter.ProfileID)
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
	var rows []persistence.CustomerSplitRecord
	if err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerSplitRecord, len(rows))
	for i := range rows {
		result[i] = *persistence.CustomerSplitRecordToDomain(&rows[i])
	}
	return result, nil
}

func (r *splitExecutionStore) CreateSplitMovedEntities(ctx context.Context, moved []domain.SplitMovedEntity) error {
	if len(moved) == 0 {
		return nil
	}
	rows := make([]persistence.SplitMovedEntity, len(moved))
	for i := range moved {
		rows[i] = *persistence.SplitMovedEntityFromDomain(&moved[i])
	}
	if err := r.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		moved[i].ID = rows[i].ID
	}
	return nil
}

func (r *splitExecutionStore) ListSplitMovedEntities(ctx context.Context, splitRecordID uint) ([]domain.SplitMovedEntity, error) {
	var rows []persistence.SplitMovedEntity
	if err := r.db.WithContext(ctx).Where("split_record_id = ?", splitRecordID).Order("move_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.SplitMovedEntity, len(rows))
	for i := range rows {
		result[i] = *persistence.SplitMovedEntityToDomain(&rows[i])
	}
	return result, nil
}

func (r *splitExecutionStore) CurrentSplitEntityState(ctx context.Context, moved domain.SplitMovedEntity) (domain.SplitEntityState, error) {
	switch moved.EntityType {
	case domain.MergeEntityIdentity:
		var row persistence.CustomerIdentity
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.SplitEntityState{}, err
		}
		return domain.SplitEntityState{Exists: true, ProfileID: splitUintPointer(row.CustomerProfileID), IsPrimary: splitBoolPointer(row.IsPrimary)}, nil
	case domain.MergeEntityAddress:
		var row persistence.CustomerAddress
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.SplitEntityState{}, err
		}
		return domain.SplitEntityState{Exists: true, ProfileID: splitUintPointer(row.CustomerProfileID), IsDefault: splitBoolPointer(row.IsDefault)}, nil
	case domain.MergeEntityDemandDocument:
		var row persistence.DemandDocument
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.SplitEntityState{}, err
		}
		return domain.SplitEntityState{Exists: true, ProfileID: row.CustomerProfileID}, nil
	case domain.MergeEntityNameObservation:
		var row persistence.CustomerNameObservation
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.SplitEntityState{}, err
		}
		return domain.SplitEntityState{Exists: true, ProfileID: splitUintPointer(row.CustomerProfileID)}, nil
	case domain.MergeEntityNameEvent:
		var row persistence.CustomerNameEvent
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.SplitEntityState{}, err
		}
		return domain.SplitEntityState{Exists: true, ProfileID: splitUintPointer(row.CustomerProfileID)}, nil
	case domain.MergeEntityOrigin:
		var row persistence.CustomerProfileOrigin
		if err := r.db.WithContext(ctx).First(&row, moved.EntityID).Error; err != nil {
			return domain.SplitEntityState{}, err
		}
		return domain.SplitEntityState{Exists: true, ProfileID: splitUintPointer(row.CustomerProfileID)}, nil
	case domain.MergeEntityProfile:
		var row persistence.CustomerProfile
		if err := r.db.WithContext(ctx).Unscoped().First(&row, moved.EntityID).Error; err != nil {
			return domain.SplitEntityState{}, err
		}
		return domain.SplitEntityState{Exists: true, ProfileID: splitUintPointer(row.ID), Status: row.Status,
			MergedIntoProfileID: row.MergedIntoProfileID, RowVersion: row.RowVersion, DisplayName: row.DisplayName,
			DisplayNameMode: row.DisplayNameMode, DisplayNameObservationID: row.DisplayNameObservationID}, nil
	default:
		return domain.SplitEntityState{}, fmt.Errorf("unsupported split entity type %q", moved.EntityType)
	}
}

func (r *splitExecutionStore) ApplySplitEntityState(ctx context.Context, moved domain.SplitMovedEntity, before, after domain.SplitEntityState) error {
	now := time.Now().UTC()
	switch moved.EntityType {
	case domain.MergeEntityIdentity:
		query := r.db.WithContext(ctx).Model(&persistence.CustomerIdentity{}).Where("id = ?", moved.EntityID)
		query = splitWhereProfile(query, before.ProfileID)
		if before.IsPrimary != nil {
			query = query.Where("is_primary = ?", *before.IsPrimary)
		}
		return splitExactUpdate(query, map[string]any{"customer_profile_id": splitRequiredProfile(after), "is_primary": splitRequiredBool(after.IsPrimary), "updated_at": now}, moved)
	case domain.MergeEntityAddress:
		query := r.db.WithContext(ctx).Model(&persistence.CustomerAddress{}).Where("id = ?", moved.EntityID)
		query = splitWhereProfile(query, before.ProfileID)
		if before.IsDefault != nil {
			query = query.Where("is_default = ?", *before.IsDefault)
		}
		return splitExactUpdate(query, map[string]any{"customer_profile_id": splitRequiredProfile(after), "is_default": splitRequiredBool(after.IsDefault), "updated_at": now}, moved)
	case domain.MergeEntityDemandDocument:
		query := splitWhereProfile(r.db.WithContext(ctx).Model(&persistence.DemandDocument{}).Where("id = ?", moved.EntityID), before.ProfileID)
		return splitExactUpdate(query, map[string]any{"customer_profile_id": after.ProfileID, "updated_at": now}, moved)
	case domain.MergeEntityNameObservation:
		query := splitWhereProfile(r.db.WithContext(ctx).Model(&persistence.CustomerNameObservation{}).Where("id = ?", moved.EntityID), before.ProfileID)
		return splitExactUpdate(query, map[string]any{"customer_profile_id": splitRequiredProfile(after), "updated_at": now}, moved)
	case domain.MergeEntityNameEvent:
		query := splitWhereProfile(r.db.WithContext(ctx).Model(&persistence.CustomerNameEvent{}).Where("id = ?", moved.EntityID), before.ProfileID)
		return splitExactUpdate(query, map[string]any{"customer_profile_id": splitRequiredProfile(after)}, moved)
	case domain.MergeEntityOrigin:
		query := splitWhereProfile(r.db.WithContext(ctx).Model(&persistence.CustomerProfileOrigin{}).Where("id = ?", moved.EntityID), before.ProfileID)
		return splitExactUpdate(query, map[string]any{"customer_profile_id": splitRequiredProfile(after), "updated_at": now}, moved)
	case domain.MergeEntityProfile:
		query := r.db.WithContext(ctx).Unscoped().Model(&persistence.CustomerProfile{}).
			Where("id = ? AND row_version = ? AND status = ?", moved.EntityID, before.RowVersion, before.Status)
		if before.MergedIntoProfileID == nil {
			query = query.Where("merged_into_profile_id IS NULL")
		} else {
			query = query.Where("merged_into_profile_id = ?", *before.MergedIntoProfileID)
		}
		return splitExactUpdate(query, map[string]any{"display_name": after.DisplayName,
			"display_name_mode": after.DisplayNameMode, "display_name_observation_id": after.DisplayNameObservationID,
			"row_version": gorm.Expr("row_version + 1"), "updated_at": now}, moved)
	default:
		return fmt.Errorf("unsupported split entity type %q", moved.EntityType)
	}
}

func (r *splitExecutionStore) CreateSplitOperationEvent(ctx context.Context, event *domain.CustomerSplitOperationEvent) error {
	row := persistence.CustomerSplitOperationEvent{SplitRecordID: event.SplitRecordID, EventKey: event.EventKey,
		OperationKey: event.OperationKey, EventType: event.EventType, Status: event.Status,
		ActorRef: event.ActorRef, ReasonCode: event.ReasonCode, Payload: event.Payload, CreatedAt: event.CreatedAt}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	event.ID = row.ID
	return nil
}

func (r *splitExecutionStore) ListSplitOperationEvents(ctx context.Context, splitRecordID uint) ([]domain.CustomerSplitOperationEvent, error) {
	var rows []persistence.CustomerSplitOperationEvent
	if err := r.db.WithContext(ctx).Where("split_record_id = ?", splitRecordID).Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerSplitOperationEvent, len(rows))
	for i := range rows {
		result[i] = domain.CustomerSplitOperationEvent{ID: rows[i].ID, SplitRecordID: rows[i].SplitRecordID,
			EventKey: rows[i].EventKey, OperationKey: rows[i].OperationKey, EventType: rows[i].EventType,
			Status: rows[i].Status, ActorRef: rows[i].ActorRef, ReasonCode: rows[i].ReasonCode,
			Payload: rows[i].Payload, CreatedAt: rows[i].CreatedAt}
	}
	return result, nil
}

func (r *splitExecutionStore) InvalidateCandidatesAfterSplit(ctx context.Context, sourceProfileID, targetProfileID uint) error {
	reviewable := []string{domain.MergeCandidateStatusPending, domain.MergeCandidateStatusBlocked}
	if err := r.db.WithContext(ctx).Model(&persistence.MergeCandidate{}).
		Where("status IN ? AND (source_profile_id IN ? OR target_profile_id IN ?)", reviewable,
			[]uint{sourceProfileID, targetProfileID}, []uint{sourceProfileID, targetProfileID}).
		Updates(map[string]any{"status": domain.MergeCandidateStatusStale, "row_version": gorm.Expr("row_version + 1")}).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&persistence.MergePolicy{}).Where("is_active = ?", true).
		Updates(map[string]any{"needs_scan": true, "row_version": gorm.Expr("row_version + 1")}).Error
}

func splitWhereProfile(query *gorm.DB, profileID *uint) *gorm.DB {
	if profileID == nil {
		return query.Where("customer_profile_id IS NULL")
	}
	return query.Where("customer_profile_id = ?", *profileID)
}

func splitExactUpdate(query *gorm.DB, updates map[string]any, moved domain.SplitMovedEntity) error {
	result := query.Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("split entity %s/%d was changed concurrently", moved.EntityType, moved.EntityID)
	}
	return nil
}

func splitRequiredProfile(state domain.SplitEntityState) uint {
	if state.ProfileID == nil {
		return 0
	}
	return *state.ProfileID
}

func splitRequiredBool(value *bool) bool {
	return value != nil && *value
}

func splitUintPointer(value uint) *uint { return &value }
func splitBoolPointer(value bool) *bool { return &value }
