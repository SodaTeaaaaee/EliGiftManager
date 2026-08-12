package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// jsonIDs renders a []uint as a JSON array literal for history patch payloads.
func jsonIDs(ids []uint) string {
	var b strings.Builder
	b.WriteString("[")
	for i, id := range ids {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(strconv.FormatUint(uint64(id), 10))
	}
	b.WriteString("]")
	return b.String()
}

// buildWaveLifecycleUC constructs a WaveLifecycleUseCase whose repositories are all
// bound to the given transaction, mirroring the pattern already used by
// controller_wave.go's CreateWave/AssignDemandToWave (infra.NewTxRepos(tx) + a fresh
// use-case per call so everything commits/rolls back atomically).
func (c *WaveController) buildWaveLifecycleUC(tx *gorm.DB) app.WaveLifecycleUseCase {
	repos := infra.NewTxRepos(tx)
	lifecycleRepo := infra.NewWaveLifecycleRepository(tx)
	return app.NewWaveLifecycleUseCase(repos.WaveRepo, lifecycleRepo, repos.DemandRepo, repos.AssignmentRepo, repos.FulfillRepo, repos.Profile)
}

// recordWaveLifecycleHistory records one history node for a lifecycle mutation
// already committed against tx. Shared by CloseWave/UnassignDemandFromWave/
// BatchAssignDemandToWave so each follows the exact same snapshot+hash+record
// sequence as the other mutating methods in controller_wave.go.
func (c *WaveController) recordWaveLifecycleHistory(ctx context.Context, tx *gorm.DB, waveID uint, preSnapshot string, input app.RecordNodeInput) error {
	repos := infra.NewTxRepos(tx)
	snapshotSvc := app.NewWaveSnapshotService(tx, repos.RuleRepo, repos.AdjustmentRepo, repos.AssignmentRepo, repos.WaveRepo, repos.FulfillRepo, repos.ClosureDecision)
	historySvc := app.NewHistoryRecordingService(repos.HistoryScope, repos.HistoryNode, repos.HistoryCheckpoint, app.WithSnapshotService(snapshotSvc))
	projHashSvc := app.NewProjectionHashService(repos.FulfillRepo, repos.RuleRepo, repos.AdjustmentRepo, repos.AssignmentRepo, repos.WaveRepo, repos.ProductRepo, repos.ClosureDecision)

	projHash, hashErr := projHashSvc.ComputeHash(ctx, waveID)
	if hashErr != nil {
		return hashErr
	}
	input.WaveID = waveID
	input.BaselineSnapshotPayload = preSnapshot
	input.ProjectionHash = projHash
	_, err := historySvc.RecordNode(ctx, input)
	return err
}

// UpdateWave renames/edits notes/levelTags for a wave (plan 3.2 / 5.2) — the missing
// write path; the underlying DTO fields already existed on Wave.
func (c *WaveController) UpdateWave(input dto.UpdateWaveInput) (dto.WaveDTO, error) {
	ctx := appContext
	var result dto.WaveDTO
	err := c.gdb.Transaction(func(tx *gorm.DB) error {
		waveLifecycleUC := c.buildWaveLifecycleUC(tx)
		updated, ucErr := waveLifecycleUC.UpdateWave(ctx, input)
		if ucErr != nil {
			return ucErr
		}
		result = updated
		return nil
	})
	if err != nil {
		return dto.WaveDTO{}, err
	}
	return result, nil
}

// CloseWave explicitly closes a wave (plan 3.2 / 5.2). Force + a note are required
// when residual (unresolved) fulfillment lines still exist.
//
// Deliberately does NOT defer c.persistLifecycle: that helper recomputes
// lifecycle_stage from live repository counts (LifecycleProjectionService) and
// would immediately overwrite the explicit "closed" transition — most visibly for
// a forced close with residual items, which would never naturally derive back to
// "closed". The explicit close is authoritative here.
func (c *WaveController) CloseWave(input dto.CloseWaveInput) (dto.CloseWaveResult, error) {
	ctx := appContext

	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
	if err != nil {
		return dto.CloseWaveResult{}, err
	}

	var result dto.CloseWaveResult
	err = c.gdb.Transaction(func(tx *gorm.DB) error {
		waveLifecycleUC := c.buildWaveLifecycleUC(tx)
		closed, ucErr := waveLifecycleUC.CloseWave(ctx, input)
		if ucErr != nil {
			return ucErr
		}
		result = closed

		summary := fmt.Sprintf("close wave %d", input.WaveID)
		if result.Forced {
			summary = fmt.Sprintf("force-close wave %d (%d unresolved lines): %s", input.WaveID, result.ResidualItemCount, input.Note)
		}
		return c.recordWaveLifecycleHistory(ctx, tx, input.WaveID, preSnapshot, app.RecordNodeInput{
			CommandKind:         "close_wave",
			CommandSummary:      summary,
			PatchPayload:        fmt.Sprintf(`{"op":"close_wave","wave_id":%d,"forced":%t}`, input.WaveID, result.Forced),
			InversePatchPayload: fmt.Sprintf(`{"op":"reopen_wave","wave_id":%d}`, input.WaveID),
		})
	})
	if err != nil {
		return dto.CloseWaveResult{}, err
	}
	return result, nil
}

