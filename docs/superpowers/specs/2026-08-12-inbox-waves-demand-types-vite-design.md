# 收件箱↔波次交互重构、需求类型分流与 Vite 8 工具链升级设计

- 日期：2026-08-12
- 状态：节 1–3 已获产品负责人批准；节 4（模板/接入配置）待专项多轮讨论后补入
- 上游文档：`docs/FRONTEND-REDESIGN-PLAN.md`、`docs/fulfillment-v2-refactor/**`（决策编号引用 `06-rollout-and-governance/06-open-decisions.md`）

## 1. 背景

前端重设计（P0–P7）切换落地后，产品侧确认三处现存痛感：

1. 收件箱与履约波次的交互不便（跨页跳转多、深链失效、同一概念两个视图）
2. config（导入模板）的创建/编辑与默认模板问题大 —— **本文档暂不覆盖，作为节 4 待产品负责人点名后专项多轮讨论**
3. 零售订单 / 会员权益两类需求在工作流上打架

另有独立工具链需求：Vite 升级至 8 最新版。

调研方式：5-agent 并行只读摸底（收件箱/波次、模板配置、需求类型、Vite 工具链、设计文档决策提取）+ v1 备份分支（`backup/pre-fulfillment-v2-refactor-2026-05-12`）前端工作流调研。

**关键结论**：三大痛区多数是「已拍板设计决策未落地」造成的偏差，而非需要新方向。本设计以补齐既定决策为主轴。

## 2. 决策账本（已批准）

| 领域 | 决策 |
| --- | --- |
| 波次类型 | 软生效：类型驱动 UI 组织与预筛，不做硬门禁 |
| 零售裁决 | 导入自动 accepted，手动编辑入口保留（收件箱行详情按 kind 分流） |
| 收件箱分区 | 按业务面三态分区（全部/会员权益/零售订单）+ demandKind 多选 |
| 待分诊卡片 | 后端加 routingDisposition 过滤维度 + pendingIntakeCount，深链直达 |
| 分派后落点 | 回执一键直达目标波次 intake |
| 波内收件箱 | 升级为「波内导入页面」：拉取未分派需求（主）+ 波内文件导入（导入即入本波，可多次）+ 收件箱能力全集 |
| 建波流程 | 建波保持最简，成功后直落 intake；不另设建波时挑单步骤 |
| 双身份快照 | 合并 mixed（启用 SnapshotTypeMixed） |
| 调整来源 | reissue/compensation 新行落 wave_adjustment，凑齐三来源 |
| 会员补发 | 走调整层 reissue，不新增导入入口 |
| waived | 豁免=扣下不发：单列统计、引擎不执行、可改回 ready |
| Vite | 核心最小集（8.2.1 + vitest 4 + patch 级），不跟 router5/pinia4 |
| 模板 | 挂起，节 4 待专项讨论 |

**文档定案直接落实（无需再决策）**：

- 仅 routingDisposition=accepted 进入稳定波次处理（FRP §3.5、04-workflows/03）
- 会员分配与零售映射是两个独立初始分配入口，写入同一套 FulfillmentLine（#2/#3）
- 不支持跨波次拆分（#34），重复分派应拦截
- 步骤向导跨步骤导航（#11，📋 待实现）
- Profile 变更绑定版本，活跃波次继续使用创建时行为（#39，📋 待实现）
- 「会员分配体验不得退化」为业务层强约束

## 3. 设计节 1 — Vite 核心最小集升级（已批准，可独立执行）

- 改动面：`frontend/package.json` 七个版本号，一个原子提交（含重生成的 `deno.lock`）：

| 包 | 当前 | 目标 |
| --- | --- | --- |
| vite | ^8.0.9 | ^8.2.1 |
| @vitejs/plugin-vue | ^6.0.6 | ^6.0.8 |
| vitest | ^3.1.3 | ^4.1.10 |
| vue | ^3.5.33 | ^3.5.41 |
| vue-tsc | ^3.2.7 | ^3.3.9 |
| vue-i18n | ^11.0.0 | ^11.4.8 |
| @vue/test-utils | ^2.4.6 | ^2.4.11 |

