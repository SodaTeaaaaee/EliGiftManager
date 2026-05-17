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
	GenerateParticipants(waveID uint) (int, error)
}

type WaveOverviewQueryUseCase interface {
	BuildBaseOverview(waveID uint) (dto.WaveOverviewDTO, error)
	GetWaveOverview(waveID uint) (dto.WaveOverviewDTO, error)
	GetWaveWorkspaceSnapshot(waveID uint) (dto.WaveWorkspaceSnapshotDTO, error)
	ListWaveFulfillmentRows(waveID uint) ([]dto.WaveFulfillmentRowDTO, error)
	ListWaveParticipantRows(waveID uint) ([]dto.WaveParticipantRowDTO, error)
	ListDashboardRows() ([]dto.WaveDashboardRowDTO, error)
}

// DemandMappingUseCase handles demand-driven mapping: converts accepted, input-ready
// DemandLines into FulfillmentLines for retail_order demand documents.
// Demand lines that require product mapping but cannot be resolved are reported as
// blocked rather than silently entering the execution layer without a ProductID.
type DemandMappingUseCase interface {
	MapDemandToFulfillment(waveID uint) (*dto.DemandMappingResult, error)
}

// ExportUseCase handles exporting supplier orders from a wave.
type ExportUseCase interface {
	ExportSupplierOrder(waveID uint) (*domain.SupplierOrder, error)
}

// ShipmentUseCase handles shipment creation and lifecycle.
type ShipmentUseCase interface {
	CreateShipment(input dto.CreateShipmentInput) (*domain.Shipment, []domain.ShipmentLine, error)
}

// ChannelSyncUseCase handles channel sync job creation.
type ChannelSyncUseCase interface {
	CreateChannelSyncJob(input dto.CreateChannelSyncJobInput) (*domain.ChannelSyncJob, []domain.ChannelSyncItem, error)
}

// ChannelClosureUseCase handles profile-driven channel closure orchestration.
type ChannelClosureUseCase interface {
	PlanChannelClosure(input dto.PlanChannelClosureInput) (*dto.PlanChannelClosureResult, error)
}

// ExecuteSyncUseCase handles executing a pending ChannelSyncJob.
type ExecuteSyncUseCase interface {
	ExecuteChannelSyncJob(jobID uint) (*dto.ExecuteSyncResult, error)
}

// RecordClosureDecisionUseCase handles persisting manual closure decisions.
type RecordClosureDecisionUseCase interface {
	RecordChannelClosureDecision(input dto.RecordClosureDecisionInput) ([]dto.ClosureDecisionRecordDTO, error)
}

// RetrySyncUseCase handles retrying failed items in a ChannelSyncJob.
type RetrySyncUseCase interface {
	RetryChannelSyncJob(jobID uint) (*dto.ExecuteSyncResult, error)
}

type WaveOverviewProjectionUseCase interface {
	ProjectWaveOverview(base dto.WaveOverviewDTO) (dto.WaveOverviewDTO, error)
}

type BasisDriftDetectionUseCase interface {
	DetectWaveBasisDrift(waveID uint, currentProjectionHash string) ([]dto.BasisDriftSignalDTO, error)
}

type HistoryHeadQueryUseCase interface {
	GetCurrentProjectionHash(waveID uint) (string, error)
	GetCurrentHeadNodeIDAndHash(waveID uint) (nodeID uint, projectionHash string, err error)
}

type AdjustmentUseCase interface {
	RecordAdjustment(input dto.RecordAdjustmentInput) (*domain.FulfillmentAdjustment, error)
	ListAdjustmentsByWave(waveID uint) ([]dto.FulfillmentAdjustmentDTO, error)
}

type UndoRedoUseCase interface {
	Undo(waveID uint) (commandSummary string, err error)
	Redo(waveID uint) (commandSummary string, err error)
}

// AllocationPolicyUseCase handles policy-driven allocation: reconcile wave (idempotent rebuild + adjustment replay) and rule CRUD.
type AllocationPolicyUseCase interface {
	ReconcileWave(waveID uint) (*dto.ReconcileResultDTO, error)
	CreateRule(input dto.CreateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error)
	UpdateRule(input dto.UpdateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error)
	DeleteRule(ruleID uint) error
	ListRulesByWave(waveID uint) ([]dto.AllocationPolicyRuleDTO, error)
}

type TemplateManagementUseCase interface {
	CreateDocumentTemplate(input dto.CreateDocumentTemplateInput) (*dto.DocumentTemplateDTO, error)
	ListDocumentTemplates() ([]dto.DocumentTemplateDTO, error)
	BindTemplateToProfile(input dto.BindTemplateToProfileInput) (*dto.ProfileTemplateBindingDTO, error)
	ListBindingsByProfile(profileID uint) ([]dto.ProfileTemplateBindingDTO, error)
	GetDefaultTemplateForProfile(profileID uint, docType string) (*dto.DocumentTemplateDTO, error)
}

// ProductUseCase handles product master CRUD and wave-scoped product snapshots.
type ProductUseCase interface {
	CreateProductMaster(input dto.CreateProductMasterInput) (*dto.ProductMasterDTO, error)
	ListProductMasters() ([]dto.ProductMasterDTO, error)
	UpdateProductMaster(input dto.UpdateProductMasterInput) (*dto.ProductMasterDTO, error)
	SnapshotProductsForWave(input dto.SnapshotProductsInput) ([]dto.ProductDTO, error)
	ListProductsByWave(waveID uint) ([]dto.ProductDTO, error)
}

// ProfileManagementUseCase handles IntegrationProfile CRUD and seeding.
type ProfileManagementUseCase interface {
	CreateProfile(input dto.CreateProfileInput) (*dto.IntegrationProfileDTO, error)
	UpdateProfile(input dto.UpdateProfileInput) (*dto.IntegrationProfileDTO, error)
	DeleteProfile(id uint) error
	GetProfile(id uint) (*dto.IntegrationProfileDTO, error)
	ListProfiles() ([]dto.IntegrationProfileDTO, error)
	SeedDefaultProfiles() ([]dto.IntegrationProfileDTO, error)
}
