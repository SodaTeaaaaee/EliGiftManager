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
