package infra

import (
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

func (r *shipmentRepository) Create(shipment *domain.Shipment) error {
	p := persistence.ToPersistenceShipment(shipment)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*shipment = *persistence.FromPersistenceShipment(p)
	return nil
}

func (r *shipmentRepository) FindByID(id uint) (*domain.Shipment, error) {
	var p persistence.Shipment
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceShipment(&p), nil
}

func (r *shipmentRepository) ListBySupplierOrder(supplierOrderID uint) ([]domain.Shipment, error) {
	var ps []persistence.Shipment
	if err := r.db.Where("supplier_order_id = ?", supplierOrderID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Shipment, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceShipment(&p)
	}
	return result, nil
}

func (r *shipmentRepository) ListByWave(waveID uint) ([]domain.Shipment, error) {
	var ps []persistence.Shipment
	if err := r.db.
		Joins("JOIN supplier_orders ON supplier_orders.id = shipments.supplier_order_id").
		Where("supplier_orders.wave_id = ?", waveID).
		Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.Shipment, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceShipment(&p)
	}
	return result, nil
}

func (r *shipmentRepository) CreateLine(line *domain.ShipmentLine) error {
	p := persistence.ToPersistenceShipmentLine(line)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*line = *persistence.FromPersistenceShipmentLine(p)
	return nil
}

func (r *shipmentRepository) AtomicCreateShipment(shipment *domain.Shipment, lines []*domain.ShipmentLine, pin *domain.BasisPinParam) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		pShip := persistence.ToPersistenceShipment(shipment)
		if err := tx.Create(pShip).Error; err != nil {
			return err
		}
		*shipment = *persistence.FromPersistenceShipment(pShip)
		for _, line := range lines {
			line.ShipmentID = shipment.ID
			pLine := persistence.ToPersistenceShipmentLine(line)
			if err := tx.Create(pLine).Error; err != nil {
				return err
			}
			*line = *persistence.FromPersistenceShipmentLine(pLine)
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

func (r *shipmentRepository) ListLinesByShipment(shipmentID uint) ([]domain.ShipmentLine, error) {
	var ps []persistence.ShipmentLine
	if err := r.db.Where("shipment_id = ?", shipmentID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ShipmentLine, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceShipmentLine(&p)
	}
	return result, nil
}
