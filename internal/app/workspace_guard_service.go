package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// WorkspaceGuardService validates cross-step invariants before allowing
// navigation or mutation in the wave workspace. Each guard method returns
// nil if the step is accessible, or an error describing what must be done first.
type WorkspaceGuardService struct {
	assignmentRepo domain.WaveDemandAssignmentRepository
	fulfillRepo    domain.FulfillmentLineRepository
	supplierRepo   domain.SupplierOrderRepository
	shipmentRepo   domain.ShipmentRepository
}

func NewWorkspaceGuardService(
	assignmentRepo domain.WaveDemandAssignmentRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	supplierRepo domain.SupplierOrderRepository,
	shipmentRepo domain.ShipmentRepository,
) *WorkspaceGuardService {
	return &WorkspaceGuardService{
		assignmentRepo: assignmentRepo,
		fulfillRepo:    fulfillRepo,
		supplierRepo:   supplierRepo,
		shipmentRepo:   shipmentRepo,
	}
}

// GuardAllocationRequiresDemandIntake checks that at least one demand document
// is assigned before entering the allocation step.
func (s *WorkspaceGuardService) GuardAllocationRequiresDemandIntake(waveID uint) error {
	assignments, err := s.assignmentRepo.ListByWave(waveID)
	if err != nil {
		return fmt.Errorf("check demand intake: %w", err)
	}
	if len(assignments) == 0 {
		return fmt.Errorf("no demand documents assigned; complete demand intake first")
	}
	return nil
}

// GuardReviewRequiresFulfillment checks that fulfillment lines exist before review.
func (s *WorkspaceGuardService) GuardReviewRequiresFulfillment(waveID uint) error {
	lines, err := s.fulfillRepo.ListByWave(waveID)
	if err != nil {
		return fmt.Errorf("check fulfillment: %w", err)
	}
	if len(lines) == 0 {
		return fmt.Errorf("no fulfillment lines; complete allocation/mapping first")
	}
	return nil
}

// GuardExecutionRequiresReview checks that supplier orders exist.
func (s *WorkspaceGuardService) GuardExecutionRequiresReview(waveID uint) error {
	orders, err := s.supplierRepo.ListByWave(waveID)
	if err != nil {
		return fmt.Errorf("check supplier orders: %w", err)
	}
	if len(orders) == 0 {
		return fmt.Errorf("no supplier orders exported; complete review first")
	}
	return nil
}

// GuardShipmentRequiresSupplierOrder checks that supplier orders exist before shipment intake.
func (s *WorkspaceGuardService) GuardShipmentRequiresSupplierOrder(waveID uint) error {
	return s.GuardExecutionRequiresReview(waveID)
}

// GuardSyncRequiresShipment checks that shipments exist before channel sync.
func (s *WorkspaceGuardService) GuardSyncRequiresShipment(waveID uint) error {
	shipments, err := s.shipmentRepo.ListByWave(waveID)
	if err != nil {
		return fmt.Errorf("check shipments: %w", err)
	}
	if len(shipments) == 0 {
		return fmt.Errorf("no shipments recorded; complete shipment intake first")
	}
	return nil
}
