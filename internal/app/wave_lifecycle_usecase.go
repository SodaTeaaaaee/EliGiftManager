package app

import (
	"context"
	"fmt"
	"time"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

// WaveLifecycleUseCase implements the lifecycle write paths missing from WaveUseCase:
// rename/notes editing, explicit closure, demand unassignment, and batch assignment
// with per-item partial-success semantics (plan 3.2 / 5.2 / 5.3).
type WaveLifecycleUseCase interface {
	UpdateWave(ctx context.Context, input dto.UpdateWaveInput) (dto.WaveDTO, error)
	CloseWave(ctx context.Context, input dto.CloseWaveInput) (dto.CloseWaveResult, error)
	UnassignDemandFromWave(ctx context.Context, waveID uint, demandDocumentID uint) error
	BatchAssignDemandToWave(ctx context.Context, waveID uint, docIDs []uint) (dto.BatchAssignDemandResult, error)
	BatchUnassignDemandFromWave(ctx context.Context, waveID uint, docIDs []uint) (dto.BatchUnassignDemandResult, error)
}

type waveLifecycleUseCase struct {
	waveRepo       domain.WaveRepository
	lifecycleRepo  domain.WaveLifecycleRepository
	demandRepo     domain.DemandDocumentRepository
	assignmentRepo domain.WaveDemandAssignmentRepository
	fulfillRepo    domain.FulfillmentLineRepository
	profileRepo    domain.IntegrationProfileRepository
}

// NewWaveLifecycleUseCase constructs a WaveLifecycleUseCase.
func NewWaveLifecycleUseCase(
	waveRepo domain.WaveRepository,
	lifecycleRepo domain.WaveLifecycleRepository,
	demandRepo domain.DemandDocumentRepository,
	assignmentRepo domain.WaveDemandAssignmentRepository,
	fulfillRepo domain.FulfillmentLineRepository,
	profileRepo domain.IntegrationProfileRepository,
) WaveLifecycleUseCase {
	return &waveLifecycleUseCase{
		waveRepo:       waveRepo,
		lifecycleRepo:  lifecycleRepo,
		demandRepo:     demandRepo,
		assignmentRepo: assignmentRepo,
		fulfillRepo:    fulfillRepo,
		profileRepo:    profileRepo,
	}
}

// UpdateWave persists the operator-editable name/notes/levelTags fields for a wave.
// Name is required non-empty — an explicitly blank name would just recreate the
// "Wave <epoch-ms>" failure mode plan 3.2 sets out to eliminate.
func (uc *waveLifecycleUseCase) UpdateWave(ctx context.Context, input dto.UpdateWaveInput) (dto.WaveDTO, error) {
	if input.Name == "" {
		return dto.WaveDTO{}, fmt.Errorf("wave name is required")
	}
	if _, err := uc.waveRepo.FindByID(ctx, input.WaveID); err != nil {
		return dto.WaveDTO{}, fmt.Errorf("wave %d not found: %w", input.WaveID, err)
	}
	if err := uc.lifecycleRepo.UpdateWaveFields(ctx, input.WaveID, input.Name, input.Notes, input.LevelTags); err != nil {
		return dto.WaveDTO{}, err
	}
	updated, err := uc.waveRepo.FindByID(ctx, input.WaveID)
	if err != nil {
		return dto.WaveDTO{}, err
	}
	return toWaveDTO(updated), nil
}

// residualFulfillmentLineCount counts fulfillment lines that are not yet fully
// resolved: allocation still draft, address missing/invalid, supplier not yet
// submitted, or channel sync still pending/failed. Used to decide whether a close
// requires force + note (plan 3.2).
func (uc *waveLifecycleUseCase) residualFulfillmentLineCount(ctx context.Context, waveID uint) (int, error) {
	lines, err := uc.fulfillRepo.ListByWave(ctx, waveID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, l := range lines {
		if l.AllocationState == string(domain.AllocationStateDraft) ||
			l.AddressState == string(domain.AddressStateMissing) ||
			l.AddressState == string(domain.AddressStateInvalid) ||
			l.SupplierState == string(domain.SupplierStateNotSubmitted) ||
			l.ChannelSyncState == string(domain.ChannelSyncStatePending) ||
			l.ChannelSyncState == string(domain.ChannelSyncStateFailed) {
			count++
		}
	}
	return count, nil
}

// CloseWave transitions a wave to the closed lifecycle stage. If residual
// (unresolved) fulfillment lines exist, the caller must pass Force=true and a
// non-empty Note; that note is appended onto the wave's Notes field as an audit
// trail (plan 3.2: "有残留项时强制关闭需填写说明（审计留痕）").
func (uc *waveLifecycleUseCase) CloseWave(ctx context.Context, input dto.CloseWaveInput) (dto.CloseWaveResult, error) {
	wave, err := uc.waveRepo.FindByID(ctx, input.WaveID)
	if err != nil {
		return dto.CloseWaveResult{}, fmt.Errorf("wave %d not found: %w", input.WaveID, err)
	}

	residualCount, err := uc.residualFulfillmentLineCount(ctx, input.WaveID)
	if err != nil {
		return dto.CloseWaveResult{}, err
	}

	forced := false
	if residualCount > 0 {
		if !input.Force {
			return dto.CloseWaveResult{}, fmt.Errorf(
				"wave %d has %d unresolved fulfillment line(s); close with force=true and a note, or resolve them first",
				input.WaveID, residualCount)
		}
		if input.Note == "" {
			return dto.CloseWaveResult{}, fmt.Errorf("force-closing wave %d requires a note", input.WaveID)
		}
		forced = true
	}

	notes := wave.Notes
	if input.Note != "" {
		if notes != "" {
			notes += "\n"
		}
		notes += fmt.Sprintf("[closure] %s", input.Note)
	}
	if notes != wave.Notes {
		if err := uc.lifecycleRepo.UpdateWaveFields(ctx, input.WaveID, wave.Name, notes, wave.LevelTags); err != nil {
			return dto.CloseWaveResult{}, err
		}
	}

	if err := uc.lifecycleRepo.TransitionLifecycleStage(ctx, input.WaveID, string(domain.LifecycleStageClosed)); err != nil {
		return dto.CloseWaveResult{}, err
	}

	updated, err := uc.waveRepo.FindByID(ctx, input.WaveID)
	if err != nil {
		return dto.CloseWaveResult{}, err
	}
	return dto.CloseWaveResult{
		Wave:              toWaveDTO(updated),
		Forced:            forced,
		ResidualItemCount: residualCount,
	}, nil
}

// UnassignDemandFromWave returns a demand document to the unassigned pool. Only
// permitted before allocation has started for that document — i.e. before any
// FulfillmentLine referencing it has been generated (plan 5.2).
func (uc *waveLifecycleUseCase) UnassignDemandFromWave(ctx context.Context, waveID uint, demandDocumentID uint) error {
	if _, err := uc.waveRepo.FindByID(ctx, waveID); err != nil {
		return fmt.Errorf("wave %d not found: %w", waveID, err)
	}
	if _, err := uc.demandRepo.FindByID(ctx, demandDocumentID); err != nil {
		return fmt.Errorf("demand document %d not found: %w", demandDocumentID, err)
	}

	lines, err := uc.fulfillRepo.ListByWave(ctx, waveID)
	if err != nil {
		return err
	}
	for _, l := range lines {
		if l.DemandDocumentID != nil && *l.DemandDocumentID == demandDocumentID {
			return fmt.Errorf(
				"demand document %d already has fulfillment lines in wave %d; allocation has started, unassign is no longer available",
				demandDocumentID, waveID)
		}
	}

	return uc.assignmentRepo.DeleteByWaveAndDocument(ctx, waveID, demandDocumentID)
}

// BatchUnassignDemandFromWave returns multiple demand documents to the unassigned
// pool with per-item partial-success semantics. A document is only removable while
// allocation has not started for it (same rule as the single-item variant).
func (uc *waveLifecycleUseCase) BatchUnassignDemandFromWave(ctx context.Context, waveID uint, docIDs []uint) (dto.BatchUnassignDemandResult, error) {
	if _, err := uc.waveRepo.FindByID(ctx, waveID); err != nil {
		return dto.BatchUnassignDemandResult{}, fmt.Errorf("wave %d not found: %w", waveID, err)
	}

	lines, err := uc.fulfillRepo.ListByWave(ctx, waveID)
	if err != nil {
		return dto.BatchUnassignDemandResult{}, err
	}
	blockedByLines := make(map[uint]bool, len(lines))
	for _, l := range lines {
		if l.DemandDocumentID != nil {
			blockedByLines[*l.DemandDocumentID] = true
		}
	}

	result := dto.BatchUnassignDemandResult{Results: make([]dto.BatchUnassignDemandItemResult, 0, len(docIDs))}
	for _, docID := range docIDs {
		item := dto.BatchUnassignDemandItemResult{DemandDocumentID: docID}
		if blockedByLines[docID] {
			item.Error = fmt.Sprintf("demand document %d already has fulfillment lines in wave %d; allocation has started, unassign is no longer available", docID, waveID)
			result.Results = append(result.Results, item)
			result.FailureCount++
			continue
		}
		if err := uc.assignmentRepo.DeleteByWaveAndDocument(ctx, waveID, docID); err != nil {
			item.Error = err.Error()
			result.Results = append(result.Results, item)
			result.FailureCount++
			continue
		}
		item.Success = true
		result.Results = append(result.Results, item)
		result.SuccessCount++
	}
	return result, nil
}

// assignOne performs the core single-document assignment: create the wave-demand
// linkage and capture the profile snapshot onto the demand document. Mirrors the
// logic in controller_wave.go's AssignDemandToWave; kept here so it is unit-testable
// without a transaction and reusable by BatchAssignDemandToWave.
func (uc *waveLifecycleUseCase) assignOne(ctx context.Context, waveID, demandDocumentID uint) error {
	doc, err := uc.demandRepo.FindByID(ctx, demandDocumentID)
	if err != nil {
		return fmt.Errorf("demand document %d not found: %w", demandDocumentID, err)
	}

	exists, err := uc.assignmentRepo.ExistsByDocument(ctx, demandDocumentID)
	if err != nil {
		return fmt.Errorf("check existing assignment for demand document %d: %w", demandDocumentID, err)
	}
	if exists {
		return fmt.Errorf("demand document %d already assigned to a wave; cross-wave split is not supported", demandDocumentID)
	}

	// membership_entitlement docs may only enter a wave once every line has been
	// triaged. retail_order is temporarily exempt: 节 3 lands retail
	// auto-acceptance and tightens this gate to all kinds.
	if doc.Kind == string(domain.DemandKindMembershipEntitlement) {
		docLines, lineErr := uc.demandRepo.ListLinesByDocument(ctx, demandDocumentID)
		if lineErr != nil {
			return fmt.Errorf("list demand lines for demand document %d: %w", demandDocumentID, lineErr)
		}
		for _, l := range docLines {
			if l.RoutingDisposition == string(domain.RoutingDispositionPendingIntake) {
				return fmt.Errorf("demand document %d has pending_intake line(s); complete triage before assigning to a wave", demandDocumentID)
			}
		}
	}

	now := time.Now()
	assignment := &domain.WaveDemandAssignment{
		WaveID:           waveID,
		DemandDocumentID: demandDocumentID,
		AcceptedAt:       &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := uc.assignmentRepo.Create(ctx, assignment); err != nil {
		return err
	}

	if doc != nil && doc.IntegrationProfileID != nil {
		if uc.profileRepo == nil {
			return fmt.Errorf("capture profile snapshot for demand document %d: profile repository is not configured", demandDocumentID)
		}
		profile, profErr := uc.profileRepo.FindByID(ctx, *doc.IntegrationProfileID)
		if profErr != nil {
			return fmt.Errorf("capture profile snapshot for demand document %d: %w", demandDocumentID, profErr)
		}
		if profile == nil {
			return fmt.Errorf("capture profile snapshot for demand document %d: profile %d not found", demandDocumentID, *doc.IntegrationProfileID)
		}
		snapshot := CaptureProfileSnapshot(profile)
		if err := uc.demandRepo.UpdateBoundProfileSnapshot(ctx, demandDocumentID, snapshot); err != nil {
			return fmt.Errorf("persist profile snapshot for demand document %d: %w", demandDocumentID, err)
		}
	}
	return nil
}

// BatchAssignDemandToWave assigns each demand document to the wave, collecting a
// per-item result rather than aborting the whole batch on the first failure (plan
// 5.3: "逐条结果返回（部分成功语义），替换前端串行循环"). Callers that need each
// assignment to be transactionally isolated (so a mid-batch failure cannot leave a
// partially-written item, and so undo/redo history can be recorded per item) should
// wrap each call in its own transaction — see controller_wave_lifecycle.go.
func (uc *waveLifecycleUseCase) BatchAssignDemandToWave(ctx context.Context, waveID uint, docIDs []uint) (dto.BatchAssignDemandResult, error) {
	if _, err := uc.waveRepo.FindByID(ctx, waveID); err != nil {
		return dto.BatchAssignDemandResult{}, fmt.Errorf("wave %d not found: %w", waveID, err)
	}

	result := dto.BatchAssignDemandResult{
		Results: make([]dto.BatchAssignDemandItemResult, 0, len(docIDs)),
	}
	for _, docID := range docIDs {
		if err := uc.assignOne(ctx, waveID, docID); err != nil {
			result.Results = append(result.Results, dto.BatchAssignDemandItemResult{
				DemandDocumentID: docID,
				Success:          false,
				Error:            err.Error(),
			})
			result.FailureCount++
			continue
		}
		result.Results = append(result.Results, dto.BatchAssignDemandItemResult{
			DemandDocumentID: docID,
			Success:          true,
		})
		result.SuccessCount++
	}
	return result, nil
}
