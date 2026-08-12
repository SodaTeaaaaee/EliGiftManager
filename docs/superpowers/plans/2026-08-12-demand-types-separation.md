# 需求类型：分引擎、分行源、分业务面 实施计划（规范节 3）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让两类需求（会员权益 / 零售订单）在工作流上彻底分流——分配引擎、行来源、业务面各自独立：会员走规则核算、零售走订单映射；零售导入免分诊；同一 profile 的双身份合并为 mixed 快照；reissue/compensation 产生独立 `wave_adjustment` 来源行；waived 统一为「豁免=扣下不发」；履约网格按 lineReason 精确过滤「已调整」行。

**Architecture:** 后端先行（Task 1-5）：快照聚合与 waived 口径 → 引擎边界修复 → 零售免分诊 → 调整来源三值补齐 → lineReasons 过滤维度；随后同步 Wails 绑定与 bridge 契约（Task 6）；前端分两片落地：分配页签 waveType 软生效与预筛联动（Task 7）、行详情 kind 分流与网格过滤升级（Task 8）；最后收紧节 2 遗留的分派门禁豁免并做六门回归（Task 9）。本计划多处依赖节 2 计划（`docs/superpowers/plans/2026-08-12-inbox-waves-interaction.md`）已落地：节 2 Task 6 的 `businessSurface.ts`、Task 7 的波内导入页面重构、Task 3 的 `assignOne` kind 豁免——后者在本计划 Task 9 收紧，节 2 的其他变更一律不回退。

**Tech Stack:** Go + GORM + SQLite（后端）；Vue 3 + TypeScript + Vite 8 + vitest 4 + Naive UI + vue-i18n（前端）；Deno 工具链。

**Spec:** `docs/superpowers/specs/2026-08-12-inbox-waves-demand-types-vite-design.md` §2（决策账本）、§5（节 3 全部 7 项）、§7（待办）。

## Global Constraints

- 所有 Wails 运行时调用必须走 `frontend/src/shared/api/bridge.ts`；页面禁止直引 `wailsjs`（guardrail rule3）。
- 前端命令只用 `deno task ...`；禁止 npm/yarn/pnpm。
- 用户可见文案必须走 vue-i18n，zh-CN 与 en-US 叶子键集合保持一致（每任务新增键两 locale 同落）。
- 枚举值渲染必须走 glossary/StatusBadge/FilterBar，禁止裸枚举（guardrail rule2）。`lineReason` 维度已在 `frontend/src/shared/i18n/glossary.ts:343-347,469` 注册（三值齐全），本计划只消费不新增维度。
- Go 代码 `gofmt`；每个后端任务跑 `go build ./... && go test -count=1 ./...` 全绿。
- 每个前端任务跑 `cd frontend && deno task typecheck && deno task test && deno task build && deno task lint:guardrails` 全绿。
- 提交风格：`feat:` / `fix:` / `refactor:` + 中文描述，与仓库历史一致；每个任务一个 commit。
- 不修改 `docs/`、不动 `frontend-legacy/`。
- 节 2 计划已改过的文件（`list_pagination*`、`wave_lifecycle*`、`useInboxGrid.ts`、`InboxPage.vue`、`WaveIntakeTab.vue`、`BatchActionBar.vue` 等）本计划可以继续改，但不得回退节 2 的任何变更；节 2 Task 3 在 `assignOne` 中的零售 kind 豁免保持到本计划 Task 9 再收紧。
- 工作目录：仓库根 `E:\Projects\Code\EliGiftManager`（Git Bash，路径用正斜杠）。

---

### Task 1: 混合快照聚合（SnapshotTypeMixed）+ waived 口径统一 + 快照查询稳定排序

**Files:**
- Modify: `internal/app/use_cases.go`（`GenerateParticipants` 重写，:128-220）
- Modify: `internal/app/entitlement_routing_usecase.go`（`GetWaveRoutingStats` waived 单列，:101-112）
- Modify: `internal/app/dto/demand.go`（`WaveRoutingStatsDTO` +waivedCount，:136-146）
- Modify: `internal/infra/demand_assignment_repo.go`（`ListDemandDocumentsByWave` 稳定排序，:77-100）
- Modify: `internal/infra/demand_repo.go`（`ListLinesByDocument` 稳定排序，:120-130）
- Modify: `internal/infra/wave_repo.go`（`ListParticipantsByWave` 稳定排序，:81-91）
- Test: `internal/app/use_cases_test.go`（追加 2 用例 + mockWaveRepo.AddParticipant 落库；harness = 文件内既有 mock，:15-402）
- Test: `internal/app/entitlement_routing_usecase_test.go`（新建，纯 mock 测试）

**Interfaces:**
- Consumes: `domain.WaveParticipantSnapshot`（SnapshotType 常量 `domain.SnapshotTypeMixed` 已在 `internal/domain/enums.go:103-110` 定义未生成，本次启用）；`isEligibleForFulfillment`（同包 `use_cases.go:239-244`）
- Produces:
  - `GenerateParticipants` 新语义：按 `CustomerProfileID` 聚合波内全部文档 → 任一文档存在 accepted+ready/not_required 行才生成快照；kind 双身份 → `SnapshotType="mixed"`；`GiftLevel` 只取 membership_entitlement 行；`SourceDocumentRefs` 按文档 ID 升序逗号连接（依赖 repo 稳定排序）
  - `dto.WaveRoutingStatsDTO.WaivedCount int`（供 Task 6 绑定；前端当前无消费者——已核实 `getWaveRoutingStats` 仅 bridge 导出无页面调用，统计 UI 一致性问题不涉及前端改动）

- [ ] **Step 1: 写失败测试（混合快照）**

`internal/app/use_cases_test.go` 末尾追加（`mockWaveRepo.AddParticipant` 目前只分配 ID 不落库——`policyWaveRepo` 的写法是落库的，先把它改成同款，否则快照无法断言）：

```go
func (m *mockWaveRepo) AddParticipant(ctx context.Context, snap *domain.WaveParticipantSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap.ID = m.next()
	cp := *snap
	m.participants = append(m.participants, cp)
	return nil
}

func TestGenerateParticipantsMergesMixedSnapshot(t *testing.T) {
	ctx := context.Background()
	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()
	uc := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo)

	profileID := uint(100)
	memberDoc := &domain.DemandDocument{
		Kind: "membership_entitlement", SourceChannel: "bilibili", SourceCustomerRef: "uid-1",
		CustomerProfileID: &profileID,
	}
	retailDoc := &domain.DemandDocument{
		Kind: "retail_order", SourceChannel: "shop", SourceCustomerRef: "buyer-1",
		CustomerProfileID: &profileID,
	}
	intakeUC := NewDemandIntakeUseCase(demandRepo)
	if err := intakeUC.ImportDemand(ctx, memberDoc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", GiftLevelSnapshot: "gold", LineType: "entitlement_rule"},
		{RoutingDisposition: "accepted", RecipientInputState: "waiting_for_input", GiftLevelSnapshot: "silver", LineType: "entitlement_rule"},
	}); err != nil {
		t.Fatalf("import member doc: %v", err)
	}
	if err := intakeUC.ImportDemand(ctx, retailDoc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 2, LineType: "sku_order"},
	}); err != nil {
		t.Fatalf("import retail doc: %v", err)
	}
	for _, doc := range []*domain.DemandDocument{memberDoc, retailDoc} {
		if err := assignmentRepo.Create(ctx, &domain.WaveDemandAssignment{WaveID: 1, DemandDocumentID: doc.ID}); err != nil {
			t.Fatalf("assign doc %d: %v", doc.ID, err)
		}
	}

	count, err := uc.GenerateParticipants(ctx, 1)
	if err != nil {
		t.Fatalf("GenerateParticipants: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 merged snapshot, got %d", count)
	}
	snaps, err := waveRepo.ListParticipantsByWave(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
	snap := snaps[0]
	if snap.SnapshotType != "mixed" {
		t.Errorf("SnapshotType = %q, want mixed", snap.SnapshotType)
	}
	if snap.GiftLevel != "gold" {
		t.Errorf("GiftLevel = %q, want gold (membership line only — waiting_for_input line must not contribute)", snap.GiftLevel)
	}
	if snap.IdentityPlatform != "bilibili" {
		t.Errorf("IdentityPlatform = %q, want membership doc's channel", snap.IdentityPlatform)
	}
	if want := fmt.Sprintf("%d,%d", memberDoc.ID, retailDoc.ID); snap.SourceDocumentRefs != want {
		t.Errorf("SourceDocumentRefs = %q, want %q (doc-id ascending)", snap.SourceDocumentRefs, want)
	}
}

func TestGenerateParticipantsSkipsLinesNotReadyForFulfillment(t *testing.T) {
	ctx := context.Background()
	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()
	uc := NewWaveUseCase(waveRepo, demandRepo, assignmentRepo)

	profileID := uint(200)
	doc := &domain.DemandDocument{
		Kind: "membership_entitlement", CustomerProfileID: &profileID,
	}
	if err := NewDemandIntakeUseCase(demandRepo).ImportDemand(ctx, doc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "waiting_for_input", LineType: "entitlement_rule"},
		{RoutingDisposition: "accepted", RecipientInputState: "waived", LineType: "entitlement_rule"},
	}); err != nil {
		t.Fatalf("import doc: %v", err)
	}
	if err := assignmentRepo.Create(ctx, &domain.WaveDemandAssignment{WaveID: 1, DemandDocumentID: doc.ID}); err != nil {
		t.Fatalf("assign doc: %v", err)
	}

	count, err := uc.GenerateParticipants(ctx, 1)
	if err != nil {
		t.Fatalf("GenerateParticipants: %v", err)
	}
	if count != 0 {
		t.Fatalf("waiting_for_input/waived lines must not produce a snapshot, got %d", count)
	}
	snaps, _ := waveRepo.ListParticipantsByWave(ctx, 1)
	if len(snaps) != 0 {
		t.Fatalf("expected 0 snapshots, got %d", len(snaps))
	}
}
```

- [ ] **Step 2: 写失败测试（waived 统计单列）**

新建 `internal/app/entitlement_routing_usecase_test.go`（`dto` import 留待 Task 3 追加该文件的第二个用例时一并补——本文件当前只用到 `domain` 与 mock）：

```go
package app

import (
	"context"
	"testing"

	"github.com/SodaTeaaaaee/EliGiftManager/internal/domain"
)

func TestGetWaveRoutingStatsSeparatesWaivedFromReady(t *testing.T) {
	ctx := context.Background()
	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	uc := NewEntitlementRoutingUseCase(demandRepo, assignmentRepo)

	doc := &domain.DemandDocument{Kind: "membership_entitlement"}
	if err := NewDemandIntakeUseCase(demandRepo).ImportDemand(ctx, doc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", LineType: "entitlement_rule"},
		{RoutingDisposition: "accepted", RecipientInputState: "not_required", LineType: "entitlement_rule"},
		{RoutingDisposition: "accepted", RecipientInputState: "waived", LineType: "entitlement_rule"},
	}); err != nil {
		t.Fatalf("import doc: %v", err)
	}
	if err := assignmentRepo.Create(ctx, &domain.WaveDemandAssignment{WaveID: 1, DemandDocumentID: doc.ID}); err != nil {
		t.Fatalf("assign doc: %v", err)
	}

	stats, err := uc.GetWaveRoutingStats(ctx, 1)
	if err != nil {
		t.Fatalf("GetWaveRoutingStats: %v", err)
	}
	if stats.TotalLines != 3 {
		t.Fatalf("TotalLines = %d, want 3", stats.TotalLines)
	}
	if stats.AcceptedReadyCount != 2 {
		t.Errorf("AcceptedReadyCount = %d, want 2 (ready + not_required, waived excluded)", stats.AcceptedReadyCount)
	}
	if stats.WaivedCount != 1 {
		t.Errorf("WaivedCount = %d, want 1", stats.WaivedCount)
	}
}
```

- [ ] **Step 3: 跑测试确认失败**

