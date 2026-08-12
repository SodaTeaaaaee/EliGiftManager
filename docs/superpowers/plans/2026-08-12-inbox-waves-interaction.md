# 收件箱↔波次双向对称 + 波内导入页面 实施计划（规范节 2）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让收件箱与波次工作区双向可达、能力对称，波次内可自足地拉取/导入需求，任务中心与总览深链真正生效。

**Architecture:** 后端先扩分页过滤维度与批量退单/门禁（Task 1-4），再同步 Wails 绑定与 bridge 契约（Task 5），随后前端分两片落地：全局收件箱改造（Task 6）与波内导入页面（Task 7），最后深链/落点/回归（Task 8-9）。节 3（需求类型）另有一份计划，本计划的分派门禁对零售文档暂按 kind 豁免，待节 3 落地零售自动裁决后收紧。

**Tech Stack:** Go + GORM + SQLite（后端）；Vue 3 + TypeScript + Vite 8 + vitest 4 + Naive UI + vue-i18n（前端）；Deno 工具链。

**Spec:** `docs/superpowers/specs/2026-08-12-inbox-waves-demand-types-vite-design.md` §2/§4（节 2）。

## Global Constraints

- 所有 Wails 运行时调用必须走 `frontend/src/shared/api/bridge.ts`；页面禁止直引 `wailsjs`（guardrail rule3）。
- 前端命令只用 `deno task ...`；禁止 npm/yarn/pnpm。
- 用户可见文案必须走 vue-i18n，zh-CN 与 en-US 叶子键集合保持一致（每任务新增键两 locale 同落）。
- 枚举值渲染必须走 glossary/StatusBadge，禁止裸枚举（guardrail rule2）。
- Go 代码 `gofmt`；每个后端任务跑 `go build ./... && go test -count=1 ./...` 全绿。
- 每个前端任务跑 `cd frontend && deno task typecheck && deno task test && deno task build && deno task lint:guardrails` 全绿。
- 提交风格：`feat:` / `fix:` / `refactor:` + 中文描述，与仓库历史一致；每个任务一个 commit。
- 不修改 `docs/`、不动 `frontend-legacy/`。
- 工作目录：仓库根 `E:\Projects\Code\EliGiftManager`（Git Bash，路径用正斜杠）。

---

### Task 1: 后端收件箱过滤维度扩展（routingDispositions / demandKinds 多值 + pendingIntakeCount）

**Files:**
- Modify: `internal/app/dto/demand.go`（`DemandInboxFilterInput`，:90-102）
- Modify: `internal/app/dto/workspace.go`（`DemandInboxRowDTO`，:84-104）
- Modify: `internal/domain/list_pagination_ports.go`（`DemandInboxPageQuery`，:26-32）
- Modify: `internal/app/list_pagination_usecase.go`（`ListDemandInboxRowsPage` 透传 + `AssembleDemandInboxRows` 计数，:99-129, :159-217）
- Modify: `internal/infra/list_pagination_repo.go`（`ListDemandDocumentsPage` SQL，:191-230）
- Test: `internal/infra/list_pagination_repo_test.go`（追加用例；harness = 文件内既有 `setupListPaginationTestDB`）
- Test: `internal/app/list_pagination_assemble_test.go`（新建，纯函数测试）

**Interfaces:**
- Consumes: `dto.DemandInboxFilterInput`（现含 Assignment/DemandKind/IntegrationProfileID/WaveID/SortBy/SortDir/Limit/Offset）
- Produces:
  - `DemandInboxFilterInput.RoutingDispositions []string`（行级路由处置过滤，多值）
  - `DemandInboxFilterInput.DemandKinds []string`（需求类型过滤，多值；非空时优先于旧单值 `DemandKind`）
  - `DemandInboxRowDTO.PendingIntakeCount int`（pending_intake 行计数）
  - `domain.DemandInboxPageQuery` 同名字段（供 Task 5 桥接层对齐）

- [ ] **Step 1: 写失败测试（repo 层）**

在 `internal/infra/list_pagination_repo_test.go` 末尾追加（复用文件内既有 `setupListPaginationTestDB` helper，`persistence.DemandDocument`/`persistence.DemandLine` 已 AutoMigrate）：

```go
func TestListPaginationRepository_DemandDocumentsRoutingDispositionFilter(t *testing.T) {
	db := setupListPaginationTestDB(t)
	repo := NewListPaginationRepository(db)
	ctx := context.Background()

	docA := persistence.DemandDocument{Kind: "membership_entitlement", SourceDocumentNo: "A"}
	docB := persistence.DemandDocument{Kind: "retail_order", SourceDocumentNo: "B"}
	if err := db.Create(&docA).Error; err != nil { t.Fatal(err) }
	if err := db.Create(&docB).Error; err != nil { t.Fatal(err) }

	lines := []persistence.DemandLine{
		{DemandDocumentID: docA.ID, RoutingDisposition: "pending_intake"},
		{DemandDocumentID: docA.ID, RoutingDisposition: "accepted"},
		{DemandDocumentID: docB.ID, RoutingDisposition: "accepted"},
	}
	for i := range lines {
		if err := db.Create(&lines[i]).Error; err != nil { t.Fatal(err) }
	}

	docs, total, err := repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{
		RoutingDispositions: []string{"pending_intake"},
	})
	if err != nil { t.Fatal(err) }
	if total != 1 || len(docs) != 1 || docs[0].ID != docA.ID {
		t.Fatalf("want only docA (has pending_intake line), got total=%d docs=%v", total, docs)
	}

	docs, total, err = repo.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{
		DemandKinds: []string{"retail_order"},
	})
	if err != nil { t.Fatal(err) }
	if total != 1 || len(docs) != 1 || docs[0].ID != docB.ID {
		t.Fatalf("want only docB (retail_order), got total=%d docs=%v", total, docs)
	}
}
```

- [ ] **Step 2: 写失败测试（Assemble 计数）**

新建 `internal/app/list_pagination_assemble_test.go`：

```go
package app

import (
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestAssembleDemandInboxRowsCountsPendingIntake(t *testing.T) {
	docs := []domain.DemandDocument{{ID: 1, Kind: "membership_entitlement"}}
	lines := []domain.DemandLine{
		{ID: 1, DemandDocumentID: 1, RoutingDisposition: "pending_intake"},
		{ID: 2, DemandDocumentID: 1, RoutingDisposition: "accepted", RecipientInputState: "ready"},
		{ID: 3, DemandDocumentID: 1, RoutingDisposition: "deferred"},
		{ID: 4, DemandDocumentID: 1, RoutingDisposition: "excluded_manual"},
	}
	rows := AssembleDemandInboxRows(docs, nil, lines, nil, nil)
	if len(rows) != 1 { t.Fatalf("want 1 row, got %d", len(rows)) }
	row := rows[0]
	if row.PendingIntakeCount != 1 || row.TotalLineCount != 4 ||
		row.AcceptedCount != 1 || row.ReadyAcceptedCount != 1 ||
		row.DeferredCount != 1 || row.ExcludedCount != 1 {
		t.Fatalf("unexpected counts: %+v", row)
	}
}
```

- [ ] **Step 3: 跑测试确认失败**

Run: `go test -count=1 ./internal/infra/ ./internal/app/ -run 'TestListPaginationRepository_DemandDocumentsRoutingDispositionFilter|TestAssembleDemandInboxRowsCountsPendingIntake'`
Expected: 编译失败（`RoutingDispositions` / `DemandKinds` / `PendingIntakeCount` 未定义）。

