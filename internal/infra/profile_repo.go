package infra

import (
	"context"

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

func NewCustomerMergeProfileRepository(db *gorm.DB) domain.CustomerMergeProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) Create(ctx context.Context, profile *domain.CustomerProfile) error {
	p := persistence.CustomerProfileFromDomain(profile)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*profile = *persistence.CustomerProfileToDomain(p)
	return nil
}

func (r *profileRepository) Update(ctx context.Context, profile *domain.CustomerProfile) error {
	p := persistence.CustomerProfileFromDomain(profile)
	p.ID = profile.ID
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *profileRepository) FindByID(ctx context.Context, id uint) (*domain.CustomerProfile, error) {
	var p persistence.CustomerProfile
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerProfileToDomain(&p), nil
}

func (r *profileRepository) List(ctx context.Context) ([]domain.CustomerProfile, error) {
	var ps []persistence.CustomerProfile
	if err := r.db.WithContext(ctx).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerProfile, len(ps))
	for i, p := range ps {
		result[i] = *persistence.CustomerProfileToDomain(&p)
	}
	return result, nil
}

func (r *profileRepository) CreateIdentity(ctx context.Context, identity *domain.CustomerIdentity) error {
	p := persistence.CustomerIdentityFromDomain(identity)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*identity = *persistence.CustomerIdentityToDomain(p)
	return nil
}

func (r *profileRepository) ListIdentitiesByProfile(ctx context.Context, profileID uint) ([]domain.CustomerIdentity, error) {
	var ps []persistence.CustomerIdentity
	if err := r.db.WithContext(ctx).Where("customer_profile_id = ?", profileID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerIdentity, len(ps))
	for i, p := range ps {
		result[i] = *persistence.CustomerIdentityToDomain(&p)
	}
	return result, nil
}

func (r *profileRepository) ListIdentitiesByIDs(ctx context.Context, identityIDs []uint) ([]domain.CustomerIdentity, error) {
	if len(identityIDs) == 0 {
		return []domain.CustomerIdentity{}, nil
	}
	var ps []persistence.CustomerIdentity
	if err := r.db.WithContext(ctx).Where("id IN ?", identityIDs).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.CustomerIdentity, len(ps))
	for i, p := range ps {
		result[i] = *persistence.CustomerIdentityToDomain(&p)
	}
	return result, nil
}

func (r *profileRepository) FindIdentityByPlatformAndValue(ctx context.Context, platform, value string) (*domain.CustomerIdentity, error) {
	var p persistence.CustomerIdentity
	if err := r.db.WithContext(ctx).Where("identity_platform = ? AND identity_value = ?", platform, value).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.CustomerIdentityToDomain(&p), nil
}

func (r *profileRepository) UpdateIdentityProfileID(ctx context.Context, identityID uint, newProfileID uint) error {
	return r.db.WithContext(ctx).Model(&persistence.CustomerIdentity{}).Where("id = ?", identityID).Update("customer_profile_id", newProfileID).Error
}

func (r *profileRepository) BulkUpdateIdentityProfileID(ctx context.Context, identityIDs []uint, newProfileID uint) error {
	if len(identityIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&persistence.CustomerIdentity{}).Where("id IN ?", identityIDs).Update("customer_profile_id", newProfileID).Error
}

func (r *profileRepository) SoftDelete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&persistence.CustomerProfile{}, id).Error
}

func (r *profileRepository) IsSoftDeleted(ctx context.Context, id uint) (bool, error) {
	var profile persistence.CustomerProfile
	if err := r.db.WithContext(ctx).Unscoped().First(&profile, id).Error; err != nil {
		return false, err
	}
	return profile.DeletedAt.Valid, nil
}

func (r *profileRepository) RestoreSoftDeleted(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Unscoped().Model(&persistence.CustomerProfile{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *profileRepository) DeleteIdentity(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&persistence.CustomerIdentity{}, id).Error
}
