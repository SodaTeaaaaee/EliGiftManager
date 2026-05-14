package infra

import (
	"errors"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type historyCheckpointRepository struct {
	db *gorm.DB
}

func NewHistoryCheckpointRepository(db *gorm.DB) domain.HistoryCheckpointRepository {
	return &historyCheckpointRepository{db: db}
}

func (r *historyCheckpointRepository) Create(cp *domain.HistoryCheckpoint) error {
	p := persistence.HistoryCheckpointFromDomain(cp)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	cp.ID = p.ID
	cp.CreatedAt = p.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	return nil
}

func (r *historyCheckpointRepository) FindByNodeID(nodeID uint) (*domain.HistoryCheckpoint, error) {
	var p persistence.HistoryCheckpoint
	if err := r.db.Where("history_node_id = ?", nodeID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return persistence.HistoryCheckpointToDomain(&p), nil
}
