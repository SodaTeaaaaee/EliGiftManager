package infra

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) domain.CustomerProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) Create(profile *domain.CustomerProfile) error {
	p := persistence.ToPersistenceCustomerProfile(profile)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*profile = *persistence.FromPersistenceCustomerProfile(p)
	return nil
}

func (r *profileRepository) Update(profile *domain.CustomerProfile) error {
	p := persistence.ToPersistenceCustomerProfile(profile)
	p.ID = profile.ID
	return r.db.Save(p).Error
}

func (r *profileRepository) FindByID(id uint) (*domain.CustomerProfile, error) {
	var p persistence.CustomerProfile
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceCustomerProfile(&p), nil
}

func (r *profileRepository) List() ([]domain.CustomerProfile, error) {
	var ps []persistence.CustomerProfile
	if err := r.db.Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerProfile, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceCustomerProfile(&p)
	}
	return result, nil
}

func (r *profileRepository) CreateIdentity(identity *domain.CustomerIdentity) error {
	p := persistence.ToPersistenceCustomerIdentity(identity)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*identity = *persistence.FromPersistenceCustomerIdentity(p)
	return nil
}

func (r *profileRepository) ListIdentitiesByProfile(profileID uint) ([]domain.CustomerIdentity, error) {
	var ps []persistence.CustomerIdentity
	if err := r.db.Where("customer_profile_id = ?", profileID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerIdentity, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceCustomerIdentity(&p)
	}
	return result, nil
}

func (r *profileRepository) FindIdentityByPlatformAndValue(platform, value string) (*domain.CustomerIdentity, error) {
	var p persistence.CustomerIdentity
	if err := r.db.Where("identity_platform = ? AND identity_value = ?", platform, value).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceCustomerIdentity(&p), nil
}

func (r *profileRepository) UpdateIdentityProfileID(identityID uint, newProfileID uint) error {
	return r.db.Model(&persistence.CustomerIdentity{}).Where("id = ?", identityID).Update("customer_profile_id", newProfileID).Error
}

func (r *profileRepository) BulkUpdateIdentityProfileID(identityIDs []uint, newProfileID uint) error {
	if len(identityIDs) == 0 {
		return nil
	}
	return r.db.Model(&persistence.CustomerIdentity{}).Where("id IN ?", identityIDs).Update("customer_profile_id", newProfileID).Error
}

func (r *profileRepository) SoftDelete(id uint) error {
	return r.db.Delete(&persistence.CustomerProfile{}, id).Error
}

func (r *profileRepository) DeleteIdentity(id uint) error {
	return r.db.Unscoped().Delete(&persistence.CustomerIdentity{}, id).Error
}
