package infra

import (
	"context"

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

func (r *supplierOrderRepository) Create(ctx context.Context, order *domain.SupplierOrder) error {
	p := persistence.SupplierOrderFromDomain(order)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*order = *persistence.SupplierOrderToDomain(p)
	return nil
}

func (r *supplierOrderRepository) FindByID(ctx context.Context, id uint) (*domain.SupplierOrder, error) {
	var p persistence.SupplierOrder
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.SupplierOrderToDomain(&p), nil
}

func (r *supplierOrderRepository) List(ctx context.Context) ([]domain.SupplierOrder, error) {
	var ps []persistence.SupplierOrder
	if err := r.db.WithContext(ctx).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.SupplierOrder, len(ps))
	for i, p := range ps {
		result[i] = *persistence.SupplierOrderToDomain(&p)
	}
	return result, nil
}

func (r *supplierOrderRepository) ListByWave(ctx context.Context, waveID uint) ([]domain.SupplierOrder, error) {
	var ps []persistence.SupplierOrder
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.SupplierOrder, len(ps))
	for i, p := range ps {
		result[i] = *persistence.SupplierOrderToDomain(&p)
	}
	return result, nil
}

func (r *supplierOrderRepository) CreateLine(ctx context.Context, line *domain.SupplierOrderLine) error {
	p := persistence.SupplierOrderLineFromDomain(line)
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return err
	}
	*line = *persistence.SupplierOrderLineToDomain(p)
	return nil
}

func (r *supplierOrderRepository) ListLinesByOrder(ctx context.Context, orderID uint) ([]domain.SupplierOrderLine, error) {
	var ps []persistence.SupplierOrderLine
	if err := r.db.WithContext(ctx).Where("supplier_order_id = ?", orderID).Find(&ps).Error; err != nil {
		return nil, err
	}
	result := make([]domain.SupplierOrderLine, len(ps))
	for i, p := range ps {
		result[i] = *persistence.SupplierOrderLineToDomain(&p)
	}
	return result, nil
}

func (r *supplierOrderRepository) FindLineByID(ctx context.Context, id uint) (*domain.SupplierOrderLine, error) {
	var p persistence.SupplierOrderLine
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return persistence.SupplierOrderLineToDomain(&p), nil
}

func (r *supplierOrderRepository) DeleteLinesByOrder(ctx context.Context, orderID uint) error {
	return r.db.WithContext(ctx).Where("supplier_order_id = ?", orderID).Delete(&persistence.SupplierOrderLine{}).Error
}

func (r *supplierOrderRepository) DeleteDraftsByWave(ctx context.Context, waveID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete lines belonging to draft orders for this wave in one batch
		if err := tx.Where("supplier_order_id IN (?)", tx.Model(&persistence.SupplierOrder{}).Select("id").Where("wave_id = ? AND status = ?", waveID, "draft")).Delete(&persistence.SupplierOrderLine{}).Error; err != nil {
			return err
		}
		// Delete the draft orders themselves
		if err := tx.Where("wave_id = ? AND status = ?", waveID, "draft").Delete(&persistence.SupplierOrder{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *supplierOrderRepository) Update(ctx context.Context, order *domain.SupplierOrder) error {
	po := persistence.SupplierOrderFromDomain(order)
	po.ID = order.ID
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *supplierOrderRepository) AtomicCreateSupplierOrder(ctx context.Context, order *domain.SupplierOrder, lines []*domain.SupplierOrderLine, pin *domain.BasisPinParam) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		p := persistence.SupplierOrderFromDomain(order)
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		*order = *persistence.SupplierOrderToDomain(p)

		for _, line := range lines {
			line.SupplierOrderID = order.ID
			pLine := persistence.SupplierOrderLineFromDomain(line)
			if err := tx.Create(pLine).Error; err != nil {
				return err
			}
			*line = *persistence.SupplierOrderLineToDomain(pLine)
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