- 流程：改版本 → `deno install` 重生成 `deno.lock` → 核对 Vite 7 影子副本（`vite@7.3.2`/rollup 条目）消失 → `typecheck` / `test` / `build` / `lint:guardrails` 四连验证 → 提交。
- 依据（调研核实）：
  - 8.0.9→8.2.1 区间官方 changelog 无 breaking/弃用条目；8.0 的 Rolldown/Oxc 大变更已在现版本消化，`vite.config.ts` 极简（仅 plugin-vue + alias + 端口），全仓 grep 零命中 Vite 5 时代 API（transformWithEsbuild/manualChunks/rollupOptions/esbuild 配置）。
  - vitest 3.x peer 仅支持 vite ^5||^6||^7，是 lock 中 Vite 7 影子副本的来源；4.1.10 起 peer 扩至 ^6||^7||^8。v4 breaking 与 23 个测试文件逐条核对零命中。
  - plugin-vue 6.0.5 起 peer 含 vite 8；TS 维持 6.0.3（vue-tsc/Volar 官方确认 TS 7.0 移除 createProgram，不可用）。
- 明确不升：TypeScript 7（等 7.1）、vue-router 5、pinia 4、happy-dom 20、@types/node 26。
- 风险面：仅 vitest 4 为行为面变化（已核对零命中）；升级基线（8.0.9）typecheck/build 已验证全绿。

## 4. 设计节 2 — 收件箱 ↔ 波次：双向对称 + 波内导入页面

目标：收件箱保留为全局导入/分诊面，波次内自足（v1 教训），两者双向可达（FRP §3.5），深链杜绝断头（FRP §3.1）。

1. **全局收件箱**（`frontend/src/pages/inbox/`）：
   - 业务面三态分区（全部/会员权益/零售订单）+ demandKind 多选过滤。
   - 后端 `DemandInboxFilterInput` 增加 `routingDisposition` 过滤维度（`internal/app/dto/demand.go:90-102` 现状缺）；`DemandInboxRowDTO` 增加 `pendingIntakeCount`（现状行组装忽略 pending_intake，`internal/app/list_pagination_usecase.go:197-213`），与任务中心「待分诊」计数对账。
   - 分派回执可点击 → 直达目标波次 intake（现状纯文本，`BatchActionBar.vue:59-66`）。
   - assignment 三态写入 URL（现状 read-once，`useInboxGrid.ts:86-98`）。
   - 导入结果步增加「发送到波次」出口（FRP §3.5 要求，现状只有关闭，`ImportFileModal.vue:334-356`）。
2. **波内导入页面**（intake 步骤升级，`WaveIntakeTab.vue` 重构）：
   - 与全局收件箱同组件同能力：业务面分区、筛选、批量勾选。
   - 「拉取需求」为主入口：浏览未分派池（按业务面过滤），批量拉入当前波次（复用 `BatchAssignDemandToWave`，`controller_wave_lifecycle.go:139-191`）。
   - 波内文件导入为次入口：文件导入 → 立即入本波（前端编排 ImportDemandCSV → BatchAssignDemandToWave 链；波内导入需选择接入配置以确定默认模板）。支持波次生命周期内多次导入（v1 步骤 1 的现代形态）。
   - 批量退单（后端补 `BatchUnassignDemandFromWave`；现状单条 `UnassignDemandFromWave`，且按 item 记 undo 历史需一并处理撤销粒度）。
3. **深链全修**：
   - 任务中心 bucket 卡/总览六桶/漏斗的深链：修正目标子路由与 query 键单复数（现状 `wave-filter-link.ts:19-26` 复数键 vs `useUrlFilters.ts:60-66` 单数读取，预过滤被静默丢弃）。
   - 「打开波次」按场景落点：分派后 → intake；任务卡 → lines 预过滤视图。
4. **门禁**：
   - accepted 才能入波：权益行未分诊禁入（前端禁用 + 后端校验，`wave_lifecycle_usecase.go:179-214` 现状零校验）；零售行自动 accepted 不受影响。
   - 重复分派拦截：后端阻止 doc 二次挂波（#34 禁跨波拆分；现状仅 pair 唯一索引、UI 只显示第一个赋值）。
5. **选波次 picker**（`BatchActionBar.vue:34-47`）：阶段过滤（排除已关闭）、名称搜索、类型筛选；去掉 200 条硬截断静默（后端 `ListWavesFiltered` 补过滤字段）。
6. **建波流程**：建波对话框保持最简；建波成功后落地页直落 intake（非总览），当场拉单或导入。

## 5. 设计节 3 — 需求类型：分引擎、分行源、分业务面

