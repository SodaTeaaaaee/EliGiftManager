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
		SourceProfileID: record.SourceProfileID,
		TargetProfileID: record.TargetProfileID,
		Payload:         record.Payload,
		CreatedAt:       record.CreatedAt,
		UndoneAt:        record.UndoneAt,
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
		Update("undone_at", undoneAt)
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
		ID:              p.ID,
		SourceProfileID: p.SourceProfileID,
		TargetProfileID: p.TargetProfileID,
		Payload:         p.Payload,
		CreatedAt:       p.CreatedAt,
		UndoneAt:        p.UndoneAt,
	}
}