// UnassignDemandFromWave returns a demand document to the unassigned pool, only
// permitted before allocation has started for it (plan 5.2).
func (c *WaveController) UnassignDemandFromWave(input dto.UnassignDemandInput) error {
	ctx := appContext
	defer c.persistLifecycle(input.WaveID)

	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
	if err != nil {
		return err
	}

	return c.gdb.Transaction(func(tx *gorm.DB) error {
		waveLifecycleUC := c.buildWaveLifecycleUC(tx)
		if ucErr := waveLifecycleUC.UnassignDemandFromWave(ctx, input.WaveID, input.DemandDocumentID); ucErr != nil {
			return ucErr
		}
		return c.recordWaveLifecycleHistory(ctx, tx, input.WaveID, preSnapshot, app.RecordNodeInput{
			CommandKind:         "unassign_demand",
			CommandSummary:      fmt.Sprintf("unassign demand %d from wave %d", input.DemandDocumentID, input.WaveID),
			PatchPayload:        fmt.Sprintf(`{"op":"unassign_demand","wave_id":%d,"demand_document_id":%d}`, input.WaveID, input.DemandDocumentID),
			InversePatchPayload: fmt.Sprintf(`{"op":"assign_demand","wave_id":%d,"demand_document_id":%d}`, input.WaveID, input.DemandDocumentID),
		})
	})
}

// BatchUnassignDemandFromWave returns multiple demand documents to the unassigned
// pool. Unlike the single-item variant, the whole batch is applied in ONE transaction
// and records ONE undo/redo history node — so a batch unassign can be undone with a
// single Ctrl+Z instead of N.
func (c *WaveController) BatchUnassignDemandFromWave(input dto.BatchUnassignDemandInput) (dto.BatchUnassignDemandResult, error) {
	ctx := appContext
	defer c.persistLifecycle(input.WaveID)

	preSnapshot, err := c.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
	if err != nil {
		return dto.BatchUnassignDemandResult{}, err
	}

	var result dto.BatchUnassignDemandResult
	txErr := c.gdb.Transaction(func(tx *gorm.DB) error {
		waveLifecycleUC := c.buildWaveLifecycleUC(tx)
		result, err = waveLifecycleUC.BatchUnassignDemandFromWave(ctx, input.WaveID, input.DocIDs)
		if err != nil {
			return err
		}
		if result.SuccessCount == 0 {
			return fmt.Errorf("batch unassign: no document could be unassigned")
		}
		successIDs := make([]uint, 0, len(result.Results))
		for _, item := range result.Results {
			if item.Success {
				successIDs = append(successIDs, item.DemandDocumentID)
			}
		}
		return c.recordWaveLifecycleHistory(ctx, tx, input.WaveID, preSnapshot, app.RecordNodeInput{
			CommandKind:         "batch_unassign_demand",
			CommandSummary:      fmt.Sprintf("batch unassign %d demand(s) from wave %d", result.SuccessCount, input.WaveID),
			PatchPayload:        fmt.Sprintf(`{"op":"batch_unassign_demand","wave_id":%d,"demand_document_ids":%s}`, input.WaveID, jsonIDs(successIDs)),
			InversePatchPayload: fmt.Sprintf(`{"op":"batch_assign_demand","wave_id":%d,"demand_document_ids":%s}`, input.WaveID, jsonIDs(successIDs)),
		})
	})
	if txErr != nil {
		return dto.BatchUnassignDemandResult{}, txErr
	}
	return result, nil
}

