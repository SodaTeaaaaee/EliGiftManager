package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type customerMergeRecordRepository struct {
	db *gorm.DB
}

func NewCustomerMergeRecordRepository(db *gorm.DB) domain.CustomerMergeRecordRepository {
	return &customerMergeRecordRepository{db: db}
}

func (r *customerMergeRecordRepository) Create(ctx context.Context, record *domain.CustomerMergeRecord) error {
	p := persistence.CustomerMergeRecord{
		SourceProfileID:        record.SourceProfileID,
		TargetProfileID:        record.TargetProfileID,
		MergeCandidateID:       record.MergeCandidateID,
		MergePolicyRevisionID:  record.MergePolicyRevisionID,
		MergeMode:              record.MergeMode,
		DecisionSource:         record.DecisionSource,
		DecisionReason:         record.DecisionReason,
		ActorRef:               record.ActorRef,
		CorrelationID:          record.CorrelationID,
		SourceRowVersion:       record.SourceRowVersion,
		TargetRowVersion:       record.TargetRowVersion,
		EvidenceSnapshot:       record.EvidenceSnapshot,
		Payload:                record.Payload,
		RowVersion:             record.RowVersion,
		OperationKey:           record.OperationKey,
		CommandHash:            record.CommandHash,
		PreviewHash:            record.PreviewHash,
		MovePlanHash:           record.MovePlanHash,
		Status:                 record.Status,
		DependsOnMergeRecordID: record.DependsOnMergeRecordID,
		SourceRowVersionAfter:  record.SourceRowVersionAfter,
		TargetRowVersionAfter:  record.TargetRowVersionAfter,
		SourceProfileSnapshot:  record.SourceProfileSnapshot,
		TargetProfileSnapshot:  record.TargetProfileSnapshot,
		CompletedAt:            record.CompletedAt,
		UndoOperationKey:       record.UndoOperationKey,
		LastUndoPlanHash:       record.LastUndoPlanHash,
		LastUndoCheckedAt:      record.LastUndoCheckedAt,
		UndoneBy:               record.UndoneBy,
		UndoReason:             record.UndoReason,
		UndoneSourceRowVersion: record.UndoneSourceRowVersion,
		UndoneTargetRowVersion: record.UndoneTargetRowVersion,
		CreatedAt:              record.CreatedAt,
		UndoneAt:               record.UndoneAt,
	}
	if err := r.db.WithContext(ctx).Create(&p).Error; err != nil {
		return err
	}
	record.ID = p.ID
	record.CreatedAt = p.CreatedAt
	return nil
}

func (r *customerMergeRecordRepository) FindByID(ctx context.Context, id uint) (*domain.CustomerMergeRecord, error) {
	var p persistence.CustomerMergeRecord
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return mergeRecordToDomain(&p), nil
}

func (r *customerMergeRecordRepository) ListActiveByTargetProfileID(ctx context.Context, targetProfileID uint) ([]domain.CustomerMergeRecord, error) {
	var records []persistence.CustomerMergeRecord
	if err := r.db.WithContext(ctx).
		Where("target_profile_id = ? AND undone_at IS NULL", targetProfileID).
		Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerMergeRecord, len(records))
	for i := range records {
		result[i] = *mergeRecordToDomain(&records[i])
	}
	return result, nil
}

func (r *customerMergeRecordRepository) MarkUndone(ctx context.Context, id uint, undoneAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&persistence.CustomerMergeRecord{}).
		Where("id = ? AND undone_at IS NULL", id).
		Updates(map[string]any{
			"undone_at":   undoneAt,
			"status":      domain.MergeRecordStatusUndone,
			"row_version": gorm.Expr("row_version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("merge record %d is missing or already undone", id)
	}
	return nil
}

func mergeRecordToDomain(p *persistence.CustomerMergeRecord) *domain.CustomerMergeRecord {
	return &domain.CustomerMergeRecord{
		ID:                     p.ID,
		SourceProfileID:        p.SourceProfileID,
		TargetProfileID:        p.TargetProfileID,
		MergeCandidateID:       p.MergeCandidateID,
		MergePolicyRevisionID:  p.MergePolicyRevisionID,
		MergeMode:              p.MergeMode,
		DecisionSource:         p.DecisionSource,
		DecisionReason:         p.DecisionReason,
		ActorRef:               p.ActorRef,
		CorrelationID:          p.CorrelationID,
		SourceRowVersion:       p.SourceRowVersion,
		TargetRowVersion:       p.TargetRowVersion,
		EvidenceSnapshot:       p.EvidenceSnapshot,
		Payload:                p.Payload,
		RowVersion:             p.RowVersion,
		OperationKey:           p.OperationKey,
		CommandHash:            p.CommandHash,
		PreviewHash:            p.PreviewHash,
		MovePlanHash:           p.MovePlanHash,
		Status:                 p.Status,
		DependsOnMergeRecordID: p.DependsOnMergeRecordID,
		SourceRowVersionAfter:  p.SourceRowVersionAfter,
		TargetRowVersionAfter:  p.TargetRowVersionAfter,
		SourceProfileSnapshot:  p.SourceProfileSnapshot,
		TargetProfileSnapshot:  p.TargetProfileSnapshot,
		CompletedAt:            p.CompletedAt,
		UndoOperationKey:       p.UndoOperationKey,
		LastUndoPlanHash:       p.LastUndoPlanHash,
		LastUndoCheckedAt:      p.LastUndoCheckedAt,
		UndoneBy:               p.UndoneBy,
		UndoReason:             p.UndoReason,
		UndoneSourceRowVersion: p.UndoneSourceRowVersion,
		UndoneTargetRowVersion: p.UndoneTargetRowVersion,
		CreatedAt:              p.CreatedAt,
		UndoneAt:               p.UndoneAt,
	}
}