Run: `go test -count=1 ./internal/app/ -run 'TestGenerateParticipants|TestGetWaveRoutingStatsSeparatesWaivedFromReady'`
Expected: 编译失败（`WaivedCount` 未定义）+ `TestGenerateParticipantsMergesMixedSnapshot` 失败（现状按 doc 各写一条 member/buyer 快照，不会合并）。

- [ ] **Step 4: 重写 GenerateParticipants**

`internal/app/use_cases.go` 的 `GenerateParticipants`（:128-220）整体替换为：

```go
func (uc *waveUseCase) GenerateParticipants(ctx context.Context, waveID uint) (int, error) {
	// Get demand documents assigned to this wave
	docs, err := uc.assignmentRepo.ListDemandDocumentsByWave(ctx, waveID)
	if err != nil {
		return 0, err
	}

	// Group assigned documents by customer profile so a profile that appears in
	// BOTH a membership_entitlement and a retail_order document merges into ONE
	// mixed snapshot (spec §5.3 双身份合并一条).
	docsByProfile := make(map[uint][]domain.DemandDocument, len(docs))
	skippedNoProfile := 0
	for i := range docs {
		doc := &docs[i]
		if doc.CustomerProfileID == nil {
			skippedNoProfile++
			continue
		}
		docsByProfile[*doc.CustomerProfileID] = append(docsByProfile[*doc.CustomerProfileID], *doc)
	}

	// Get existing participants for idempotency check
	existingSnaps, err := uc.waveRepo.ListParticipantsByWave(ctx, waveID)
	if err != nil {
		return 0, err
	}
	existingProfiles := make(map[uint]bool, len(existingSnaps))
	for _, snap := range existingSnaps {
		existingProfiles[snap.CustomerProfileID] = true
	}

	// Iterate profiles in ascending ID order so snapshot creation order (and thus
	// downstream reads) is deterministic.
	profileIDs := make([]uint, 0, len(docsByProfile))
	for profileID := range docsByProfile {
		profileIDs = append(profileIDs, profileID)
	}
	sort.Slice(profileIDs, func(i, j int) bool { return profileIDs[i] < profileIDs[j] })

	count := 0
	for _, profileID := range profileIDs {
		if existingProfiles[profileID] {
			continue
		}
		profileDocs := docsByProfile[profileID]

		// Aggregate eligible lines across the profile's documents. A line is
		// eligible only when accepted AND its recipient input is ready/not_required
		// — waiting_for_input / waived / expired lines never enter a snapshot.
		hasEligible := false
		kinds := make(map[string]bool, 2)
		var giftLevel string
		var docRefs []string
		var identityPlatform, identityValue string
		identityFromFallback := false
		for docIdx := range profileDocs {
			doc := &profileDocs[docIdx]
			kinds[doc.Kind] = true
			docRefs = append(docRefs, fmt.Sprintf("%d", doc.ID))
			if !identityFromFallback && doc.SourceChannel != "" {
				identityPlatform, identityValue, identityFromFallback = doc.SourceChannel, doc.SourceCustomerRef, true
			}
			if doc.Kind == "membership_entitlement" {
				// Primary identity always comes from the membership side for mixed snapshots.
				identityPlatform, identityValue = doc.SourceChannel, doc.SourceCustomerRef
			}
			lines, err := uc.demandRepo.ListLinesByDocument(ctx, doc.ID)
			if err != nil {
				return count, err
			}
			for lineIdx := range lines {
				line := &lines[lineIdx]
				if !isEligibleForFulfillment(line) {
					continue
				}
				hasEligible = true
				if doc.Kind == "membership_entitlement" && giftLevel == "" {
					giftLevel = line.GiftLevelSnapshot
				}
			}
		}

		// Only generate a snapshot when at least one eligible line exists.
		if !hasEligible {
			continue
		}

		snapshotType := "member"
		switch {
		case kinds["membership_entitlement"] && kinds["retail_order"]:
			snapshotType = "mixed"
		case kinds["retail_order"]:
			snapshotType = "buyer"
		}

		snap := domain.WaveParticipantSnapshot{
			WaveID:             waveID,
			CustomerProfileID:  profileID,
			SnapshotType:       snapshotType,
			IdentityPlatform:   identityPlatform,
			IdentityValue:      identityValue,
			DisplayName:        "",
			GiftLevel:          giftLevel,
			SourceDocumentRefs: strings.Join(docRefs, ","), // 升序依赖 Step 6 的 Order("id") 稳定排序——不要在此排序、不要依赖文档遍历序
			SourceProfileRefs:  "",
			CreatedAt:          time.Now(),
		}

		if err := uc.waveRepo.AddParticipant(ctx, &snap); err != nil {
			return count, err
		}
		count++
	}

	// If documents were assigned but all lacked CustomerProfileID, signal explicitly
	if count == 0 && skippedNoProfile > 0 {
		return 0, fmt.Errorf("all %d assigned demand documents lack a CustomerProfileID; cannot generate participant snapshots", skippedNoProfile)
	}

	return count, nil
}
```

- [ ] **Step 5: waived 统计单列 + DTO**

`internal/app/dto/demand.go` 的 `WaveRoutingStatsDTO`（:136-146）在 `AcceptedPartialCount` 后追加：

```go
	// WaivedCount counts accepted lines whose recipient input was waived.
	// Waived means "withhold — do not ship", so it is reported separately and
	// never folded into AcceptedReadyCount (spec §5.6).
	WaivedCount int `json:"waivedCount"`
```

`internal/app/entitlement_routing_usecase.go` 的 switch（:103-112）替换为：

```go
				switch line.RecipientInputState {
				case "ready", "not_required":
					stats.AcceptedReadyCount++
				case "waived":
					stats.WaivedCount++
				case "waiting_for_input":
					stats.AcceptedWaitingCount++
				case "partially_collected":
					stats.AcceptedPartialCount++
				default:
					stats.AcceptedReadyCount++
				}
```

- [ ] **Step 6: 三个 repo 查询加稳定排序**

`internal/infra/demand_assignment_repo.go` 的 `ListDemandDocumentsByWave`（:78-80、:92-94）两处查询各加 `Order("id")`：

```go
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Order("id").Find(&assignments).Error; err != nil {
```

```go
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Order("id").Find(&ps).Error; err != nil {
```

`internal/infra/demand_repo.go` 的 `ListLinesByDocument`（:122）改为：

```go
	if err := r.db.WithContext(ctx).Where("demand_document_id = ?", docID).Order("id").Find(&ps).Error; err != nil {
```

`internal/infra/wave_repo.go` 的 `ListParticipantsByWave`（:83）改为：

```go
	if err := r.db.WithContext(ctx).Where("wave_id = ?", waveID).Order("id").Find(&ps).Error; err != nil {
```

（排序目的：快照类型与 `SourceDocumentRefs` 的拼接序不再依赖 SQLite 插入序，`GenerateParticipants` 与 `GetWaveRoutingStats` 的输出可复现。）

- [ ] **Step 7: 跑测试确认通过**

Run: `gofmt -w internal/app/use_cases.go internal/app/entitlement_routing_usecase.go internal/app/dto/demand.go internal/app/use_cases_test.go internal/app/entitlement_routing_usecase_test.go internal/infra/demand_assignment_repo.go internal/infra/demand_repo.go internal/infra/wave_repo.go && go build ./... && go test -count=1 ./...`
Expected: 全绿（新增 3 用例 + 既有用例；`mockWaveRepo.AddParticipant` 落库化不影响既有用例——既有用例均走 `SetParticipants` 或断言 count）。

- [ ] **Step 8: Commit**

```bash
git add internal/app/use_cases.go internal/app/entitlement_routing_usecase.go internal/app/dto/demand.go internal/app/use_cases_test.go internal/app/entitlement_routing_usecase_test.go internal/infra/demand_assignment_repo.go internal/infra/demand_repo.go internal/infra/wave_repo.go
git commit -m "feat(wave): 混合快照聚合与 waived 口径统一——双身份合并 mixed 快照、快照只收 ready/not_required、统计单列 waived、查询稳定排序"
```

---

### Task 2: 引擎边界修复——reconcile 只扫权益行、零售映射解除参与者前置

**Files:**
- Modify: `internal/app/allocation_policy_usecase.go`（`ReconcileWave` 资格判定 kind 过滤，:92-107）
- Modify: `internal/app/use_cases.go`（`MapDemandToFulfillment` 移除参与者快照前置，:282-312、:331）
- Test: `internal/app/allocation_policy_usecase_test.go`（追加 1 用例；harness = 文件内 `newPolicyWaveRepo` 等 + 同包 `newMockDemandRepo`/`newMockAssignmentRepo`）
- Test: `internal/app/use_cases_test.go`（重写 `TestMapDemandToFulfillmentFailsOnPartialSnapshotMissing` :831-889 为通过语义；其余既有映射用例不动）

**Interfaces:**
- Consumes: `isEligibleForFulfillment`（同包）；`domain.DemandKindMembershipEntitlement`（`internal/domain/enums.go:27`）
- Produces:
  - `ReconcileWave` 资格判定：只把 membership_entitlement 文档的行计入 eligible（零售买家不再被会员规则白送礼品）
  - `MapDemandToFulfillment`：不再要求每个零售 profile 已有参与者快照；有快照则软链接 `WaveParticipantSnapshotID`，无快照则 `nil`；`CustomerProfileID` 缺失仍报错（履约行必须有 profile）

- [ ] **Step 1: 写失败测试（reconcile 资格）**

`internal/app/allocation_policy_usecase_test.go` 末尾追加（`uint_ptr` helper 同包 `replay_test.go` 已定义）：

```go
func TestReconcileWave_EligibilityIgnoresRetailDocuments(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()
	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)

	waveID := uint(1)
	waveRepo.participants[waveID] = []domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: waveID, CustomerProfileID: 100, IdentityPlatform: "bilibili", GiftLevel: "L1"},
		{ID: 2, WaveID: waveID, CustomerProfileID: 101, IdentityPlatform: "shop", GiftLevel: ""},
	}

	if err := ruleRepo.Create(context.Background(), &domain.AllocationPolicyRule{
		WaveID: waveID, ProductID: 10,
		SelectorPayload:      domain.SelectorPayload{Type: "wave_all"},
		ContributionQuantity: 3, RuleKind: "entitlement", Priority: 1, Active: true,
	}); err != nil {
		t.Fatalf("setup rule Create failed: %v", err)
	}

	// Profile 100: membership doc with an accepted+ready line -> eligible.
	// Profile 101: retail_order doc with an accepted+ready line -> must NOT become eligible.
	intakeUC := NewDemandIntakeUseCase(demandRepo)
	memberDoc := &domain.DemandDocument{Kind: "membership_entitlement", CustomerProfileID: uint_ptr(100)}
	retailDoc := &domain.DemandDocument{Kind: "retail_order", CustomerProfileID: uint_ptr(101)}
	if err := intakeUC.ImportDemand(context.Background(), memberDoc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", LineType: "entitlement_rule"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := intakeUC.ImportDemand(context.Background(), retailDoc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", LineType: "sku_order"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []*domain.DemandDocument{memberDoc, retailDoc} {
		if err := assignmentRepo.Create(context.Background(), &domain.WaveDemandAssignment{WaveID: waveID, DemandDocumentID: doc.ID}); err != nil {
			t.Fatalf("assign doc %d: %v", doc.ID, err)
		}
	}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, demandRepo, assignmentRepo, nil)
	result, err := uc.ReconcileWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("ReconcileWave failed: %v", err)
	}
	if result.Created != 1 {
		t.Fatalf("expected only the membership profile to receive a line, got %d created", result.Created)
	}
	lines, _ := fulfillRepo.ListByWave(context.Background(), waveID)
	if len(lines) != 1 {
		t.Fatalf("expected 1 persisted line, got %d", len(lines))
	}
	if lines[0].CustomerProfileID == nil || *lines[0].CustomerProfileID != 100 {
		t.Errorf("expected the line to belong to membership profile 100, got %+v", lines[0])
	}
}
```

- [ ] **Step 2: 写失败测试（映射解除前置）**

