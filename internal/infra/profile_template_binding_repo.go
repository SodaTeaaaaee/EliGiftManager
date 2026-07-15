package infra

import (
	"context"
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

func (r *profileTemplateBindingRepository) Create(ctx context.Context, b *domain.IntegrationProfileTemplateBinding) error {
	p := persistence.ProfileTemplateBindingFromDomain(b)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	b.ID = p.ID
	b.CreatedAt = p.CreatedAt
	return nil
}

func (r *profileTemplateBindingRepository) FindByID(ctx context.Context, id uint) (*domain.IntegrationProfileTemplateBinding, error) {
	var p persistence.IntegrationProfileTemplateBinding
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return persistence.ProfileTemplateBindingToDomain(&p), nil
}

func (r *profileTemplateBindingRepository) ListByProfile(ctx context.Context, profileID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	var records []persistence.IntegrationProfileTemplateBinding
	if err := r.db.WithContext(ctx).Where("integration_profile_id = ?", profileID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.IntegrationProfileTemplateBinding, len(records))
	for i := range records {
		out[i] = *persistence.ProfileTemplateBindingToDomain(&records[i])
	}
	return out, nil
}

func (r *profileTemplateBindingRepository) ListByTemplateID(ctx context.Context, templateID uint) ([]domain.IntegrationProfileTemplateBinding, error) {
	var records []persistence.IntegrationProfileTemplateBinding
	if err := r.db.WithContext(ctx).Where("template_id = ?", templateID).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	out := make([]domain.IntegrationProfileTemplateBinding, len(records))
	for i := range records {
		out[i] = *persistence.ProfileTemplateBindingToDomain(&records[i])
	}
	return out, nil
}

func (r *profileTemplateBindingRepository) FindDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) (*domain.IntegrationProfileTemplateBinding, error) {
	var p persistence.IntegrationProfileTemplateBinding
	err := r.db.WithContext(ctx).Where("integration_profile_id = ? AND document_type = ? AND is_default = ?", profileID, docType, true).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return persistence.ProfileTemplateBindingToDomain(&p), nil
}

func (r *profileTemplateBindingRepository) ClearDefaultByProfileAndType(ctx context.Context, profileID uint, docType string) error {
	return r.db.WithContext(ctx).
		Model(&persistence.IntegrationProfileTemplateBinding{}).
		Where("integration_profile_id = ? AND document_type = ? AND is_default = ?", profileID, docType, true).
		Update("is_default", false).Error
}

func (r *profileTemplateBindingRepository) Update(ctx context.Context, b *domain.IntegrationProfileTemplateBinding) error {
	p := persistence.ProfileTemplateBindingFromDomain(b)
	p.ID = b.ID
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return err
	}
	*b = *persistence.ProfileTemplateBindingToDomain(p)
	return nil
}

func (r *profileTemplateBindingRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&persistence.IntegrationProfileTemplateBinding{}, id).Error
}

func (r *profileTemplateBindingRepository) CountByProfileID(ctx context.Context, profileID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&persistence.IntegrationProfileTemplateBinding{}).Where("integration_profile_id = ?", profileID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
