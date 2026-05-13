package infra

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type fulfillmentRepository struct {
	db *gorm.DB
}

func NewFulfillmentRepository(db *gorm.DB) domain.FulfillmentLineRepository {
	return &fulfillmentRepository{db: db}
}

func (r *fulfillmentRepository) Create(line *domain.FulfillmentLine) error {
	p := persistence.ToPersistenceFulfillmentLine(line)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*line = *persistence.FromPersistenceFulfillmentLine(p)
	return nil
}

func (r *fulfillmentRepository) FindByID(id uint) (*domain.FulfillmentLine, error) {
	var p persistence.FulfillmentLine
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceFulfillmentLine(&p), nil
}

func (r *fulfillmentRepository) ListByWave(waveID uint) ([]domain.FulfillmentLine, error) {
	var ps []persistence.FulfillmentLine
	if err := r.db.Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.FulfillmentLine, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceFulfillmentLine(&p)
	}
	return result, nil
}