`internal/app/use_cases_test.go` 的 `TestMapDemandToFulfillmentFailsOnPartialSnapshotMissing`（:831-889）整体替换为（import 块补 `"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"`——本文件当前未导入）：

```go
func TestMapDemandToFulfillmentProceedsWithoutParticipantSnapshots(t *testing.T) {
	t.Parallel()

	demandRepo := newMockDemandRepo()
	fulfillRepo := newMockFulfillRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	waveRepo := newMockWaveRepo()

	profileA := uint(100)
	profileB := uint(200)
	// Only profileA has a snapshot; profileB does not — mapping must proceed anyway.
	waveRepo.SetParticipants([]domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: 1, CustomerProfileID: profileA, SnapshotType: "buyer"},
	})

	demandUC := NewDemandIntakeUseCase(demandRepo)
	docA := &domain.DemandDocument{
		Kind: "retail_order", CaptureMode: "manual_entry",
		SourceChannel: "test", SourceDocumentNo: "PARTIAL-A",
		CustomerProfileID: &profileA,
	}
	docB := &domain.DemandDocument{
		Kind: "retail_order", CaptureMode: "manual_entry",
		SourceChannel: "test", SourceDocumentNo: "PARTIAL-B",
		CustomerProfileID: &profileB,
	}
	if err := demandUC.ImportDemand(context.Background(), docA, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 1, LineType: "sku_order"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := demandUC.ImportDemand(context.Background(), docB, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "ready", RequestedQuantity: 1, LineType: "sku_order"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []*domain.DemandDocument{docA, docB} {
		if err := assignmentRepo.Create(context.Background(), &domain.WaveDemandAssignment{
			WaveID: 1, DemandDocumentID: doc.ID,
		}); err != nil {
			t.Fatal(err)
		}
	}

	dmUC := NewDemandMappingUseCase(demandRepo, fulfillRepo, assignmentRepo, waveRepo, nil, nil)
	dmResult, err := dmUC.MapDemandToFulfillment(context.Background(), 1)
	if err != nil {
		t.Fatalf("MapDemandToFulfillment must not require participant snapshots, got: %v", err)
	}
	if len(dmResult.CreatedLines) != 2 {
		t.Fatalf("expected 2 fulfillment lines, got %d", len(dmResult.CreatedLines))
	}
	byProfile := map[uint]dto.FulfillmentLineDTO{}
	for _, line := range dmResult.CreatedLines {
		byProfile[*line.CustomerProfileID] = line
	}
	if a := byProfile[profileA]; a.WaveParticipantSnapshotID == nil || *a.WaveParticipantSnapshotID != 1 {
		t.Errorf("profileA line should soft-link snapshot 1, got %v", a.WaveParticipantSnapshotID)
	}
	if b := byProfile[profileB]; b.WaveParticipantSnapshotID != nil {
		t.Errorf("profileB line has no snapshot yet — snapshot id must be nil, got %v", b.WaveParticipantSnapshotID)
	}
}
```

（`TestMapDemandToFulfillmentFailsOnMissingCustomerProfileID` :891-923 保持不变——profile 缺失仍然必须失败。）

- [ ] **Step 3: 跑测试确认失败**

Run: `go test -count=1 ./internal/app/ -run 'TestReconcileWave_EligibilityIgnoresRetailDocuments|TestMapDemandToFulfillmentProceedsWithoutParticipantSnapshots'`
Expected: `TestReconcileWave_...` 失败（现状零售 doc 使 profile 101 具备资格，Created=2）；`TestMapDemandToFulfillmentProceeds...` 失败（现状报 `run GenerateParticipants first`）。

- [ ] **Step 4: 实现 reconcile kind 过滤**

`internal/app/allocation_policy_usecase.go` 的资格扫描循环（:92-107）在 `if doc.CustomerProfileID == nil` 之后插入：

```go
			if doc.Kind != string(domain.DemandKindMembershipEntitlement) {
				// Rules engine serves the membership side only — a retail buyer must
				// never become eligible via membership rules (spec §5.4 引擎边界修复).
				continue
			}
```

- [ ] **Step 5: 实现映射解除前置**

`internal/app/use_cases.go` 的 `MapDemandToFulfillment` 预检块（:282-312）替换为：

```go
	// Pre-check: every retail_order with eligible lines must be associable to a profile.
	// Participant snapshots are NO LONGER required (spec §5.4): order lines flow straight
	// through; a snapshot is soft-linked when one exists for the profile, nil otherwise.
	var missingProfileDocs []uint
	for docIdx := range docs {
		doc := &docs[docIdx]
		if doc.Kind != "retail_order" {
			continue
		}
		hasEligible, err := uc.docHasEligibleLines(ctx, doc.ID)
		if err != nil {
			return nil, err
		}
		if !hasEligible {
			continue
		}
		if doc.CustomerProfileID == nil {
			missingProfileDocs = append(missingProfileDocs, doc.ID)
		}
	}
	if len(missingProfileDocs) > 0 {
		return nil, fmt.Errorf("retail demand documents %v have eligible lines but no CustomerProfileID; cannot generate fulfillment lines", missingProfileDocs)
	}
```

主循环（:331）的 `snapID := profileToSnapshot[*doc.CustomerProfileID]` 替换为软链接：

```go
		var snapID *uint
		if profileToSnapshot != nil {
			if id, ok := profileToSnapshot[*doc.CustomerProfileID]; ok {
				snapID = &id
			}
		}
```

**连带修改（同一步必须做，否则编译失败）**：主循环内 `snapID` 的消费者原来写作 `WaveParticipantSnapshotID: &snapID`（use_cases.go:381 附近）——`snapID` 变为 `*uint` 后 `&snapID` 是 `**uint`，必须同步改为：

```go
			WaveParticipantSnapshotID: snapID,
```

（其余消费者若存在同样写法一并改；grep `&snapID` 确认零残留。）

- [ ] **Step 6: 跑测试确认通过**

Run: `gofmt -w internal/app/allocation_policy_usecase.go internal/app/use_cases.go internal/app/allocation_policy_usecase_test.go internal/app/use_cases_test.go && go build ./... && go test -count=1 ./...`
Expected: 全绿（`TestMapDemandToFulfillmentDemandDriven` 既有断言 `WaveParticipantSnapshotID==1` 不受影响——profileA 有快照仍软链接）。

- [ ] **Step 7: Commit**

```bash
git add internal/app/allocation_policy_usecase.go internal/app/use_cases.go internal/app/allocation_policy_usecase_test.go internal/app/use_cases_test.go
git commit -m "fix(allocation): 引擎边界修复——reconcile 资格只扫会员权益行、零售映射解除参与者快照前置（订单行直入，快照软链接）"
```

---

### Task 3: 零售免分诊——导入自动裁决 + 路由校验 kind 感知

**Files:**
- Modify: `controller_demand.go`（新增 `applyRetailAutoAdjudication` helper；`ImportDemandDocument` :109-126 与 `ImportDemandFromCSV` :199-237 两处调用）
- Modify: `controller_demand_csv_import.go`（`ImportDemandCSV` 行组装后调用，:269-272）
- Modify: `internal/app/entitlement_routing_usecase.go`（`UpdateDemandLineRouting` 零售行豁免 input state，:48-64）
- Test: `controller_demand_csv_import_test.go`（追加 2 用例；harness = 文件内 `setupDemandCSVImportTestDB`/`newDemandCSVImportTestController`/`writeTempCSV`，零售 profile 构造照抄 :363-406 的既有零售用例）
- Test: `internal/app/entitlement_routing_usecase_test.go`（追加 1 用例）

**Interfaces:**
- Consumes: `domain.DemandKindRetailOrder` / `RoutingDispositionAccepted` / `RecipientInputStateNotRequired`（`internal/domain/enums.go`）；`DemandDocumentRepository.FindByID`（同包接口已有，`assignOne` 已用）
- Produces:
  - `applyRetailAutoAdjudication(kind string, lines []*domain.DemandLine)`（package main，Task 3 定义）
  - `UpdateDemandLineRouting` 新语义：零售行允许 `RecipientInputState` 为空（保留现值）；非空值仍须合法（会员行行为完全不变）

- [ ] **Step 1: 写失败测试（导入自动裁决）**

`controller_demand_csv_import_test.go` 末尾追加：

```go
func TestImportDemandCSV_RetailLinesAutoAccepted(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	ctx := appContext

	profileRepo := infra.NewIntegrationProfileRepository(gdb)
	profile := &domain.IntegrationProfile{
		ProfileKey:              "auto-accept-retail",
		SourceChannel:           "bilibili",
		SourceSurface:           string(domain.SourceSurfaceRetail),
		DemandKind:              string(domain.DemandKindRetailOrder),
		IdentityStrategy:        app.IdentityStrategyOrderScopedProvisional,
		RequiresExternalOrderNo: true,
	}
	if err := profileRepo.Create(ctx, profile); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	const rules = `{
		"version":2,
		"mode":"header",
		"hasHeader":true,
		"columns":{
			"document.source_document_no":"Order",
			"document.source_customer_ref":"Buyer",
			"line.external_title":"Item",
			"line.requested_quantity":"Qty"
		},
		"defaults":{"line.line_type":"sku_order"},
		"required":["document.source_document_no"]
	}`
	templateRepo := infra.NewDocumentTemplateRepository(gdb)
	tmpl := &domain.DocumentTemplate{
		TemplateKey: "auto-accept-retail-template", DocumentType: "import_sales_order", Format: "xls", MappingRules: rules,
	}
	if err := templateRepo.Create(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	bindingRepo := infra.NewProfileTemplateBindingRepository(gdb)
	if err := bindingRepo.Create(ctx, &domain.IntegrationProfileTemplateBinding{
		IntegrationProfileID: profile.ID, DocumentType: "import_sales_order", TemplateID: tmpl.ID, IsDefault: true,
	}); err != nil {
		t.Fatalf("create binding: %v", err)
	}

	result, err := c.ImportDemandCSV(dto.ImportDemandCSVInput{
		IntegrationProfileID: profile.ID,
		ImportMode:           "reject_all",
		Rows: []map[string]string{
			{"Order": "ORDER-A", "Buyer": "buyer-a", "Item": "Standee", "Qty": "1"},
			{"Order": "ORDER-A", "Buyer": "buyer-a", "Item": "Badge", "Qty": "2"},
		},
	})
	if err != nil {
		t.Fatalf("ImportDemandCSV: %v", err)
	}
	if result.SuccessCount != 2 || result.Document == nil {
		t.Fatalf("unexpected result: %+v", result)
	}

	lines, err := c.demandRepo.ListLinesByDocument(ctx, result.Document.ID)
	if err != nil {
		t.Fatalf("ListLinesByDocument: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if line.RoutingDisposition != string(domain.RoutingDispositionAccepted) {
			t.Errorf("line %d RoutingDisposition = %q, want accepted (retail auto-adjudication)", i, line.RoutingDisposition)
		}
		if line.RecipientInputState != string(domain.RecipientInputStateNotRequired) {
			t.Errorf("line %d RecipientInputState = %q, want not_required", i, line.RecipientInputState)
		}
	}
}

func TestImportDemandDocument_RetailLinesAutoAccepted(t *testing.T) {
	gdb := setupDemandCSVImportTestDB(t)
	c := newDemandCSVImportTestController(gdb)
	ctx := appContext

	docDTO, err := c.ImportDemandDocument(dto.CreateDemandInput{
		Kind: "retail_order", CaptureMode: "manual_entry", SourceChannel: "shop", SourceSurface: "retail",
		SourceDocumentNo: "ORDER-M-1", SourceCustomerRef: "buyer-x",
		Lines: []dto.CreateDemandLineInput{
			{LineType: "sku_order", RoutingDisposition: "pending_intake", RequestedQuantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("ImportDemandDocument: %v", err)
	}

	lines, err := c.demandRepo.ListLinesByDocument(ctx, docDTO.ID)
	if err != nil {
		t.Fatalf("ListLinesByDocument: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].RoutingDisposition != string(domain.RoutingDispositionAccepted) {
		t.Errorf("RoutingDisposition = %q, want accepted", lines[0].RoutingDisposition)
	}
	if lines[0].RecipientInputState != string(domain.RecipientInputStateNotRequired) {
		t.Errorf("RecipientInputState = %q, want not_required", lines[0].RecipientInputState)
	}
}
```

