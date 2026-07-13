package infra

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

// shipmentWriterRepository implements domain.ShipmentWriteRepository against
// the same "shipments" table as shipmentRepository (shipment_repo.go), but
// lives in its own file/constructor per the parallel-unit file-boundary rule
// — shipment_repo.go is owned by another unit and must not be edited here.
type shipmentWriterRepository struct {
	db *gorm.DB
}

// NewShipmentWriterRepository constructs a domain.ShipmentWriteRepository
// bound to db (or a *gorm.DB transaction handle).
func NewShipmentWriterRepository(db *gorm.DB) domain.ShipmentWriteRepository {
	return &shipmentWriterRepository{db: db}
}

func (r *shipmentWriterRepository) Update(ctx context.Context, shipment *domain.Shipment) error {
	p := persistence.ShipmentFromDomain(shipment)
	p.ID = shipment.ID
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return err
	}
	*shipment = *persistence.ShipmentToDomain(p)
	return nil
}
