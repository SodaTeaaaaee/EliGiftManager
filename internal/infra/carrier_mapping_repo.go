package infra

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type carrierMappingRepository struct {
	db *gorm.DB
}

// NewCarrierMappingRepository returns a domain.CarrierMappingRepository backed by GORM.
func NewCarrierMappingRepository(db *gorm.DB) domain.CarrierMappingRepository {
	return &carrierMappingRepository{db: db}
}

func (r *carrierMappingRepository) Create(ctx context.Context, mapping *domain.CarrierMapping) error {
	p := persistence.CarrierMappingFromDomain(mapping)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*mapping = *persistence.CarrierMappingToDomain(p)
	return nil
}

func (r *carrierMappingRepository) Update(ctx context.Context, mapping *domain.CarrierMapping) error {
	p := persistence.CarrierMappingFromDomain(mapping)
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return err
	}
	*mapping = *persistence.CarrierMappingToDomain(p)
	return nil
}

func (r *carrierMappingRepository) ListByProfile(ctx context.Context, profileID uint) ([]domain.CarrierMapping, error) {
	var ps []persistence.CarrierMapping
	if err := r.db.WithContext(ctx).Where("integration_profile_id = ?", profileID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CarrierMapping, len(ps))
	for i, p := range ps {
		result[i] = *persistence.CarrierMappingToDomain(&p)
	}
	return result, nil
}

func (r *carrierMappingRepository) FindByProfileAndInternal(ctx context.Context, profileID uint, internalCode string) (*domain.CarrierMapping, error) {
	var p persistence.CarrierMapping
	if err := r.db.WithContext(ctx).Where("integration_profile_id = ? AND internal_carrier_code = ?", profileID, internalCode).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.CarrierMappingToDomain(&p), nil
}

func (r *carrierMappingRepository) FindByProfileAndExternal(ctx context.Context, profileID uint, externalCode string) (*domain.CarrierMapping, error) {
	var p persistence.CarrierMapping
	if err := r.db.WithContext(ctx).Where("integration_profile_id = ? AND external_carrier_code = ?", profileID, externalCode).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.CarrierMappingToDomain(&p), nil
}

func (r *carrierMappingRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&persistence.CarrierMapping{}, id).Error
}
