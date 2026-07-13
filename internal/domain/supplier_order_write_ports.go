package domain

import (
	"context"
	"time"
)

// SupplierOrderLineWriter defines the write operations needed for supplier
// order lifecycle transitions (submit / accept) that are not covered by the
// generic SupplierOrderRepository.Update (which replaces the whole order
// row) or by SupplierOrderRepository.CreateLine/DeleteLinesByOrder (which
// operate on whole lines, not a single accepted-quantity field).
//
// It is implemented by the same underlying repository type as
// SupplierOrderRepository (see infra.NewSupplierOrderLineWriter), so a
// caller that already holds a *gorm.DB / transaction can construct both
// without duplicating connection wiring.
type SupplierOrderLineWriter interface {
	// UpdateOrderSubmission records the factory-assigned external order
	// number and the submission timestamp on a supplier order, and moves
	// its status forward (e.g. draft -> submitted).
	UpdateOrderSubmission(ctx context.Context, orderID uint, externalOrderNo string, submittedAt time.Time, status string) error

	// UpdateLineAcceptance records the factory-accepted quantity for a
	// single supplier order line and moves its status forward (e.g.
	// draft/submitted -> accepted).
	UpdateLineAcceptance(ctx context.Context, lineID uint, acceptedQuantity int, status string) error
}
