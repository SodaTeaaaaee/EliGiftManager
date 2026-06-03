package app

import (
	"context"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// DemandIntakeUseCase handles importing demand documents and their lines.
type DemandIntakeUseCase interface {
	ImportDemand(ctx context.Context, doc *domain.DemandDocument, lines []*domain.DemandLine) error
}

// WaveUseCase handles wave lifecycle operations.
type WaveUseCase interface {
	CreateWave(ctx context.Context, wave *domain.Wave) error
	ListWaves(ctx context.Context) ([]domain.Wave, error)
	ListWavesPaginated(ctx context.Context, offset, limit int) ([]domain.Wave, int64, error)
	GetWave(ctx context.Context, id uint) (*domain.Wave, error)
	GenerateParticipants(ctx context.Context, waveID uint) (int, error)
}

type WaveOverviewQueryUseCase interface {
	BuildBaseOverview(ctx context.Context, waveID uint) (dto.WaveOverviewDTO, error)
	GetWaveOverview(ctx context.Context, waveID uint) (dto.WaveOverviewDTO, error)
	GetWaveWorkspaceSnapshot(ctx context.Context, waveID uint) (dto.WaveWorkspaceSnapshotDTO, error)
	ListWaveFulfillmentRows(ctx context.Context, waveID uint) ([]dto.WaveFulfillmentRowDTO, error)
	ListWaveParticipantRows(ctx context.Context, waveID uint) ([]dto.WaveParticipantRowDTO, error)
	ListDashboardRows(ctx context.Context) ([]dto.WaveDashboardRowDTO, error)
	ListRecentHistory(ctx context.Context, waveID uint, limit int) ([]dto.HistoryNodeDTO, error)
}

// DemandMappingUseCase handles demand-driven mapping: converts accepted, input-ready
// DemandLines into FulfillmentLines for retail_order demand documents.
// Demand lines that require product mapping but cannot be resolved are reported as
// blocked rather than silently entering the execution layer without a ProductID.
type DemandMappingUseCase interface {
	MapDemandToFulfillment(ctx context.Context, waveID uint) (*dto.DemandMappingResult, error)
}

// ExportUseCase handles exporting supplier orders from a wave.
type ExportUseCase interface {
	ExportSupplierOrder(ctx context.Context, waveID uint) ([]*domain.SupplierOrder, error)
}

// ShipmentUseCase handles shipment creation and lifecycle.
type ShipmentUseCase interface {
	CreateShipment(ctx context.Context, input dto.CreateShipmentInput) (*domain.Shipment, []domain.ShipmentLine, error)
}

// ShipmentImportUseCase handles bulk shipment import from factory return data.
type ShipmentImportUseCase interface {
	ImportShipments(ctx context.Context, input dto.ImportShipmentInput) (*dto.ImportShipmentResult, error)
}

// ChannelSyncUseCase handles channel sync job creation.
type ChannelSyncUseCase interface {
	CreateChannelSyncJob(ctx context.Context, input dto.CreateChannelSyncJobInput) (*domain.ChannelSyncJob, []domain.ChannelSyncItem, error)
}

// ChannelClosureUseCase handles profile-driven channel closure orchestration.
type ChannelClosureUseCase interface {
	PlanChannelClosure(ctx context.Context, input dto.PlanChannelClosureInput) (*dto.PlanChannelClosureResult, error)
}

// ExecuteSyncUseCase handles executing a pending ChannelSyncJob.
type ExecuteSyncUseCase interface {
	ExecuteChannelSyncJob(ctx context.Context, jobID uint) (*dto.ExecuteSyncResult, error)
}

// RecordClosureDecisionUseCase handles persisting manual closure decisions.
type RecordClosureDecisionUseCase interface {
	RecordChannelClosureDecision(ctx context.Context, input dto.RecordClosureDecisionInput) ([]dto.ClosureDecisionRecordDTO, error)
}

// RetrySyncUseCase handles retrying failed items in a ChannelSyncJob.
type RetrySyncUseCase interface {
	RetryChannelSyncJob(ctx context.Context, jobID uint) (*dto.ExecuteSyncResult, error)
}

type WaveOverviewProjectionUseCase interface {
	ProjectWaveOverview(ctx context.Context, base dto.WaveOverviewDTO) (dto.WaveOverviewDTO, error)
}

type BasisDriftDetectionUseCase interface {
	DetectWaveBasisDrift(ctx context.Context, waveID uint, currentProjectionHash string) ([]dto.BasisDriftSignalDTO, error)
}

type HistoryHeadQueryUseCase interface {
	GetCurrentProjectionHash(ctx context.Context, waveID uint) (string, error)
	GetCurrentHeadNodeIDAndHash(ctx context.Context, waveID uint) (nodeID uint, projectionHash string, err error)
}

type AdjustmentUseCase interface {
	RecordAdjustment(ctx context.Context, input dto.RecordAdjustmentInput) (*domain.FulfillmentAdjustment, error)
	ListAdjustmentsByWave(ctx context.Context, waveID uint) ([]dto.FulfillmentAdjustmentDTO, error)
}

type UndoRedoUseCase interface {
	Undo(ctx context.Context, waveID uint) (commandSummary string, err error)
	Redo(ctx context.Context, waveID uint) (commandSummary string, err error)
}

// AllocationPolicyUseCase handles policy-driven allocation: reconcile wave (idempotent rebuild + adjustment replay) and rule CRUD.
type AllocationPolicyUseCase interface {
	ReconcileWave(ctx context.Context, waveID uint) (*dto.ReconcileResultDTO, error)
	CreateRule(ctx context.Context, input dto.CreateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error)
	UpdateRule(ctx context.Context, input dto.UpdateAllocationPolicyRuleInput) (*dto.AllocationPolicyRuleDTO, error)
	DeleteRule(ctx context.Context, ruleID uint) error
	ListRulesByWave(ctx context.Context, waveID uint) ([]dto.AllocationPolicyRuleDTO, error)
}

type TemplateManagementUseCase interface {
	CreateDocumentTemplate(ctx context.Context, input dto.CreateDocumentTemplateInput) (*dto.DocumentTemplateDTO, error)
	ListDocumentTemplates(ctx context.Context) ([]dto.DocumentTemplateDTO, error)
	BindTemplateToProfile(ctx context.Context, input dto.BindTemplateToProfileInput) (*dto.ProfileTemplateBindingDTO, error)
	ListBindingsByProfile(ctx context.Context, profileID uint) ([]dto.ProfileTemplateBindingDTO, error)
	GetDefaultTemplateForProfile(ctx context.Context, profileID uint, docType string) (*dto.DocumentTemplateDTO, error)
}

// ProductUseCase handles product master CRUD and wave-scoped product snapshots.
type ProductUseCase interface {
	CreateProductMaster(ctx context.Context, input dto.CreateProductMasterInput) (*dto.ProductMasterDTO, error)
	ListProductMasters(ctx context.Context) ([]dto.ProductMasterDTO, error)
	UpdateProductMaster(ctx context.Context, input dto.UpdateProductMasterInput) (*dto.ProductMasterDTO, error)
	SnapshotProductsForWave(ctx context.Context, input dto.SnapshotProductsInput) ([]dto.ProductDTO, error)
	ListProductsByWave(ctx context.Context, waveID uint) ([]dto.ProductDTO, error)
}

// AddressManagementUseCase handles CustomerAddress CRUD, binding, and derivation.
type AddressManagementUseCase interface {
	CreateAddress(ctx context.Context, input dto.CreateAddressInput) (*dto.CustomerAddressDTO, error)
	UpdateAddress(ctx context.Context, input dto.UpdateAddressInput) (*dto.CustomerAddressDTO, error)
	DeleteAddress(ctx context.Context, id uint) error
	GetAddress(ctx context.Context, id uint) (*dto.CustomerAddressDTO, error)
	ListAddressesByProfile(ctx context.Context, profileID uint) ([]dto.CustomerAddressDTO, error)
	BindAddressToLine(ctx context.Context, input dto.BindAddressInput) (*dto.CustomerAddressDTO, error)
	UnbindAddressFromLine(ctx context.Context, fulfillmentLineID uint) error
}

// ProfileManagementUseCase handles IntegrationProfile CRUD and seeding.
type ProfileManagementUseCase interface {
	CreateProfile(ctx context.Context, input dto.CreateProfileInput) (*dto.IntegrationProfileDTO, error)
	UpdateProfile(ctx context.Context, input dto.UpdateProfileInput) (*dto.IntegrationProfileDTO, error)
	DeleteProfile(ctx context.Context, id uint) error
	GetProfile(ctx context.Context, id uint) (*dto.IntegrationProfileDTO, error)
	ListProfiles(ctx context.Context) ([]dto.IntegrationProfileDTO, error)
	SeedDefaultProfiles(ctx context.Context) ([]dto.IntegrationProfileDTO, error)
}

// ProfileMergeUseCase handles merging a source profile into a target profile,
// migrating all identities, addresses, and references.
type ProfileMergeUseCase interface {
	MergeProfiles(ctx context.Context, input dto.MergeProfilesInput) (*dto.MergeProfilesResult, error)
}

// EntitlementRoutingUseCase handles demand line routing disposition and input state management.
type EntitlementRoutingUseCase interface {
	UpdateDemandLineRouting(ctx context.Context, input dto.UpdateDemandLineRoutingInput) error
	BatchUpdateDemandLineRouting(ctx context.Context, input dto.BatchUpdateDemandLineRoutingInput) (*dto.BatchUpdateDemandLineRoutingResult, error)
	GetWaveRoutingStats(ctx context.Context, waveID uint) (*dto.WaveRoutingStatsDTO, error)
}
