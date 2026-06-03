package infra

import (
	"context"

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

func (r *historyScopeRepository) Create(ctx context.Context, scope *domain.HistoryScope) error {
	p := persistence.HistoryScopeFromDomain(scope)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	scope.ID = p.ID
	scope.CreatedAt = p.CreatedAt
	scope.UpdatedAt = p.UpdatedAt
	return nil
}

func (r *historyScopeRepository) FindByID(ctx context.Context, id uint) (*domain.HistoryScope, error) {
	var p persistence.HistoryScope
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return persistence.HistoryScopeToDomain(&p), nil
}

func (r *historyScopeRepository) FindByScopeTypeAndKey(ctx context.Context, scopeType string, scopeKey string) (*domain.HistoryScope, error) {
	var p persistence.HistoryScope
	if err := r.db.WithContext(ctx).Where("scope_type = ? AND scope_key = ?", scopeType, scopeKey).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return persistence.HistoryScopeToDomain(&p), nil
}

func (r *historyScopeRepository) UpdateHead(ctx context.Context, scopeID uint, headNodeID uint) error {
	return r.db.WithContext(ctx).Model(&persistence.HistoryScope{}).Where("id = ?", scopeID).Update("current_head_node_id", headNodeID).Error
}

func (r *historyScopeRepository) FindOrCreate(ctx context.Context, scopeType string, scopeKey string) (*domain.HistoryScope, error) {
	existing, err := r.FindByScopeTypeAndKey(ctx, scopeType, scopeKey)
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
	if err := r.Create(ctx, scope); err != nil {
		return nil, err
	}
	return scope, nil
}
