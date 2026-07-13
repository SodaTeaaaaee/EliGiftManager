package infra

import (
	"context"

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

func NewCustomerMergeAddressRepository(db *gorm.DB) domain.CustomerMergeAddressRepository {
	return &addressRepository{db: db}
}

func (r *addressRepository) Create(ctx context.Context, addr *domain.CustomerAddress) error {
	p := persistence.CustomerAddressFromDomain(addr)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*addr = *persistence.CustomerAddressToDomain(p)
	return nil
}

func (r *addressRepository) FindByID(ctx context.Context, id uint) (*domain.CustomerAddress, error) {
	var p persistence.CustomerAddress
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerAddressToDomain(&p), nil
}

func (r *addressRepository) ListByProfile(ctx context.Context, profileID uint) ([]domain.CustomerAddress, error) {
	var ps []persistence.CustomerAddress
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerAddress, len(ps))
	for i, p := range ps {
		result[i] = *persistence.CustomerAddressToDomain(&p)
	}
	return result, nil
}

func (r *addressRepository) ListByIDs(ctx context.Context, addressIDs []uint) ([]domain.CustomerAddress, error) {
	if len(addressIDs) == 0 {
		return []domain.CustomerAddress{}, nil
	}
	var ps []persistence.CustomerAddress
	if err := r.db.WithContext(ctx).Where("id IN ?", addressIDs).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerAddress, len(ps))
	for i, p := range ps {
		result[i] = *persistence.CustomerAddressToDomain(&p)
	}
	return result, nil
}

func (r *addressRepository) Update(ctx context.Context, addr *domain.CustomerAddress) error {
	p := persistence.CustomerAddressFromDomain(addr)
	p.ID = addr.ID
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *addressRepository) SoftDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&persistence.CustomerAddress{}, id).Error
}

func (r *addressRepository) BulkUpdateProfileID(ctx context.Context, oldProfileID, newProfileID uint) error {
	return r.db.WithContext(ctx).Model(&persistence.CustomerAddress{}).
		Where("customer_profile_id = ?", oldProfileID).
		Update("customer_profile_id", newProfileID).Error
}

func (r *addressRepository) BulkUpdateProfileIDByIDs(ctx context.Context, addressIDs []uint, newProfileID uint) error {
	if len(addressIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&persistence.CustomerAddress{}).
		Where("id IN ?", addressIDs).
		Update("customer_profile_id", newProfileID).Error
}

func (r *addressRepository) ClearDefaultByProfile(ctx context.Context, profileID uint) error {
	return r.db.WithContext(ctx).Model(&persistence.CustomerAddress{}).
		Where("customer_profile_id = ? AND is_default = ?", profileID, true).
		Update("is_default", false).Error
}