- [ ] **Step 2: 写失败测试（零售行路由校验豁免）**

`internal/app/entitlement_routing_usecase_test.go` 追加（import 块补 `"github.com/SodaTeaaaaee/EliGiftManager/internal/app/dto"`）：

```go
func TestUpdateDemandLineRouting_RetailLinesSkipInputStateRequirement(t *testing.T) {
	ctx := context.Background()
	demandRepo := newMockDemandRepo()
	assignmentRepo := newMockAssignmentRepo(demandRepo)
	uc := NewEntitlementRoutingUseCase(demandRepo, assignmentRepo)

	intakeUC := NewDemandIntakeUseCase(demandRepo)
	retailDoc := &domain.DemandDocument{Kind: "retail_order", SourceDocumentNo: "R-1"}
	if err := intakeUC.ImportDemand(ctx, retailDoc, []*domain.DemandLine{
		{RoutingDisposition: "accepted", RecipientInputState: "not_required", LineType: "sku_order"},
	}); err != nil {
		t.Fatalf("import retail doc: %v", err)
	}
	lines, err := demandRepo.ListLinesByDocument(ctx, retailDoc.ID)
	if err != nil || len(lines) != 1 {
		t.Fatalf("list retail lines: %v (%d)", err, len(lines))
	}
	line := lines[0]

	// 空 input state -> 保留现值，不报错
	if err := uc.UpdateDemandLineRouting(ctx, dto.UpdateDemandLineRoutingInput{
		DemandLineID: line.ID, RoutingDisposition: "deferred", RecipientInputState: "",
	}); err != nil {
		t.Fatalf("retail line must accept empty recipient input state, got: %v", err)
	}
	updated, err := demandRepo.FindLineByID(ctx, line.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.RecipientInputState != "not_required" {
		t.Errorf("empty input state must keep the existing value, got %q", updated.RecipientInputState)
	}
	if updated.RoutingDisposition != "deferred" {
		t.Errorf("RoutingDisposition = %q, want deferred", updated.RoutingDisposition)
	}

	// 会员行仍要求合法 input state
	memberDoc := &domain.DemandDocument{Kind: "membership_entitlement", SourceDocumentNo: "M-1"}
	if err := intakeUC.ImportDemand(ctx, memberDoc, []*domain.DemandLine{
		{RoutingDisposition: "pending_intake", RecipientInputState: "waiting_for_input", LineType: "entitlement_rule"},
	}); err != nil {
		t.Fatalf("import member doc: %v", err)
	}
	memberLines, err := demandRepo.ListLinesByDocument(ctx, memberDoc.ID)
	if err != nil || len(memberLines) != 1 {
		t.Fatalf("list member lines: %v (%d)", err, len(memberLines))
	}
	if err := uc.UpdateDemandLineRouting(ctx, dto.UpdateDemandLineRoutingInput{
		DemandLineID: memberLines[0].ID, RoutingDisposition: "accepted", RecipientInputState: "bogus_state",
	}); err == nil {
		t.Fatal("membership line must still reject an invalid recipient input state")
	}
}
```

- [ ] **Step 3: 跑测试确认失败**

Run: `go test -count=1 . ./internal/app/ -run 'TestImportDemandCSV_RetailLinesAutoAccepted|TestImportDemandDocument_RetailLinesAutoAccepted|TestUpdateDemandLineRouting_RetailLinesSkipInputStateRequirement'`
Expected: 两个导入用例失败（现状零售行保持模板映射值，多为空 disposition）；路由校验用例失败（空 input state 被 `validRecipientInputStates` 拒绝）。

- [ ] **Step 4: 实现导入自动裁决**

`controller_demand.go` 在 `domainLineSliceToPtrs` 附近追加 helper：

```go
// applyRetailAutoAdjudication forces retail_order lines to accepted + not_required:
// retail orders bypass inbox triage entirely (spec §5.2 零售免分诊). Membership
// lines pass through untouched — their triage stays a manual workflow.
func applyRetailAutoAdjudication(kind string, lines []*domain.DemandLine) {
	if kind != string(domain.DemandKindRetailOrder) {
		return
	}
	for _, l := range lines {
		if l == nil {
			continue
		}
		l.RoutingDisposition = string(domain.RoutingDispositionAccepted)
		l.RecipientInputState = string(domain.RecipientInputStateNotRequired)
	}
}
```

三个调用点：

1. `ImportDemandDocument`（:109-126）行组装循环之后、事务之前：

```go
	applyRetailAutoAdjudication(effectiveKind, lines)
```

2. `ImportDemandFromCSV`（:199-206）`mappedLines` 就绪后、事务内 `ImportDemand` 调用之前：

```go
		applyRetailAutoAdjudication(profile.DemandKind, mappedLines)
```

3. `controller_demand_csv_import.go` 的 `ImportDemandCSV`（:269-272）每组建行之后：

```go
			applyRetailAutoAdjudication(profile.DemandKind, lines)
```

- [ ] **Step 5: 实现路由校验 kind 感知**

`internal/app/entitlement_routing_usecase.go` 的 `UpdateDemandLineRouting`（:48-64）整体替换为：

```go
// UpdateDemandLineRouting validates and applies routing field updates to a single demand line.
// Retail lines carry no recipient-input semantics — the simplified three-state retail
// adjudication (accepted/deferred/excluded) leaves recipient_input_state untouched, so an
// empty input state is accepted for retail documents and the existing value is preserved.
func (uc *entitlementRoutingUseCase) UpdateDemandLineRouting(ctx context.Context, input dto.UpdateDemandLineRoutingInput) error {
	line, err := uc.demandRepo.FindLineByID(ctx, input.DemandLineID)
	if err != nil {
		return fmt.Errorf("demand line %d not found: %w", input.DemandLineID, err)
	}
	if !validRoutingDispositions[input.RoutingDisposition] {
		return fmt.Errorf("invalid routing_disposition %q: must be one of pending_intake, accepted, deferred, excluded_manual, excluded_duplicate, excluded_revoked", input.RoutingDisposition)
	}

	isRetail := false
	if doc, docErr := uc.demandRepo.FindByID(ctx, line.DemandDocumentID); docErr == nil && doc != nil {
		isRetail = doc.Kind == string(domain.DemandKindRetailOrder)
	}

	recipientInputState := input.RecipientInputState
	if recipientInputState != "" && !validRecipientInputStates[recipientInputState] {
		return fmt.Errorf("invalid recipient_input_state %q: must be one of not_required, waiting_for_input, partially_collected, ready, waived, expired", input.RecipientInputState)
	}
	if isRetail && recipientInputState == "" {
		recipientInputState = line.RecipientInputState
	}

	return uc.demandRepo.UpdateLineRoutingFields(ctx,
		input.DemandLineID,
		input.RoutingDisposition,
		recipientInputState,
		input.RoutingReasonCode,
	)
}
```

- [ ] **Step 6: 跑测试确认通过**

Run: `gofmt -w controller_demand.go controller_demand_csv_import.go internal/app/entitlement_routing_usecase.go controller_demand_csv_import_test.go internal/app/entitlement_routing_usecase_test.go && go build ./... && go test -count=1 ./...`
Expected: 全绿（既有 CSV 导入用例对 membership profile 行为不变——helper 对非零售 kind 是 no-op）。

- [ ] **Step 7: Commit**

```bash
git add controller_demand.go controller_demand_csv_import.go internal/app/entitlement_routing_usecase.go controller_demand_csv_import_test.go internal/app/entitlement_routing_usecase_test.go
git commit -m "feat(demand): 零售免分诊——三个导入入口对 retail_order 行自动置 accepted+not_required，路由校验对零售行豁免输入态要求"
```

---

### Task 4: 调整来源三值补齐——reissue/compensation 重放产生 wave_adjustment 新行

**Files:**
- Modify: `internal/app/replay.go`（`ReplayAdjustments` 增加新行分支 + `createsNewLine`/`spawnWaveAdjustmentLine`，:38-83、:208-222）
- Modify: `internal/app/allocation_policy_usecase.go`（re-anchor 匹配排除 wave_adjustment 行，:265-276）
- Test: `internal/app/replay_test.go`（追加 2 用例；harness = 文件内 `uint_ptr`）
- Test: `internal/app/allocation_policy_usecase_test.go`（追加 1 用例）

**Interfaces:**
- Consumes: `domain.AdjustmentKindReissue` / `AdjustmentKindCompensation` / `LineReasonWaveAdjustment`（`internal/domain/enums.go`）；`ReplayOptions`（同包）
- Produces:
  - `ReplayAdjustments` 新语义：participant-target 的 reissue/compensation 不再原地加数量，而是产生一条独立新行——克隆匹配基线（同 participant/product/address 等），`Quantity=delta`、`LineReason="wave_adjustment"`、`ID=0`、`GeneratedBy` 继承基线（reconcile 重建时经重放确定性再生）；delta<=0 不产生行；目标解析仍只对基线行单匹配（连续多条 reissue 不互相干扰）
  - reconcile re-anchor 匹配规则：`LineReason=="wave_adjustment"` 的行不参与 (participant, product) 匹配（避免新行使 matchCount 变成 2 导致重锚失效）

- [ ] **Step 1: 写失败测试（replay 层）**

`internal/app/replay_test.go` 末尾追加：

```go
func TestReplayAdjustments_ParticipantReissueSpawnsWaveAdjustmentLine(t *testing.T) {
	productID := uint(9)
	baselines := []domain.FulfillmentLine{
		{ID: 1, WaveParticipantSnapshotID: uint_ptr(300), ProductID: &productID, Quantity: 10, LineReason: "entitlement", GeneratedBy: "allocation_policy_driven"},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                        70,
			TargetKind:                "participant",
			WaveParticipantSnapshotID: uint_ptr(300),
			AdjustmentKind:            string(domain.AdjustmentKindReissue),
			QuantityDelta:             2,
		},
	}

	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
	if len(result) != 2 {
		t.Fatalf("expected baseline + spawned line = 2, got %d", len(result))
	}
	// Baseline untouched.
	if result[0].Quantity != 10 || result[0].LineReason != "entitlement" {
		t.Fatalf("baseline must stay untouched, got %+v", result[0])
	}
	spawned := result[1]
	if spawned.ID != 0 {
		t.Errorf("spawned line must have ID 0 (assigned at persist), got %d", spawned.ID)
	}
	if spawned.Quantity != 2 {
		t.Errorf("spawned line Quantity = %d, want 2", spawned.Quantity)
	}
	if spawned.LineReason != string(domain.LineReasonWaveAdjustment) {
		t.Errorf("spawned line LineReason = %q, want wave_adjustment", spawned.LineReason)
	}
	if spawned.GeneratedBy != "allocation_policy_driven" {
		t.Errorf("spawned line GeneratedBy = %q, want allocation_policy_driven (inherited)", spawned.GeneratedBy)
	}
	if spawned.WaveParticipantSnapshotID == nil || *spawned.WaveParticipantSnapshotID != 300 {
		t.Errorf("spawned line must carry the participant, got %v", spawned.WaveParticipantSnapshotID)
	}
	if spawned.ProductID == nil || *spawned.ProductID != productID {
		t.Errorf("spawned line must inherit the product, got %v", spawned.ProductID)
	}
}

func TestReplayAdjustments_ParticipantCompensationZeroDeltaSpawnsNoLine(t *testing.T) {
	baselines := []domain.FulfillmentLine{
		{ID: 1, WaveParticipantSnapshotID: uint_ptr(300), Quantity: 10},
	}
	adjustments := []domain.FulfillmentAdjustment{
		{
			ID:                        71,
			TargetKind:                "participant",
			WaveParticipantSnapshotID: uint_ptr(300),
			AdjustmentKind:            string(domain.AdjustmentKindCompensation),
			QuantityDelta:             -2,
		},
	}

	result, failures := ReplayAdjustments(baselines, adjustments)
	if len(failures) != 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
	if len(result) != 1 || result[0].Quantity != 10 {
		t.Fatalf("non-positive delta must not spawn a line nor touch the baseline, got %+v", result)
	}
}
```

