package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// LifecycleProjectionService computes and persists Wave.lifecycle_stage
// and Wave.progress_snapshot after every state-changing command.
type LifecycleProjectionService struct {
	waveRepo       domain.WaveRepository
	fulfillRepo    domain.FulfillmentLineRepository
	supplierRepo   domain.SupplierOrderRepository
	shipmentRepo   domain.ShipmentRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
	channelSync    domain.ChannelSyncRepository
}

func NewLifecycleProjectionService(
	waveRepo domain.WaveRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	supplierRepo domain.SupplierOrderRepository,
	shipmentRepo domain.ShipmentRepository,
	assignmentRepo domain.WaveDemandAssignmentRepository,
	channelSync domain.ChannelSyncRepository,
) *LifecycleProjectionService {
	return &LifecycleProjectionService{
		waveRepo:       waveRepo,
		fulfillRepo:    fulfillRepo,
		supplierRepo:   supplierRepo,
		shipmentRepo:   shipmentRepo,
		assignmentRepo: assignmentRepo,
		channelSync:    channelSync,
	}
}

// ProgressSnapshot captures a lightweight state summary for Wave.progress_snapshot.
type ProgressSnapshot struct {
	DemandCount        int `json:"demandCount"`
	FulfillmentCount   int `json:"fulfillmentCount"`
	SupplierOrderCount int `json:"supplierOrderCount"`
	ShipmentCount      int `json:"shipmentCount"`
	SyncJobCount       int `json:"syncJobCount"`
	ParticipantCount   int `json:"participantCount"`
}

// ProjectAndPersist computes the current lifecycle stage and progress snapshot
// from repository state, then persists them to the Wave record.
func (s *LifecycleProjectionService) ProjectAndPersist(ctx context.Context, waveID uint) error {
	stage, snap, err := s.compute(ctx, waveID)
	if err != nil {
		return err
	}
	snapJSON, _ := json.Marshal(snap)
	return s.waveRepo.UpdateLifecycle(ctx, waveID, stage, string(snapJSON))
}

func (s *LifecycleProjectionService) compute(ctx context.Context, waveID uint) (string, ProgressSnapshot, error) {
	var snap ProgressSnapshot

	// Count demand documents assigned
	assignments, err := s.assignmentRepo.ListByWave(ctx, waveID)
	if err != nil {
		return "", snap, fmt.Errorf("list assignments: %w", err)
	}
	snap.DemandCount = len(assignments)

	// Count fulfillment lines
	fulfillLines, err := s.fulfillRepo.ListByWave(ctx, waveID)
	if err != nil {
		return "", snap, fmt.Errorf("list fulfillment lines: %w", err)
	}
	snap.FulfillmentCount = len(fulfillLines)

	// Count supplier orders
	orders, err := s.supplierRepo.ListByWave(ctx, waveID)
	if err != nil {
		return "", snap, fmt.Errorf("list supplier orders: %w", err)
	}
	snap.SupplierOrderCount = len(orders)

	// Count shipments
	shipments, err := s.shipmentRepo.ListByWave(ctx, waveID)
	if err != nil {
		return "", snap, fmt.Errorf("list shipments: %w", err)
	}
	snap.ShipmentCount = len(shipments)

	// Count sync jobs
	jobs, err := s.channelSync.ListJobsByWave(ctx, waveID)
	if err != nil {
		return "", snap, fmt.Errorf("list sync jobs: %w", err)
	}
	snap.SyncJobCount = len(jobs)

	// Count participants
	participants, err := s.waveRepo.ListParticipantsByWave(ctx, waveID)
	if err != nil {
		return "", snap, fmt.Errorf("list participants: %w", err)
	}
	snap.ParticipantCount = len(participants)

	// Derive lifecycle stage
	stage := deriveStageFromCounts(snap, jobs)
	return stage, snap, nil
}

func deriveStageFromCounts(snap ProgressSnapshot, jobs []domain.ChannelSyncJob) string {
	if snap.DemandCount == 0 {
		return "intake"
	}
	if snap.FulfillmentCount == 0 {
		return "allocation"
	}
	if snap.SupplierOrderCount == 0 {
		return "review"
	}
	if snap.ShipmentCount == 0 {
		return "execution"
	}
	// Check sync state
	hasActive := false
	hasFailed := false
	for _, j := range jobs {
		switch j.Status {
		case "pending", "running":
			hasActive = true
		case "failed", "partial_success":
			hasFailed = true
		}
	}
	if hasActive {
		return "syncing_back"
	}
	if hasFailed || snap.SyncJobCount > 0 {
		return "awaiting_manual_closure"
	}
	return "closed"
}
