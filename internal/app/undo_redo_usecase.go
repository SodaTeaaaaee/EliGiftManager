package app

import (
	"errors"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type undoRedoUseCase struct {
	scopeRepo     domain.HistoryScopeRepository
	nodeRepo      domain.HistoryNodeRepository
	patchExecutor *PatchExecutor
}

func NewUndoRedoUseCase(
	scopeRepo domain.HistoryScopeRepository,
	nodeRepo domain.HistoryNodeRepository,
	patchExecutor ...*PatchExecutor,
) UndoRedoUseCase {
	uc := &undoRedoUseCase{scopeRepo: scopeRepo, nodeRepo: nodeRepo}
	if len(patchExecutor) > 0 && patchExecutor[0] != nil {
		uc.patchExecutor = patchExecutor[0]
	}
	return uc
}

func (uc *undoRedoUseCase) Undo(waveID uint) (string, error) {
	scope, err := uc.scopeRepo.FindByScopeTypeAndKey("wave", fmt.Sprintf("%d", waveID))
	if err != nil {
		return "", err
	}
	if scope == nil || scope.CurrentHeadNodeID == 0 {
		return "", fmt.Errorf("no history for wave %d", waveID)
	}

	currentNode, err := uc.nodeRepo.FindByID(scope.CurrentHeadNodeID)
	if err != nil {
		return "", err
	}
	if currentNode == nil || currentNode.ParentNodeID == 0 {
		return "", fmt.Errorf("nothing to undo")
	}

	if uc.patchExecutor != nil && currentNode.InversePatchPayload != "" {
		if err := uc.patchExecutor.ApplyInversePatch(currentNode.InversePatchPayload); err != nil {
			if errors.Is(err, ErrOperationNotUndoable) {
				return "", fmt.Errorf("cannot undo %q: %w", currentNode.CommandSummary, err)
			}
			return "", fmt.Errorf("undo failed for %q: %w", currentNode.CommandSummary, err)
		}
	}

	if err := uc.scopeRepo.UpdateHead(scope.ID, currentNode.ParentNodeID); err != nil {
		return "", err
	}
	return currentNode.CommandSummary, nil
}

func (uc *undoRedoUseCase) Redo(waveID uint) (string, error) {
	scope, err := uc.scopeRepo.FindByScopeTypeAndKey("wave", fmt.Sprintf("%d", waveID))
	if err != nil {
		return "", err
	}
	if scope == nil || scope.CurrentHeadNodeID == 0 {
		return "", fmt.Errorf("no history for wave %d", waveID)
	}

	currentNode, err := uc.nodeRepo.FindByID(scope.CurrentHeadNodeID)
	if err != nil {
		return "", err
	}
	if currentNode == nil || currentNode.PreferredRedoChildID == 0 {
		return "", fmt.Errorf("nothing to redo")
	}

	childNode, err := uc.nodeRepo.FindByID(currentNode.PreferredRedoChildID)
	if err != nil {
		return "", err
	}
	if childNode == nil {
		return "", fmt.Errorf("nothing to redo")
	}

	if uc.patchExecutor != nil && childNode.PatchPayload != "" {
		if err := uc.patchExecutor.ApplyPatch(childNode.PatchPayload); err != nil {
			if errors.Is(err, ErrOperationNotUndoable) {
				return "", fmt.Errorf("cannot redo %q: %w", childNode.CommandSummary, err)
			}
			return "", fmt.Errorf("redo failed for %q: %w", childNode.CommandSummary, err)
		}
	}

	if err := uc.scopeRepo.UpdateHead(scope.ID, currentNode.PreferredRedoChildID); err != nil {
		return "", err
	}
	return childNode.CommandSummary, nil
}