- [ ] **Step 4: 实现 DTO / domain query**

`internal/app/dto/demand.go` 的 `DemandInboxFilterInput` 在 `WaveID` 字段后追加：

```go
	// RoutingDispositions narrows to documents having at least one demand line whose
	// routing_disposition is one of the given values (multi-value; AND'd with other
	// filter dimensions). Backs the "待分诊" task-center deep link and the inbox
	// triage filter.
	RoutingDispositions []string `json:"routingDispositions"`
	// DemandKinds narrows to the given kinds (multi-value). Takes precedence over the
	// legacy single-value DemandKind field when non-empty; DemandKind remains accepted
	// for backward compatibility with existing callers.
	DemandKinds []string `json:"demandKinds"`
```

`internal/app/dto/workspace.go` 的 `DemandInboxRowDTO` 在 `ExcludedCount` 后追加：

```go
	PendingIntakeCount       int       `json:"pendingIntakeCount"`
```

`internal/domain/list_pagination_ports.go` 的 `DemandInboxPageQuery` 在 `WaveID` 后追加：

```go
	RoutingDispositions []string
	DemandKinds         []string
```

- [ ] **Step 5: 实现 usecase 透传与计数**

`internal/app/list_pagination_usecase.go` 的 `ListDemandInboxRowsPage`（:101-104）改为：

```go
	docs, total, err := uc.demand.ListDemandDocumentsPage(ctx, domain.DemandInboxPageQuery{
		ListPageQuery: domain.ListPageQuery{SortBy: input.SortBy, SortDir: input.SortDir, Limit: input.Limit, Offset: input.Offset},
		Assignment:    input.Assignment, DemandKind: input.DemandKind, DemandKinds: input.DemandKinds,
		RoutingDispositions: input.RoutingDispositions, IntegrationProfileID: input.IntegrationProfileID, WaveID: input.WaveID,
	})
```

`AssembleDemandInboxRows` 的 switch（:199-212）在 `case "deferred":` 前插入：

```go
			case "pending_intake":
				row.PendingIntakeCount++
```

- [ ] **Step 6: 实现 repo SQL**

`internal/infra/list_pagination_repo.go` 的 `ListDemandDocumentsPage`（:193-195）替换为：

```go
	if len(q.DemandKinds) > 0 {
		query = query.Where("demand_documents.kind IN ?", q.DemandKinds)
	} else if q.DemandKind != "" {
		query = query.Where("demand_documents.kind = ?", q.DemandKind)
	}
	if len(q.RoutingDispositions) > 0 {
		query = query.Where(`EXISTS (SELECT 1 FROM demand_lines dl
			WHERE dl.demand_document_id = demand_documents.id AND dl.deleted_at IS NULL
			AND dl.routing_disposition IN ?)`, q.RoutingDispositions)
	}
```

（表名 `demand_lines` 与仓库内既有 `wave_demand_assignments` 裸 SQL 同风格；若实现时发现实际表名不同，以 `persistence.DemandLine` 的 TableName 为准修正。）

- [ ] **Step 7: 跑测试确认通过**

Run: `gofmt -w internal/app/dto/demand.go internal/app/dto/workspace.go internal/domain/list_pagination_ports.go internal/app/list_pagination_usecase.go internal/infra/list_pagination_repo.go && go test -count=1 ./...`
Expected: 全绿。

- [ ] **Step 8: Commit**

```bash
git add internal/app/dto/demand.go internal/app/dto/workspace.go internal/domain/list_pagination_ports.go internal/app/list_pagination_usecase.go internal/app/list_pagination_assemble_test.go internal/infra/list_pagination_repo.go internal/infra/list_pagination_repo_test.go
git commit -m "feat(inbox): 收件箱后端补 routingDispositions/demandKinds 多值过滤与 pendingIntakeCount 计数"
```

---

### Task 2: 后端批量退单 BatchUnassignDemandFromWave（单历史节点）

**Files:**
- Modify: `internal/app/dto/wave_lifecycle.go`（新 DTO）
- Modify: `internal/app/wave_lifecycle_usecase.go`（接口 + 实现，:15-20, :149-173 附近）
- Modify: `controller_wave_lifecycle.go`（端点，:107-130 单条退单之后）
- Test: `internal/app/wave_lifecycle_usecase_test.go`（追加；沿用文件内既有 harness）

**Interfaces:**
- Consumes: `domain.WaveLifecycleRepository`、`domain.FulfillmentLineRepository`（既有多余量检查逻辑）、`domain.WaveDemandAssignmentRepository.DeleteByWaveAndDocument`
- Produces（供 Task 5 绑定）:
  - `dto.BatchUnassignDemandInput{WaveID uint; DocIDs []uint}`
  - `dto.BatchUnassignDemandItemResult{DemandDocumentID uint; Success bool; Error string}`
  - `dto.BatchUnassignDemandResult{Results []BatchUnassignDemandItemResult; SuccessCount int; FailureCount int}`
  - `WaveLifecycleUseCase.BatchUnassignDemandFromWave(ctx, waveID uint, docIDs []uint) (dto.BatchUnassignDemandResult, error)`
  - `WaveController.BatchUnassignDemandFromWave(input dto.BatchUnassignDemandInput) (dto.BatchUnassignDemandResult, error)`

- [ ] **Step 1: 写失败测试**

`internal/app/wave_lifecycle_usecase_test.go` 末尾追加（复用文件内既有 fake/DB harness——文件 :182、:227 已有批量分配/退单测试，直接抄其 setup）：

```go
func TestBatchUnassignDemandFromWaveReturnsPerItemResults(t *testing.T) {
	// setup: 复制本文件 TestBatchAssignDemandToWaveReturnsPerItemResultsIncludingAFailure 的
	// harness，预置 wave 1 + doc 10（无履约行）、doc 11（无履约行）、doc 12（已有履约行）。
	result, err := uc.BatchUnassignDemandFromWave(ctx, 1, []uint{10, 11, 12})
	if err != nil { t.Fatal(err) }
	if result.SuccessCount != 2 || result.FailureCount != 1 { t.Fatalf("want 2/1, got %+v", result) }
	// doc 12 的失败原因必须包含 "allocation has started"
	if !result.Results[2].Success || !strings.Contains(result.Results[2].Error, "allocation has started") {
		t.Fatalf("doc 12 should fail with allocation-started reason, got %+v", result.Results[2])
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -count=1 ./internal/app/ -run TestBatchUnassignDemandFromWaveReturnsPerItemResults`
Expected: 编译失败（`BatchUnassignDemandFromWave` 未定义）。

- [ ] **Step 3: 实现 DTO**

`internal/app/dto/wave_lifecycle.go` 末尾追加：

```go
// BatchUnassignDemandInput carries a batch of demand documents to unassign from one wave.
type BatchUnassignDemandInput struct {
	WaveID uint   `json:"waveId"`
	DocIDs []uint `json:"docIds"`
}

// BatchUnassignDemandItemResult reports the per-item outcome of a batch unassignment.
type BatchUnassignDemandItemResult struct {
	DemandDocumentID uint   `json:"demandDocumentId"`
	Success          bool   `json:"success"`
	Error            string `json:"error,omitempty"`
}

// BatchUnassignDemandResult aggregates per-item results for BatchUnassignDemandFromWave,
// mirroring BatchAssignDemandResult's partial-success contract.
type BatchUnassignDemandResult struct {
	Results      []BatchUnassignDemandItemResult `json:"results"`
	SuccessCount int                             `json:"successCount"`
	FailureCount int                             `json:"failureCount"`
}
```

