package infra

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type closureDecisionRepository struct {
	db *gorm.DB
}

func NewClosureDecisionRepository(db *gorm.DB) domain.ChannelClosureDecisionRepository {
	return &closureDecisionRepository{db: db}
}

func (r *closureDecisionRepository) Create(ctx context.Context, record *domain.ChannelClosureDecisionRecord) error {
	p := persistence.ChannelClosureDecisionRecordFromDomain(record)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*record = *persistence.ChannelClosureDecisionRecordToDomain(p)
	return nil
}

func (r *closureDecisionRepository) AtomicCreate(ctx context.Context, records []*domain.ChannelClosureDecisionRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			p := persistence.ChannelClosureDecisionRecordFromDomain(record)
			if err := tx.Create(p).Error; err != nil {
				return err
			}
			*record = *persistence.ChannelClosureDecisionRecordToDomain(p)
		}
		return nil
	})
}

func (r *closureDecisionRepository) ListByFulfillmentLine(ctx context.Context, fulfillmentLineID uint) ([]domain.ChannelClosureDecisionRecord, error) {
	var ps []persistence.ChannelClosureDecisionRecord
	if err := r.db.WithContext(ctx).Where("fulfillment_line_id = ?", fulfillmentLineID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ChannelClosureDecisionRecord, len(ps))
	for i, p := range ps {
		result[i] = *persistence.ChannelClosureDecisionRecordToDomain(&p)
	}
	return result, nil
}

func (r *closureDecisionRepository) ListByWave(ctx context.Context, waveID uint) ([]domain.ChannelClosureDecisionRecord, error) {
	var ps []persistence.ChannelClosureDecisionRecord
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ChannelClosureDecisionRecord, len(ps))
	for i, p := range ps {
		result[i] = *persistence.ChannelClosureDecisionRecordToDomain(&p)
	}
	return result, nil
}

func (r *closureDecisionRepository) CountByProfileID(ctx context.Context, profileID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&persistence.ChannelClosureDecisionRecord{}).Where("integration_profile_id = ?", profileID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
