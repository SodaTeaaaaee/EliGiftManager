# 决策记录与继续讨论事项

> 标注说明：`✅ 已实现` = 已在当前代码中落地；`📋 待实现` = 已确认但尚未在代码中体现。

## 已确认的决策

### 1. `WaveAllocationStep` ✅ 已实现

文档和代码正式采用 `WaveAllocationStep`，`WaveTagStep` 仅作历史参考。

### 2. 混合波次按工作流分区 ✅ 已实现

- `Membership Allocation` + `Demand Mapping` 是两个独立初始分配入口
- `Wave Overview` 是波次内独立只读总览页
- `Adjustment Review` 是共享收敛层

### 3. 零售渠道允许进入共享调整层 ✅ 已实现

允许加送/减送/替换/补发，但不能覆盖原始订单语义。

### 4. 模板系统升级为 Profile 系统 ✅ 已实现

V2 采用 `IntegrationProfile` + `DocumentTemplate` + `IntegrationProfileTemplateBinding`。

### 5. 人工闭环只记录决策，不覆盖事实状态 ✅ 已实现

人工进度记录闭环决策，不覆盖工厂发货、物流、渠道同步等事实状态。

### 6. 支持关系导出的实体礼物归 `membership_entitlement` ✅ 已实现

通过 `ObligationTriggerKind`、`EntitlementAuthority`、`RecipientInputState` 区分内部差异。

### 7. 会员限定购买归 `retail_order` ✅ 已实现

通过 `eligibility_context_ref` 保留资格来源，`ObligationTriggerKind = supporter_only_purchase`。

### 8. 平台权威成就交由上游平台判定 ✅ 已实现

`EntitlementAuthority = upstream_platform`，本系统负责输入采集与履约执行。

### 9. 共享总览页采用方案 B ✅ 已实现

`Wave Overview` 是波次内独立页面，不与 `Adjustment Review` 合并。

### 10. `Wave Overview` 只读优先 ✅ 已实现

主要负责观察、汇总、诊断、导航。编辑职责在 `Adjustment Review`。

### 11. 步骤向导与跨步骤导航 📋 待实现

波次内需要更强的步骤向导，用户可在任意步骤间跳转。后端数据结构已支持。

### 12. 非破坏性编辑 ✅ 已实现

不同步骤是同一波次数据的不同视角，页面切换不隐式重写数据。

### 13. `Wave Overview` 作为调整层入口 📋 待实现

可承担进入 `Adjustment Review` 的显式关口，但非强制卡点。

### 14. "三问分流"规则 ✅ 已实现

- 改上游真相 → 回前置页面
- 改默认生成逻辑 → 回分配/映射页面
- 改最终履约例外 → 进 `Adjustment Review`

### 15. `Adjustment Review` 首版范围 ✅ 已实现

- 允许：加送、减送、替换、移除、补发/补偿、一次性数量修正
- 不允许：改写原始订单/权益/路由/输入状态/规则/映射

### 16. 不允许在调整层新增全新参与者 ✅ 已实现

共享调整层只修改已进入当前波次的对象。

### 17. 不为旧数据兼容背负约束 ✅ 已实现

greenfield 重建，文档和代码直接按目标语义收敛。

### 18. 工厂提交后仍允许编辑 ✅ 已实现

软件以当前工作区结果为准，外部对象通过 basis 偏离提示。

### 19. 动态集合规则属于 `Membership Allocation` ✅ 已实现

`Adjustment Review` 不支持 selector 级动态批量规则。

### 20. 有符号贡献可以存在，最终结果不为负 ✅ 已实现

`resolved = max(base + delta, 0)`。负号存在于贡献层和 delta 层。

### 21. 调整层只处理具体对象 ✅ 已实现

不持有会随重算扩张/收缩的动态 selector 目标。

### 22. 持久化树状工作区历史 ✅ 已实现

`HistoryScope` + `HistoryNode` + `HistoryCheckpoint` + `HistoryPin`。

