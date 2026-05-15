# Task Record: Health-First Stabilization and Phase 9 Advance

**Date**: 2026-05-15
**Task ID**: 2026-05-15-v2-health-first-stabilization-and-phase-9-advance
**Status**: Completed
**Operator**: Claude Opus 4.6 (1M context)

---

## Scope

Execute the full 7-stage execution plan (Stage A through G) from `.agents/codex-task/2026-05-15-v2-health-first-stabilization-and-phase-9-advance.md`, covering:

1. Baseline assessment and phase positioning
2. Frontend stabilization and crash containment
3. Frontend regression guardrails (test infrastructure)
4. Backend/bridge correctness hardening
5. Phase 8 read-side UI advancement
6. Phase 9 history system materialization (patch-first, checkpoint-backed)
7. Frontend undo/redo bridge activation

---

## Work Performed

### Stage A — Baseline Assessment

- Ran `go test ./...` → all pass
- Ran `deno task typecheck` → no errors
- Ran `deno task build` → success
- Identified: 1 functional bug (MembershipAllocationPage missing imports), no error boundary, no test infra, history system entirely placeholder

### Stage B — Frontend Stabilization

| File | Change |
|---|---|
| `frontend/src/app/App.vue` | Added `onErrorCaptured` error boundary with NResult UI; chunk error listener; Wails bridge error exemption |
| `frontend/src/app/router/index.ts` | Added `router.onError()` for chunk load failures → dispatches CustomEvent |
| `frontend/src/pages/membership-allocation/MembershipAllocationPage.vue` | Added missing imports: NCollapse, NCollapseItem, NList, NListItem |
| `frontend/src/pages/wave-workspace/WaveOverviewStep.vue` | Changed waveId from `ref` to `computed` |

### Stage C — Frontend Regression Guardrails

| File | Change |
|---|---|
| `frontend/package.json` | Added vitest, @vue/test-utils, happy-dom |
| `frontend/deno.json` | Added `test` and `test:watch` tasks |
| `frontend/vitest.config.ts` | New — vitest config with happy-dom, @alias, setup file |
| `frontend/src/__tests__/setup.ts` | New — global mocks for wails bridge + naive-ui composition APIs |
| `frontend/src/__tests__/route-smoke.test.ts` | New — 9 smoke tests covering 7 routes + 2 bridge rejection scenarios |

### Stage D — Backend Correctness Hardening

| File | Change |
|---|---|
| `internal/app/history_head_query_test.go` | New — tests for no-history-state safe defaults |
| `internal/app/wave_overview_error_test.go` | New — tests for overview error propagation, no-scope behavior |
| `internal/app/profile_delete_fk_test.go` | New — table-driven tests for all 4 FK blocking paths |

### Stage E — Phase 8 Read-Side Advancement

| File | Change |
|---|---|
| `frontend/src/pages/dashboard/DashboardPage.vue` | Rewritten — stats cards, create wave button, lifecycle stage tags, row click navigation |
| `frontend/src/pages/wave-workspace/WaveOverviewStep.vue` | Rewritten — workflow checkpoint with stage tag, demand/fulfillment/export/shipment stats, channel sync breakdown, closure decisions, basis drift, next-step guidance |

### Stage F — Phase 9 History System Materialization

| File | Change |
|---|---|
| `internal/domain/enums.go` | Added 7 CommandKind constants |
| `internal/domain/ports.go` | Added FindOrCreate to HistoryScopeRepository; FindByID + Delete to FulfillmentAdjustmentRepository; DeleteByWaveAndDocument to WaveDemandAssignmentRepository |
| `internal/infra/history_scope_repo.go` | Implemented FindOrCreate |
| `internal/infra/adjustment_repo.go` | Implemented FindByID + Delete |
| `internal/infra/demand_assignment_repo.go` | Implemented DeleteByWaveAndDocument |
| `internal/app/history_recording_service.go` | New — HistoryRecordingService with RecordNode (find/create scope, create node, advance head, optional checkpoint) |
| `internal/app/patch_executor.go` | New — PatchExecutor with ApplyPatch/ApplyInversePatch supporting 8 operation types |
| `internal/app/undo_redo_usecase.go` | Enhanced — now applies inverse/forward patches before moving head pointer |
| `controller_wave.go` | Added historyRecordingSvc; instrumented AssignDemandToWave, GenerateParticipants, ApplyAllocationRules; wired PatchExecutor into UndoRedoUseCase |
| `controller_allocation_policy.go` | Added historyRecordingSvc; instrumented CreateRule, UpdateRule, DeleteRule with full data payloads |
| `controller_adjustment.go` | Added historyRecordingSvc; instrumented RecordAdjustment with full data payload |

### Stage G — Frontend Undo/Redo Bridge Activation

| File | Change |
|---|---|
| `frontend/src/shared/composables/useUndoRedo.ts` | Rewritten — now calls undoWaveAction/redoWaveAction bridge functions; onSuccess/onError callbacks |
| `frontend/src/pages/wave-workspace/WaveWorkspaceLayout.vue` | Updated — wires onSuccess (toast), onError (warning), onNotReady (info) |

### Test Mock Updates (interface compliance)

| File | Change |
|---|---|
| `internal/app/adjustment_test.go` | Added FindByID + Delete to mockAdjustmentRepo |
| `internal/app/allocation_policy_usecase_test.go` | Added FindByID + Delete to policyAdjRepo |
| `internal/app/use_cases_test.go` | Added DeleteByWaveAndDocument to mockAssignmentRepo |
| `internal/app/history_head_query_test.go` | Added FindOrCreate to mockHistoryScopeRepo |

---

## Verification Results

| Check | Result |
|---|---|
| `go test ./...` | ✅ All pass |
| `deno task typecheck` | ✅ No errors |
| `deno task build` | ✅ Success (chunk size warning only) |
| `deno task test` | ✅ 9/9 pass |
| `wails dev` manual exercise | ❌ Not possible (non-interactive environment) |

---

## Phase Assessment

**Before this task**: Early Phase 8 (read-side infrastructure existed but was inert)

**After this task**: Mid Phase 9

Evidence:
- 5 wave operations have real local undo/redo with DB state restoration
- History nodes are persistently recorded across app restart
- BasisStampService now has real nodes to stamp against (drift detection activated)
- Frontend is connected to real undo/redo backend
- Regression guardrails exist for route-mount stability

---

## Remaining Work

- `generate_participants` / `apply_allocation_rules` undo (batch operations, need checkpoint restore)
- Periodic checkpoint every ~20 nodes (only CheckpointHint path implemented)
- ProjectionHash computation
- Frontend refreshKey after undo/redo for automatic data reload
- History graph UI (not required for mid Phase 9)
- Full scope coverage beyond wave (global scope)

---

## Architecture Decisions Made

1. **Patch-first, checkpoint-backed hybrid** (user-confirmed): daily nodes store patch/inverse patch; periodic checkpoints; heavy ops can hint checkpoint
2. **History recording in controller layer**: avoids breaking existing use case test constructors; minimal invasion
3. **RecordNode failure silently ignored**: history is side-channel, must not block main operations
4. **PatchExecutor uses direct GORM access**: avoids circular dependencies with use case layer
5. **ErrOperationNotUndoable for complex batch ops**: clear error rather than silent failure or fake undo
6. **HistoryNode = user intent only**: ReconcileWave and other derived operations are NOT recorded