- [ ] **Step 2: 写失败测试（reconcile 级）**

`internal/app/allocation_policy_usecase_test.go` 追加：

```go
func TestReconcileWave_ReissueSpawnsWaveAdjustmentLine(t *testing.T) {
	t.Parallel()

	waveRepo := newPolicyWaveRepo()
	ruleRepo := newPolicyRuleRepo()
	fulfillRepo := newMockFulfillRepo()
	adjRepo := newPolicyAdjRepo()

	waveID := uint(1)
	waveRepo.participants[waveID] = []domain.WaveParticipantSnapshot{
		{ID: 1, WaveID: waveID, CustomerProfileID: 100, IdentityPlatform: "bilibili", GiftLevel: "L1"},
	}

	if err := ruleRepo.Create(context.Background(), &domain.AllocationPolicyRule{
		WaveID: waveID, ProductID: 10,
		SelectorPayload:      domain.SelectorPayload{Type: "wave_all"},
		ContributionQuantity: 3, RuleKind: "entitlement", Priority: 1, Active: true,
	}); err != nil {
		t.Fatalf("setup rule Create failed: %v", err)
	}
	if err := adjRepo.Create(context.Background(), &domain.FulfillmentAdjustment{
		WaveID: waveID, TargetKind: "participant", WaveParticipantSnapshotID: uint_ptr(1),
		AdjustmentKind: string(domain.AdjustmentKindReissue), QuantityDelta: 2,
	}); err != nil {
		t.Fatalf("setup adjustment Create failed: %v", err)
	}

	uc := NewAllocationPolicyUseCase(ruleRepo, fulfillRepo, waveRepo, adjRepo, nil, nil, nil)
	result, err := uc.ReconcileWave(context.Background(), waveID)
	if err != nil {
		t.Fatalf("ReconcileWave failed: %v", err)
	}
	if result.Created != 2 {
		t.Fatalf("expected entitlement line + reissue line = 2 created, got %d", result.Created)
	}

	lines, _ := fulfillRepo.ListByWave(context.Background(), waveID)
	var adjustmentCount, entitlementCount int
	for _, line := range lines {
		switch line.LineReason {
		case string(domain.LineReasonWaveAdjustment):
			adjustmentCount++
			if line.Quantity != 2 {
				t.Errorf("wave_adjustment line Quantity = %d, want 2", line.Quantity)
			}
		case "entitlement":
			entitlementCount++
		}
	}
	if adjustmentCount != 1 || entitlementCount != 1 {
		t.Fatalf("want 1 entitlement + 1 wave_adjustment line, got %d/%d", entitlementCount, adjustmentCount)
	}
}
```

- [ ] **Step 3: 跑测试确认失败**

Run: `go test -count=1 ./internal/app/ -run 'TestReplayAdjustments_ParticipantReissueSpawnsWaveAdjustmentLine|TestReplayAdjustments_ParticipantCompensationZeroDeltaSpawnsNoLine|TestReconcileWave_ReissueSpawnsWaveAdjustmentLine'`
Expected: 三个用例失败（现状 reissue 走 `applyAdjustment` 原地 +2，无新行）。

- [ ] **Step 4: 实现 replay 新行分支**

`internal/app/replay.go` 的 `ReplayAdjustments`（:38-83）替换为：

```go
// ReplayAdjustments applies a chronologically-ordered slice of adjustments onto
// the given baselines.
//
// Participant-target reissue/compensation do NOT mutate the matched baseline:
// they spawn a SEPARATE line with LineReason=wave_adjustment (spec §5.5 来源三值
// 补齐) so the manual re-send stays visible as its own source. Spawned lines are
// collected in extraLines and appended to the result AFTER resolution, so they
// never enter the target-resolution universe (successive reissues stay
// single-match against the entitlement baselines).
//
// Caller guarantees: adjustments are sorted by CreatedAt ascending.
func ReplayAdjustments(
	baselines []domain.FulfillmentLine,
	adjustments []domain.FulfillmentAdjustment,
	opts ...ReplayOptions,
) ([]domain.FulfillmentLine, []ReplayFailure) {
	if len(adjustments) == 0 {
		return baselines, nil
	}

	var opt ReplayOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Build index: line ID → slice index for O(1) lookup.
	idIndex := make(map[uint]int, len(baselines))
	for i := range baselines {
		idIndex[baselines[i].ID] = i
	}

	var failures []ReplayFailure
	var extraLines []domain.FulfillmentLine

	for _, adj := range adjustments {
		idx, ok := resolveTarget(baselines, idIndex, adj, opt.LineHints)
		if !ok {
			failures = append(failures, ReplayFailure{
				AdjustmentID: adj.ID,
				Reason:       resolveFailureReason(baselines, adj, opt.LineHints),
			})
			if opt.Mode == ReplayHaltOnFirstFailure {
				break
			}
			continue
		}
		if createsNewLine(adj) {
			if extra := spawnWaveAdjustmentLine(&baselines[idx], adj); extra != nil {
				extraLines = append(extraLines, *extra)
			}
			continue
		}
		applyAdjustment(&baselines[idx], adj)
	}

	// Final clamp: quantity must be >= 0.
	for i := range baselines {
		if baselines[i].Quantity < 0 {
			baselines[i].Quantity = 0
		}
	}
	kept := extraLines[:0]
	for i := range extraLines {
		if extraLines[i].Quantity > 0 {
			kept = append(kept, extraLines[i])
		}
	}

	return append(baselines, kept...), failures
}

// createsNewLine reports whether the adjustment spawns a separate wave_adjustment
// line instead of mutating the matched baseline. Only participant-target
// reissue/compensation spawn lines — production validation (adjustment_usecase.go)
// guarantees these two kinds are never recorded with a fulfillment_line target.
func createsNewLine(adj domain.FulfillmentAdjustment) bool {
	if adj.TargetKind != "participant" {
		return false
	}
	return adj.AdjustmentKind == string(domain.AdjustmentKindReissue) ||
		adj.AdjustmentKind == string(domain.AdjustmentKindCompensation)
}

// spawnWaveAdjustmentLine clones the matched baseline into a fresh re-send line
// carrying the adjustment's quantity delta and LineReason=wave_adjustment. The
// baseline's GeneratedBy namespace is preserved so the next reconcile rebuilds the
// line deterministically via replay (reconcile only replaces policy_driven lines).
func spawnWaveAdjustmentLine(base *domain.FulfillmentLine, adj domain.FulfillmentAdjustment) *domain.FulfillmentLine {
	if base == nil || adj.QuantityDelta <= 0 {
		return nil
	}
	clone := *base
	clone.ID = 0
	clone.Quantity = adj.QuantityDelta
	clone.LineReason = string(domain.LineReasonWaveAdjustment)
	return &clone
}
```

- [ ] **Step 5: re-anchor 排除 wave_adjustment 行**

`internal/app/allocation_policy_usecase.go` 的重锚匹配循环（:265-276）在 `if line.GeneratedBy != "allocation_policy_driven"` 之后插入：

```go
				if line.LineReason == string(domain.LineReasonWaveAdjustment) {
					// Adjustment-spawned lines share the (participant, product) identity of
					// their entitlement base — never let them shadow the re-anchor match.
					continue
				}
```

- [ ] **Step 6: 跑测试确认通过**

Run: `gofmt -w internal/app/replay.go internal/app/allocation_policy_usecase.go internal/app/replay_test.go internal/app/allocation_policy_usecase_test.go && go build ./... && go test -count=1 ./...`
Expected: 全绿（既有 `TestReplayAdjustments_AddReduceCompensation` 用 fulfillment_line-target 的 compensation，仍走原地 delta；`TestReplayAdjustments_ParticipantTarget_SingleMatch` 用 kind=add，不受影响）。

- [ ] **Step 7: Commit**

```bash
git add internal/app/replay.go internal/app/allocation_policy_usecase.go internal/app/replay_test.go internal/app/allocation_policy_usecase_test.go
git commit -m "feat(allocation): 调整来源三值补齐——reissue/compensation 重放产生独立 wave_adjustment 行，re-anchor 匹配排除调整行"
```

---

### Task 5: 履约网格 lineReasons 后端过滤维度

**Files:**
- Modify: `internal/app/dto/wave_fulfillment_filter.go`（`WaveFulfillmentFilterInput` +lineReasons，:9-19）
- Modify: `internal/app/wave_fulfillment_filter_usecase.go`（过滤循环 + 维度判断，:74-98）
- Test: `internal/app/wave_lifecycle_usecase_test.go`（扩展 `TestListWaveFulfillmentRowsFilteredReturnsExpectedSubset`，:250-325）

**Interfaces:**
- Consumes: `dto.WaveFulfillmentRowDTO.LineReason`（`internal/app/dto/workspace.go:57` 已有）
- Produces: `dto.WaveFulfillmentFilterInput.LineReasons []string`（Task 5 定义 → Task 6 绑定 → Task 8 消费；空切片 = 不过滤，与其余维度同约定）

- [ ] **Step 1: 写失败测试**

`internal/app/wave_lifecycle_usecase_test.go` 的 `TestListWaveFulfillmentRowsFilteredReturnsExpectedSubset`：在 seed 两条行之后追加第三条（与既有 seed 行同构——不带 ProductID，避免行组装触发商品查找路径；仅 `LineReason` 不同）：

```go
	if err := fulfillRepo.Create(ctx, &domain.FulfillmentLine{
		WaveID:           wave.ID,
		Quantity:         2,
		AllocationState:  string(domain.AllocationStateReady),
		AddressState:     string(domain.AddressStateReady),
		SupplierState:    string(domain.SupplierStateNotSubmitted),
		ChannelSyncState: string(domain.ChannelSyncStateNotRequired),
		LineReason:       string(domain.LineReasonWaveAdjustment),
	}); err != nil {
		t.Fatalf("seed wave_adjustment line: %v", err)
	}
```

在既有「unfiltered 断言」之后追加：

```go
	// lineReasons dimension: only the wave_adjustment line matches.
	reasonPage, err := filterUC.ListWaveFulfillmentRowsFiltered(ctx, dto.WaveFulfillmentFilterInput{
		WaveID:      wave.ID,
		LineReasons: []string{string(domain.LineReasonWaveAdjustment)},
		Pagination:  dto.PaginationInput{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatalf("ListWaveFulfillmentRowsFiltered (lineReasons): %v", err)
	}
	if reasonPage.Pagination.TotalCount != 1 || len(reasonPage.Items) != 1 ||
		reasonPage.Items[0].LineReason != string(domain.LineReasonWaveAdjustment) {
		t.Fatalf("lineReasons filter: want only the wave_adjustment line, got %+v", reasonPage)
	}
```

并把上面的 unfiltered 断言 `TotalCount != 2` 改为 `!= 3`（注释同步：unfiltered 应见三条）。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -count=1 ./internal/app/ -run TestListWaveFulfillmentRowsFilteredReturnsExpectedSubset`
Expected: 编译失败（`LineReasons` 未定义）。

- [ ] **Step 3: 实现 DTO + 过滤**

`internal/app/dto/wave_fulfillment_filter.go` 的 `WaveFulfillmentFilterInput` 在 `DriftStatuses` 后追加：

```go
	// LineReasons narrows to rows whose line_reason is one of the given values
	// (entitlement / retail_order / wave_adjustment). Backs the "已调整" saved view
	// as an EXACT predicate, replacing the former reviewRequirement approximation
	// (spec §5.5).
	LineReasons []string `json:"lineReasons"`
```

`internal/app/wave_fulfillment_filter_usecase.go` 的过滤循环（:88-91 之后）插入：

```go
		if !waveFilterStringInSet(row.LineReason, input.LineReasons) {
			continue
		}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `gofmt -w internal/app/dto/wave_fulfillment_filter.go internal/app/wave_fulfillment_filter_usecase.go internal/app/wave_lifecycle_usecase_test.go && go build ./... && go test -count=1 ./...`
