package controller

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// UpdateShipment corrects fields on an existing, non-voided shipment
// (carrier/tracking/shipment numbers, shipped-at). This is a compensating
// write path outside the undo/redo command history (plan 5.2) — no
// HistoryRecordingService.RecordNode call here, by design, so the undo
// boundary stays honestly represented rather than silently faked for an
// operation with no meaningful inverse patch.
func (c *ShipmentController) UpdateShipment(input dto.UpdateShipmentInput) (dto.ShipmentDTO, error) {
	ctx := appContext
	var shipment *domain.Shipment
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		writeRepo := infra.NewShipmentWriterRepository(tx)
		uc := newShipmentLifecycleUC(repos, writeRepo)

		updated, ucErr := uc.UpdateShipment(ctx, input)
		if ucErr != nil {
			return ucErr
		}
		shipment = updated
		return nil
	})
	if err != nil {
		return dto.ShipmentDTO{}, err
	}
	return domainToShipmentDTO(shipment), nil
}

// VoidShipment marks a shipment as voided (terminal, compensating state —
// not a deletion). See dto.VoidShipmentInput for the ExtraData audit-trail
// followup note; this is likewise outside the undo/redo command history.
func (c *ShipmentController) VoidShipment(input dto.VoidShipmentInput) (dto.ShipmentDTO, error) {
	ctx := appContext
	var shipment *domain.Shipment
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		repos := infra.NewTxRepos(tx)
		writeRepo := infra.NewShipmentWriterRepository(tx)
		uc := newShipmentLifecycleUC(repos, writeRepo)

		voided, ucErr := uc.VoidShipment(ctx, input)
		if ucErr != nil {
			return ucErr
		}
		shipment = voided
		return nil
	})
	if err != nil {
		return dto.ShipmentDTO{}, err
	}
	return domainToShipmentDTO(shipment), nil
}
