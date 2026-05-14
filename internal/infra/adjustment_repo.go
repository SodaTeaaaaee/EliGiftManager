package infra

import (
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

func (r *fulfillmentAdjustmentRepository) Create(adj *domain.FulfillmentAdjustment) error {
	p := persistence.FulfillmentAdjustmentFromDomain(adj)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	adj.ID = p.ID
	adj.CreatedAt = p.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	adj.UpdatedAt = p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	return nil
}

func (r *fulfillmentAdjustmentRepository) ListByWave(waveID uint) ([]domain.FulfillmentAdjustment, error) {
	var records []persistence.FulfillmentAdjustment
	if err := r.db.Where("wave_id = ?", waveID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.FulfillmentAdjustment, len(records))
	for i := range records {
		out[i] = *persistence.FulfillmentAdjustmentToDomain(&records[i])
	}
	return out, nil
}

func (r *fulfillmentAdjustmentRepository) ListByFulfillmentLine(fulfillmentLineID uint) ([]domain.FulfillmentAdjustment, error) {
	var records []persistence.FulfillmentAdjustment
	if err := r.db.Where("fulfillment_line_id = ?", fulfillmentLineID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.FulfillmentAdjustment, len(records))
	for i := range records {
		out[i] = *persistence.FulfillmentAdjustmentToDomain(&records[i])
	}
	return out, nil
}