Expected: 全绿。

- [ ] **Step 5: Commit**

```bash
git add internal/app/dto/wave_fulfillment_filter.go internal/app/wave_fulfillment_filter_usecase.go internal/app/wave_lifecycle_usecase_test.go
git commit -m "feat(filter): 履约网格 WaveFulfillmentFilterInput 增加 lineReasons 过滤维度（wave_adjustment 精确命中）"
```

---

### Task 6: Wails 绑定 + bridge 契约同步

**Files:**
- Modify: `frontend/wailsjs/go/models.ts`（`WaveFulfillmentFilterInput` +lineReasons，:5654-5680；`WaveRoutingStatsDTO` +waivedCount，:5946-5973）
- Modify: `frontend/src/shared/api/bridge.ts`（`listWaveFulfillmentRowsFiltered` 入参扩展，:459-474）
- Modify: `frontend/src/pages/waves/workspace/tabs/fulfillment-grid/useFulfillmentGrid.ts`（`buildFilterInput` 临时透传 `lineReasons: []`——typecheck 硬性要求，正式落点见 Task 8 Step 2，:134-146）

**Interfaces:**
- Consumes: Task 1 的 `waivedCount`、Task 5 的 `lineReasons`（Go DTO 形状）
- Produces（Task 8 消费）:
  - `bridge.listWaveFulfillmentRowsFiltered(input: { waveId; allocationStates; addressStates; supplierStates; channelSyncStates; reviewRequirements; driftStatuses; lineReasons: string[]; keyword; pagination })`
  - `dto.WaveFulfillmentFilterInput.createFrom` 携带 `lineReasons`；`dto.WaveRoutingStatsDTO` 携带 `waivedCount`

- [ ] **Step 1: 手改 wailsjs 绑定**

`frontend/wailsjs/go/models.ts`（仓库惯例手维护，风格照抄同文件既有类）：

`WaveFulfillmentFilterInput` 类字段区与构造函数各补一行：

```ts
    driftStatuses: string[];
    lineReasons: string[];
    keyword: string;
```

```ts
        this.driftStatuses = source["driftStatuses"];
        this.lineReasons = source["lineReasons"];
        this.keyword = source["keyword"];
```

`WaveRoutingStatsDTO` 类字段区与构造函数各补一行：

```ts
    acceptedPartialCount: number;
    waivedCount: number;
    deferredCount: number;
```

```ts
        this.acceptedPartialCount = source["acceptedPartialCount"];
        this.waivedCount = source["waivedCount"];
        this.deferredCount = source["deferredCount"];
```

- [ ] **Step 2: bridge 扩展**

`frontend/src/shared/api/bridge.ts` 的 `listWaveFulfillmentRowsFiltered`（:459-474）：

```ts
/** Server-side filtered + paginated fulfillment-line grid rows for a wave. */
export async function listWaveFulfillmentRowsFiltered(input: {
  waveId: number;
  allocationStates: string[];
  addressStates: string[];
  supplierStates: string[];
  channelSyncStates: string[];
  reviewRequirements: string[];
  driftStatuses: string[];
  lineReasons: string[];
  keyword: string;
  pagination: PaginationInput;
}): Promise<dto.WaveFulfillmentRowsPage> {
  assertWailsRuntime();
  return ListWaveFulfillmentRowsFiltered(
    dto.WaveFulfillmentFilterInput.createFrom(input),
  );
}
```

（`createFrom` 直通对象，构造函数按 `source["lineReasons"]` 取值；无运行时降级行为与既有硬失败约定保持一致。）

- [ ] **Step 3: useFulfillmentGrid 临时透传（typecheck 门）**

`frontend/src/pages/waves/workspace/tabs/fulfillment-grid/useFulfillmentGrid.ts` 的 `buildFilterInput`（:134-146）补：

```ts
      reviewRequirements: filters.state.reviewRequirement,
      driftStatuses: filters.state.driftStatus,
      lineReasons: [], // 临时空透传——schema 的 lineReason 维度在节 3 Task 8 落地后替换为 filters.state.lineReason
      keyword: filters.state.keyword,
```

（这是硬性要求而非可选：bridge 入参类型收紧后，`buildFilterInput` 的返回对象缺少 `lineReasons` 会直接 TS2322。）

Run: `cd frontend && deno task typecheck`
Expected: exit 0。

- [ ] **Step 4: Commit**

```bash
git add frontend/wailsjs/go/models.ts frontend/src/shared/api/bridge.ts frontend/src/pages/waves/workspace/tabs/fulfillment-grid/useFulfillmentGrid.ts
git commit -m "feat(frontend): 同步 lineReasons 过滤维度与 waivedCount 统计字段的 Wails 绑定与 bridge 契约"
```

---

### Task 7: 分配页签 waveType 软生效 + intake 业务面预筛联动 + 分派 kind×type 提示

**Files:**
- Create: `frontend/src/pages/waves/workspace/tabs/allocation/engineVisibility.ts`（纯函数模块，可测）
- Create: `frontend/src/pages/waves/workspace/tabs/allocation/engineVisibility.test.ts`
- Modify: `frontend/src/pages/waves/workspace/tabs/WaveAllocationTab.vue`（两引擎改折叠区块，按 waveType 默认展开；映射按钮解除参与者门禁）
- Modify: `frontend/src/pages/waves/workspace/tabs/WaveIntakeTab.vue`（业务面预筛与 waveType 联动——依赖节 2 Task 6/7 已落地该页的 surface segmented 与 `businessSurface.ts`）
- Modify: `frontend/src/pages/inbox/inbox-grid/BatchActionBar.vue`（kind×type 不匹配提示，不拦截）
- Modify: `frontend/src/shared/i18n/locales/zh-CN.ts` / `en-US.ts`（新键，两 locale 同落）

**Interfaces:**
- Consumes: `ctx.snapshot.value?.wave.waveType`（`WaveWorkspaceSnapshotDTO.wave`，`internal/app/dto/workspace.go:26`）；节 2 的 `businessSurface.ts`（`surfaceFromKinds`/`kindsFromSurface`/`BusinessSurface`）；`dto.WaveDTO.waveType`
- Produces:
  - `WaveAllocationTab`：`expandedNames: Ref<string[]>`（'rules' | 'mapping' 折叠区块，waveType 决定默认集；membership→['rules']、retail→['mapping']、mixed→两者）
  - `WaveIntakeTab`：waveType 联动业务面预筛（未手动选过业务面时，membership→membership_entitlement、retail→retail_order、mixed→all）
  - `BatchActionBar`：`mismatchHint` computed（仅提示不拦截）
  - i18n 键：`allocation.engineHint.membership/retail/mixed`、`inbox.batch.kindTypeMismatchHint`（waveType 文案复用既有 `waves.waveType.*`，`zh-CN.ts:740`）

- [ ] **Step 1: 写失败测试（软生效纯逻辑）**

软生效的默认展开与提示判定抽成纯函数模块，新建 `frontend/src/pages/waves/workspace/tabs/allocation/engineVisibility.ts` 与测试：

```ts
// frontend/src/pages/waves/workspace/tabs/allocation/engineVisibility.ts
/** 波次类型软生效（spec §5.1）：返回默认展开的引擎区块名集合。 */
export function defaultExpandedEngines(waveType: string): string[] {
  if (waveType === 'membership') return ['rules']
  if (waveType === 'retail') return ['mapping']
  return ['rules', 'mapping']
}

/**
 * kind×type 不匹配检测（仅提示不拦截）：单一类型波次 vs 另一业务面的文档 kind。
 * mixed 波次永不提示；空 waveType 按 mixed 处理（老数据安全）。
 */
export function kindTypeMismatch(waveType: string, kinds: readonly string[]): boolean {
  if (waveType !== 'membership' && waveType !== 'retail') return false
  const opposite = waveType === 'membership' ? 'retail_order' : 'membership_entitlement'
  return kinds.includes(opposite)
}
```

```ts
// frontend/src/pages/waves/workspace/tabs/allocation/engineVisibility.test.ts
import { describe, expect, it } from 'vitest'
import { defaultExpandedEngines, kindTypeMismatch } from './engineVisibility'

describe('defaultExpandedEngines', () => {
  it('membership opens rules only', () => {
    expect(defaultExpandedEngines('membership')).toEqual(['rules'])
  })
  it('retail opens mapping only', () => {
    expect(defaultExpandedEngines('retail')).toEqual(['mapping'])
  })
  it('mixed opens both', () => {
    expect(defaultExpandedEngines('mixed')).toEqual(['rules', 'mapping'])
  })
})

describe('kindTypeMismatch', () => {
  it('flags retail docs going into a membership wave', () => {
    expect(kindTypeMismatch('membership', ['retail_order'])).toBe(true)
  })
  it('flags membership docs going into a retail wave', () => {
    expect(kindTypeMismatch('retail', ['membership_entitlement'])).toBe(true)
  })
  it('never flags mixed waves or matching kinds', () => {
    expect(kindTypeMismatch('mixed', ['retail_order'])).toBe(false)
    expect(kindTypeMismatch('membership', ['membership_entitlement'])).toBe(false)
  })
})
```

Run: `cd frontend && deno task test -- engineVisibility`
Expected: 失败（模块不存在）。

- [ ] **Step 2: WaveAllocationTab 软生效改造**

script 部分：

```ts
import { computed, ref, watch } from 'vue'
import { NCollapse, NCollapseItem, NButton } from 'naive-ui'
import { defaultExpandedEngines } from './allocation/engineVisibility'

// ── 波次类型软生效（spec §5.1）：默认展开对应引擎，另一侧可展开；mixed 双可见 ──
const waveType = computed<string>(() => ctx.snapshot.value?.wave.waveType ?? 'mixed')
const expandedNames = ref<string[]>([])
watch(waveType, (type) => { expandedNames.value = defaultExpandedEngines(type) }, { immediate: true })
const engineHintKey = computed<string>(() => `allocation.engineHint.${waveType.value === 'mixed' ? 'mixed' : waveType.value}`)
```

template：两个 `SectionCard`（rules/mapping）改为 `NCollapse` 折叠区块，动作按钮移入 `#header-extra`；映射按钮的 `:disabled="!allocation.hasParticipants.value"` 移除（零售引擎不再需要参与者前置，后端 Task 2 已同步解除），规则核算按钮保留该门禁；`mapping` 的 `needsParticipantsHint` CalloutBar 删除。要点结构：

```html
      <CalloutBar tone="info" :message="t(engineHintKey)" />

      <NCollapse v-model:expanded-names="expandedNames">
        <NCollapseItem name="rules" :title="t('allocation.tabs.rules')">
          <template #header-extra>
            <NButton size="tiny" @click="openPickFromMaster">{{ t('products.pickFromMasterAction') }}</NButton>
            <NButton size="tiny" @click="openCreateRule">{{ t('allocation.rules.addRule') }}</NButton>
            <NButton
              size="tiny"
              :loading="allocation.reconciling.value"
              :disabled="!allocation.hasParticipants.value"
              @click="handleReconcile"
            >
              {{ t('allocation.rules.reconcile') }}
            </NButton>
          </template>
          <CalloutBar v-if="reconcileSummaryMessage" tone="info" :message="reconcileSummaryMessage" />
          <CalloutBar
            v-if="!allocation.hasParticipants.value"
            tone="warning"
            :message="t('allocation.rules.needsParticipantsHint')"
          />
          <DataGrid
            :columns="columns"
            :rows="allocation.rules.value"
            row-key="id"
            :loading="allocation.loadingRules.value"
            pagination="client"
            :empty="{ title: t('allocation.rules.empty') }"
          />
        </NCollapseItem>

        <NCollapseItem name="mapping" :title="t('allocation.tabs.mapping')">
          <template #header-extra>
            <NButton
              size="tiny"
              type="primary"
              :loading="allocation.mappingRunning.value"
              @click="handleRunMapping"
            >
              {{ t('allocation.mapping.run') }}
            </NButton>
          </template>
          <p class="wave-allocation-tab__assigned-count">
            {{ t('allocation.mapping.assignedDemandCount', { count: allocation.assignedDemands.value.length }) }}
          </p>
          <ul v-if="allocation.assignedDemands.value.length > 0" class="wave-allocation-tab__assigned-list">
            <li v-for="doc in allocation.assignedDemands.value" :key="doc.id" class="wave-allocation-tab__assigned-item">
              <span class="wave-allocation-tab__assigned-doc-no">{{ doc.sourceDocumentNo }}</span>
              <StatusBadge dimension="demandKind" :value="doc.kind" size="sm" />
            </li>
          </ul>
          <MappingResultPanel :result="allocation.lastMappingResult.value" />
        </NCollapseItem>
      </NCollapse>
```