1. **波次类型软生效**：类型保留三选一（membership/retail/mixed），语义改为「默认视图与预筛」：
   - 分配页签默认落在对应引擎区块（membership→规则，retail→映射，mixed→并排）；另一侧可展开。
   - 收件箱/intake 按业务面预筛；分派时 kind×type 不匹配仅提示不拦截。
   - 行级共存保留（GeneratedBy 命名空间：reconcile 只删 policy_driven、mapping 只删 demand_driven）。
2. **零售免分诊**：
   - 导入管线对 retail_order 行自动置 accepted（`controller_demand.go` 导入路径按 kind 分支）。
   - 手动裁决入口保留在收件箱行详情，按 kind 分流：零售行精简三态（accepted/deferred/excluded），权益行保持完整分诊（disposition + input state 六态）；零售行不再要求 recipient_input_state。
3. **混合快照**：`GenerateParticipants` 聚合两类行后写 `SnapshotTypeMixed`（双身份合并一条，启用已定义未生成的枚举）；GiftLevel 只从权益行取；查询加稳定排序（现状 `demand_assignment_repo.go:77-99` 无 ORDER BY，快照类型看插入序）。
4. **引擎边界修复**（bug 级）：
   - reconcile 资格判定只扫会员权益行（现状 `allocation_policy_usecase.go:86-116` 不区分 kind，零售买家被会员规则白送礼品）。
   - 零售映射解除「必须先生成参与者」前置（现状 `use_cases.go:282-312` 报错门），订单行直入：按 doc 行解析商品/地址门直接生成履约行；UI 的 hasParticipants 门禁对零售引擎移除。
5. **来源三值补齐**：
   - reissue/compensation 产生的新行写 `LineReason=wave_adjustment`（现状三值只落两值，无写入路径）。
   - 后端 `WaveFulfillmentFilterInput` 加 lineReasons 过滤维度；「adjusted」保存视图从 reviewRequirement 近似升级为精确过滤（P3 期已知近似，`fulfillment-grid/filter-schema.ts:54-62` 自认注释）。
6. **waived 口径统一**（豁免=扣下不发）：
   - 统计单列（不计入 ready，现状 `entitlement_routing_usecase.go:103-105` 计入）。
   - 引擎不执行（现状一致，保持）。
   - 可显式改回 ready 恢复执行。
   - `GenerateParticipants` 只收 ready/not_required 行（现状 `use_cases.go:174-185` 无视 input state，waiting_for_input 也进快照——连带修复）。
7. **调整层是唯一手工补发路径**：会员补发走波内 reissue（#15 钦定），不新增导入入口；`manual_grant` 枚举保留不接 UI。

## 6. v1 工作流调研结论（`backup/pre-fulfillment-v2-refactor-2026-05-12`）

- v1 无独立收件箱页面，一切在波次内发生：步骤 1 两个导入面板（商品 CSV/ZIP + 会员数据 CSV）、步骤 3「添加礼物」弹窗（选会员+商品+数量 → `syncUserTagForTargetQuantity` → 后端自动 ReconcileWave 生成 DispatchRecord）。
- v1 无零售/会员之分（旧模型统一为 会员×商品×数量），零售语义只在导出模板表头出现。
- v1 也没有独立的手工授予入口——其手工补发形态（波内弹窗改标签数量 + 自动 reconcile）正是本设计「调整层 reissue」的直系祖先；`AddDispatchToMember` 后端绑定前端零调用，职能已被调整层覆盖，无需复活。
- 吸收：波内自足（→波内导入页面）、建波后直落工作流起点（→直落 intake）、波内多次导入（→波内文件导入入口）。
- 不吸收：无收件箱的一级入口设计（设计宪章 §3.1/§3.5 定案保留全局收件箱）、波内才做商品导入（商品目录已有全局主数据入口）。

## 7. 待办与挂起

- 节 4 模板/接入配置：待产品负责人点名后多轮专项讨论（机制现状已调研完毕，见调研产物）。
- 波内导入页面的接入配置选择交互（波内文件导入时如何确定 profile/默认模板）——实现规划时细化。
- 批量退单的 undo 粒度：现状按 item 记历史节点，批量退单的撤销需一次 Ctrl+Z（后端配合）。

## 8. 范围外（明确不做）

- vue-router 5 / pinia 4 / happy-dom 20 / @types/node 26 / TypeScript 7（理由见节 1）。
- 会员手工授予独立导入入口（调整层为唯一补发路径）。
- 跨波次拆分与多波次挂载 UI（#34 定案不支持）。
- 模板编辑历史 scope（06a-history-scope-deferral.md 显式延后）。
