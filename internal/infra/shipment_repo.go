package infra

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type shipmentRepository struct {
	db *gorm.DB
}

func NewShipmentRepository(db *gorm.DB) domain.ShipmentRepository {
	return &shipmentRepository{db: db}
}

func (r *shipmentRepository) Create(ctx context.Context, shipment *domain.Shipment) error {
	p := persistence.ShipmentFromDomain(shipment)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*shipment = *persistence.ShipmentToDomain(p)
	return nil
}

func (r *shipmentRepository) FindByID(ctx context.Context, id uint) (*domain.Shipment, error) {
	var p persistence.Shipment
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.ShipmentToDomain(&p), nil
}

func (r *shipmentRepository) ListBySupplierOrder(ctx context.Context, supplierOrderID uint) ([]domain.Shipment, error) {
	var ps []persistence.Shipment
	if err := r.db.WithContext(ctx).Where("supplier_order_id = ?", supplierOrderID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Shipment, len(ps))
	for i, p := range ps {
		result[i] = *persistence.ShipmentToDomain(&p)
	}
	return result, nil
}

func (r *shipmentRepository) ListByWave(ctx context.Context, waveID uint) ([]domain.Shipment, error) {
	var ps []persistence.Shipment
	if err := r.db.WithContext(ctx).
		Joins("JOIN supplier_orders ON supplier_orders.id = shipments.supplier_order_id").
		Where("supplier_orders.wave_id = ?", waveID).
		Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Shipment, len(ps))
	for i, p := range ps {
		result[i] = *persistence.ShipmentToDomain(&p)
	}
	return result, nil
}

func (r *shipmentRepository) CreateLine(ctx context.Context, line *domain.ShipmentLine) error {
	p := persistence.ShipmentLineFromDomain(line)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*line = *persistence.ShipmentLineToDomain(p)
	return nil
}

func (r *shipmentRepository) AtomicCreateShipment(ctx context.Context, shipment *domain.Shipment, lines []*domain.ShipmentLine, pin *domain.BasisPinParam) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		pShip := persistence.ShipmentFromDomain(shipment)
		if err := tx.Create(pShip).Error; err != nil {
			return err
		}
		*shipment = *persistence.ShipmentToDomain(pShip)
		for _, line := range lines {
			line.ShipmentID = shipment.ID
			pLine := persistence.ShipmentLineFromDomain(line)
			if err := tx.Create(pLine).Error; err != nil {
				return err
			}
			*line = *persistence.ShipmentLineToDomain(pLine)
		}
		if pin != nil && pin.HistoryNodeID != 0 {
			pPin := &persistence.HistoryPin{
				HistoryNodeID: pin.HistoryNodeID,
				PinKind:       pin.PinKind,
				RefType:       pin.RefType,
				RefID:         shipment.ID,
			}
			if err := tx.Create(pPin).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *shipmentRepository) ListLinesByShipment(ctx context.Context, shipmentID uint) ([]domain.ShipmentLine, error) {
	var ps []persistence.ShipmentLine
	if err := r.db.WithContext(ctx).Where("shipment_id = ?", shipmentID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ShipmentLine, len(ps))
	for i, p := range ps {
		result[i] = *persistence.ShipmentLineToDomain(&p)
	}
	return result, nil
}

func (r *shipmentRepository) SumShippedQuantityBySOL(ctx context.Context, supplierOrderLineID uint) (int, error) {
	var total int
	err := r.db.WithContext(ctx).Model(&persistence.ShipmentLine{}).
		Where("supplier_order_line_id = ?", supplierOrderLineID).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&total).Error
	return total, err
}