（`SectionCard` import 若不再用于 rules/mapping 区块仍被 participants 卡片使用，保留；`useAllocationTab.ts` 无需改动——`hasParticipants` 继续供规则引擎门禁使用。）

- [ ] **Step 3: WaveIntakeTab 业务面预筛联动**

script 补（依赖节 2 已引入 surface segmented；`BusinessSurface` 类型与函数来自节 2 Task 6 的 `businessSurface.ts`）：

```ts
import { computed, watch } from 'vue'
import { kindsFromSurface, surfaceFromKinds, type BusinessSurface } from '@/pages/inbox/inbox-grid/businessSurface'

// ── waveType 联动业务面预筛（spec §5.1）：软生效，仅当操作员未手动选过业务面时生效 ──
const waveType = computed<string>(() => ctx.snapshot.value?.wave.waveType ?? 'mixed')
watch(
  waveType,
  (type) => {
    if (surfaceFromKinds(grid.filters.state.demandKind) !== 'all') return
    const target: BusinessSurface =
      type === 'membership' ? 'membership_entitlement' : type === 'retail' ? 'retail_order' : 'all'
    grid.filters.setEnumValues('demandKind', kindsFromSurface(target))
  },
  { immediate: true },
)
```

（`setEnumValues`/`applySnapshot` 是 `UseUrlFiltersApi` 既有 API——`frontend/src/shared/ui/filter-bar/useUrlFilters.ts:74,92`；`grid` 为页面内既有 `useInboxGrid({ waveId: ctx.waveId })` 实例。）

- [ ] **Step 4: BatchActionBar 分派提示**

script 补（既有 import 行修改而非新增——`ref` 已在 `import { computed, ref } from 'vue'` 中，补 `type Ref`；`NAlert` 加入 `naive-ui` 导入行）：

```ts
import { computed, ref, type Ref } from 'vue'
import { NAlert, NButton, NModal, NSelect } from 'naive-ui'
import { kindTypeMismatch } from '@/pages/waves/workspace/tabs/allocation/engineVisibility'
import type { dto } from '@/../wailsjs/go/models'

const waves = ref<dto.WaveDTO[]>([]) as Ref<dto.WaveDTO[]>
```

`openPicker`（:39-40）改为同时保存 waves 列表：

```ts
    const page = await listWavesFiltered({ page: 1, pageSize: 200, sortBy: 'updatedAt', sortDesc: true })
    waves.value = page.items
    waveOptions.value = page.items.map((wave) => ({ label: `${wave.name} (${wave.waveNo})`, value: wave.id }))
```

追加 computed 与模板提示：

```ts
const selectedKinds = computed<string[]>(() => Array.from(new Set(props.selectedRows.map((row) => row.kind))))
const mismatchHint = computed<string | null>(() => {
  const wave = waves.value.find((w) => w.id === targetWaveId.value)
  if (!wave) return null
  if (!kindTypeMismatch(wave.waveType, selectedKinds.value)) return null
  return t('inbox.batch.kindTypeMismatchHint', { waveType: t(`waves.waveType.${wave.waveType}`) })
})
```

模板在 picker 的 `NSelect` 之后插入（提示不拦截——`canConfirm` 保持现状不变）：

```html
      <NAlert v-if="mismatchHint" type="warning" :show-icon="true" class="inbox-batch-action-bar__mismatch">
        {{ mismatchHint }}
      </NAlert>
```

样式补：

```css
.inbox-batch-action-bar__mismatch {
  margin-top: var(--space-3);
}
```

（`waves.waveType.*` 键已存在，`zh-CN.ts:740`；节 2 Task 8 若已把该 picker 改为带过滤/搜索的版本，则在其 `listWavesFiltered` 调用处同样补 `waves.value = page.items`，其余不变。）

- [ ] **Step 5: i18n 键（zh-CN.ts 与 en-US.ts 同落）**

```ts
// allocation 命名空间（zh-CN.ts:1715 区块内）
    engineHint: {
      membership: '本波次为会员权益波：默认使用「分配规则」引擎，订单映射可展开使用。',
      retail: '本波次为零售订单波：默认使用「订单映射」引擎，分配规则可展开使用。',
      mixed: '本波次为混合波：规则与映射两个引擎并排可见。',
    },
```

```ts
// inbox.batch 命名空间
      kindTypeMismatchHint: '所选波次类型为 {waveType}，选中需求包含另一业务面的文档——仍可分派，仅作提示。',
```

en-US.ts 对应：

```ts
    engineHint: {
      membership: 'This is a membership wave: the allocation-rules engine opens by default; order mapping can be expanded.',
      retail: 'This is a retail wave: the order-mapping engine opens by default; allocation rules can be expanded.',
      mixed: 'This is a mixed wave: both the rules and mapping engines are visible.',
    },
```

```ts
      kindTypeMismatchHint: 'The selected wave type is {waveType} while the selection includes documents from the other business surface — assignment is still allowed; this is a hint only.',
```

- [ ] **Step 6: 前端门 + Commit**

Run: `cd frontend && deno task typecheck && deno task test && deno task build && deno task lint:guardrails`
Expected: 全绿；guardrails 零违规（新提示文案全走 i18n，枚举渲染仍走 StatusBadge）。

```bash
git add frontend/src/pages/waves/workspace/tabs/WaveAllocationTab.vue frontend/src/pages/waves/workspace/tabs/allocation/engineVisibility.ts frontend/src/pages/waves/workspace/tabs/allocation/engineVisibility.test.ts frontend/src/pages/waves/workspace/tabs/WaveIntakeTab.vue frontend/src/pages/inbox/inbox-grid/BatchActionBar.vue frontend/src/shared/i18n/locales/zh-CN.ts frontend/src/shared/i18n/locales/en-US.ts
git commit -m "feat(allocation): 波次类型软生效——分配页签按 waveType 默认展开对应引擎、零售映射解除参与者门禁、intake 业务面预筛联动、分派 kind×type 仅提示不拦截"
```

---

### Task 8: 行详情 kind 分流（零售精简三态）+ 网格 lineReason 过滤与 adjusted 预设升级

**Files:**
- Modify: `frontend/src/pages/inbox/inbox-grid/RowDetailPanel.vue`（kind 分流：零售行三态、隐藏输入态选择器）
- Modify: `frontend/src/pages/waves/workspace/tabs/fulfillment-grid/filter-schema.ts`（schema +lineReason 维度；`adjusted` 预设从 reviewRequirement 近似升级为精确 lineReasons）
- Modify: `frontend/src/pages/waves/workspace/tabs/fulfillment-grid/useFulfillmentGrid.ts`（`buildFilterInput` 透传 lineReasons，:134-146）
- Modify: `frontend/src/shared/i18n/locales/zh-CN.ts` / `en-US.ts`（新键，两 locale 同落）

**Interfaces:**
- Consumes: Task 5/6 的 `lineReasons` 契约；后端 Task 3 的零售行 input-state 豁免（零售行编辑不再携带输入态要求）；`glossaryTables.lineReason`（已注册，`glossary.ts:343-347,469`）
- Produces:
  - `RowDetailPanel`：`isRetail` computed；`dispositionOptions`（零售行 = accepted/deferred/excluded_manual 三值）；零售行编辑/批量编辑不再要求 `recipientInputState`（编辑时回传原值、批量时传空串由后端保留）
  - `FULFILLMENT_GRID_FILTER_SCHEMA` 第 7 维度 `lineReason`；`adjusted` 预设 = `{ lineReason: ['wave_adjustment'] }`
  - i18n 键：`fulfillmentGrid.filters.lineReason`、`inbox.detail.retailTriageHint`

- [ ] **Step 1: RowDetailPanel kind 分流**

script 部分：

```ts
/** 零售行免分诊（spec §5.2）：精简三态 accepted/deferred/excluded，无收货输入态。 */
const isRetail = computed(() => props.row?.kind === 'retail_order')

const retailDispositionOptions = computed<SelectOption[]>(() =>
  (['accepted', 'deferred', 'excluded_manual'] as const).map((value) => ({
    label: glossaryLabel('routingDisposition', value),
    value,
  })),
)
const dispositionOptions = computed<SelectOption[]>(() =>
  isRetail.value ? retailDispositionOptions.value : routingDispositionOptions.value,
)
```

`startEdit` 保持双字段回填（零售行 `recipientInputState` 恒为 `not_required`，回填后无需选择）；`saveEdit` 的守卫与提交改：

```ts
async function saveEdit(line: DemandLine): Promise<void> {
  if (!editDraft.routingDisposition || (!isRetail.value && !editDraft.recipientInputState)) return
  savingLineId.value = line.id
  try {
    await updateDemandLineRouting({
      demandLineId: line.id,
      routingDisposition: editDraft.routingDisposition,
      recipientInputState: isRetail.value ? (line.recipientInputState ?? 'not_required') : editDraft.recipientInputState!,
      routingReasonCode: line.routingReasonCode ?? '',
    })
    ...
```

`canApplyBulk` 与 `applyBulkRouting` 改：

```ts
const canApplyBulk = computed(
  () =>
    selectedLineIds.value.length > 0 &&
    !!bulkRoutingDisposition.value &&
    (isRetail.value || !!bulkRecipientInputState.value) &&
    !applyingBulk.value,
)
```

```ts
    const result = await batchUpdateDemandLineRouting({
      updates: selectedLineIds.value.map((demandLineId) => ({
        demandLineId,
        routingDisposition: bulkRoutingDisposition.value as string,
        recipientInputState: isRetail.value ? '' : (bulkRecipientInputState.value as string),
        routingReasonCode: '',
      })),
    })
```

template 改动（三处，均有精确对照）：

1. 批量栏（:232-238）第二个 `NSelect`（recipientInputState）加 `v-if="!isRetail"`。
2. 编辑态（:253-268）中 `editDraft.recipientInputState` 的 `NSelect` 加 `v-if="!isRetail"`；`editDraft.routingDisposition` 的 `NSelect` 的 `:options` 改为 `dispositionOptions`。
3. 展示态（:271-275）：`StatusBadge dimension="recipientInputState"` 加 `v-if="!isRetail"`（零售行不渲染输入态徽章）。

另在 `SectionCard`（routing 卡）头部下方加提示行（仅零售行）：

```html
      <CalloutBar v-if="isRetail" tone="info" :message="t('inbox.detail.retailTriageHint')" />
```

（`CalloutBar` 补 import：`import { CalloutBar } from '@/shared/ui/guidance'`。）

- [ ] **Step 2: 网格过滤 schema 与预设升级**

`frontend/src/pages/waves/workspace/tabs/fulfillment-grid/filter-schema.ts`：

schema（:28-36）在 `driftStatus` 后插入：

```ts
  { key: 'lineReason', type: 'enum-multi', dimension: 'lineReason' },
```

`adjusted` 条目及其「近似说明」（:54-62，即 `* - \`adjusted\`:` 行至注释闭合 `*/`）替换为：

```ts
 * - `adjusted`: NOW EXACT — the Go side exposes `lineReasons` on
 *   `WaveFulfillmentFilterInput` (round-2 节 3 Task 5), and participant-target
 *   reissue/compensation spawn dedicated `lineReason='wave_adjustment'` lines
 *   (Task 4), so this preset pins exactly the manually re-sent lines.
 */
export const FULFILLMENT_GRID_PRESET_SNAPSHOTS: Record<FulfillmentGridPresetId, FilterSnapshot> = {
  blocked: { addressState: ['missing', 'invalid'] },
  submittable: { addressState: ['ready'], supplierState: ['not_submitted'] },
  backfillFailed: { channelSyncState: ['failed'] },
  adjusted: { lineReason: ['wave_adjustment'] },
}
```

