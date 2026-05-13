package app

import (
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// DemandIntakeUseCase handles importing demand documents and their lines.
type DemandIntakeUseCase interface {
	ImportDemand(doc *domain.DemandDocument, lines []*domain.DemandLine) error
}

// WaveUseCase handles wave lifecycle operations.
type WaveUseCase interface {
	CreateWave(wave *domain.Wave) error
	ListWaves() ([]domain.Wave, error)
	GetWave(id uint) (*domain.Wave, error)
}

// AllocationUseCase handles applying allocation policy rules to a wave.
type AllocationUseCase interface {
	ApplyRules(waveID uint) ([]domain.FulfillmentLine, error)
}

// ExportUseCase handles exporting supplier orders from a wave.
type ExportUseCase interface {
	ExportSupplierOrder(waveID uint) (*domain.SupplierOrder, error)
}

// ShipmentUseCase handles shipment creation and lifecycle.
type ShipmentUseCase interface {
	CreateShipment(input dto.CreateShipmentInput) (*domain.Shipment, []domain.ShipmentLine, error)
}
