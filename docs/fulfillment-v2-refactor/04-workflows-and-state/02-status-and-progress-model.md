# 状态与进度模型重构

本文件用于重新定义行级状态、波次聚合状态和未来进度展示方式，以支撑更长的履约生命周期。

## 8. 状态与进度模型重构

### 8.1 现有状态模型的问题

现有 `Wave.Status` 有以下问题：

1. 只有单维状态
2. 只反映“是否缺地址”和“是否已导出”
3. 无法表达工厂执行阶段
4. 无法表达物流回填阶段
5. 无法表达部分完成
6. 前端进度条是固定映射，不是业务真实进度

### 8.1.1 在履约行状态之前，先有需求接手与输入状态

对 `membership_entitlement` 或更复杂的 `DemandLine` 而言，在生成 `FulfillmentLine` 之前，建议先有两层前置状态：

1. `recipient_input_state`

- `not_required`
- `waiting_for_input`
- `partially_collected`
- `ready`
- `waived`
- `expired`

2. `routing_disposition`

- `pending_intake`
- `accepted`
- `deferred`
- `excluded_manual`
- `excluded_duplicate`
- `excluded_revoked`

这两层主要服务：

- 需求导入侧
- 权益候选侧
- 波次接手前的路由判断

它们不应被直接塞进 `FulfillmentLine` 执行状态。

### 8.2 V2 行级状态模型

建议把 `FulfillmentLine` 的生命周期拆成多维状态。

#### allocation_state

- `draft`
- `ready`
- `changed_after_submit`
- `locked`

#### address_state

- `missing`
- `ready`
- `invalid`

#### supplier_state

- `not_submitted`
- `submitted`
- `accepted`
- `producing`
- `partially_shipped`
- `shipped`
- `canceled`

#### channel_sync_state

- `not_required`
- `unsupported`
- `pending`
- `synced`
- `manual_confirmed`
- `skipped`
- `failed`

这里需要特别区分：

- `not_required`
  - 业务上本来就不要求回填
- `unsupported`
  - 理论上有闭环需求，但当前业务面或集成能力不支持自动回填
- `manual_confirmed`
  - 没有自动成功回填，但操作员已完成人工闭环确认

### 8.2.1 轻量手动进度能力应该如何存在

当前更稳妥的方向，不是允许用户任意改写底层状态枚举，而是：

- 保持底层事实状态尽量由真实对象自动推导
- 只在“闭环判定”这类边界问题上允许显式人工决策

建议引入一类轻量决策记录，例如：

- `mark_sync_unsupported`
- `mark_sync_skipped`
- `mark_sync_completed_manually`
- `confirm_offline_handover`
- `reopen_closure`

每条决策记录至少应带：

- `reason_code`
- `operator_id`
- `note`
- `evidence_ref`
- `created_at`

这个模型的重点是：

- 人工动作记录“为什么允许闭环”
- 而不是直接把真实状态字段改成一个更好看的值

### 8.3 V2 波次聚合状态

`Wave` 不再直接表达底层细节，而是表达聚合阶段：

- `draft`
- `allocating`
- `address_blocked`
- `ready_to_submit`
- `submitted_to_supplier`
- `partially_shipped`
- `shipped`
- `syncing_back`
- `awaiting_manual_closure`
- `closed`

其中：

- `syncing_back` 表示系统仍在等待自动或文档式闭环结果
- `awaiting_manual_closure` 表示自动事实已基本完成，但还缺少人工闭环决策

### 8.4 V2 波次进度展示

前端应放弃“伪百分比硬编码”，改成可解释的漏斗指标：

- 总履约行数
- 地址就绪行数
- 已提交工厂行数
- 已回传快递行数
- 已完成来源渠道回填行数
- 已人工确认闭环行数
- 失败回填行数

波次首页和 Dashboard 最终应显示：

- 阶段标签
- 分业务面执行摘要
- 风险计数
- 回填失败计数
- 待人工闭环计数

而不是只显示一个含糊的“已导出/待补全”。

对于导入侧、共享总览页或相关 Dashboard，还应单独显示：

- `accepted`
- `waiting_for_input`
- `deferred`
- `excluded_manual`
- `excluded_duplicate`
- `excluded_revoked`

---
