package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type integrationProfileRepository struct {
	db *gorm.DB
}

func NewIntegrationProfileRepository(db *gorm.DB) domain.IntegrationProfileRepository {
	return &integrationProfileRepository{db: db}
}

func (r *integrationProfileRepository) Create(ctx context.Context, profile *domain.IntegrationProfile) error {
	p := persistence.IntegrationProfileFromDomain(profile)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*profile = *persistence.IntegrationProfileToDomain(p)
	return nil
}

func (r *integrationProfileRepository) FindByID(ctx context.Context, id uint) (*domain.IntegrationProfile, error) {
	var p persistence.IntegrationProfile
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.IntegrationProfileToDomain(&p), nil
}

func (r *integrationProfileRepository) FindByProfileKey(ctx context.Context, profileKey string) (*domain.IntegrationProfile, error) {
	var p persistence.IntegrationProfile
	if err := r.db.WithContext(ctx).Where("profile_key = ?", profileKey).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.IntegrationProfileToDomain(&p), nil
}

func (r *integrationProfileRepository) List(ctx context.Context) ([]domain.IntegrationProfile, error) {
	var ps []persistence.IntegrationProfile
	if err := r.db.WithContext(ctx).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.IntegrationProfile, len(ps))
	for i, p := range ps {
		result[i] = *persistence.IntegrationProfileToDomain(&p)
	}
	return result, nil
}

func (r *integrationProfileRepository) Update(ctx context.Context, profile *domain.IntegrationProfile) error {
	p := persistence.IntegrationProfileFromDomain(profile)
	p.ID = profile.ID
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *integrationProfileRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var profile persistence.IntegrationProfile
		if err := tx.First(&profile, id).Error; err != nil {
			return err
		}
		// SQLite unique indexes include soft-deleted rows. Rewrite the key before
		// soft deletion so operators may recreate the same stable profile key.
		deletedKey := fmt.Sprintf("%s__deleted_%d_%d", profile.ProfileKey, id, time.Now().UTC().UnixNano())
		if err := tx.Model(&profile).Update("profile_key", deletedKey).Error; err != nil {
			return err
		}
		return tx.Delete(&profile).Error
	})
}
