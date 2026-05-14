package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type undoRedoUseCase struct {
	scopeRepo domain.HistoryScopeRepository
	nodeRepo  domain.HistoryNodeRepository
}

func NewUndoRedoUseCase(
	scopeRepo domain.HistoryScopeRepository,
	nodeRepo domain.HistoryNodeRepository,
) UndoRedoUseCase {
	return &undoRedoUseCase{scopeRepo: scopeRepo, nodeRepo: nodeRepo}
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

	if err := uc.scopeRepo.UpdateHead(scope.ID, currentNode.PreferredRedoChildID); err != nil {
		return "", err
	}

	childNode, err := uc.nodeRepo.FindByID(currentNode.PreferredRedoChildID)
	if err != nil {
		return "", err
	}
	if childNode != nil {
		return childNode.CommandSummary, nil
	}
	return "", nil
}
