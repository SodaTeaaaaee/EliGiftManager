package main

import (
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/app"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"github.com/SodaTeaaaaee/EliGiftManager/internal/infra"
	"gorm.io/gorm"
)

// BatchRecordAdjustments records a batch of adjustments in a single call (plan 5.3).
// Each entry is applied in its own nested savepoint so one invalid entry does not
// roll back the entries that already succeeded — the caller gets a per-item result.
func (c *AdjustmentController) BatchRecordAdjustments(input dto.BatchRecordAdjustmentsInput) (dto.BatchRecordAdjustmentsResult, error) {
	ctx := appContext

	// Batch calls may span multiple waves; capture baseline snapshots per wave
	// lazily, once per distinct wave encountered.
	baselineSnapshots := make(map[uint]string)

	result := dto.BatchRecordAdjustmentsResult{
		Results: make([]dto.BatchAdjustmentItemResult, 0, len(input.Entries)),
	}

	for i, entry := range input.Entries {
		if _, captured := baselineSnapshots[entry.WaveID]; !captured {
			snap, err := c.captureBaselineSnapshot(entry.WaveID)
			if err != nil {
				result.Results = append(result.Results, dto.BatchAdjustmentItemResult{
					Index:   i,
					Success: false,
					Error:   err.Error(),
				})
				result.FailureCount++
				continue
			}
			baselineSnapshots[entry.WaveID] = snap
		}
		preSnapshot := baselineSnapshots[entry.WaveID]

		var adj *domain.FulfillmentAdjustment
		txErr := c.gdb.Transaction(func(tx *gorm.DB) error {
			repos := infra.NewTxRepos(tx)
			adjustmentRepo := repos.AdjustmentRepo
			fulfillRepo := repos.FulfillRepo
			waveRepo := repos.WaveRepo
			ruleRepo := repos.RuleRepo
			assignmentRepo := repos.AssignmentRepo
			closureDecisionRepo := repos.ClosureDecision
			historyScopeRepo := repos.HistoryScope
			historyNodeRepo := repos.HistoryNode
			historyCheckpointRepo := repos.HistoryCheckpoint
			productRepo := repos.ProductRepo

			adjustmentUC := app.NewAdjustmentUseCase(adjustmentRepo, fulfillRepo, waveRepo)
			snapshotSvc := app.NewWaveSnapshotService(tx, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, fulfillRepo, closureDecisionRepo)
			historySvc := app.NewHistoryRecordingService(historyScopeRepo, historyNodeRepo, historyCheckpointRepo, app.WithSnapshotService(snapshotSvc))
			projHashSvc := app.NewProjectionHashService(fulfillRepo, ruleRepo, adjustmentRepo, assignmentRepo, waveRepo, productRepo, closureDecisionRepo)

			recordedAdj, recordErr := adjustmentUC.RecordAdjustment(ctx, entry)
			if recordErr != nil {
				return recordErr
			}
			adj = recordedAdj

			patchPayload, patchErr := app.BuildAdjustmentPatch("record_adjustment", adj)
			if patchErr != nil {
				return patchErr
			}

			projHash, hashErr := projHashSvc.ComputeHash(ctx, adj.WaveID)
			if hashErr != nil {
				return hashErr
			}
			_, historyErr := historySvc.RecordNode(ctx, app.RecordNodeInput{
				WaveID:                  adj.WaveID,
				CommandKind:             domain.CmdRecordAdjustment,
				CommandSummary:          fmt.Sprintf("batch record adjustment %d (%s) for wave %d", adj.ID, adj.AdjustmentKind, adj.WaveID),
				PatchPayload:            patchPayload,
				InversePatchPayload:     fmt.Sprintf(`{"op":"delete_adjustment","adjustment_id":%d}`, adj.ID),
				BaselineSnapshotPayload: preSnapshot,
				ProjectionHash:          projHash,
			})
			return historyErr
		})

		if txErr != nil {
			result.Results = append(result.Results, dto.BatchAdjustmentItemResult{
				Index:   i,
				Success: false,
				Error:   txErr.Error(),
			})
			result.FailureCount++
			continue
		}

		itemDTO := domainToFulfillmentAdjustmentDTO(*adj)
		result.Results = append(result.Results, dto.BatchAdjustmentItemResult{
			Index:      i,
			Success:    true,
			Adjustment: &itemDTO,
		})
		result.SuccessCount++

		// Subsequent entries targeting the same wave should snapshot against the
		// post-entry state, not the pre-batch state, so undo history stays coherent.
		if entry.WaveID != 0 {
			if refreshedSnap, err := c.captureBaselineSnapshot(entry.WaveID); err == nil {
				baselineSnapshots[entry.WaveID] = refreshedSnap
			}
		}
	}

	return result, nil
}
