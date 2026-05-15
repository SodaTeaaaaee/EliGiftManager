# Undo/Redo Frontend Wiring (V2)

## What it is

Full undo/redo system for wave-scoped operations. Patch-first, checkpoint-backed hybrid architecture with real DB state restoration.

## Current status: FULLY OPERATIONAL (all 7 wave operations)

All user-intent wave operations support bidirectional undo/redo. Batch operations use checkpoint-based full state restore.

## Supported operations

| Operation | CommandKind | Undo mechanism | Redo mechanism |
|---|---|---|---|
| Create allocation rule | `create_rule` | delete_rule patch | restore_rule patch |
| Update allocation rule | `update_rule` | update_rule (old data) patch | update_rule (new data) patch |
| Delete allocation rule | `delete_rule` | restore_rule patch | delete_rule patch |
| Record adjustment | `record_adjustment` | delete_adjustment patch | record_adjustment patch |
| Assign demand to wave | `assign_demand` | unassign_demand patch | assign_demand patch |
| Generate participants | `generate_participants` | restore_checkpoint (pre-snapshot) | restore_checkpoint (post-snapshot) |
| Apply allocation rules | `apply_allocation_rules` | restore_checkpoint (pre-snapshot) | restore_checkpoint (post-snapshot) |

## Architecture

- **Patch-first**: each HistoryNode stores PatchPayload (forward) + InversePatchPayload (inverse) as JSON
- **Checkpoint-backed**: periodic every ~20 nodes + CheckpointHint for batch ops
- **WaveSnapshotService**: captures/restores full mutable wave state (rules + adjustments + assignments)
- **ProjectionHash**: SHA-256 over sorted rules + fulfillment lines + adjustments — enables basis drift detection
- **HistoryNode = user intent only**: ReconcileWave and other derived operations are NOT recorded
- **Branch-preserving**: undo-then-edit creates new branch; old branch retained in DB; preferred_redo_child updated

## Key components

| Layer | File | Role |
|---|---|---|
| Composable | `frontend/src/shared/composables/useUndoRedo.ts` | Captures Ctrl+Z/Y, calls bridge, returns handleUndo/handleRedo |
| Layout integration | `frontend/src/pages/wave-workspace/WaveWorkspaceLayout.vue` | Wires composable; refreshKey++ on success forces child remount |
| Bridge | `frontend/src/shared/lib/wails/app.ts` | undoWaveAction/redoWaveAction/listRecentHistory (all typed imports) |
| Recording Service | `internal/app/history_recording_service.go` | Creates HistoryNodes; auto-captures checkpoint snapshots |
| Snapshot Service | `internal/app/wave_snapshot_service.go` | CaptureSnapshot / RestoreSnapshot (rules + adjustments + assignments) |
| Patch Executor | `internal/app/patch_executor.go` | Applies forward/inverse patches + restore_checkpoint |
| Projection Hash | `internal/app/projection_hash_service.go` | SHA-256 hash for drift detection |
| Undo/Redo UseCase | `internal/app/undo_redo_usecase.go` | Orchestrates patch apply + head pointer move |
| Controller bindings | `controller_wave.go` | UndoWaveAction/RedoWaveAction/ListRecentHistory |
| CommandKind enums | `internal/domain/enums.go` | CmdCreateRule, CmdUpdateRule, etc. |

## Patch payload format

```json
// Simple operations — direct DB manipulation
{"op": "restore_rule", "rule_id": 5, "wave_id": 1, "data": {...full rule object...}}
{"op": "delete_rule", "rule_id": 5}
{"op": "update_rule", "rule_id": 5, "wave_id": 1, "data": {...field values...}}
{"op": "delete_adjustment", "adjustment_id": 3}
{"op": "record_adjustment", "adjustment_id": 3, "wave_id": 1, "data": {...full adj...}}
{"op": "assign_demand", "wave_id": 1, "demand_document_id": 7}
{"op": "unassign_demand", "wave_id": 1, "demand_document_id": 7}

// Batch operations — full state checkpoint restore
{"op": "restore_checkpoint", "data": "<escaped WaveSnapshot JSON>"}
```

## WaveSnapshot schema (v1)

```json
{
  "wave_id": 1,
  "rules": [...AllocationPolicyRule objects...],
  "adjustments": [...FulfillmentAdjustment objects...],
  "assignments": [...WaveDemandAssignment objects...],
  "schema_version": "1"
}
```

RestoreSnapshot uses Unscoped hard-delete (bypasses GORM soft-delete) to avoid ghost records on re-create.

## Design decisions

- History recording lives in controller layer (not use case) to avoid breaking existing test constructors
- RecordNode failure is silently ignored (`_, _`) — history is a side-channel, must not block main operations
- PatchExecutor uses direct GORM DB access (not through use cases) to avoid circular dependencies
- NewUndoRedoUseCase / NewPatchExecutor / NewHistoryRecordingService accept optional services via variadic params for backward compatibility
- Batch ops capture pre-snapshot AND post-snapshot: undo restores pre, redo restores post
- Frontend refreshKey via provide/inject + router-view :key forces child route remount after undo/redo
- All bridge calls use typed Wails imports (no runtime fallback remaining)

## Gotchas

- Snapshot does NOT include fulfillment lines (derived via ApplyRules — regeneratable)
- Snapshot does NOT include external facts (supplier orders, shipments, channel sync jobs)
- DeleteByWave methods use Unscoped hard-delete — necessary to avoid unique constraint violations on restore
- Periodic checkpoint walks parent chain to count nodes since last checkpoint — O(20) max per write
- ProjectionHash changes after every write → BasisStamp comparison detects drift correctly
