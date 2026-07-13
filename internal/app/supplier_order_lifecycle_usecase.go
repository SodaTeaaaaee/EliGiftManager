package app

import (
	"context"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// SupplierOrderLifecycleUseCase completes the supplier-order lifecycle
// (plan 5.2 / 3.3.4): draft -> submitted -> accepted. It is deliberately a
// separate interface from ExportUseCase (which only creates draft orders)
// so the two responsibilities can evolve independently.
type SupplierOrderLifecycleUseCase interface {
	// MarkSupplierOrderSubmitted transitions a supplier order from draft to
	// submitted, recording the factory-assigned external order number and
	// the submission timestamp.
	MarkSupplierOrderSubmitted(ctx context.Context, input dto.MarkSupplierOrderSubmittedInput) (*domain.SupplierOrder, error)

	// RecordSupplierOrderAcceptance transitions a supplier order from
	// submitted to accepted, recording the factory-accepted quantity for
	// each line.
	RecordSupplierOrderAcceptance(ctx context.Context, input dto.RecordSupplierOrderAcceptanceInput) (*domain.SupplierOrder, error)
}

type supplierOrderLifecycleUseCase struct {
	supplierRepo domain.SupplierOrderRepository
	lineWriter   domain.SupplierOrderLineWriter
}

// NewSupplierOrderLifecycleUseCase constructs a SupplierOrderLifecycleUseCase.
// supplierRepo and lineWriter should be constructed against the same
// *gorm.DB (or the same transaction handle) so reads and writes observe a
// consistent view.
func NewSupplierOrderLifecycleUseCase(supplierRepo domain.SupplierOrderRepository, lineWriter domain.SupplierOrderLineWriter) SupplierOrderLifecycleUseCase {
	return &supplierOrderLifecycleUseCase{
		supplierRepo: supplierRepo,
		lineWriter:   lineWriter,
	}
}

func (uc *supplierOrderLifecycleUseCase) MarkSupplierOrderSubmitted(ctx context.Context, input dto.MarkSupplierOrderSubmittedInput) (*domain.SupplierOrder, error) {
	if input.OrderID == 0 {
		return nil, fmt.Errorf("order_id is required")
	}
	if input.ExternalOrderNo == "" {
		return nil, fmt.Errorf("external_order_no is required")
	}

	order, err := uc.supplierRepo.FindByID(ctx, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("supplier order %d lookup failed: %w", input.OrderID, err)
	}
	if order == nil {
		return nil, fmt.Errorf("supplier order %d not found", input.OrderID)
	}
	if order.Status != string(domain.SupplierOrderStatusDraft) {
		return nil, fmt.Errorf("supplier order %d is %q, expected %q to mark submitted", input.OrderID, order.Status, domain.SupplierOrderStatusDraft)
	}

	submittedAt := time.Now()
	if input.SubmittedAt != nil {
		submittedAt = *input.SubmittedAt
	}

	if err := uc.lineWriter.UpdateOrderSubmission(ctx, input.OrderID, input.ExternalOrderNo, submittedAt, string(domain.SupplierOrderStatusSubmitted)); err != nil {
		return nil, fmt.Errorf("supplier order %d submission update failed: %w", input.OrderID, err)
	}

	updated, err := uc.supplierRepo.FindByID(ctx, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("supplier order %d reload after submission failed: %w", input.OrderID, err)
	}
	return updated, nil
}

func (uc *supplierOrderLifecycleUseCase) RecordSupplierOrderAcceptance(ctx context.Context, input dto.RecordSupplierOrderAcceptanceInput) (*domain.SupplierOrder, error) {
	if input.OrderID == 0 {
		return nil, fmt.Errorf("order_id is required")
	}
	if len(input.Lines) == 0 {
		return nil, fmt.Errorf("at least one line acceptance entry is required")
	}

	order, err := uc.supplierRepo.FindByID(ctx, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("supplier order %d lookup failed: %w", input.OrderID, err)
	}
	if order == nil {
		return nil, fmt.Errorf("supplier order %d not found", input.OrderID)
	}
	if order.Status != string(domain.SupplierOrderStatusSubmitted) {
		return nil, fmt.Errorf("supplier order %d is %q, expected %q to record acceptance", input.OrderID, order.Status, domain.SupplierOrderStatusSubmitted)
	}

	for _, entry := range input.Lines {
		if entry.LineID == 0 {
			return nil, fmt.Errorf("line_id is required for every acceptance entry")
		}
		if entry.AcceptedQuantity < 0 {
			return nil, fmt.Errorf("accepted_quantity for line %d cannot be negative", entry.LineID)
		}

		line, err := uc.supplierRepo.FindLineByID(ctx, entry.LineID)
		if err != nil {
			return nil, fmt.Errorf("supplier order line %d lookup failed: %w", entry.LineID, err)
		}
		if line == nil {
			return nil, fmt.Errorf("supplier order line %d not found", entry.LineID)
		}
		if line.SupplierOrderID != input.OrderID {
			return nil, fmt.Errorf("supplier order line %d does not belong to supplier order %d", entry.LineID, input.OrderID)
		}

		if err := uc.lineWriter.UpdateLineAcceptance(ctx, entry.LineID, entry.AcceptedQuantity, string(domain.SupplierOrderStatusAccepted)); err != nil {
			return nil, fmt.Errorf("supplier order line %d acceptance update failed: %w", entry.LineID, err)
		}
	}

	order.Status = string(domain.SupplierOrderStatusAccepted)
	if err := uc.supplierRepo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("supplier order %d status update failed: %w", input.OrderID, err)
	}

	updated, err := uc.supplierRepo.FindByID(ctx, input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("supplier order %d reload after acceptance failed: %w", input.OrderID, err)
	}
	return updated, nil
}
