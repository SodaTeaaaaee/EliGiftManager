package infra

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type ruleRepository struct {
	db *gorm.DB
}

func NewRuleRepository(db *gorm.DB) domain.AllocationPolicyRuleRepository {
	return &ruleRepository{db: db}
}

func (r *ruleRepository) Create(ctx context.Context, rule *domain.AllocationPolicyRule) error {
	p, err := persistence.AllocationPolicyRuleFromDomain(rule)
	if err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	mapped, err := persistence.AllocationPolicyRuleToDomain(p)
	if err != nil {
		return err
	}
	*rule = *mapped
	return nil
}

func (r *ruleRepository) FindByID(ctx context.Context, id uint) (*domain.AllocationPolicyRule, error) {
	var p persistence.AllocationPolicyRule
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.AllocationPolicyRuleToDomain(&p)
}

func (r *ruleRepository) ListByWave(ctx context.Context, waveID uint) ([]domain.AllocationPolicyRule, error) {
	var ps []persistence.AllocationPolicyRule
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.AllocationPolicyRule, len(ps))
	for i, p := range ps {
		mapped, err := persistence.AllocationPolicyRuleToDomain(&p)
		if err != nil {
			return nil, err
		}
		result[i] = *mapped
	}
	return result, nil
}

func (r *ruleRepository) Update(ctx context.Context, rule *domain.AllocationPolicyRule) error {
	p, err := persistence.AllocationPolicyRuleFromDomain(rule)
	if err != nil {
		return err
	}
	p.ID = rule.ID
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return err
	}
	mapped, err := persistence.AllocationPolicyRuleToDomain(p)
	if err != nil {
		return err
	}
	*rule = *mapped
	return nil
}

func (r *ruleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&persistence.AllocationPolicyRule{}, id).Error
}

func (r *ruleRepository) DeleteByWave(ctx context.Context, waveID uint) error {
	return r.db.WithContext(ctx).Unscoped().Where("wave_id = ?", waveID).Delete(&persistence.AllocationPolicyRule{}).Error
}
