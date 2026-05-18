package infra

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type addressRepository struct {
	db *gorm.DB
}

func NewAddressRepository(db *gorm.DB) domain.CustomerAddressRepository {
	return &addressRepository{db: db}
}

func (r *addressRepository) Create(addr *domain.CustomerAddress) error {
	p := persistence.ToPersistenceCustomerAddress(addr)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*addr = *persistence.FromPersistenceCustomerAddress(p)
	return nil
}

func (r *addressRepository) FindByID(id uint) (*domain.CustomerAddress, error) {
	var p persistence.CustomerAddress
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceCustomerAddress(&p), nil
}

func (r *addressRepository) ListByProfile(profileID uint) ([]domain.CustomerAddress, error) {
	var ps []persistence.CustomerAddress
	if err := r.db.Where("customer_profile_id = ?", profileID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerAddress, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceCustomerAddress(&p)
	}
	return result, nil
}

func (r *addressRepository) Update(addr *domain.CustomerAddress) error {
	p := persistence.ToPersistenceCustomerAddress(addr)
	p.ID = addr.ID
	return r.db.Save(p).Error
}

func (r *addressRepository) SoftDelete(id uint) error {
	return r.db.Delete(&persistence.CustomerAddress{}, id).Error
}

func (r *addressRepository) BulkUpdateProfileID(oldProfileID, newProfileID uint) error {
	return r.db.Model(&persistence.CustomerAddress{}).
		Where("customer_profile_id = ?", oldProfileID).
		Update("customer_profile_id", newProfileID).Error
}

func (r *addressRepository) ClearDefaultByProfile(profileID uint) error {
	return r.db.Model(&persistence.CustomerAddress{}).
		Where("customer_profile_id = ? AND is_default = ?", profileID, true).
		Update("is_default", false).Error
}
