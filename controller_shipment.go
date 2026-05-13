package main

import (
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra/persistence"

	"gorm.io/gorm"
)

// ShipmentController exposes shipment-management Wails bindings.
type ShipmentController struct {
	shipmentRepo domain.ShipmentRepository
	supplierRepo domain.SupplierOrderRepository
	fulfillRepo  domain.FulfillmentLineRepository
	db           *gorm.DB
}

func NewShipmentController() *ShipmentController {
	gdb := db.GetDB()
	shipmentRepo := infra.NewShipmentRepository(gdb)
	supplierRepo := infra.NewSupplierOrderRepository(gdb)
	fulfillRepo := infra.NewFulfillmentRepository(gdb)
	return &ShipmentController{
		shipmentRepo: shipmentRepo,
		supplierRepo: supplierRepo,
		fulfillRepo:  fulfillRepo,
		db:           gdb,
	}
}

// CreateShipment creates a shipment with its lines.
func (c *ShipmentController) CreateShipment(input dto.CreateShipmentInput) (dto.ShipmentDTO, error) {
	now := time.Now().Format(time.RFC3339)

	// Validate supplier order existence
	supplierOrder, err := c.supplierRepo.FindByID(input.SupplierOrderID)
	if err != nil {
		return dto.ShipmentDTO{}, fmt.Errorf("supplier order %d not found: %w", input.SupplierOrderID, err)
	}

	shipment := &domain.Shipment{
		SupplierOrderID:      input.SupplierOrderID,
		SupplierPlatform:     input.SupplierPlatform,
		ShipmentNo:           input.ShipmentNo,
		ExternalShipmentNo:   input.ExternalShipmentNo,
		CarrierCode:          input.CarrierCode,
		CarrierName:          input.CarrierName,
		TrackingNo:           input.TrackingNo,
		Status:               input.Status,
		ShippedAt:            input.ShippedAt,
		BasisHistoryNodeID:   input.BasisHistoryNodeID,
		BasisProjectionHash:  input.BasisProjectionHash,
		BasisPayloadSnapshot: input.BasisPayloadSnapshot,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	lineDTOs := make([]dto.ShipmentLineDTO, 0, len(input.Lines))

	// Atomic creation: shipment + all lines in one transaction
	err = c.db.Transaction(func(tx *gorm.DB) error {
		pShip := persistence.ToPersistenceShipment(shipment)
		if err := tx.Create(pShip).Error; err != nil {
			return err
		}
		*shipment = *persistence.FromPersistenceShipment(pShip)

		for _, li := range input.Lines {
			// Validate supplier order line existence
			sol, err := c.supplierRepo.FindLineByID(li.SupplierOrderLineID)
			if err != nil {
				return fmt.Errorf("supplier order line %d not found: %w", li.SupplierOrderLineID, err)
			}
			// Validate fulfillment line existence
			fl, err := c.fulfillRepo.FindByID(li.FulfillmentLineID)
			if err != nil {
				return fmt.Errorf("fulfillment line %d not found: %w", li.FulfillmentLineID, err)
			}
			// Validate supplier order line belongs to this supplier order
			if sol.SupplierOrderID != shipment.SupplierOrderID {
				return fmt.Errorf("supplier order line %d belongs to order %d, not %d", li.SupplierOrderLineID, sol.SupplierOrderID, shipment.SupplierOrderID)
			}
			// Validate supplier order line references the correct fulfillment line
			if sol.FulfillmentLineID != li.FulfillmentLineID {
				return fmt.Errorf("supplier order line %d references fulfillment line %d, not %d", li.SupplierOrderLineID, sol.FulfillmentLineID, li.FulfillmentLineID)
			}
			// Validate cross-wave consistency
			if fl.WaveID != supplierOrder.WaveID {
				return fmt.Errorf("fulfillment line %d belongs to wave %d, not wave %d", li.FulfillmentLineID, fl.WaveID, supplierOrder.WaveID)
			}

			line := &domain.ShipmentLine{
				ShipmentID:          shipment.ID,
				SupplierOrderLineID: li.SupplierOrderLineID,
				FulfillmentLineID:   li.FulfillmentLineID,
				Quantity:            li.Quantity,
				CreatedAt:           now,
			}
			pLine := persistence.ToPersistenceShipmentLine(line)
			if err := tx.Create(pLine).Error; err != nil {
				return err
			}
			*line = *persistence.FromPersistenceShipmentLine(pLine)
			lineDTOs = append(lineDTOs, domainToShipmentLineDTO(line))
		}
		return nil
	})
	if err != nil {
		return dto.ShipmentDTO{}, err
	}

	result := domainToShipmentDTO(shipment)
	result.Lines = lineDTOs
	return result, nil
}

// ListShipmentsByWave lists all shipments for a given wave.
func (c *ShipmentController) ListShipmentsByWave(waveID uint) ([]dto.ShipmentDTO, error) {
	shipments, err := c.shipmentRepo.ListByWave(waveID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ShipmentDTO, len(shipments))
	for i, s := range shipments {
		shipmentDTO := domainToShipmentDTO(&s)
		lines, err := c.shipmentRepo.ListLinesByShipment(s.ID)
		if err != nil {
			return nil, err
		}
		shipmentDTO.Lines = make([]dto.ShipmentLineDTO, len(lines))
		for j, l := range lines {
			shipmentDTO.Lines[j] = domainToShipmentLineDTO(&l)
		}
		result[i] = shipmentDTO
	}
	return result, nil
}

// domainToShipmentDTO converts a domain Shipment to a DTO.
func domainToShipmentDTO(s *domain.Shipment) dto.ShipmentDTO {
	if s == nil {
		return dto.ShipmentDTO{}
	}
	return dto.ShipmentDTO{
		ID:                   s.ID,
		SupplierOrderID:      s.SupplierOrderID,
		SupplierPlatform:     s.SupplierPlatform,
		ShipmentNo:           s.ShipmentNo,
		ExternalShipmentNo:   s.ExternalShipmentNo,
		CarrierCode:          s.CarrierCode,
		CarrierName:          s.CarrierName,
		TrackingNo:           s.TrackingNo,
		Status:               s.Status,
		ShippedAt:            s.ShippedAt,
		BasisHistoryNodeID:   s.BasisHistoryNodeID,
		BasisProjectionHash:  s.BasisProjectionHash,
		BasisPayloadSnapshot: s.BasisPayloadSnapshot,
		ExtraData:            s.ExtraData,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}

// domainToShipmentLineDTO converts a domain ShipmentLine to a DTO.
func domainToShipmentLineDTO(l *domain.ShipmentLine) dto.ShipmentLineDTO {
	if l == nil {
		return dto.ShipmentLineDTO{}
	}
	return dto.ShipmentLineDTO{
		ID:                  l.ID,
		ShipmentID:          l.ShipmentID,
		SupplierOrderLineID: l.SupplierOrderLineID,
		FulfillmentLineID:   l.FulfillmentLineID,
		Quantity:            l.Quantity,
		CreatedAt:           l.CreatedAt,
	}
}
