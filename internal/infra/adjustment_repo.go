package infra

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type fulfillmentAdjustmentRepository struct {
	db *gorm.DB
}

func NewFulfillmentAdjustmentRepository(db *gorm.DB) domain.FulfillmentAdjustmentRepository {
	return &fulfillmentAdjustmentRepository{db: db}
}

func (r *fulfillmentAdjustmentRepository) Create(ctx context.Context, adj *domain.FulfillmentAdjustment) error {
	p := persistence.FulfillmentAdjustmentFromDomain(adj)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	adj.ID = p.ID
	adj.CreatedAt = p.CreatedAt
	adj.UpdatedAt = p.UpdatedAt
	return nil
}

func (r *fulfillmentAdjustmentRepository) ListByWave(ctx context.Context, waveID uint) ([]domain.FulfillmentAdjustment, error) {
	var records []persistence.FulfillmentAdjustment
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Order("created_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.FulfillmentAdjustment, len(records))
	for i := range records {
		out[i] = *persistence.FulfillmentAdjustmentToDomain(&records[i])
	}
	return out, nil
}

func (r *fulfillmentAdjustmentRepository) FindByID(ctx context.Context, id uint) (*domain.FulfillmentAdjustment, error) {
	var p persistence.FulfillmentAdjustment
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return persistence.FulfillmentAdjustmentToDomain(&p), nil
}

func (r *fulfillmentAdjustmentRepository) Update(ctx context.Context, adj *domain.FulfillmentAdjustment) error {
	p := persistence.FulfillmentAdjustmentFromDomain(adj)
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return err
	}
	updated := persistence.FulfillmentAdjustmentToDomain(p)
	*adj = *updated
	return nil
}

func (r *fulfillmentAdjustmentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&persistence.FulfillmentAdjustment{}, id).Error
}

func (r *fulfillmentAdjustmentRepository) DeleteByWave(ctx context.Context, waveID uint) error {
	return r.db.WithContext(ctx).Unscoped().Where("wave_id = ?", waveID).Delete(&persistence.FulfillmentAdjustment{}).Error
}

func (r *fulfillmentAdjustmentRepository) ListByFulfillmentLine(ctx context.Context, fulfillmentLineID uint) ([]domain.FulfillmentAdjustment, error) {
	var records []persistence.FulfillmentAdjustment
	if err := r.db.WithContext(ctx).Where("fulfillment_line_id = ?", fulfillmentLineID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.FulfillmentAdjustment, len(records))
	for i := range records {
		out[i] = *persistence.FulfillmentAdjustmentToDomain(&records[i])
	}
	return out, nil
}