- [ ] **Step 4: 实现 usecase**

`internal/app/wave_lifecycle_usecase.go`：
接口（:18 之后）加：

```go
	BatchUnassignDemandFromWave(ctx context.Context, waveID uint, docIDs []uint) (dto.BatchUnassignDemandResult, error)
```

实现（`UnassignDemandFromWave` 之后追加；履约行检查只跑一次，循环内复用）：

```go
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
```

- [ ] **Step 5: 实现 controller 端点（单个历史节点）**

`controller_wave_lifecycle.go` 在 `UnassignDemandFromWave`（:109-130）之后追加：

```go
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
		return c.recordWaveLifecycleHistory(ctx, tx, input.WaveID, preSnapshot, app.RecordNodeInput{
			CommandKind:         "batch_unassign_demand",
			CommandSummary:      fmt.Sprintf("batch unassign %d demand(s) from wave %d", result.SuccessCount, input.WaveID),
			PatchPayload:        fmt.Sprintf(`{"op":"batch_unassign_demand","wave_id":%d,"demand_document_ids":%s}`, input.WaveID, jsonIDs(input.DocIDs)),
			InversePatchPayload: fmt.Sprintf(`{"op":"batch_assign_demand","wave_id":%d,"demand_document_ids":%s}`, input.WaveID, jsonIDs(input.DocIDs)),
		})
	})
	if txErr != nil {
		return dto.BatchUnassignDemandResult{}, txErr
	}
	return result, nil
}
```

同一文件顶部区域（import 块附近）加辅助函数：

```go
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
```

（补 import：`strconv`、`strings`。）

- [ ] **Step 6: 跑测试确认通过**

Run: `gofmt -w internal/app/dto/wave_lifecycle.go internal/app/wave_lifecycle_usecase.go controller_wave_lifecycle.go && go build ./... && go test -count=1 ./...`
Expected: 全绿（新增用例 + 既有用例）。

- [ ] **Step 7: Commit**

```bash
git add internal/app/dto/wave_lifecycle.go internal/app/wave_lifecycle_usecase.go internal/app/wave_lifecycle_usecase_test.go controller_wave_lifecycle.go
git commit -m "feat(wave): 批量退单端点——单事务单历史节点，一次撤销即可回滚整批"
```

---

### Task 3: 分派门禁（重复分派拦截 + kind 感知的 accepted 校验）

**Files:**
- Modify: `internal/app/wave_lifecycle_usecase.go`（`assignOne`，:179-214）
- Modify: `internal/domain/ports.go` 或对应 repo 接口定义处（`WaveDemandAssignmentRepository` 加 `ExistsByDocument`；`DemandDocumentRepository` 加 `ListLinesByDocument`——先 `Grep 'type (WaveDemandAssignmentRepository|DemandDocumentRepository) interface' internal/domain/` 定位）
- Modify: 对应 infra 实现文件（`Grep -l 'func (r .*WaveDemandAssignment)' internal/infra/` 定位）
- Test: `internal/app/wave_lifecycle_usecase_test.go`（追加）

**Interfaces:**
- Consumes: `use_cases.go` 旧 `AssignDemandToWave` 的重复分派检查逻辑（`:1070` 测试对应实现，照搬语义）
- Produces:
  - `WaveDemandAssignmentRepository.ExistsByDocument(ctx, demandDocumentID uint) (bool, error)`
  - `DemandDocumentRepository.ListLinesByDocument(ctx, demandDocumentID uint) ([]domain.DemandLine, error)`
  - `assignOne` 新规则：文档已挂任意波 → 拒绝（#34 禁跨波拆分）；文档为 membership_entitlement 且存在 `pending_intake` 行 → 拒绝（retail_order 暂豁免，节 3 落地零售自动裁决后收紧为全 kind）

- [ ] **Step 1: 写失败测试**

`internal/app/wave_lifecycle_usecase_test.go` 追加（harness 同上）：

```go
func TestAssignOneRejectsAlreadyAssignedDocument(t *testing.T) {
	// setup: wave 1 + wave 2 + doc 10；先把 doc 10 分派给 wave 1。
	// act: BatchAssignDemandToWave(ctx, 2, []uint{10})
	// expect: 该 item Success=false，Error 含 "already assigned"
}

func TestAssignOneRejectsMembershipDocWithPendingIntakeLines(t *testing.T) {
	// setup: wave 1 + doc 20（kind=membership_entitlement，带一条 pending_intake 行）
	// act: BatchAssignDemandToWave(ctx, 1, []uint{20})
	// expect: 该 item Success=false，Error 含 "pending_intake"
}
```

（实现时按文件内既有 fake 形态把注释替换为实际 setup 代码；断言写成具体字段比较。）

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -count=1 ./internal/app/ -run 'TestAssignOneRejects'`
Expected: 两个用例失败（当前 assignOne 不拦截）。

- [ ] **Step 3: 实现 repo 接口方法**

`domain` 侧接口加两方法（定位文件后追加到对应 interface）：

```go
	// ExistsByDocument reports whether the demand document is linked to any wave.
	ExistsByDocument(ctx context.Context, demandDocumentID uint) (bool, error)
```

```go
	// ListLinesByDocument returns a demand document's lines in stable id order.
	ListLinesByDocument(ctx context.Context, demandDocumentID uint) ([]DemandLine, error)
