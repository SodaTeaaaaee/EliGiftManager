package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type HistoryRecordingService struct {
	scopeRepo      domain.HistoryScopeRepository
	nodeRepo       domain.HistoryNodeRepository
	checkpointRepo domain.HistoryCheckpointRepository
}

func NewHistoryRecordingService(
	scopeRepo domain.HistoryScopeRepository,
	nodeRepo domain.HistoryNodeRepository,
	checkpointRepo domain.HistoryCheckpointRepository,
) *HistoryRecordingService {
	return &HistoryRecordingService{
		scopeRepo:      scopeRepo,
		nodeRepo:       nodeRepo,
		checkpointRepo: checkpointRepo,
	}
}

type RecordNodeInput struct {
	WaveID              uint
	CommandKind         string
	CommandSummary      string
	PatchPayload        string
	InversePatchPayload string
	CheckpointHint      bool
	SnapshotPayload     string
	ProjectionHash      string
	CreatedBy           string
}

func (s *HistoryRecordingService) RecordNode(input RecordNodeInput) (*domain.HistoryNode, error) {
	scope, err := s.scopeRepo.FindOrCreate("wave", fmt.Sprintf("%d", input.WaveID))
	if err != nil {
		return nil, fmt.Errorf("history: find or create scope: %w", err)
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

	if err := s.nodeRepo.Create(node); err != nil {
		return nil, fmt.Errorf("history: create node: %w", err)
	}

	if scope.CurrentHeadNodeID != 0 {
		if err := s.nodeRepo.UpdatePreferredRedoChild(scope.CurrentHeadNodeID, node.ID); err != nil {
			return nil, fmt.Errorf("history: update preferred redo child: %w", err)
		}
	}

	if err := s.scopeRepo.UpdateHead(scope.ID, node.ID); err != nil {
		return nil, fmt.Errorf("history: update head: %w", err)
	}

	if input.CheckpointHint && input.SnapshotPayload != "" {
		cp := &domain.HistoryCheckpoint{
			HistoryScopeID:  scope.ID,
			HistoryNodeID:   node.ID,
			SnapshotPayload: input.SnapshotPayload,
		}
		if err := s.checkpointRepo.Create(cp); err != nil {
			return nil, fmt.Errorf("history: create checkpoint: %w", err)
		}
	}

	return node, nil
}
