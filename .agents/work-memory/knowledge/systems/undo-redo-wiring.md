# Undo/Redo Frontend Wiring (V2)

## What it is

Keyboard shortcut (Ctrl+Z / Ctrl+Shift+Z / Ctrl+Y) infrastructure with real backend state restoration for supported wave operations.

## Current status: OPERATIONAL (subset)

The system is now live for a meaningful subset of wave operations. Undo/redo performs actual state restoration via a patch-first, checkpoint-backed hybrid architecture.

## Supported operations (real undo/redo)

| Operation | CommandKind | Undoable |
|---|---|---|
| Create allocation rule | `create_rule` | ✅ |
| Update allocation rule | `update_rule` | ✅ |
| Delete allocation rule | `delete_rule` | ✅ |
| Record adjustment | `record_adjustment` | ✅ |
| Assign demand to wave | `assign_demand` | ✅ |
| Generate participants | `generate_participants` | ❌ returns ErrOperationNotUndoable |
| Apply allocation rules | `apply_allocation_rules` | ❌ returns ErrOperationNotUndoable |

## Architecture

- **Patch-first**: each HistoryNode stores PatchPayload (forward) + InversePatchPayload (inverse) as JSON
- **Checkpoint-backed**: CheckpointHint triggers full snapshot storage (periodic every ~20 nodes planned but not yet implemented)
- **HistoryNode = user intent only**: ReconcileWave and other derived operations are NOT recorded

## Key components

| Layer | File | Role |
|---|---|---|
| Composable | `frontend/src/shared/composables/useUndoRedo.ts` | Captures Ctrl+Z/Y, calls bridge |
| Layout integration | `frontend/src/pages/wave-workspace/WaveWorkspaceLayout.vue` | Wires composable with toast feedback |
| Bridge | `frontend/src/shared/lib/wails/app.ts` | undoWaveAction/redoWaveAction |
| Recording Service | `internal/app/history_recording_service.go` | Creates HistoryNodes on write operations |
| Patch Executor | `internal/app/patch_executor.go` | Applies forward/inverse patches to DB |
| Undo/Redo UseCase | `internal/app/undo_redo_usecase.go` | Orchestrates patch apply + head pointer move |
| Controller bindings | `controller_wave.go` | UndoWaveAction/RedoWaveAction exposed to frontend |
| CommandKind enums | `internal/domain/enums.go` | CmdCreateRule, CmdUpdateRule, etc. |

## Patch payload format

```json
{"op": "restore_rule", "rule_id": 5, "wave_id": 1, "data": {...full rule object...}}
{"op": "delete_rule", "rule_id": 5}
{"op": "update_rule", "rule_id": 5, "wave_id": 1, "data": {...field values...}}
{"op": "delete_adjustment", "adjustment_id": 3}
{"op": "assign_demand", "wave_id": 1, "demand_document_id": 7}
{"op": "unassign_demand", "wave_id": 1, "demand_document_id": 7}
```

## Design decisions

- History recording lives in controller layer (not use case) to avoid breaking existing test constructors
- RecordNode failure is silently ignored (`_, _`) — history is a side-channel, must not block main operations
- PatchExecutor uses direct GORM DB access (not through use cases) to avoid circular dependencies
- NewUndoRedoUseCase accepts optional PatchExecutor via variadic param for backward compatibility with tests
- Frontend shows toast on success/error; text inputs preserve native undo (focus guard)

## Remaining work

- Periodic checkpoint every ~20 nodes (currently only CheckpointHint path)
- ProjectionHash computation (field exists, always empty string)
- generate_participants / apply_allocation_rules undo support (batch operations, complex)
- refreshKey mechanism for forcing page data reload after undo/redo
- BasisStampService now has real nodes to stamp against (drift detection activated)