### 23. 撤销/重做只回滚本地工作区 ✅ 已实现

外部对象保留，与工作区脱节时进入 basis 偏离提示。

### 24. 全应用共用 history 基础设施 📋 待实现

优先完善 `wave` scope，`templates/products` 逐步纳入。

### 25. Undo/Redo 即时反馈 📋 待实现

即时 toast + 短期回执托盘。

### 26. 全局快捷键不劫持文本输入 📋 待实现

焦点在 input/textarea 时，Ctrl+Z 优先服务文本输入。

### 27. 命名尽早收敛 ✅ 已实现

代码直接使用 V2 业务语言（`CustomerProfile`、`FulfillmentLine` 等）。

### 28. Basis Drift 与 Review Requirement 双轴 ✅ 已实现

`basis_drift_status`（`in_sync`/`drifted`）+ `review_requirement`（`none`/`recommended`/`required`）。

### 29. 强提示以"是否丢失可用性"为基准 ✅ 已实现

`drifted + required` 时进入强提示与复核入口。

### 30. 调整层失配以"能否正确重放"为核心 ✅ 已实现

target 删除/歧义/basis 映射丢失 → `review_requirement = required`。

### 31. 继续沿用现有技术栈 ✅ 已实现

Wails + Go + Vue/TypeScript + SQLite + 本地工作区模式。

### 32. 工程壳子只保留中性部分 ✅ 已实现

旧业务代码已删除，按 V2 重建。

### 33. 第一条端到端纵切面 ✅ 已实现

`Demand Intake → Initial Allocation → Wave Overview → SupplierOrder Export`。

### 34. 保留 `WaveDemandAssignment` ✅ 已实现

不支持跨波次拆分，但保留显式关系对象。

### 35. 单写者假设 ✅ 已实现

同一时刻只有一个实例写入，树状历史天然为未来多端保留扩展空间。

### 36. 导入部分成功支持两种模式 ✅ 已实现

整体拒绝（全部回滚）或跳过错误行（成功行落库）。

### 37. 身份归并策略 ✅ 已实现

手动归并可用，自动归并默认关闭。归并只影响未来波次。操作可撤销。

### 38. Adjustment 重放默认整体暂停 📋 待实现

重放顺序：跨层级按步骤顺序，同层级内按 `created_at` 升序。默认暂停，可切换为标记并继续。

### 39. Profile 变更采用绑定版本 📋 待实现

活跃波次继续使用创建时绑定的 profile 行为，直到用户显式刷新。首版不引入版本号。

### 40. 外部事实不可通过 undo 撤销 ✅ 已实现

undo 不能回退外部事实状态，但可通过正向操作创建新对象替代旧的。

## 仍需继续讨论的问题

### 1. 灰区动作的归属规则

主规则已够用（改上游回前置，改例外进调整层），但"合理批量例外"与"应前置的批量逻辑修正"的界线仍需细化。

### 2. `Wave Overview` 首版聚合视角

首版最小视角由当前工作流推导决定，后续通过反馈迭代调整。

### 3. 调整层重放精确顺序

前置层先重建基础 → 共享调整层重放。`drift_reason_codes` 首版范围和双轴信号聚合展示方式待细化。

### 4. 导出/回传对象与工作区脱节提示

basis 偏离已建模。`SupplierOrder`/`Shipment`/`ChannelSyncJob` 各保留 basis 快照。导出 basis、物流 basis、回填 basis 的 reason code 首版范围待定。

### 5. 反馈循环边界

反馈循环适合改"看法"（分组命名、导航样式、计数位置），不适合替代"语义边界"。

### 6. 工作区历史参数

已确认：历史必须持久化、分支不能丢、toast 需要存在。具体参数（checkpoint 频率、GC 条件、toast 时长）在实现阶段调整。

### 7. Greenfield 开工前最后确认

技术栈已确认，工程壳子已清理。剩余：具体 SQLite pragma 偏好。