（模块顶部 :16-21 的 CAVEAT 注释一并更新：`lineReason` 是逐行谓词，不受「整波盖章」限制。）

`frontend/src/pages/waves/workspace/tabs/fulfillment-grid/useFulfillmentGrid.ts` 的 `buildFilterInput`（:134-146）：

```ts
      reviewRequirements: filters.state.reviewRequirement,
      driftStatuses: filters.state.driftStatus,
      lineReasons: filters.state.lineReason,
      keyword: filters.state.keyword,
```

（`filters.state.lineReason` 由 `useUrlFilters` 按新 schema 自动派生；Task 6 Step 3 的临时 `lineReasons: []` 在此替换为正式取值。）

- [ ] **Step 3: i18n 键（zh-CN.ts 与 en-US.ts 同落）**

```ts
// fulfillmentGrid.filters（zh-CN.ts:1238 区块内）
      lineReason: '来源',
```

```ts
// inbox.detail 命名空间
      retailTriageHint: '零售行免分诊：仅需选择受理/推迟/排除，无需填写收货输入状态。',
```

en-US.ts 对应：

```ts
      lineReason: 'Source',
```

```ts
      retailTriageHint: 'Retail lines skip triage: only accepted/deferred/excluded apply; no recipient input state is required.',
```

- [ ] **Step 4: 前端门 + Commit**

Run: `cd frontend && deno task typecheck && deno task test && deno task build && deno task lint:guardrails`
Expected: 全绿；guardrails 零违规（lineReason 过滤走 FilterBar 维度，零售行徽章仍走 StatusBadge）。

```bash
git add frontend/src/pages/inbox/inbox-grid/RowDetailPanel.vue frontend/src/pages/waves/workspace/tabs/fulfillment-grid/filter-schema.ts frontend/src/pages/waves/workspace/tabs/fulfillment-grid/useFulfillmentGrid.ts frontend/src/shared/i18n/locales/zh-CN.ts frontend/src/shared/i18n/locales/en-US.ts
git commit -m "feat(inbox): 行详情按 kind 分流（零售精简三态、免输入态）+ 履约网格 lineReason 过滤维度与已调整预设精确化"
```

---

### Task 9: 分派门禁收紧（节 2 依赖收口）+ 六门全量回归

**Files:**
- Modify: `internal/app/wave_lifecycle_usecase.go`（`assignOne` 移除零售 kind 豁免，:179-214）
- Test: `internal/app/wave_lifecycle_usecase_test.go`（追加 1 用例；harness = `newWaveLifecycleTestUC`，:50-60）

**Interfaces:**
- Consumes: 节 2 Task 3 已落地的 `assignOne` 门禁（`ExistsByDocument` 重复分派拦截 + membership 专属 pending_intake 拦截）；本计划 Task 3 的零售自动裁决（收紧的安全前提）
- Produces: `assignOne` 最终规则——pending_intake 门禁对所有 kind 统一生效（零售行导入即 accepted，`pending_intake` 只可能来自历史数据或手改，此时应显式拦截而非静默放行）

**依赖注明（节 2 → 节 3 收口）:** 节 2 计划 Task 3 在 `assignOne` 中对 retail_order 的 pending_intake 豁免是**暂时的**（当时零售自动裁决尚未落地，豁免是为避免零售单无法分派）。本计划 Task 3 已让零售行导入即 accepted，收紧不再有误伤面，故作为本计划最后一个任务落地。若本计划执行时节 2 Task 3 尚未合入，本任务必须等它先行——收紧的代码就写在它落地的文件与行号上。

- [ ] **Step 1: 写失败测试**

`internal/app/wave_lifecycle_usecase_test.go` 追加：

```go
func TestAssignOneRejectsRetailDocWithPendingIntakeLines(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uc, waveRepo, demandRepo, _ := newWaveLifecycleTestUC()

	wave := &domain.Wave{Name: "retail-wave"}
	if err := waveRepo.Create(ctx, wave); err != nil {
		t.Fatalf("seed wave: %v", err)
	}
	// 历史零售文档：导入于自动裁决上线之前，仍带 pending_intake 行。
	doc := &domain.DemandDocument{Kind: "retail_order", SourceDocumentNo: "LEGACY-1"}
	if err := NewDemandIntakeUseCase(demandRepo).ImportDemand(ctx, doc, []*domain.DemandLine{
		{RoutingDisposition: "pending_intake", RecipientInputState: "not_required", LineType: "sku_order"},
	}); err != nil {
		t.Fatalf("import doc: %v", err)
	}

	result, err := uc.BatchAssignDemandToWave(ctx, wave.ID, []uint{doc.ID})
	if err != nil {
		t.Fatalf("BatchAssignDemandToWave: %v", err)
	}
	if result.FailureCount != 1 || len(result.Results) != 1 || result.Results[0].Success {
		t.Fatalf("retail doc with pending_intake must be rejected, got %+v", result)
	}
	if !strings.Contains(result.Results[0].Error, "pending_intake") {
		t.Errorf("error should mention pending_intake, got: %q", result.Results[0].Error)
	}
}
```

（`strings` 已在文件 import 块内。节 2 Task 3 的既有两个用例 `TestAssignOneRejectsAlreadyAssignedDocument` / `TestAssignOneRejectsMembershipDocWithPendingIntakeLines` 保持通过。）

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -count=1 ./internal/app/ -run TestAssignOneRejectsRetailDocWithPendingIntakeLines`
Expected: 失败（现状 retail kind 豁免，分派成功）。

- [ ] **Step 3: 收紧 assignOne**

`internal/app/wave_lifecycle_usecase.go` 的 `assignOne`（节 2 Task 3 落地后的形态）中，把 membership 专属的 pending_intake 检查改为全 kind 统一：

```go
	// All kinds share the triage gate: retail lines are auto-adjudicated at import
	// (round-2 节 3 Task 3), so a pending_intake line here means triage is
	// genuinely incomplete — legacy retail rows must be triaged (via the retail
	// three-state flow) before assignment.
	docLines, lineErr := uc.demandRepo.ListLinesByDocument(ctx, demandDocumentID)
	if lineErr != nil {
		return fmt.Errorf("list demand lines for demand document %d: %w", demandDocumentID, lineErr)
	}
	for _, l := range docLines {
		if l.RoutingDisposition == string(domain.RoutingDispositionPendingIntake) {
			return fmt.Errorf("demand document %d has pending_intake line(s); complete triage before assigning to a wave", demandDocumentID)
		}
	}
```

（即删除节 2 加的 `if doc.Kind == string(domain.DemandKindMembershipEntitlement) {` 包裹层与对应右花括号，循环体与错误文案不变；重复分派拦截 `ExistsByDocument` 保持不动。）

- [ ] **Step 4: 后端全量门**

Run: `gofmt -w internal/app/wave_lifecycle_usecase.go internal/app/wave_lifecycle_usecase_test.go && go build ./... && go test -count=1 ./...`
Expected: 全绿。

- [ ] **Step 5: i18n 叶子键 parity 检查**

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

Expected: 两行均输出 0。（若 locale 文件导出结构非默认导出对象，按实际导出形式调整——先 `Read` 两个 locale 文件头部确认导出形态，与节 2 计划 Task 9 同一手法。）

- [ ] **Step 6: 六门全量回归**

Run（仓库根）:

```bash
go build ./... && go test -count=1 ./... && cd frontend && deno task typecheck && deno task test && deno task build && deno task lint:guardrails
```

Expected: 六门全绿（vitest 全文件过；guardrails `no violations found`；enum check 通过——本计划未改 `internal/domain/enums.go`，`SnapshotTypeMixed` 等均为既有常量）。

- [ ] **Step 7: 手工走查清单（无需自动化）**

1. 建 retail 波 → 分配页签默认只展开「订单映射」，规则区块可手动展开；映射按钮不再被「先生成参与者」禁用。
2. 零售 CSV / 手工录入导入 → 行自动 accepted；收件箱行详情显示精简三态与「零售行免分诊」提示，无输入态选择器；改为 deferred 后回改 accepted 正常。
3. 混合波：同一 profile 的权益+零售双文档分派后「生成参与者」→ 参与者列表单条 `mixed` 快照，GiftLevel 来自权益行。
4. 零售波不生成参与者直接映射 → 履约行照常生成，快照 ID 为空（软链接）。
5. 会员波 reconcile：零售买家 profile 不再被规则白送礼品；waived 行不计入 ready 统计、不进快照。
6. 对某会员行执行「重新发放」（reissue）→ reconcile 后网格出现独立的「来源=wave_adjustment」行；「已调整」预设视图精确命中该行。
7. 全局收件箱批量分派到类型不匹配的波次 → 出现警告提示但可确认分派。
8. 历史零售文档（含 pending_intake 行）分派 → 被后端拒绝并提示先分诊。

- [ ] **Step 8: Commit**

```bash
git add internal/app/wave_lifecycle_usecase.go internal/app/wave_lifecycle_usecase_test.go
git commit -m "feat(wave): 分派门禁收紧——pending_intake 校验对全 kind 统一生效，零售自动裁决落地后移除节 2 临时豁免"
```

---

## Self-Review 记录

- **Spec 覆盖**：§5.1（波次类型软生效）→ Task 7（页签默认/预筛/提示）+ 回归约束 Task 2/4 的行级共存（GeneratedBy 命名空间不改，reconcile 只替换 policy_driven、mapping 只替换 demand_driven）；§5.2（零售免分诊）→ Task 3（后端）+ Task 8（行详情分流）；§5.3（混合快照）→ Task 1；§5.4（引擎边界修复）→ Task 2；§5.5（来源三值补齐）→ Task 4（wave_adjustment 写入路径）+ Task 5（lineReasons 过滤）+ Task 8（adjusted 预设精确化）；§5.6（waived 口径）→ Task 1（快照资格 + 统计单列，前端无统计 UI 消费者——已核实 `getWaveRoutingStats` 无页面调用，展示一致性问题不存在）；§5.7（调整层唯一补发路径）→ 回归约束，无新入口，Task 9 走查清单第 6 条覆盖。节 2 依赖收口 → Task 9（assignOne 豁免收紧，已注明前置顺序）。
- **占位符扫描**：全文无 TBD/TODO/「待实现」注释；每个代码步骤给出真实代码。唯一带「临时」字样的代码是 Task 6 Step 3 的 `lineReasons: []` 透传——那是跨任务 staged 变更的必经中间态（Task 8 Step 2 给出替换落点与代码），非占位。留给实现时核对的仅有「以实际行号/实际 API 为准」类提示（`waves.waveType.*` 键路径、locale 导出形态），均附带验证命令。
- **类型一致性**：`WaveFulfillmentFilterInput.LineReasons []string` 在 Task 5 定义、Task 6 绑定、Task 8 消费，字段名三处一致；`WaveRoutingStatsDTO.WaivedCount` 在 Task 1 定义、Task 6 绑定；`applyRetailAutoAdjudication(kind string, lines []*domain.DemandLine)` 在 Task 3 定义并在同任务三个调用点消费；`createsNewLine`/`spawnWaveAdjustmentLine` 在 Task 4 定义、同任务 reconcile 路径消费；前端 `defaultExpandedEngines`/`kindTypeMismatch` 在 Task 7 定义并在同任务消费；`businessSurface.ts` 的三态类型由节 2 定义、Task 7 消费，命名一致。枚举值全部引用 `domain.*` 常量（非裸字符串）。
- **已知前置**：Task 7/8/9 依赖节 2 计划已落地（`businessSurface.ts`、intake 重构、`assignOne` 门禁、picker 增强）；Task 9 的收紧必须在 Task 3（零售自动裁决）之后执行，计划顺序已保证；Task 1 对 `mockWaveRepo.AddParticipant` 的落库化改动属于测试 harness 修正，已在对应 Step 注明对既有用例无影响。
