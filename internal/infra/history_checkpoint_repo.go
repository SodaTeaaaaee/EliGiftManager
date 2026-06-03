package infra

import (
	"context"
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

func (r *historyCheckpointRepository) Create(ctx context.Context, cp *domain.HistoryCheckpoint) error {
	p := persistence.HistoryCheckpointFromDomain(cp)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	cp.ID = p.ID
	cp.CreatedAt = p.CreatedAt
	return nil
}

func (r *historyCheckpointRepository) FindByNodeID(ctx context.Context, nodeID uint) (*domain.HistoryCheckpoint, error) {
	var p persistence.HistoryCheckpoint
	if err := r.db.WithContext(ctx).Where("history_node_id = ?", nodeID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return persistence.HistoryCheckpointToDomain(&p), nil
}

func (r *historyCheckpointRepository) DeleteByNodeID(ctx context.Context, nodeID uint) error {
	return r.db.WithContext(ctx).Where("history_node_id = ?", nodeID).Delete(&persistence.HistoryCheckpoint{}).Error
}
