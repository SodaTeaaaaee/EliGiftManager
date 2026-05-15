package infra

import (
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

func (r *ruleRepository) Create(rule *domain.AllocationPolicyRule) error {
	p := persistence.ToPersistenceAllocationPolicyRule(rule)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*rule = *persistence.FromPersistenceAllocationPolicyRule(p)
	return nil
}

func (r *ruleRepository) FindByID(id uint) (*domain.AllocationPolicyRule, error) {
	var p persistence.AllocationPolicyRule
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceAllocationPolicyRule(&p), nil
}

func (r *ruleRepository) ListByWave(waveID uint) ([]domain.AllocationPolicyRule, error) {
	var ps []persistence.AllocationPolicyRule
	if err := r.db.Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.AllocationPolicyRule, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceAllocationPolicyRule(&p)
	}
	return result, nil
}

func (r *ruleRepository) Update(rule *domain.AllocationPolicyRule) error {
	p := persistence.ToPersistenceAllocationPolicyRule(rule)
	p.ID = rule.ID
	if err := r.db.Save(p).Error; err != nil {
		return err
	}
	*rule = *persistence.FromPersistenceAllocationPolicyRule(p)
	return nil
}

func (r *ruleRepository) Delete(id uint) error {
	return r.db.Unscoped().Delete(&persistence.AllocationPolicyRule{}, id).Error
}

func (r *ruleRepository) DeleteByWave(waveID uint) error {
	return r.db.Unscoped().Where("wave_id = ?", waveID).Delete(&persistence.AllocationPolicyRule{}).Error
}
