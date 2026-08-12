package infra

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type integrationProfileReferenceRepository struct {
	db *gorm.DB
}

func NewIntegrationProfileReferenceRepository(db *gorm.DB) domain.IntegrationProfileReferenceRepository {
	return &integrationProfileReferenceRepository{db: db}
}

func (r *integrationProfileReferenceRepository) CountReferences(ctx context.Context, profileID uint) (map[string]int64, error) {
	targets := []struct {
		kind  string
		model any
		where string
	}{
		{kind: "carrier mappings", model: &persistence.CarrierMapping{}, where: "integration_profile_id = ?"},
		{kind: "supplier orders", model: &persistence.SupplierOrder{}, where: "factory_integration_profile_id = ?"},
		{kind: "customer identities", model: &persistence.CustomerIdentity{}, where: "source_integration_profile_id = ?"},
		{kind: "customer profile origins", model: &persistence.CustomerProfileOrigin{}, where: "source_integration_profile_id = ?"},
		{kind: "customer name observations", model: &persistence.CustomerNameObservation{}, where: "source_integration_profile_id = ?"},
	}
	counts := make(map[string]int64, len(targets))
	for _, target := range targets {
		var count int64
		if err := r.db.WithContext(ctx).Model(target.model).Where(target.where, profileID).Count(&count).Error; err != nil {
			return nil, fmt.Errorf("count %s: %w", target.kind, err)
		}
		counts[target.kind] = count
	}
	return counts, nil
}