// BatchAssignDemandToWave assigns multiple demand documents to a wave in one call,
// replacing the frontend's serial-loop workaround with per-item partial-success
// results (plan 5.3). Each item is applied in its own transaction so one failure
// does not roll back items that already succeeded; a history node is recorded once
// per successfully-assigned item, matching the granularity of the existing
// single-item AssignDemandToWave so undo/redo can still step through individual
// assignments.
func (c *WaveController) BatchAssignDemandToWave(input dto.BatchAssignDemandInput) (dto.BatchAssignDemandResult, error) {
	ctx := appContext
	defer c.persistLifecycle(input.WaveID)

	if _, err := c.waveUC.GetWave(ctx, input.WaveID); err != nil {
		return dto.BatchAssignDemandResult{}, err
	}

	result := dto.BatchAssignDemandResult{
		Results: make([]dto.BatchAssignDemandItemResult, 0, len(input.DocIDs)),
	}

	for _, docID := range input.DocIDs {
		preSnapshot, snapErr := c.snapshotSvc.CaptureSnapshot(ctx, input.WaveID)
		if snapErr != nil {
			result.Results = append(result.Results, dto.BatchAssignDemandItemResult{DemandDocumentID: docID, Success: false, Error: snapErr.Error()})
			result.FailureCount++
			continue
		}

		txErr := c.gdb.Transaction(func(tx *gorm.DB) error {
			waveLifecycleUC := c.buildWaveLifecycleUC(tx)
			itemResult, ucErr := waveLifecycleUC.BatchAssignDemandToWave(ctx, input.WaveID, []uint{docID})
			if ucErr != nil {
				return ucErr
			}
			if len(itemResult.Results) != 1 || !itemResult.Results[0].Success {
				reason := "assignment failed"
				if len(itemResult.Results) == 1 {
					reason = itemResult.Results[0].Error
				}
				return fmt.Errorf("%s", reason)
			}

			return c.recordWaveLifecycleHistory(ctx, tx, input.WaveID, preSnapshot, app.RecordNodeInput{
				CommandKind:         domain.CmdAssignDemand,
				CommandSummary:      fmt.Sprintf("batch assign demand %d to wave %d", docID, input.WaveID),
				PatchPayload:        fmt.Sprintf(`{"op":"assign_demand","wave_id":%d,"demand_document_id":%d}`, input.WaveID, docID),
				InversePatchPayload: fmt.Sprintf(`{"op":"unassign_demand","wave_id":%d,"demand_document_id":%d}`, input.WaveID, docID),
			})
		})

		if txErr != nil {
			result.Results = append(result.Results, dto.BatchAssignDemandItemResult{DemandDocumentID: docID, Success: false, Error: txErr.Error()})
			result.FailureCount++
			continue
		}
		result.Results = append(result.Results, dto.BatchAssignDemandItemResult{DemandDocumentID: docID, Success: true})
		result.SuccessCount++
	}

	return result, nil
}

// ListWaveFulfillmentRowsFiltered returns a server-side filtered/paginated page of
// the wave's fulfillment grid rows (plan 3.3.2 / 5.4): four state-dim multi-selects,
// reviewRequirement, drift status, and keyword, AND'd across dimensions.
func (c *WaveController) ListWaveFulfillmentRowsFiltered(input dto.WaveFulfillmentFilterInput) (dto.WaveFulfillmentRowsPage, error) {
	ctx := appContext
	filterUC := app.NewWaveFulfillmentFilterUseCase(infra.NewWaveRepository(c.gdb), c.overviewQueryUC)
	return filterUC.ListWaveFulfillmentRowsFiltered(ctx, input)
}

// ListWavesFiltered returns a typed, sorted, paginated page of waves (plan 5.4).
// Named distinctly from controller_wave.go's existing ListWavesPaginated (which
// returns an untyped map[string]any and ignores SortBy) to avoid redeclaring that
// bound method from a different file; the Integrate phase should decide whether to
// retire the old method in favor of this one or keep both during a transition.
func (c *WaveController) ListWavesFiltered(input dto.PaginationInput) (dto.WavesPage, error) {
	ctx := appContext
	filterUC := app.NewWaveFulfillmentFilterUseCase(infra.NewWaveRepository(c.gdb), c.overviewQueryUC)
	return filterUC.ListWavesPaginatedTyped(ctx, input)
}
