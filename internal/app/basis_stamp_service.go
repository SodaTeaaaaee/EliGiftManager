package app

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// BasisStampService resolves the current history head for a wave and creates
// HistoryPin records that anchor external objects (SupplierOrder, Shipment,
// ChannelSyncJob) to the projection snapshot they were created against.
//
// Usage — two-step pattern:
//  1. Call ResolveBasis BEFORE persisting the object to populate its basis fields.
//  2. Call CreatePin AFTER persisting the object (once its ID is available).
type BasisStampService struct {
	historyHeadUC HistoryHeadQueryUseCase
	pinRepo       domain.HistoryPinRepository
}

func NewBasisStampService(
	historyHeadUC HistoryHeadQueryUseCase,
	pinRepo domain.HistoryPinRepository,
) *BasisStampService {
	return &BasisStampService{
		historyHeadUC: historyHeadUC,
		pinRepo:       pinRepo,
	}
}

// ResolveBasis returns the current head node ID (as a string) and projection
// hash for the given wave. Returns empty strings when no history scope exists
// yet — callers should treat that as "no basis to stamp" and skip CreatePin.
//
// Call this BEFORE persisting the external object so its basis fields can be
// set before the INSERT.
func (s *BasisStampService) ResolveBasis(waveID uint) (nodeID string, projectionHash string, err error) {
	nid, hash, err := s.historyHeadUC.GetCurrentHeadNodeIDAndHash(waveID)
	if err != nil {
		return "", "", err
	}
	if nid == 0 {
		return "", "", nil
	}
	return fmt.Sprintf("%d", nid), hash, nil
}

// CreatePin creates a HistoryPin that records which external object was created
// against a given basis node. basisNodeID must be the value returned by
// ResolveBasis; if it is empty the call is a no-op (no history scope existed).
//
// Call this AFTER persisting the external object so refID is populated.
func (s *BasisStampService) CreatePin(basisNodeID string, pinKind string, refType string, refID uint) error {
	if basisNodeID == "" {
		return nil
	}
	var nodeID uint
	if _, err := fmt.Sscanf(basisNodeID, "%d", &nodeID); err != nil {
		return fmt.Errorf("invalid basisNodeID %q: %w", basisNodeID, err)
	}
	if nodeID == 0 {
		return nil
	}
	return s.pinRepo.Create(&domain.HistoryPin{
		HistoryNodeID: nodeID,
		PinKind:       pinKind,
		RefType:       refType,
		RefID:         refID,
	})
}
