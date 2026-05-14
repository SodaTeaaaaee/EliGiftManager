package infra

import (
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

func (r *closureDecisionRepository) Create(record *domain.ChannelClosureDecisionRecord) error {
	p := persistence.ToPersistenceChannelClosureDecisionRecord(record)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*record = *persistence.FromPersistenceChannelClosureDecisionRecord(p)
	return nil
}

func (r *closureDecisionRepository) AtomicCreate(records []*domain.ChannelClosureDecisionRecord) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			p := persistence.ToPersistenceChannelClosureDecisionRecord(record)
			if err := tx.Create(p).Error; err != nil {
				return err
			}
			*record = *persistence.FromPersistenceChannelClosureDecisionRecord(p)
		}
		return nil
	})
}

func (r *closureDecisionRepository) ListByFulfillmentLine(fulfillmentLineID uint) ([]domain.ChannelClosureDecisionRecord, error) {
	var ps []persistence.ChannelClosureDecisionRecord
	if err := r.db.Where("fulfillment_line_id = ?", fulfillmentLineID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ChannelClosureDecisionRecord, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceChannelClosureDecisionRecord(&p)
	}
	return result, nil
}

func (r *closureDecisionRepository) ListByWave(waveID uint) ([]domain.ChannelClosureDecisionRecord, error) {
	var ps []persistence.ChannelClosureDecisionRecord
	if err := r.db.Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ChannelClosureDecisionRecord, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceChannelClosureDecisionRecord(&p)
	}
	return result, nil
}
