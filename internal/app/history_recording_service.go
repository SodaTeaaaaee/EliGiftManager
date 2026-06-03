package app

import (
	"context"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// HistoryRecordingOption configures optional dependencies for HistoryRecordingService.
type HistoryRecordingOption func(*HistoryRecordingService)

// WithSnapshotService injects the wave snapshot service used to capture
// point-in-time snapshots during history recording.
func WithSnapshotService(svc *WaveSnapshotService) HistoryRecordingOption {
	return func(s *HistoryRecordingService) { s.snapshotSvc = svc }
}

type HistoryRecordingService struct {
	scopeRepo      domain.HistoryScopeRepository
	nodeRepo       domain.HistoryNodeRepository
	checkpointRepo domain.HistoryCheckpointRepository
	snapshotSvc    *WaveSnapshotService
}

func NewHistoryRecordingService(
	scopeRepo domain.HistoryScopeRepository,
	nodeRepo domain.HistoryNodeRepository,
	checkpointRepo domain.HistoryCheckpointRepository,
	opts ...HistoryRecordingOption,
) *HistoryRecordingService {
	svc := &HistoryRecordingService{
		scopeRepo:      scopeRepo,
		nodeRepo:       nodeRepo,
		checkpointRepo: checkpointRepo,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

func (s *HistoryRecordingService) FindScope(ctx context.Context, waveID uint) (*domain.HistoryScope, error) {
	return s.scopeRepo.FindByScopeTypeAndKey(ctx, "wave", fmt.Sprintf("%d", waveID))
}

type RecordNodeInput struct {
	WaveID                  uint
	CommandKind             string
	CommandSummary          string
	PatchPayload            string
	InversePatchPayload     string
	CheckpointHint          bool
	BaselineSnapshotPayload string
	SnapshotPayload         string
	ProjectionHash          string
	CreatedBy               string
}

func (s *HistoryRecordingService) RecordNode(ctx context.Context, input RecordNodeInput) (*domain.HistoryNode, error) {
	scope, err := s.scopeRepo.FindOrCreate(ctx, "wave", fmt.Sprintf("%d", input.WaveID))
	if err != nil {
		return nil, fmt.Errorf("history: find or create scope: %w", err)
	}

	if scope.CurrentHeadNodeID == 0 {
		if _, err := s.createSystemBaseline(ctx, scope, input.WaveID, input.BaselineSnapshotPayload); err != nil {
			return nil, err
		}
		scope, err = s.scopeRepo.FindByID(ctx, scope.ID)
		if err != nil {
			return nil, fmt.Errorf("history: reload scope after baseline: %w", err)
		}
		if scope == nil {
			return nil, fmt.Errorf("history: scope disappeared after baseline creation")
		}
	}

	node := &domain.HistoryNode{
		HistoryScopeID:      scope.ID,
		ParentNodeID:        scope.CurrentHeadNodeID,
		CommandKind:         input.CommandKind,
		CommandSummary:      input.CommandSummary,
		PatchPayload:        input.PatchPayload,
		InversePatchPayload: input.InversePatchPayload,
		CheckpointHint:      input.CheckpointHint,
		ProjectionHash:      input.ProjectionHash,
		CreatedBy:           input.CreatedBy,
	}

	if err := s.nodeRepo.Create(ctx, node); err != nil {
		return nil, fmt.Errorf("history: create node: %w", err)
	}

	if scope.CurrentHeadNodeID != 0 {
		if err := s.nodeRepo.UpdatePreferredRedoChild(ctx, scope.CurrentHeadNodeID, node.ID); err != nil {
			return nil, fmt.Errorf("history: update preferred redo child: %w", err)
		}
	}

	if err := s.scopeRepo.UpdateHead(ctx, scope.ID, node.ID); err != nil {
		return nil, fmt.Errorf("history: update head: %w", err)
	}

	needsCheckpoint := input.CheckpointHint
	if !needsCheckpoint {
		needsCheckpoint = s.shouldCreatePeriodicCheckpoint(ctx, node)
	}

	if needsCheckpoint {
		snapshotPayload := input.SnapshotPayload
		if snapshotPayload == "" && s.snapshotSvc != nil {
			var captureErr error
			snapshotPayload, captureErr = s.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
			if captureErr != nil {
				return nil, fmt.Errorf("history: capture checkpoint snapshot: %w", captureErr)
			}
		}
		if snapshotPayload != "" {
			cp := &domain.HistoryCheckpoint{
				HistoryScopeID:  scope.ID,
				HistoryNodeID:   node.ID,
				SnapshotPayload: snapshotPayload,
				SchemaVersion:   snapshotSchemaVersion,
			}
			if err := s.checkpointRepo.Create(ctx, cp); err != nil {
				return nil, fmt.Errorf("history: create checkpoint: %w", err)
			}
		}
	}

	return node, nil
}

func (s *HistoryRecordingService) createSystemBaseline(ctx context.Context, scope *domain.HistoryScope, waveID uint, snapshotPayload string) (*domain.HistoryNode, error) {
	node := &domain.HistoryNode{
		HistoryScopeID:      scope.ID,
		ParentNodeID:        0,
		CommandKind:         domain.CmdSystemBaseline,
		CommandSummary:      "system baseline",
		PatchPayload:        "",
		InversePatchPayload: "",
		CheckpointHint:      true,
		CreatedBy:           "system",
	}

	if snapshotPayload == "" && s.snapshotSvc != nil {
		capturedPayload, err := s.snapshotSvc.CaptureSnapshot(ctx, waveID)
		if err != nil {
			return nil, fmt.Errorf("history: capture baseline snapshot: %w", err)
		}
		snapshotPayload = capturedPayload
	}

	if snapshotPayload != "" {
		node.CheckpointHint = true
		if err := s.nodeRepo.Create(ctx, node); err != nil {
			return nil, fmt.Errorf("history: create baseline node: %w", err)
		}
		cp := &domain.HistoryCheckpoint{
			HistoryScopeID:  scope.ID,
			HistoryNodeID:   node.ID,
			SnapshotPayload: snapshotPayload,
			SchemaVersion:   snapshotSchemaVersion,
		}
		if err := s.checkpointRepo.Create(ctx, cp); err != nil {
			return nil, fmt.Errorf("history: create baseline checkpoint: %w", err)
		}
		if err := s.scopeRepo.UpdateHead(ctx, scope.ID, node.ID); err != nil {
			return nil, fmt.Errorf("history: update head to baseline: %w", err)
		}
		return node, nil
	}

	if err := s.nodeRepo.Create(ctx, node); err != nil {
		return nil, fmt.Errorf("history: create baseline node: %w", err)
	}
	if err := s.scopeRepo.UpdateHead(ctx, scope.ID, node.ID); err != nil {
		return nil, fmt.Errorf("history: update head to baseline: %w", err)
	}
	return node, nil
}

const checkpointInterval = 20

func (s *HistoryRecordingService) shouldCreatePeriodicCheckpoint(ctx context.Context, node *domain.HistoryNode) bool {
	count := 0
	current := node
	for current != nil && current.ParentNodeID != 0 {
		cp, _ := s.checkpointRepo.FindByNodeID(ctx, current.ID)
		if cp != nil {
			return false
		}
		count++
		if count >= checkpointInterval {
			return true
		}
		parent, err := s.nodeRepo.FindByID(ctx, current.ParentNodeID)
		if err != nil || parent == nil {
			break
		}
		current = parent
	}
	return count >= checkpointInterval
}