```

infra 实现（各自 repo 文件内，照既有 GORM 风格）：

```go
func (r *xxxRepository) ExistsByDocument(ctx context.Context, demandDocumentID uint) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&persistence.WaveDemandAssignment{}).
		Where("demand_document_id = ?", demandDocumentID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
```

```go
func (r *xxxRepository) ListLinesByDocument(ctx context.Context, demandDocumentID uint) ([]domain.DemandLine, error) {
	var rows []persistence.DemandLine
	if err := r.db.WithContext(ctx).Where("demand_document_id = ?", demandDocumentID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.DemandLine, len(rows))
	for i := range rows {
		items[i] = *persistence.DemandLineToDomain(&rows[i])
	}
	return items, nil
}
```

（`xxxRepository` 换成实际接收者类型；软删除列名 `deleted_at` 由 GORM 自动处理。）

- [ ] **Step 4: 实现 assignOne 门禁**

`internal/app/wave_lifecycle_usecase.go` 的 `assignOne`（:179-185 之后，创建 assignment 之前）插入：

```go
	exists, err := uc.assignmentRepo.ExistsByDocument(ctx, demandDocumentID)
	if err != nil {
		return fmt.Errorf("check existing assignment for demand document %d: %w", demandDocumentID, err)
	}
	if exists {
		return fmt.Errorf("demand document %d already assigned to a wave; cross-wave split is not supported", demandDocumentID)
	}

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
```

（枚举常量名以 `internal/domain/enums.go` 实际定义为准；若为字符串常量或直接字面量，则改用对应形式。）

- [ ] **Step 5: 跑测试确认通过**

Run: `gofmt -w . && go build ./... && go test -count=1 ./...`
Expected: 全绿。

- [ ] **Step 6: Commit**

```bash
git add internal/app/wave_lifecycle_usecase.go internal/app/wave_lifecycle_usecase_test.go internal/domain/ internal/infra/
git commit -m "feat(wave): 分派门禁——禁跨波重复分派、会员文档未分诊禁入波（零售按 kind 暂豁免，节 3 收紧）"
```

---

### Task 4: ListWavesFiltered 过滤字段（阶段/类型/名称）

**Files:**
- Modify: `internal/app/dto/wave_fulfillment_filter.go`（新 `WaveListFilterInput`）
- Modify: `internal/app/wave_fulfillment_filter_usecase.go`（`ListWavesPaginatedTyped` 签名与内存过滤，:149-187）
- Modify: `controller_wave_lifecycle.go`（`ListWavesFiltered`，:207-211）
- Test: 追加到 `wave_fulfillment_filter_usecase` 对应既有测试文件（`Grep -l 'ListWavesPaginatedTyped' internal/app/*_test.go` 定位）

**Interfaces:**
- Consumes: `dto.PaginationInput`、`domain.WaveRepository.List`
- Produces: `dto.WaveListFilterInput{Page int; PageSize int; SortBy string; SortDesc bool; LifecycleStage string; NameKeyword string; WaveType string}`（嵌入 `PaginationInput`）；`WaveController.ListWavesFiltered(input dto.WaveListFilterInput) (dto.WavesPage, error)`

- [ ] **Step 1: 写失败测试**

定位既有测试文件后追加（沿用其 fake repo harness）：

```go
func TestListWavesPaginatedTypedFiltersByStageTypeAndKeyword(t *testing.T) {
	// seed 三个 wave：A(active, membership, name="舰长回馈"), B(closed, retail, "6月零售"), C(active, retail, "无关")
	// act: ListWavesPaginatedTyped(ctx, dto.WaveListFilterInput{
	//   PaginationInput: dto.PaginationInput{Page:1, PageSize:20},
	//   LifecycleStage: "active", WaveType: "retail", NameKeyword: "6月"})
	// expect: 只返回 B？——不：stage=active 排除 B；期望返回 C（retail+active，名称含"6月"的只有 B 被排除后为空集则期望空）。
	// 分两段断言：(1) {LifecycleStage:"active",WaveType:"retail"} -> 只有 C
	//             (2) {NameKeyword:"舰长"} -> 只有 A
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -count=1 ./internal/app/ -run TestListWavesPaginatedTypedFiltersByStageTypeAndKeyword`
Expected: 编译失败（`WaveListFilterInput` 未定义）。

- [ ] **Step 3: 实现 DTO 与过滤**

`internal/app/dto/wave_fulfillment_filter.go` 追加：

```go
// WaveListFilterInput extends pagination with wave-list filters for the assign-to-wave
// picker and the waves list page (plan §4.5 of the round-2 spec).
type WaveListFilterInput struct {
	PaginationInput
	LifecycleStage string `json:"lifecycleStage"`
	NameKeyword    string `json:"nameKeyword"`
	WaveType       string `json:"waveType"`
}
```

`internal/app/wave_fulfillment_filter_usecase.go` 的 `ListWavesPaginatedTyped` 签名改为 `input dto.WaveListFilterInput`，在 `waves, err := uc.waveRepo.List(ctx)` 之后插入：

```go
	waves = filterWaves(waves, input)
```

文件内追加：

```go
// filterWaves applies the picker/list filters in memory. The wave list is small by
// design (single operator, tens of waves); server-side SQL filtering is not worth
// the extra repository surface (spec §2 follow-ups).
func filterWaves(waves []domain.Wave, input dto.WaveListFilterInput) []domain.Wave {
	out := waves[:0]
	keyword := strings.ToLower(strings.TrimSpace(input.NameKeyword))
	for _, w := range waves {
		if input.LifecycleStage != "" && w.LifecycleStage != input.LifecycleStage {
			continue
		}
		if input.WaveType != "" && w.WaveType != input.WaveType {
			continue
		}
		if keyword != "" &&
			!strings.Contains(strings.ToLower(w.Name), keyword) &&
			!strings.Contains(strings.ToLower(w.WaveNo), keyword) {
			continue
		}
		out = append(out, w)
	}
	return out
}
```

（字段名 `LifecycleStage`/`WaveType`/`WaveNo` 以 `toWaveDTO` 实际读取的 domain.Wave 字段为准；缺 `strings` import 则补。）

`controller_wave_lifecycle.go` 的 `ListWavesFiltered` 签名同步改为 `input dto.WaveListFilterInput`（调用处不变，仅类型）。

- [ ] **Step 4: 跑测试确认通过**

Run: `gofmt -w . && go build ./... && go test -count=1 ./...`
Expected: 全绿。

- [ ] **Step 5: Commit**

```bash
git add internal/app/dto/wave_fulfillment_filter.go internal/app/wave_fulfillment_filter_usecase.go controller_wave_lifecycle.go
git commit -m "feat(wave): 波次列表/选波 picker 支持阶段、类型、名称过滤（内存过滤，波次量小）"
```

---

### Task 5: Wails 绑定 + bridge + entities 契约同步

**Files:**
- Modify: `frontend/wailsjs/go/main/WaveController.d.ts` / `WaveController.js`（+`BatchUnassignDemandFromWave`；`ListWavesFiltered` 签名改 `WaveListFilterInput`）
- Modify: `frontend/wailsjs/go/main/DemandController.d.ts` / `.js`（`ListDemandInboxRowsPage` 的 input 类型已有 `DemandInboxFilterInput`——只需 models 里补字段）
- Modify: `frontend/wailsjs/go/models.ts`（`DemandInboxFilterInput` +routingDispositions/demandKinds；`DemandInboxRowDTO` +pendingIntakeCount；新增 `BatchUnassignDemandInput/ItemResult/Result`、`WaveListFilterInput`）
- Modify: `frontend/src/shared/api/bridge.ts`（`listDemandInboxRowsPage` 入参扩展；新增 `batchUnassignDemandFromWave`；`listWavesFiltered` 入参扩展）
- Modify: `frontend/src/entities/demand.ts`（`DemandInboxRow` +pendingIntakeCount）

**Interfaces:**
- Consumes: Task 1-4 的 DTO 形状
- Produces（Task 6-8 消费）:
  - `bridge.listDemandInboxRowsPage(input: { assignment?; demandKind?; demandKinds?: string[]; routingDispositions?: string[]; integrationProfileId?; waveId?; sortBy?; sortDir?; limit; offset })`
  - `bridge.batchUnassignDemandFromWave(input: { waveId: number; docIds: number[] }): Promise<dto.BatchUnassignDemandResult>`（硬失败，assertWailsRuntime）
  - `bridge.listWavesFiltered(input: { page; pageSize; sortBy?; sortDesc?; lifecycleStage?; nameKeyword?; waveType? })`

- [ ] **Step 1: 手改 wailsjs 绑定**

按仓库惯例手维护（参考 `UnassignDemandFromWave` 在 `WaveController.d.ts` 的现有写法，字母序插入 `BatchUnassignDemandFromWave`）：

```ts
// WaveController.d.ts
export function BatchUnassignDemandFromWave(arg1: models.BatchUnassignDemandInput): Promise<models.BatchUnassignDemandResult>;
```

```js
// WaveController.js
export function BatchUnassignDemandFromWave(arg1) {
  return window['go']['main']['WaveController']['BatchUnassignDemandFromWave'](arg1);
}
```

`ListWavesFiltered` 两处签名 `models.PaginationInput` → `models.WaveListFilterInput`。
`models.ts` 按 DTO 镜像新增类（`createFrom`/`convertValues` 风格照抄同文件既有类），并给 `DemandInboxFilterInput` 补 `routingDispositions: string[]`、`demandKinds: string[]`，给 `DemandInboxRowDTO` 补 `pendingIntakeCount: number`。

- [ ] **Step 2: bridge 扩展**

`frontend/src/shared/api/bridge.ts`：

`listDemandInboxRowsPage`（:226-240）入参类型与 createFrom 调用补：

```ts
export async function listDemandInboxRowsPage(input: {
  assignment?: string
  demandKind?: string
  demandKinds?: string[]
  routingDispositions?: string[]
  integrationProfileId?: number
  waveId?: number
  sortBy?: string
  sortDir?: 'asc' | 'desc'
  limit: number
  offset: number
}): Promise<dto.DemandInboxPageResult> {
  if (!isWailsRuntimeAvailable()) {
    return dto.DemandInboxPageResult.createFrom({ items: [], totalCount: 0 })
  }
  return ListDemandInboxRowsPage(dto.DemandInboxFilterInput.createFrom(input))
}
```

`unassignDemandFromWave` 之后追加：

```ts
/** Batch-return demand documents to the unassigned pool (single undo node). Hard-fail. */
export async function batchUnassignDemandFromWave(input: {
  waveId: number
  docIds: number[]
}): Promise<dto.BatchUnassignDemandResult> {
  assertWailsRuntime()
  return BatchUnassignDemandFromWave(dto.BatchUnassignDemandInput.createFrom(input))
}
```

`listWavesFiltered`（:411-419）入参扩展并透传：

```ts
export async function listWavesFiltered(input: {
  page: number
  pageSize: number
  sortBy?: string
  sortDesc?: boolean
  lifecycleStage?: string
  nameKeyword?: string
  waveType?: string
}): Promise<dto.WavesPage> {
  if (!isWailsRuntimeAvailable()) return dto.WavesPage.createFrom({ items: [], pagination: {} })
  return ListWavesFiltered(dto.WaveListFilterInput.createFrom(input))
}
```

（保持与既有实现一致的软失败行为；检查既有 wrapper 的软/硬失败约定后沿用。）

`frontend/src/entities/demand.ts` 的 `DemandInboxRow` 接口补 `pendingIntakeCount: number`。

- [ ] **Step 3: typecheck 门**

Run: `cd frontend && deno task typecheck`
Expected: exit 0。

- [ ] **Step 4: Commit**

```bash
git add frontend/wailsjs/go/main/WaveController.d.ts frontend/wailsjs/go/main/WaveController.js frontend/wailsjs/go/models.ts frontend/src/shared/api/bridge.ts frontend/src/entities/demand.ts
git commit -m "feat(frontend): 同步收件箱过滤/批量退单/波次列表过滤的 Wails 绑定与 bridge 契约"
```

---

### Task 6: 全局收件箱——业务面分区、待分诊过滤、assignment 三态 URL 同步、计数列

**Files:**
- Create: `frontend/src/pages/inbox/inbox-grid/businessSurface.ts`（纯函数，可测）
- Create: `frontend/src/pages/inbox/inbox-grid/businessSurface.test.ts`
- Modify: `frontend/src/pages/inbox/inbox-grid/filter-schema.ts`（+routingDisposition 维度）
- Modify: `frontend/src/shared/i18n/glossary.ts`（+routingDisposition dimension 及 StatusTone/desc 表）
- Modify: `frontend/src/pages/inbox/inbox-grid/useInboxGrid.ts`（透传新过滤字段；assignment 写 URL）
- Modify: `frontend/src/pages/inbox/InboxPage.vue`（业务面 segmented 控件）
- Modify: `frontend/src/pages/inbox/inbox-grid/columns.ts`（+待分诊列）
- Modify: `frontend/src/shared/i18n/locales/zh-CN.ts` / `en-US.ts`（新键，两 locale 同落）

**Interfaces:**
- Consumes: Task 5 的 bridge 入参形状；`glossary.ts` 的 dimension 注册模式（照 `reviewRequirement` 维度写法）
- Produces:
  - `businessSurface.ts`: `export type BusinessSurface = 'all' | 'membership_entitlement' | 'retail_order'`；`surfaceFromKinds(kinds: readonly string[]): BusinessSurface`；`kindsFromSurface(surface: BusinessSurface): string[]`

- [ ] **Step 1: 写失败测试（businessSurface）**

```ts
// frontend/src/pages/inbox/inbox-grid/businessSurface.test.ts
import { describe, expect, it } from 'vitest'
import { kindsFromSurface, surfaceFromKinds } from './businessSurface'

describe('businessSurface', () => {
  it('derives all from 0 or 2 selected kinds', () => {
    expect(surfaceFromKinds([])).toBe('all')
    expect(surfaceFromKinds(['membership_entitlement', 'retail_order'])).toBe('all')
  })
  it('derives the single selected kind', () => {
    expect(surfaceFromKinds(['retail_order'])).toBe('retail_order')
    expect(surfaceFromKinds(['membership_entitlement'])).toBe('membership_entitlement')
  })
  it('maps surface back to kinds', () => {
    expect(kindsFromSurface('all')).toEqual([])
    expect(kindsFromSurface('retail_order')).toEqual(['retail_order'])
  })
})
```

Run: `cd frontend && deno task test -- businessSurface`（vitest 按文件名过滤）
Expected: 失败（模块不存在）。

- [ ] **Step 2: 实现 businessSurface.ts**

```ts
/** 收件箱业务面三态：全部 / 会员权益 / 零售订单。纯函数模块，供 InboxPage 与波内导入页共用。 */
export type BusinessSurface = 'all' | 'membership_entitlement' | 'retail_order'

export function surfaceFromKinds(kinds: readonly string[]): BusinessSurface {
  if (kinds.length === 1 && (kinds[0] === 'membership_entitlement' || kinds[0] === 'retail_order')) {
    return kinds[0]
  }
  return 'all'
}

export function kindsFromSurface(surface: BusinessSurface): string[] {
  return surface === 'all' ? [] : [surface]
}
```

- [ ] **Step 3: filter-schema + glossary**

`filter-schema.ts` 改为：

```ts
export const INBOX_GRID_FILTER_SCHEMA = [
  { key: 'demandKind', type: 'enum-multi', dimension: 'demandKind' },
  { key: 'routingDisposition', type: 'enum-multi', dimension: 'routingDisposition' },
] as const satisfies FilterSchema
```

`glossary.ts`：照 `reviewRequirement` 维度的注册模式新增 `routingDisposition` 维度，六个值 `pending_intake / accepted / deferred / excluded_manual / excluded_duplicate / excluded_revoked`（tone 与 label/desc 键严格镜像 `internal/domain/enums.go` 权威值集；`statusKit.dimensionNames` 补 `routingDisposition`）。

- [ ] **Step 4: useInboxGrid 透传 + assignment 写 URL**

`useInboxGrid.ts`：
`resolveDemandKindParam` 替换为多值直传：

```ts
  /** demandKind 与 routingDisposition 多值直传；空数组即不筛。 */
  function resolveInboxFilterParams(): { demandKinds: string[] | undefined; routingDispositions: string[] | undefined } {
    const kinds = filters.state.demandKind
    const dispositions = filters.state.routingDisposition
    return {
      demandKinds: kinds.length > 0 ? kinds : undefined,
      routingDispositions: dispositions.length > 0 ? dispositions : undefined,
    }
  }
```

`fetchPage` 的调用改为：

```ts
      const { demandKinds, routingDispositions } = resolveInboxFilterParams()
      const result = await listDemandInboxRowsPage({
        assignment: scopedWaveId != null ? 'assigned' : assignment.value === 'all' ? undefined : assignment.value,
        demandKinds,
        routingDispositions,
        waveId: scopedWaveId?.value,
        sortBy: sortBy.value ?? undefined,
        sortDir: sortDir.value ?? undefined,
        limit: pageSize.value,
        offset: (page.value - 1) * pageSize.value,
      })
```

assignment URL 同步（仅 unscoped）：`useRoute` 已在 init 读取；再取 `useRouter`，在既有的 `watch(assignment, ...)`（:192-196）内补：

```ts
  watch(assignment, () => {
    if (isWaveScoped) return
    page.value = 1
    const nextQuery = { ...route.query }
    if (assignment.value === 'all') delete nextQuery.assignment
    else nextQuery.assignment = assignment.value
    void router.replace({ query: nextQuery }).catch(() => { /* duplicated navigation */ })
    void fetchPage()
  })
```

（`route`/`router` 提升到函数作用域；保持 FilterBar 深链只读初始化的既有注释语义。）

- [ ] **Step 5: InboxPage 业务面 segmented + columns 待分诊列**

`InboxPage.vue` script 补：

```ts
import { kindsFromSurface, surfaceFromKinds, type BusinessSurface } from './inbox-grid/businessSurface'

const businessSurface = computed<BusinessSurface>(() => surfaceFromKinds(filters.state.demandKind))

function handleSurfaceChange(surface: BusinessSurface): void {
  filters.setEnumValues('demandKind', kindsFromSurface(surface))
}
```

模板在 assignment 行之后插入：

```html
    <div class="inbox-page__surface">
      <span class="inbox-page__assignment-label">{{ t('inbox.filters.businessSurface') }}</span>
      <NRadioGroup :value="businessSurface" @update:value="handleSurfaceChange">
        <NRadioButton value="all">{{ t('inbox.surface.all') }}</NRadioButton>
        <NRadioButton value="membership_entitlement">{{ t('inbox.surface.membership') }}</NRadioButton>
        <NRadioButton value="retail_order">{{ t('inbox.surface.retail') }}</NRadioButton>
      </NRadioGroup>
    </div>
```

（`businessSurface` 用 `computed` 时补 `computed` import；`NRadioButton` 已 import。样式复用 `.inbox-page__assignment` 的 gap 规则。）

`columns.ts` 在 `totalLineCount` 列之后插入：

```ts
    {
      type: 'number',
      key: 'pendingIntakeCount',
      title: t('inbox.columns.pendingTriage'),
      width: 90,
      sortable: false,
      getValue: (row) => row.pendingIntakeCount,
    },
```

- [ ] **Step 6: i18n 键（zh-CN.ts 与 en-US.ts 同落）**

`inbox.filters.businessSurface`、`inbox.surface.all/membership/retail`、`inbox.columns.pendingTriage`、`glossary.routingDisposition.{pending_intake,accepted,deferred,excluded_manual,excluded_duplicate,excluded_revoked}.{label,desc}`、`statusKit.dimensionNames.routingDisposition`。文案与既有 glossary 键风格一致（中性）。

- [ ] **Step 7: 全量前端门**

Run: `cd frontend && deno task typecheck && deno task test && deno task build && deno task lint:guardrails`
Expected: 全绿；guardrails 零违规（新维度渲染必须经 FilterBar/StatusBadge，不得裸枚举）。

- [ ] **Step 8: Commit**

```bash
git add frontend/src/pages/inbox/ frontend/src/shared/i18n/
git commit -m "feat(inbox): 收件箱业务面分区、待分诊过滤维度与计数列、assignment 三态写入 URL"
```

---

### Task 7: 波内导入页面（WaveIntakeTab 重构 + 拉取需求 + 波内导入）

**Files:**
- Modify: `frontend/src/pages/waves/workspace/tabs/WaveIntakeTab.vue`（重构为波内导入页面）
- Create: `frontend/src/pages/waves/workspace/tabs/intake/PullDemandsDialog.vue`
- Modify: `frontend/src/pages/inbox/ImportFileModal.vue`（`targetWaveId` prop：导入成功后把新单据分派进目标波）
- Test: `frontend/src/pages/inbox/ImportFileModal.test.ts`（新建，mock bridge）

**Interfaces:**
- Consumes: Task 5 的 `bridge.batchUnassignDemandFromWave` / `batchAssignDemandToWave` / `listWavesFiltered`；`useInboxGrid`（unscoped 实例的 `assignment` ref 可手动置 `'unassigned'`）
- Produces:
  - `ImportFileModal` 新 prop `targetWaveId?: number`；新 emit `assignedToWave: [docIds: number[]]`
  - `PullDemandsDialog`（props: `show: boolean`；emits: `update:show` / `pulled: [count: number]`）——内部用 `useInboxGrid()` unscoped 实例、初始 `assignment='unassigned'`、业务面 segmented 复用 `businessSurface.ts`、批量拉取调 `bridge.batchAssignDemandToWave({waveId, docIds})`

- [ ] **Step 1: 写失败测试（ImportFileModal 的波内导入流）**

```ts
// frontend/src/pages/inbox/ImportFileModal.test.ts
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { flushPromises } from '@vue/test-utils'
import ImportFileModal from './ImportFileModal.vue'

vi.mock('@/shared/api/bridge', () => ({
  batchAssignDemandToWave: vi.fn(),
  importDemandCSV: vi.fn(),
  parseCSVFile: vi.fn(),
  // ...其余 bridge 导出按组件实际 import 补齐 mock
}))

import { batchAssignDemandToWave, importDemandCSV } from '@/shared/api/bridge'
const assignMock = vi.mocked(batchAssignDemandToWave)
const importMock = vi.mocked(importDemandCSV)

describe('ImportFileModal targetWaveId 波内导入', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('导入成功后把新单据分派到 targetWave 并 emit assignedToWave', async () => {
    importMock.mockResolvedValue({ document: { id: 42 }, errors: [], successCount: 1 } as never)
    assignMock.mockResolvedValue({ results: [{ demandDocumentId: 42, success: true }], successCount: 1, failureCount: 0 } as never)
    const wrapper = mount(ImportFileModal, { props: { show: true, targetWaveId: 7 } })
    // 走到导入执行（组件内部流程：预览 -> importDemandCSV）
    // 以组件实际暴露的流程触发；若导入入口需预览状态，按组件实现的步骤构造。
    await flushPromises()
    expect(assignMock).toHaveBeenCalledWith({ waveId: 7, docIds: [42] })
    expect(wrapper.emitted('assignedToWave')).toBeTruthy()
  })
})
```

Run: `cd frontend && deno task test -- ImportFileModal`
Expected: 失败（prop/流程不存在）。

- [ ] **Step 2: ImportFileModal 支持 targetWaveId**

`ImportFileModal.vue`：
props 加 `targetWaveId?: number`；导入成功分支（`ImportDemandCSV` 返回 `result.document` 时）追加：

```ts
    if (props.targetWaveId != null && result.document != null) {
      const assignResult = await batchAssignDemandToWave({ waveId: props.targetWaveId, docIds: [result.document.id] })
      if (assignResult.failureCount > 0) {
        feedback.error(t('inbox.import.assignToWaveFailed'))
      } else {
        emit('assignedToWave', [result.document.id])
      }
    }
