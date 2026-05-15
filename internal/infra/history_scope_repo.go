package infra

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type historyScopeRepository struct {
	db *gorm.DB
}

func NewHistoryScopeRepository(db *gorm.DB) domain.HistoryScopeRepository {
	return &historyScopeRepository{db: db}
}

func (r *historyScopeRepository) Create(scope *domain.HistoryScope) error {
	p := persistence.HistoryScopeFromDomain(scope)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	scope.ID = p.ID
	scope.CreatedAt = p.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	scope.UpdatedAt = p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	return nil
}

func (r *historyScopeRepository) FindByID(id uint) (*domain.HistoryScope, error) {
	var p persistence.HistoryScope
	if err := r.db.First(&p, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return persistence.HistoryScopeToDomain(&p), nil
}

func (r *historyScopeRepository) FindByScopeTypeAndKey(scopeType string, scopeKey string) (*domain.HistoryScope, error) {
	var p persistence.HistoryScope
	if err := r.db.Where("scope_type = ? AND scope_key = ?", scopeType, scopeKey).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return persistence.HistoryScopeToDomain(&p), nil
}

func (r *historyScopeRepository) UpdateHead(scopeID uint, headNodeID uint) error {
	return r.db.Model(&persistence.HistoryScope{}).Where("id = ?", scopeID).Update("current_head_node_id", headNodeID).Error
}

func (r *historyScopeRepository) FindOrCreate(scopeType string, scopeKey string) (*domain.HistoryScope, error) {
	existing, err := r.FindByScopeTypeAndKey(scopeType, scopeKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	scope := &domain.HistoryScope{
		ScopeType: scopeType,
		ScopeKey:  scopeKey,
	}
	if err := r.Create(scope); err != nil {
		return nil, err
	}
	return scope, nil
}
