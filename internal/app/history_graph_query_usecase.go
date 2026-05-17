package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// HistoryGraphQueryUseCase queries the history graph for a wave scope.
type HistoryGraphQueryUseCase interface {
	GetHistoryGraph(waveID uint) (*dto.HistoryGraphDTO, error)
	ListNodeChildren(nodeID uint) ([]dto.HistoryNodeDTO, error)
	GetNodeDetail(nodeID uint) (*dto.HistoryNodeDetailDTO, error)
}

type historyGraphQueryUseCase struct {
	scopeRepo      domain.HistoryScopeRepository
	nodeRepo       domain.HistoryNodeRepository
	checkpointRepo domain.HistoryCheckpointRepository
	pinRepo        domain.HistoryPinRepository
}

func NewHistoryGraphQueryUseCase(
	scopeRepo domain.HistoryScopeRepository,
	nodeRepo domain.HistoryNodeRepository,
	checkpointRepo domain.HistoryCheckpointRepository,
	pinRepo domain.HistoryPinRepository,
) HistoryGraphQueryUseCase {
	return &historyGraphQueryUseCase{
		scopeRepo:      scopeRepo,
		nodeRepo:       nodeRepo,
		checkpointRepo: checkpointRepo,
		pinRepo:        pinRepo,
	}
}

// GetHistoryGraph returns the full node graph for the given wave.
func (uc *historyGraphQueryUseCase) GetHistoryGraph(waveID uint) (*dto.HistoryGraphDTO, error) {
	scope, err := uc.scopeRepo.FindByScopeTypeAndKey("wave", fmt.Sprintf("%d", waveID))
	if err != nil {
		return nil, fmt.Errorf("history graph: find scope: %w", err)
	}
	if scope == nil {
		// No history yet — return empty graph
		return &dto.HistoryGraphDTO{Nodes: []dto.HistoryGraphNodeDTO{}}, nil
	}

	nodes, err := uc.nodeRepo.ListByScope(scope.ID)
	if err != nil {
		return nil, fmt.Errorf("history graph: list nodes: %w", err)
	}

	// Build child-count index
	childCount := make(map[uint]int, len(nodes))
	for _, n := range nodes {
		if n.ParentNodeID != 0 {
			childCount[n.ParentNodeID]++
		}
	}

	// Collect pinned node IDs for this scope
	pinnedIDs, err := uc.pinRepo.ListPinnedNodeIDsByScope(scope.ID)
	if err != nil {
		return nil, fmt.Errorf("history graph: list pinned nodes: %w", err)
	}
	pinned := make(map[uint]bool, len(pinnedIDs))
	for _, id := range pinnedIDs {
		pinned[id] = true
	}

	graphNodes := make([]dto.HistoryGraphNodeDTO, len(nodes))
	for i, n := range nodes {
		graphNodes[i] = dto.HistoryGraphNodeDTO{
			ID:                   n.ID,
			ParentNodeID:         n.ParentNodeID,
			PreferredRedoChildID: n.PreferredRedoChildID,
			CommandKind:          n.CommandKind,
			CommandSummary:       n.CommandSummary,
			ProjectionHash:       n.ProjectionHash,
			CheckpointHint:       n.CheckpointHint,
			CreatedAt:            n.CreatedAt,
			CreatedBy:            n.CreatedBy,
			IsCurrentHead:        n.ID == scope.CurrentHeadNodeID,
			IsPinned:             pinned[n.ID],
			ChildCount:           childCount[n.ID],
		}
	}

	return &dto.HistoryGraphDTO{
		ScopeID:       scope.ID,
		CurrentHeadID: scope.CurrentHeadNodeID,
		Nodes:         graphNodes,
	}, nil
}

// ListNodeChildren returns all direct children of a node.
func (uc *historyGraphQueryUseCase) ListNodeChildren(nodeID uint) ([]dto.HistoryNodeDTO, error) {
	node, err := uc.nodeRepo.FindByID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("history graph: find node: %w", err)
	}
	if node == nil {
		return nil, fmt.Errorf("history graph: node %d not found", nodeID)
	}

	// List all nodes in the same scope, then filter for parent = nodeID
	allNodes, err := uc.nodeRepo.ListByScope(node.HistoryScopeID)
	if err != nil {
		return nil, fmt.Errorf("history graph: list scope nodes: %w", err)
	}

	var children []dto.HistoryNodeDTO
	for _, n := range allNodes {
		if n.ParentNodeID == nodeID {
			children = append(children, domainToHistoryNodeDTO(&n))
		}
	}
	if children == nil {
		children = []dto.HistoryNodeDTO{}
	}
	return children, nil
}

// GetNodeDetail returns a node with its pins and checkpoint presence.
func (uc *historyGraphQueryUseCase) GetNodeDetail(nodeID uint) (*dto.HistoryNodeDetailDTO, error) {
	node, err := uc.nodeRepo.FindByID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("history graph: find node: %w", err)
	}
	if node == nil {
		return nil, fmt.Errorf("history graph: node %d not found", nodeID)
	}

	pins, err := uc.pinRepo.ListByNodeID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("history graph: list pins: %w", err)
	}

	cp, err := uc.checkpointRepo.FindByNodeID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("history graph: find checkpoint: %w", err)
	}

	pinDTOs := make([]dto.HistoryPinDTO, len(pins))
	for i, p := range pins {
		pinDTOs[i] = dto.HistoryPinDTO{
			ID:            p.ID,
			HistoryNodeID: p.HistoryNodeID,
			PinKind:       p.PinKind,
			RefType:       p.RefType,
			RefID:         p.RefID,
			CreatedAt:     p.CreatedAt,
		}
	}

	return &dto.HistoryNodeDetailDTO{
		HistoryNodeDTO: domainToHistoryNodeDTO(node),
		Pins:           pinDTOs,
		HasCheckpoint:  cp != nil,
	}, nil
}

func domainToHistoryNodeDTO(n *domain.HistoryNode) dto.HistoryNodeDTO {
	return dto.HistoryNodeDTO{
		ID:                   n.ID,
		ParentNodeID:         n.ParentNodeID,
		PreferredRedoChildID: n.PreferredRedoChildID,
		CommandKind:          n.CommandKind,
		CommandSummary:       n.CommandSummary,
		ProjectionHash:       n.ProjectionHash,
		CheckpointHint:       n.CheckpointHint,
		CreatedAt:            n.CreatedAt,
		CreatedBy:            n.CreatedBy,
	}
}
