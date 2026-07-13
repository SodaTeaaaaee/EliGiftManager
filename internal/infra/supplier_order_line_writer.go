package infra

import (
	"context"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"
	"gorm.io/gorm"
)

// NewSupplierOrderLineWriter constructs a domain.SupplierOrderLineWriter
// backed by the same underlying repository type as
// NewSupplierOrderRepository. Both can be constructed against the same
// *gorm.DB (or the same transaction handle) without duplicating wiring.
func NewSupplierOrderLineWriter(db *gorm.DB) domain.SupplierOrderLineWriter {
	return &supplierOrderRepository{db: db}
}

// UpdateOrderSubmission implements domain.SupplierOrderLineWriter.
func (r *supplierOrderRepository) UpdateOrderSubmission(ctx context.Context, orderID uint, externalOrderNo string, submittedAt time.Time, status string) error {
	return r.db.WithContext(ctx).
		Model(&persistence.SupplierOrder{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"external_order_no": externalOrderNo,
			"submitted_at":      submittedAt,
			"status":            persistence.SupplierOrderStatus(status),
		}).Error
}

// UpdateLineAcceptance implements domain.SupplierOrderLineWriter.
func (r *supplierOrderRepository) UpdateLineAcceptance(ctx context.Context, lineID uint, acceptedQuantity int, status string) error {
	return r.db.WithContext(ctx).
		Model(&persistence.SupplierOrderLine{}).
		Where("id = ?", lineID).
		Updates(map[string]interface{}{
			"accepted_quantity": acceptedQuantity,
			"status":            status,
		}).Error
}
