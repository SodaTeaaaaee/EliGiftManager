package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// HistoryGCService performs pin-aware garbage collection on history nodes for a scope.
type HistoryGCService struct {
	scopeRepo      domain.HistoryScopeRepository
	nodeRepo       domain.HistoryNodeRepository
	checkpointRepo domain.HistoryCheckpointRepository
	pinRepo        domain.HistoryPinRepository
}

func NewHistoryGCService(
	scopeRepo domain.HistoryScopeRepository,
	nodeRepo domain.HistoryNodeRepository,
	checkpointRepo domain.HistoryCheckpointRepository,
	pinRepo domain.HistoryPinRepository,
) *HistoryGCService {
	return &HistoryGCService{
		scopeRepo:      scopeRepo,
		nodeRepo:       nodeRepo,
		checkpointRepo: checkpointRepo,
		pinRepo:        pinRepo,
	}
}

// CollectGarbageForWave runs GC for the history scope of the given wave.
// Returns the number of deleted nodes, or 0 if no history scope exists yet.
func (s *HistoryGCService) CollectGarbageForWave(waveID uint, keepCount int) (int, error) {
	scope, err := s.scopeRepo.FindByScopeTypeAndKey("wave", fmt.Sprintf("%d", waveID))
	if err != nil {
		return 0, fmt.Errorf("history gc: find scope for wave %d: %w", waveID, err)
	}
	if scope == nil {
		return 0, nil
	}
	return s.CollectGarbage(scope.ID, keepCount)
}

// CollectGarbage removes unreachable, unpinned nodes beyond keepCount positions from head.
// Reachable = on the parent chain from current head, or is a preferredRedoChild of a reachable node.
// Pinned nodes are always preserved regardless of position.
// Returns the number of deleted nodes.
func (s *HistoryGCService) CollectGarbage(scopeID uint, keepCount int) (int, error) {
	scope, err := s.scopeRepo.FindByID(scopeID)
	if err != nil {
		return 0, fmt.Errorf("history gc: load scope: %w", err)
	}
	if scope == nil {
		return 0, fmt.Errorf("history gc: scope %d not found", scopeID)
	}

	allNodes, err := s.nodeRepo.ListByScope(scopeID)
	if err != nil {
		return 0, fmt.Errorf("history gc: list nodes: %w", err)
	}
	if len(allNodes) == 0 {
		return 0, nil
	}

	// Build index: id → node
	byID := make(map[uint]*domain.HistoryNode, len(allNodes))
	for i := range allNodes {
		byID[allNodes[i].ID] = &allNodes[i]
	}

	// Collect pinned node IDs
	pinnedIDs, err := s.pinRepo.ListPinnedNodeIDsByScope(scopeID)
	if err != nil {
		return 0, fmt.Errorf("history gc: list pinned nodes: %w", err)
	}
	pinned := make(map[uint]bool, len(pinnedIDs))
	for _, id := range pinnedIDs {
		pinned[id] = true
	}

	// BFS/walk from current head to build reachable set.
	// Walk 1: parent chain from head (undo direction).
	// Walk 2: for each reachable node, also include its preferredRedoChild (redo direction).
	reachable := make(map[uint]bool)
	queue := []uint{}
	if scope.CurrentHeadNodeID != 0 {
		queue = append(queue, scope.CurrentHeadNodeID)
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if reachable[cur] {
			continue
		}
		reachable[cur] = true

		node, ok := byID[cur]
		if !ok {
			continue
		}
		// Follow parent chain (undo direction)
		if node.ParentNodeID != 0 && !reachable[node.ParentNodeID] {
			queue = append(queue, node.ParentNodeID)
		}
		// Follow preferred redo child (redo branch tip)
		if node.PreferredRedoChildID != 0 && !reachable[node.PreferredRedoChildID] {
			queue = append(queue, node.PreferredRedoChildID)
		}
	}

	// Determine the head chain depth: nodes at position > keepCount from head are candidates.
	// We assign depth by walking the parent chain from head.
	depth := make(map[uint]int)
	cur := scope.CurrentHeadNodeID
	d := 0
	for cur != 0 {
		depth[cur] = d
		d++
		node, ok := byID[cur]
		if !ok {
			break
		}
		cur = node.ParentNodeID
	}

	// Collect candidates for deletion:
	// - not reachable via head-BFS
	// - not pinned
	// - if reachable but on parent chain, must be deeper than keepCount
	var toDelete []uint
	for _, node := range allNodes {
		if pinned[node.ID] {
			continue
		}
		if !reachable[node.ID] {
			// Completely orphaned node — safe to delete
			toDelete = append(toDelete, node.ID)
			continue
		}
		// Reachable but on the parent chain beyond keepCount
		if nodeDepth, onChain := depth[node.ID]; onChain {
			if nodeDepth >= keepCount {
				// Only delete if not pinned (already checked) and not current head
				if node.ID != scope.CurrentHeadNodeID {
					toDelete = append(toDelete, node.ID)
				}
			}
		}
	}

	// Delete collected nodes: checkpoints first, then nodes
	deleted := 0
	for _, nodeID := range toDelete {
		if err := s.checkpointRepo.DeleteByNodeID(nodeID); err != nil {
			return deleted, fmt.Errorf("history gc: delete checkpoint for node %d: %w", nodeID, err)
		}
		if err := s.nodeRepo.DeleteByID(nodeID); err != nil {
			return deleted, fmt.Errorf("history gc: delete node %d: %w", nodeID, err)
		}
		deleted++
	}

	return deleted, nil
}
