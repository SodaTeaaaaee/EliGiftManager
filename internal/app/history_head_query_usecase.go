package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

type historyHeadQueryUseCase struct {
	scopeRepo domain.HistoryScopeRepository
	nodeRepo  domain.HistoryNodeRepository
}

func NewHistoryHeadQueryUseCase(
	scopeRepo domain.HistoryScopeRepository,
	nodeRepo domain.HistoryNodeRepository,
) HistoryHeadQueryUseCase {
	return &historyHeadQueryUseCase{
		scopeRepo: scopeRepo,
		nodeRepo:  nodeRepo,
	}
}

func (uc *historyHeadQueryUseCase) GetCurrentProjectionHash(waveID uint) (string, error) {
	scope, err := uc.scopeRepo.FindByScopeTypeAndKey("wave", fmt.Sprintf("%d", waveID))
	if err != nil {
		return "", err
	}
	if scope == nil || scope.CurrentHeadNodeID == 0 {
		return "", nil
	}

	node, err := uc.nodeRepo.FindByID(scope.CurrentHeadNodeID)
	if err != nil {
		return "", err
	}
	if node == nil {
		return "", nil
	}
	return node.ProjectionHash, nil
}

func (uc *historyHeadQueryUseCase) GetCurrentHeadNodeIDAndHash(waveID uint) (uint, string, error) {
	scope, err := uc.scopeRepo.FindByScopeTypeAndKey("wave", fmt.Sprintf("%d", waveID))
	if err != nil {
		return 0, "", err
	}
	if scope == nil || scope.CurrentHeadNodeID == 0 {
		return 0, "", nil
	}

	node, err := uc.nodeRepo.FindByID(scope.CurrentHeadNodeID)
	if err != nil {
		return 0, "", err
	}
	if node == nil {
		return 0, "", nil
	}
	return node.ID, node.ProjectionHash, nil
}