```

结果步新增「发送到波次」CTA（无 targetWaveId 时打开选波 picker，复用 `BatchActionBar` 的 picker 逻辑抽取或直接内联 `listWavesFiltered`+`batchAssignDemandToWave`）；有 targetWaveId 时自动执行上段逻辑。i18n 新键 `inbox.import.assignToWaveFailed`、`inbox.import.sendToWave` 双语同落。

- [ ] **Step 3: PullDemandsDialog**

新组件骨架（复用 `useInboxGrid` unscoped + `businessSurface.ts` + `DataGrid` selectable + 底部确认）：

```vue
<script setup lang="ts">
/**
 * PullDemandsDialog — 波内导入页面的「拉取需求」弹窗：浏览未分派池（业务面三态 +
 * FilterBar + server 分页），批量勾选后调 batchAssignDemandToWave 拉入当前波次。
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NRadioButton, NRadioGroup } from 'naive-ui'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import { FilterBar } from '@/shared/ui/filter-bar'
import { useFeedback } from '@/shared/ui/feedback'
import { batchAssignDemandToWave } from '@/shared/api/bridge'
import { useInboxGrid } from '@/pages/inbox/inbox-grid/useInboxGrid'
import { buildInboxColumns } from '@/pages/inbox/inbox-grid/columns'
import { kindsFromSurface, surfaceFromKinds, type BusinessSurface } from '@/pages/inbox/inbox-grid/businessSurface'

const props = defineProps<{ show: boolean; waveId: number }>()
const emit = defineEmits<{ 'update:show': [boolean]; pulled: [count: number] }>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const grid = useInboxGrid()
grid.assignment.value = 'unassigned'

const surface = computed<BusinessSurface>(() => surfaceFromKinds(grid.filters.state.demandKind))
function handleSurfaceChange(next: BusinessSurface): void {
  grid.filters.setEnumValues('demandKind', kindsFromSurface(next))
}

const pulling = ref(false)
async function handlePull(): Promise<void> {
  if (grid.selectedKeys.value.length === 0) return
  pulling.value = true
  try {
    const result = await batchAssignDemandToWave({ waveId: props.waveId, docIds: grid.selectedKeys.value })
    if (result.failureCount > 0) feedback.error(t('waveWorkspace.intake.pullSomeFailed', { count: result.failureCount }))
    else feedback.success(t('feedback.success'))
    emit('pulled', result.successCount)
    emit('update:show', false)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    pulling.value = false
  }
}
</script>
```

模板：`NModal`（preset="card"、宽 960）内 `NRadioGroup`（surface 三态）+ `FilterBar :filters="grid.filters"` + `DataGrid`（selectable、server 分页、`:selected-keys="grid.selectedKeys"` @update:selected-keys 转 number[]）+ footer 取消/拉取（`NButton type="primary" :loading="pulling"`）。

- [ ] **Step 4: WaveIntakeTab 重构**

`WaveIntakeTab.vue` 改版要点（保留行详情与单行退单）：
- toolbar 三个动作：`拉取需求`（primary，打开 PullDemandsDialog）、`导入文件`（打开 ImportFileModal 且 `:target-wave-id="ctx.waveId.value"`）、`从收件箱分派更多`（降级为次要，深链 `/inbox?assignment=unassigned`）。
- 加 `SavedViews :filters="grid.filters" scope-id="wave-intake"` 与 `FilterBar :filters="grid.filters"`（波内实例不再只读）。
- `DataGrid` 开 `selectable` + `:selected-keys="grid.selectedKeys"` + `#selection-toolbar` 插批量退单按钮（调 `bridge.batchUnassignDemandFromWave({waveId, docIds: grid.selectedKeys.value})`，成功后 `grid.mutationDone()` + `ctx.refresh()` + `feedback.receipt`）。
- 「拉取成功」后 `grid.mutationDone()` + `ctx.refresh()`。
- i18n 新键 `waveWorkspace.intake.pullDemands/pullSomeFailed/importIntoWave/unassignSelected` 双语同落。

- [ ] **Step 5: 前端门 + Commit**

Run: `cd frontend && deno task typecheck && deno task test && deno task build && deno task lint:guardrails`
Expected: 全绿。

```bash
git add frontend/src/pages/waves/workspace/tabs/WaveIntakeTab.vue frontend/src/pages/waves/workspace/tabs/intake/ frontend/src/pages/inbox/ImportFileModal.vue frontend/src/pages/inbox/ImportFileModal.test.ts frontend/src/shared/i18n/
git commit -m "feat(wave): 波内导入页面——拉取需求弹窗、波内文件导入（导入即入波）、批量退单"
```

---

### Task 8: 深链全修 + 分派回执直达 + picker 增强 + 建波直落 intake

**Files:**
- Modify: `frontend/src/shared/lib/wave-filter-link.ts`（单数键 + 目标子路由）
- Create: `frontend/src/shared/lib/wave-filter-link.test.ts`
- Modify: `frontend/src/pages/waves/workspace/tabs/WaveOverviewTab.vue`（漏斗/六桶点击目标改 lines 路由）
- Modify: `frontend/src/pages/inbox/inbox-grid/BatchActionBar.vue`（回执直达 intake；picker 过滤 + 排除已关闭 + 截断提示）
- Modify: `frontend/src/pages/waves/components/CreateWaveDialog.vue`（创建成功直落 intake）

**Interfaces:**
- Consumes: Task 4/5 的 `listWavesFiltered` 过滤入参；路由名 `wave-workspace-intake` / `wave-workspace-lines`（`frontend/src/app/router/index.ts:48-66` 已确认）
- Produces: `buildWaveFilterLink` 新契约——query 键用**单数**（与 fulfillment-grid `useUrlFilters` schema 键一致），目标 `wave-workspace-lines`；`filter.stepKey === 'intake'` 时目标 `wave-workspace-intake` 且不带过滤 query。

- [ ] **Step 1: 写失败测试**

```ts
// frontend/src/shared/lib/wave-filter-link.test.ts
import { describe, expect, it } from 'vitest'
import { buildWaveFilterLink } from './wave-filter-link'

const filter = (overrides: Record<string, unknown> = {}) =>
  ({ allocationState: 'missing', stepKey: '', ...overrides }) as never

describe('buildWaveFilterLink', () => {
  it('targets the lines tab with singular query keys', () => {
    const link = buildWaveFilterLink(3, filter())
    expect(link).toMatchObject({ name: 'wave-workspace-lines', params: { id: 3 }, query: { allocationState: 'missing' } })
  })
  it('targets intake tab when stepKey is intake and drops grid filters', () => {
    const link = buildWaveFilterLink(3, filter({ stepKey: 'intake' }))
    expect(link).toMatchObject({ name: 'wave-workspace-intake', params: { id: 3 }, query: {} })
  })
})
```

Run: `cd frontend && deno task test -- wave-filter-link`
Expected: 失败（现实现返回 `wave-workspace` + 复数键）。

- [ ] **Step 2: 修 buildWaveFilterLink**

```ts
const FIELD_TO_QUERY_KEY = {
  allocationState: 'allocationState',
  addressState: 'addressState',
  supplierState: 'supplierState',
  channelSyncState: 'channelSyncState',
  reviewRequirement: 'reviewRequirement',
  drift: 'driftStatus',
} as const

export function buildWaveFilterLink(waveId: number, filter: dto.ActionCenterBucketFilterDTO): RouteLocationRaw {
  const query: Record<string, string> = {}
  const isIntake = filter.stepKey === 'intake'

  if (!isIntake) {
    for (const [field, queryKey] of Object.entries(FIELD_TO_QUERY_KEY) as Array<[keyof typeof FIELD_TO_QUERY_KEY, string]>) {
      const value = filter[field]
      if (!value) continue
      const serialized = serializeEnumMultiQuery([value])
      if (serialized !== undefined) query[queryKey] = serialized
    }
  }

  return {
    name: isIntake ? 'wave-workspace-intake' : 'wave-workspace-lines',
    params: { id: waveId },
    query,
  }
}
```

- [ ] **Step 3: 总览漏斗/六桶点击落 lines**

`WaveOverviewTab.vue`（:76-93、:128-130 两处 push）统一改为经 `buildWaveFilterLink(waveId, filter)` 导航（把现有复数键构造删掉）；总览自身不再作为过滤深链的落点。

- [ ] **Step 4: BatchActionBar 回执直达 + picker 增强**

picker（:34-47）改：

```ts
    const page = await listWavesFiltered({
      page: 1,
      pageSize: 200,
      sortBy: 'updatedAt',
      sortDesc: true,
      lifecycleStage: 'active',
      nameKeyword: waveKeyword.value.trim() || undefined,
    })
```

（加 `waveKeyword` ref + NSelect 上方的 NInput 搜索框；若后端 `active` 值与 `domain.LifecycleStageActive` 不一致，以 `internal/domain/enums.go` 实际值为准。截断提示：`page.pagination.totalCount > 200` 时在 modal 内渲染 `NAlert`，文案键 `inbox.batch.waveListTruncated`。）

分派成功后（:59-66 分支）替换 receipt 为带跳转的 action：

```ts
    const targetId = targetWaveId.value
    feedback.receipt({
      kind: 'action',
      summary: t('inbox.batch.assignToWaveDone', { n: result.successCount }),
      actionLabel: t('inbox.batch.openWave'),
      onAction: () => void router.push({ name: 'wave-workspace-intake', params: { id: targetId } }),
    })
```

（`useRouter` import；`feedback.receipt` 的 action 参数以 `shared/ui/feedback` 实际 API 为准——若 receipt 不支持 action 回调，则改用 `feedback.receipt` 渲染带 `RouterLink` 的自定义内容，或成功后直接 `router.push`。实现时先看 ReceiptTray 的 props。）

- [ ] **Step 5: 建波直落 intake**

`CreateWaveDialog.vue` 创建成功后：`void router.push({ name: 'wave-workspace-intake', params: { id: createdWave.id } })`（保持现有 close dialog 行为）。

- [ ] **Step 6: 前端门 + Commit**

Run: `cd frontend && deno task typecheck && deno task test && deno task build && deno task lint:guardrails`
Expected: 全绿。

```bash
git add frontend/src/shared/lib/wave-filter-link.ts frontend/src/shared/lib/wave-filter-link.test.ts frontend/src/pages/waves/workspace/tabs/WaveOverviewTab.vue frontend/src/pages/inbox/inbox-grid/BatchActionBar.vue frontend/src/pages/waves/components/CreateWaveDialog.vue frontend/src/shared/i18n/
git commit -m "fix(wave): 深链单复数键对齐与落点修正、分派回执一键直达 intake、picker 过滤、建波直落 intake"
```

---

### Task 9: 收尾——i18n parity、guardrails、六门回归

**Files:**
- Modify: `frontend/src/shared/i18n/locales/zh-CN.ts` / `en-US.ts`（补齐 Task 6-8 遗漏键）

- [ ] **Step 1: i18n 叶子键 parity 检查**

Run（仓库根）:

```bash
cd frontend && deno eval "
import zh from './src/shared/i18n/locales/zh-CN.ts';
import en from './src/shared/i18n/locales/en-US.ts';
const leaves = (o: Record<string, unknown>, p = ''): string[] => Object.entries(o).flatMap(([k, v]) => v && typeof v === 'object' && !Array.isArray(v) ? leaves(v as Record<string, unknown>, p + k + '.') : [p + k]);
const z = new Set(leaves(zh)), e = new Set(leaves(en));
const onlyZh = [...z].filter(k => !e.has(k)), onlyEn = [...e].filter(k => !z.has(k));
console.log('zh-only:', onlyZh.length, onlyZh.slice(0, 10));
console.log('en-only:', onlyEn.length, onlyEn.slice(0, 10));
"
```

Expected: 两行均输出 0。（若 locale 文件导出结构非默认导出对象，按实际导出形式调整上面的 eval——先 `Read` 两个 locale 文件头部确认导出形态。）

- [ ] **Step 2: 六门全量回归**

Run（仓库根）:

```bash
go build ./... && go test -count=1 ./... && cd frontend && deno task typecheck && deno task test && deno task build && deno task lint:guardrails
```

Expected: 六门全绿（vitest 23+ 文件全过；guardrails `no violations found`；enum check 通过）。

- [ ] **Step 3: 手工走查清单（无需自动化）**

1. 任务中心 →「收件箱 N 单待分诊」卡 → 落收件箱且 `routingDisposition=pending_intake` 已预筛。
2. 任务中心 bucket 卡 → 落 lines tab 且网格预过滤生效（URL 挂单数键）。
3. 全局收件箱：业务面三态切换、批量分派 → 回执点击 → 直达波次 intake。
4. 波内 intake：拉取需求弹窗勾选未分派单 → 拉入 → 行出现在本波；波内导入文件 → 导入即入波；批量退单 → 回全局收件箱 unassigned。
5. 未分诊会员单分派 → 被后端拒绝且提示；重复分派 → 被拒绝。
6. 选波 picker：关闭的波次不出现、名称搜索生效。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/shared/i18n/ && git commit -m "chore(i18n): 节 2 波内导入与收件箱改造的收尾键补齐与 parity"
```

---

## Self-Review 记录

- **Spec 覆盖**：§4.1（全局收件箱五项）→ Task 1/6/8；§4.2（波内导入页面）→ Task 2/7；§4.3（深链全修）→ Task 8；§4.4（门禁）→ Task 3；§4.5（picker）→ Task 4/8；§4.6（建波流程）→ Task 8；文档定案项（禁跨波拆分 #34）→ Task 3。节 3 内容（零售自动裁决、mixed 快照、引擎边界、waived、lineReason 过滤）不在本计划，另立计划。
- **类型一致性**：`BatchUnassignDemandResult` 在 Task 2 定义、Task 5 绑定、Task 7 消费，命名一致；`WaveListFilterInput` 在 Task 4 定义、Task 5/8 消费；`businessSurface` 函数名在 Task 6 定义、Task 7 消费。
- **已知前置**：Task 3 的 kind 豁免注释依赖节 3 计划落地后收紧；Task 8 的 `feedback.receipt` action 参数与 `persistence.DemandLine` 表名、`domain.Wave` 字段名需实现时以仓库实际为准（均在对应步骤内给出了验证命令）。
