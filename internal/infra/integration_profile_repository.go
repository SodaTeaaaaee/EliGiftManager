package infra

import (
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

func (r *integrationProfileRepository) Create(profile *domain.IntegrationProfile) error {
	p := persistence.ToPersistenceIntegrationProfile(profile)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*profile = *persistence.FromPersistenceIntegrationProfile(p)
	return nil
}

func (r *integrationProfileRepository) FindByID(id uint) (*domain.IntegrationProfile, error) {
	var p persistence.IntegrationProfile
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceIntegrationProfile(&p), nil
}

func (r *integrationProfileRepository) FindByProfileKey(profileKey string) (*domain.IntegrationProfile, error) {
	var p persistence.IntegrationProfile
	if err := r.db.Where("profile_key = ?", profileKey).First(&p).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceIntegrationProfile(&p), nil
}

func (r *integrationProfileRepository) List() ([]domain.IntegrationProfile, error) {
	var ps []persistence.IntegrationProfile
	if err := r.db.Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.IntegrationProfile, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceIntegrationProfile(&p)
	}
	return result, nil
}
