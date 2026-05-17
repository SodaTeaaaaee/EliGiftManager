package infra

import (
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

func (r *carrierMappingRepository) Create(mapping *domain.CarrierMapping) error {
	p := persistence.ToPersistenceCarrierMapping(mapping)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*mapping = *persistence.FromPersistenceCarrierMapping(p)
	return nil
}

func (r *carrierMappingRepository) ListByProfile(profileID uint) ([]domain.CarrierMapping, error) {
	var ps []persistence.CarrierMapping
	if err := r.db.Where("integration_profile_id = ?", profileID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CarrierMapping, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceCarrierMapping(&p)
	}
	return result, nil
}

func (r *carrierMappingRepository) FindByProfileAndInternal(profileID uint, internalCode string) (*domain.CarrierMapping, error) {
	var p persistence.CarrierMapping
	if err := r.db.Where("integration_profile_id = ? AND internal_carrier_code = ?", profileID, internalCode).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceCarrierMapping(&p), nil
}

func (r *carrierMappingRepository) Delete(id uint) error {
	return r.db.Delete(&persistence.CarrierMapping{}, id).Error
}
