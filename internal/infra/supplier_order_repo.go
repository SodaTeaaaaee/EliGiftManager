package infra

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

type supplierOrderRepository struct {
	db *gorm.DB
}

func NewSupplierOrderRepository(db *gorm.DB) domain.SupplierOrderRepository {
	return &supplierOrderRepository{db: db}
}

func (r *supplierOrderRepository) Create(order *domain.SupplierOrder) error {
	p := persistence.ToPersistenceSupplierOrder(order)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*order = *persistence.FromPersistenceSupplierOrder(p)
	return nil
}

func (r *supplierOrderRepository) FindByID(id uint) (*domain.SupplierOrder, error) {
	var p persistence.SupplierOrder
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceSupplierOrder(&p), nil
}

func (r *supplierOrderRepository) List() ([]domain.SupplierOrder, error) {
	var ps []persistence.SupplierOrder
	if err := r.db.Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.SupplierOrder, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceSupplierOrder(&p)
	}
	return result, nil
}

func (r *supplierOrderRepository) ListByWave(waveID uint) ([]domain.SupplierOrder, error) {
	var ps []persistence.SupplierOrder
	if err := r.db.Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.SupplierOrder, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceSupplierOrder(&p)
	}
	return result, nil
}

func (r *supplierOrderRepository) CreateLine(line *domain.SupplierOrderLine) error {
	p := persistence.ToPersistenceSupplierOrderLine(line)
	if err := r.db.Create(p).Error; err != nil {
		return err
	}
	*line = *persistence.FromPersistenceSupplierOrderLine(p)
	return nil
}

func (r *supplierOrderRepository) ListLinesByOrder(orderID uint) ([]domain.SupplierOrderLine, error) {
	var ps []persistence.SupplierOrderLine
	if err := r.db.Where("supplier_order_id = ?", orderID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.SupplierOrderLine, len(ps))
	for i, p := range ps {
		result[i] = *persistence.FromPersistenceSupplierOrderLine(&p)
	}
	return result, nil
}

func (r *supplierOrderRepository) FindLineByID(id uint) (*domain.SupplierOrderLine, error) {
	var p persistence.SupplierOrderLine
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.FromPersistenceSupplierOrderLine(&p), nil
}

func (r *supplierOrderRepository) DeleteLinesByOrder(orderID uint) error {
	return r.db.Where("supplier_order_id = ?", orderID).Delete(&persistence.SupplierOrderLine{}).Error
}

func (r *supplierOrderRepository) DeleteDraftsByWave(waveID uint) error {
	// Find all draft orders for this wave
	var orders []persistence.SupplierOrder
	if err := r.db.Where("wave_id = ? AND status = ?", waveID, "draft").Find(&orders).Error; err != nil {
		return err
	}
	for _, o := range orders {
		if err := r.db.Where("supplier_order_id = ?", o.ID).Delete(&persistence.SupplierOrderLine{}).Error; err != nil {
			return err
		}
	}
	if err := r.db.Where("wave_id = ? AND status = ?", waveID, "draft").Delete(&persistence.SupplierOrder{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *supplierOrderRepository) AtomicCreateSupplierOrder(order *domain.SupplierOrder, lines []*domain.SupplierOrderLine, pin *domain.BasisPinParam) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		p := persistence.ToPersistenceSupplierOrder(order)
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		*order = *persistence.FromPersistenceSupplierOrder(p)

		for _, line := range lines {
			line.SupplierOrderID = order.ID
			pLine := persistence.ToPersistenceSupplierOrderLine(line)
			if err := tx.Create(pLine).Error; err != nil {
				return err
			}
			*line = *persistence.FromPersistenceSupplierOrderLine(pLine)
		}

		if pin != nil && pin.HistoryNodeID != 0 {
			pPin := &persistence.HistoryPin{
				HistoryNodeID: pin.HistoryNodeID,
				PinKind:       pin.PinKind,
				RefType:       pin.RefType,
				RefID:         order.ID,
			}
			if err := tx.Create(pPin).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
