package infra

import (
	"errors"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type historyNodeRepository struct {
	db *gorm.DB
}

func NewHistoryNodeRepository(db *gorm.DB) domain.HistoryNodeRepository {
	return &historyNodeRepository{db: db}
}

func (r *historyNodeRepository) Create(node *domain.HistoryNode) error {
	p := persistence.HistoryNodeFromDomain(node)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	node.ID = p.ID
	node.CreatedAt = p.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	return nil
}

func (r *historyNodeRepository) FindByID(id uint) (*domain.HistoryNode, error) {
	var p persistence.HistoryNode
	if err := r.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return persistence.HistoryNodeToDomain(&p), nil
}

func (r *historyNodeRepository) UpdatePreferredRedoChild(nodeID uint, childID uint) error {
	return r.db.Model(&persistence.HistoryNode{}).Where("id = ?", nodeID).Update("preferred_redo_child_id", childID).Error
}

func (r *historyNodeRepository) ListByScopeRecent(scopeID uint, limit int) ([]domain.HistoryNode, error) {
	var ps []persistence.HistoryNode
	if err := r.db.Where("history_scope_id = ?", scopeID).Order("created_at DESC").Limit(limit).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.HistoryNode, len(ps))
	for i := range ps {
		result[i] = *persistence.HistoryNodeToDomain(&ps[i])
	}
	return result, nil
}

func (r *historyNodeRepository) ListByScope(scopeID uint) ([]domain.HistoryNode, error) {
	var ps []persistence.HistoryNode
	if err := r.db.Where("history_scope_id = ?", scopeID).Order("created_at ASC").Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.HistoryNode, len(ps))
	for i := range ps {
		result[i] = *persistence.HistoryNodeToDomain(&ps[i])
	}
	return result, nil
}

func (r *historyNodeRepository) DeleteByID(nodeID uint) error {
	return r.db.Delete(&persistence.HistoryNode{}, nodeID).Error
}
