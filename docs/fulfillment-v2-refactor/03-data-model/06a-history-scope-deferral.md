# History Scope: Deferred Extensions

## Current Implementation (as of 2026-05-17)

The workspace history system supports exactly one scope:

- **`wave`** — Full undo/redo, checkpoint, GC, and basis pinning for wave workspace operations

All history services (`HistoryRecordingService`, `HistoryGCService`, `UndoRedoUseCase`, `HistoryHeadQueryUseCase`) hardcode `"wave"` as the scope type.

## Deferred Scopes

The following scopes are described in the original design but are NOT yet implemented:

- **`template`** — Document template editing history
- **`product_library`** — Product catalog editing history
- **`profile`** — Integration profile editing history (not in original design, identified during remediation)

## Why Deferral Is Acceptable

1. Wave is the primary execution scope — all lifecycle-critical operations (export, shipment, sync) operate within wave context
2. Profile and template edits are low-frequency configuration changes, not high-frequency workspace operations
3. Profile binding (BoundProfileSnapshot on DemandDocument) already protects active waves from profile drift
4. The history infrastructure is scope-agnostic by design — adding new scopes requires only:
   - A new scope type string constant
   - A snapshot service for the new scope
   - Controller-level recording calls at edit points

## Technical Boundary for Future Implementation

To add a new history scope (e.g., `"template"`):

1. Define a `TemplateSnapshotService` implementing the same capture/restore pattern as `WaveSnapshotService`
2. In the template editing controller, call `HistoryRecordingService.RecordNode` with `scopeType: "template"` and `scopeKey: fmt.Sprintf("%d", templateID)`
3. Add undo/redo controller methods that delegate to `UndoRedoUseCase` with the new scope
4. No changes needed to `HistoryScope`, `HistoryNode`, `HistoryCheckpoint`, or `HistoryPin` domain models — they are already scope-agnostic

## Status

This deferral is intentional and tracked. It does NOT represent a gap in execution safety — only in configuration-level auditability.
