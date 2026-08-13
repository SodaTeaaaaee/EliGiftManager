package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
	"gorm.io/gorm"
)

type undoRedoUseCase struct {
	scopeRepo     domain.HistoryScopeRepository
	nodeRepo      domain.HistoryNodeRepository
	patchExecutor *PatchExecutor
}

func NewUndoRedoUseCase(
	scopeRepo domain.HistoryScopeRepository,
	nodeRepo domain.HistoryNodeRepository,
	patchExecutor ...*PatchExecutor,
) UndoRedoUseCase {
	uc := &undoRedoUseCase{scopeRepo: scopeRepo, nodeRepo: nodeRepo}
	if len(patchExecutor) > 0 && patchExecutor[0] != nil {
		uc.patchExecutor = patchExecutor[0]
	}
	return uc
}

func (uc *undoRedoUseCase) Undo(ctx context.Context, waveID uint) (string, error) {
	scope, err := uc.scopeRepo.FindByScopeTypeAndKey(ctx, "wave", fmt.Sprintf("%d", waveID))
	if err != nil {
		return "", err
	}
	if scope == nil || scope.CurrentHeadNodeID == 0 {
		return "", fmt.Errorf("no history for wave %d", waveID)
	}

	currentNode, err := uc.nodeRepo.FindByID(ctx, scope.CurrentHeadNodeID)
	if err != nil {
		return "", err
	}
	if currentNode == nil || currentNode.ParentNodeID == 0 {
		return "", fmt.Errorf("nothing to undo")
	}

	if err := uc.runPatchAndHead(
		ctx,
		currentNode.InversePatchPayload,
		func(ctx context.Context, payload string) error {
			return uc.patchExecutor.ApplyInversePatch(ctx, payload)
		},
		func() error {
			if uc.patchExecutor == nil || currentNode.PatchPayload == "" {
				return nil
			}
			return uc.patchExecutor.ApplyPatch(ctx, currentNode.PatchPayload)
		},
		"undo",
		currentNode.CommandSummary,
		scope.ID,
		currentNode.ParentNodeID,
	); err != nil {
		return "", err
	}
	return currentNode.CommandSummary, nil
}

func (uc *undoRedoUseCase) Redo(ctx context.Context, waveID uint) (string, error) {
	scope, err := uc.scopeRepo.FindByScopeTypeAndKey(ctx, "wave", fmt.Sprintf("%d", waveID))
	if err != nil {
		return "", err
	}
	if scope == nil || scope.CurrentHeadNodeID == 0 {
		return "", fmt.Errorf("no history for wave %d", waveID)
	}

	currentNode, err := uc.nodeRepo.FindByID(ctx, scope.CurrentHeadNodeID)
	if err != nil {
		return "", err
	}
	if currentNode == nil || currentNode.PreferredRedoChildID == 0 {
		return "", fmt.Errorf("nothing to redo")
	}

	childNode, err := uc.nodeRepo.FindByID(ctx, currentNode.PreferredRedoChildID)
	if err != nil {
		return "", err
	}
	if childNode == nil {
		return "", fmt.Errorf("nothing to redo")
	}

	if err := uc.runPatchAndHead(
		ctx,
		childNode.PatchPayload,
		func(ctx context.Context, payload string) error {
			return uc.patchExecutor.ApplyPatch(ctx, payload)
		},
		func() error {
			if uc.patchExecutor == nil || childNode.InversePatchPayload == "" {
				return nil
			}
			return uc.patchExecutor.ApplyInversePatch(ctx, childNode.InversePatchPayload)
		},
		"redo",
		childNode.CommandSummary,
		scope.ID,
		currentNode.PreferredRedoChildID,
	); err != nil {
		return "", err
	}
	return childNode.CommandSummary, nil
}

// runPatchAndHead applies the patch (when present) and then updates the history
// head. WaveController binds the patch executor and history repos to one
// gorm.DB.Transaction, so production undo/redo is atomic. When that outer
// transaction is present, apply+UpdateHead run in a nested savepoint: if
// UpdateHead fails, the patch rolls back with it. Callers without an outer
// transaction (tests using a root *gorm.DB) apply then UpdateHead sequentially
// and compensate with the opposite patch if the head update fails, so a failed
// UpdateHead cannot leave the patch applied.
func (uc *undoRedoUseCase) runPatchAndHead(
	ctx context.Context,
	payload string,
	apply func(context.Context, string) error,
	compensate func() error,
	action string,
	commandSummary string,
	scopeID uint,
	newHeadNodeID uint,
) error {
	applyPatch := func() error {
		if uc.patchExecutor == nil || payload == "" {
			return nil
		}
		if err := apply(ctx, payload); err != nil {
			if errors.Is(err, ErrOperationNotUndoable) {
				return fmt.Errorf("cannot %s %q: %w", action, commandSummary, err)
			}
			return fmt.Errorf("%s failed for %q: %w", action, commandSummary, err)
		}
		return nil
	}

	run := func() error {
		if err := applyPatch(); err != nil {
			return err
		}
		if err := uc.scopeRepo.UpdateHead(ctx, scopeID, newHeadNodeID); err != nil {
			if compensate != nil && uc.patchExecutor != nil && payload != "" {
				_ = compensate()
			}
			return err
		}
		return nil
	}

	if uc.patchExecutor != nil && dbIsGormTransaction(uc.patchExecutor.db) {
		return uc.patchExecutor.db.Transaction(func(tx *gorm.DB) error {
			orig := uc.patchExecutor.db
			uc.patchExecutor.db = tx
			defer func() { uc.patchExecutor.db = orig }()
			// Nested savepoint rolls the patch back if UpdateHead fails; do not
			// compensate on top of that rollback.
			if err := applyPatch(); err != nil {
				return err
			}
			return uc.scopeRepo.UpdateHead(ctx, scopeID, newHeadNodeID)
		})
	}
	return run()
}

func dbIsGormTransaction(db *gorm.DB) bool {
	if db == nil || db.Statement == nil || db.Statement.ConnPool == nil {
		return false
	}
	_, ok := db.Statement.ConnPool.(interface {
		Commit() error
		Rollback() error
	})
	return ok
}
