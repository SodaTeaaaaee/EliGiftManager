package main

import (
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/db"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
)

// ShipmentController exposes shipment-management Wails bindings.
type ShipmentController struct {
	shipmentRepo domain.ShipmentRepository
}

func NewShipmentController() *ShipmentController {
	gdb := db.GetDB()
	shipmentRepo := infra.NewShipmentRepository(gdb)
	return &ShipmentController{
		shipmentRepo: shipmentRepo,
	}
}

// CreateShipment creates a shipment with its lines.
func (c *ShipmentController) CreateShipment(input dto.CreateShipmentInput) (dto.ShipmentDTO, error) {
	now := time.Now().Format(time.RFC3339)

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

	if err := c.shipmentRepo.Create(shipment); err != nil {
		return dto.ShipmentDTO{}, err
	}

	lineDTOs := make([]dto.ShipmentLineDTO, 0, len(input.Lines))
	for _, li := range input.Lines {
		line := &domain.ShipmentLine{
			ShipmentID:          shipment.ID,
			SupplierOrderLineID: li.SupplierOrderLineID,
			FulfillmentLineID:   li.FulfillmentLineID,
			Quantity:            li.Quantity,
			CreatedAt:           now,
		}
		if err := c.shipmentRepo.CreateLine(line); err != nil {
			return dto.ShipmentDTO{}, err
		}
		lineDTOs = append(lineDTOs, domainToShipmentLineDTO(line))
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
