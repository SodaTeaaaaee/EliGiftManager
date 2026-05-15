package infra

import (
	"errors"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type profileTemplateBindingRepository struct {
	db *gorm.DB
}

func NewProfileTemplateBindingRepository(db *gorm.DB) domain.ProfileTemplateBindingRepository {
	return &profileTemplateBindingRepository{db: db}
}

func (r *profileTemplateBindingRepository) Create(b *domain.IntegrationProfileTemplateBinding) error {
	p := persistence.ProfileTemplateBindingFromDomain(b)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	b.ID = p.ID
	b.CreatedAt = p.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	return nil
}

func (r *profileTemplateBindingRepository) ListByProfile(profileID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	var records []persistence.IntegrationProfileTemplateBinding
	if err := r.db.Where("integration_profile_id = ?", profileID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.IntegrationProfileTemplateBinding, len(records))
	for i := range records {
		out[i] = *persistence.ProfileTemplateBindingToDomain(&records[i])
	}
	return out, nil
}

func (r *profileTemplateBindingRepository) FindDefaultByProfileAndType(profileID uint, docType string) (*domain.IntegrationProfileTemplateBinding, error) {
	var p persistence.IntegrationProfileTemplateBinding
	err := r.db.Where("integration_profile_id = ? AND document_type = ? AND is_default = ?", profileID, docType, true).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return persistence.ProfileTemplateBindingToDomain(&p), nil
}

func (r *profileTemplateBindingRepository) Delete(id uint) error {
	return r.db.Delete(&persistence.IntegrationProfileTemplateBinding{}, id).Error
}

func (r *profileTemplateBindingRepository) CountByProfileID(profileID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&persistence.IntegrationProfileTemplateBinding{}).Where("integration_profile_id = ?", profileID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
