package infra

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type historyPinRepository struct {
	db *gorm.DB
}

func NewHistoryPinRepository(db *gorm.DB) domain.HistoryPinRepository {
	return &historyPinRepository{db: db}
}

func (r *historyPinRepository) Create(pin *domain.HistoryPin) error {
	p := persistence.HistoryPinFromDomain(pin)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	pin.ID = p.ID
	pin.CreatedAt = p.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	return nil
}

func (r *historyPinRepository) ListByNodeID(nodeID uint) ([]domain.HistoryPin, error) {
	var records []persistence.HistoryPin
	if err := r.db.Where("history_node_id = ?", nodeID).Order("created_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.HistoryPin, len(records))
	for i := range records {
		out[i] = *persistence.HistoryPinToDomain(&records[i])
	}
	return out, nil
}

func (r *historyPinRepository) CountByNodeID(nodeID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&persistence.HistoryPin{}).Where("history_node_id = ?", nodeID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
