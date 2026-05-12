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
- `export_snapshot_outdated`

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

这里还要强调：

- 这些状态表达的是当前工作区的最新已知执行投影
- 不是“因为进入某状态所以后续禁止编辑”
- 即使已经出现工厂提交、物流回传或渠道回填，操作者仍可继续修改波次内容

其中 `export_snapshot_outdated` 的含义应理解为：

- 当前履约结果已经偏离最近一次导出或提交时的本地快照
- 它是辅助提示
- 不是锁定信号，也不是历史真相声明

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

### 8.2.2 事实状态之外，还需要辅助提示信号

在 V2 中，某些“看起来像状态”的内容更适合做成计算型辅助信号，而不是新的硬状态枚举。

建议先保留以下工作名级别的辅助信号：

- `export_snapshot_outdated`
- `shipment_basis_outdated`
- `channel_sync_basis_outdated`
- `adjustment_requires_review`

这些信号只表达：

- 当前工作区结果是否已经偏离最近一次导出、回传或回填所依据的基础
- 当前已有的共享调整层例外是否需要复核

它们应当：

- 只用于 UI 提示、波次总览聚合和复核入口
- 不改变编辑权限
- 不替代底层事实状态
- 不把系统变成历史账本锁

其中，是否要把这些信号放到 `Wave Overview` 以外的行级页面展示，首版可以先按最小可用原则处理，但“是否存在偏离”和“是否需要复核”的语义本身应在实施前先定清。

### 8.2.3 什么情况属于复核，什么情况不属于

当前已经可以先定清一条重要边界：

- “出现负数中间量”本身不等于需要复核

更准确地说：

- 规则贡献层允许正负贡献
- 共享调整层允许正负 delta
- 最终结果被压到 0 也可能是合理业务语义

因此以下情况更适合视为“正常但可观察”的现象，而不是 `requires_review`：

- 某条规则贡献为负
- 某条共享调整 delta 为负
- `base + delta < 0`，最终执行结果被压到 0

更适合进入 `adjustment_requires_review` 或更强复核提示的，是以下情况：

- 共享调整层引用的基础对象已经不存在
- 共享调整层引用的对象仍存在，但 target 已变得歧义
- 最近一次工厂导出所依赖的履约投影，与当前工作区已出现结构性错配
- 最近一次物流导入或渠道回填所依赖的 basis，已无法无歧义映射到当前对象

可以把它理解为：

- 复核信号优先关注“对象和 basis 是否 still valid”
- 而不是优先关注“算术结果看起来是否刺眼”

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

其中波次内独立的 `Wave Overview` 页面还应承担：

- 作为 `Membership Allocation` / `Demand Mapping` 与 `Adjustment Review` 之间的会合页
- 采用只读优先原则，不承担主要编辑职责
- 汇总显示需求接手状态、输入等待状态、异常分桶和进入下一步的阻塞原因
- 明确区分“需要回前置页面处理的问题”和“可以进入共享调整层处理的问题”
- 在无需继续编辑时，允许直接进入后续执行准备阶段

这里的一个重要判断原则是：

- `Adjustment Review` 无法表达或无法安全处理的问题，应明确引导回前置页面
- 能由 `Adjustment Review` 处理的问题，才适合在该页之后进入共享调整层

对于导入侧、`Wave Overview` 或相关 Dashboard，还应单独显示：

- `accepted`
- `waiting_for_input`
- `deferred`
- `excluded_manual`
- `excluded_duplicate`
- `excluded_revoked`

关于 `Wave Overview` 的具体聚合维度，当前还不建议一次性定死完整信息架构。

更稳妥的策略是：

- 先以最小可用的一组业务分桶、异常分桶和阻塞指标落地
- 再根据真实使用反馈迭代收敛

首版更建议先保证以下最小可用分组：

- 可直接继续
- 需要返回 `Membership Allocation`
- 需要返回 `Demand Mapping`
- 建议进入 `Adjustment Review`
- 等待输入 / 地址 / 资格补齐
- 未纳入本系统处理

---
