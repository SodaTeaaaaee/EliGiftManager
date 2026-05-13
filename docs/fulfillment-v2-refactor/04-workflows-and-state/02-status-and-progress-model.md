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

这里还要特别强调：

- `export_snapshot_outdated` 不应再被放入 `allocation_state`
- 它更适合留在辅助提示信号层
- 否则会把“事实状态”和“basis 偏离提示”重新混成一个状态机

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

### 8.2.2 事实状态之外，还需要两条独立提示轴

在 V2 中，某些“看起来像状态”的内容更适合做成计算型提示轴，而不是新的硬状态枚举。

更稳妥的底层模型是两条独立轴：

1. `basis_drift_status`

- `in_sync`
- `drifted`

它回答：

- “当前工作区结果，是否已经偏离这个 basis 当时依赖的基础？”

2. `review_requirement`

- `none`
- `recommended`
- `required`

它回答：

- “这种偏离现在是否需要人工复核？”

这两条轴分别表达：

- 事实信号
- 处理要求

不应在底层直接揉成一个综合枚举。

这符合当前 V2 已经采用的模式边界：

- state 与 signal 分层
- 事实判断与处理建议分层
- 底层双轴，UI 可再投影成单一总结状态

建议再补一个可选的原因列表：

- `drift_reason_codes`
  - `projection_changed`
  - `target_deleted`
  - `target_ambiguous`
  - `basis_mapping_lost`
  - `external_basis_stale`

这些轴与 reason code 只用于：

- UI 提示
- 波次总览聚合
- 复核入口
- 强提示触发

它们不应：

- 改变编辑权限本身
- 替代底层事实状态
- 把系统变成历史账本锁

这里还要明确命名习惯：

- 内部语义优先使用 `drift` / `stale` / `requires_review`
- UI 文案再投影成“已偏离最近一次导出基础”“需复核”等更自然的表达

### 8.2.2.1 为什么不直接做成一个综合状态枚举

如果把这些内容直接合成一个字段，例如：

- `healthy`
- `drifted`
- `warning`
- `requires_review`

短期看实现更省事，但很快会混掉两类问题：

- 当前到底发生了什么事实
- 系统建议用户下一步怎么做

对本项目来说，更稳妥的模式是：

- 底层保留 `basis_drift_status + review_requirement`
- UI 再按场景投影成一个总结状态

这样既能保持可扩展性，也能让界面保持简洁。

### 8.2.3 什么情况属于复核，什么情况不属于

当前已经可以先定清一条重要边界：

- “出现负数中间量”本身不等于需要复核

更准确地说：

- 规则贡献层允许正负贡献
- 共享调整层允许正负 delta
- 最终结果被压到 0 也可能是合理业务语义

因此以下情况更适合视为“正常但可观察”的现象，而不是直接进入 `review_requirement = required`：

- 某条规则贡献为负
- 某条共享调整 delta 为负
- `base + delta < 0`，最终执行结果被压到 0

更适合进入 `review_requirement = required` 的，是以下情况：

- 共享调整层引用的基础对象已经不存在
- 共享调整层引用的对象仍存在，但 target 已变得歧义
- 最近一次工厂导出所依赖的履约投影，与当前工作区已出现结构性错配
- 最近一次物流导入或渠道回填所依赖的 basis，已无法无歧义映射到当前对象

可以把它理解为：

- 复核信号优先关注“对象和 basis 是否 still valid”
- 而不是优先关注“算术结果看起来是否刺眼”

### 8.2.3.1 双轴的推荐合法组合

当前更稳妥的组合是：

1. `in_sync + none`

- 正常

2. `drifted + none`

- 已偏离，但当前还不影响可用性

3. `drifted + recommended`

- 已偏离，建议用户留意或在合适时复核

4. `drifted + required`

- 已偏离，且系统已经无法安全自动解释

通常不建议出现：

- `in_sync + required`

因为如果基础仍完全同步，复核要求通常不应凭空出现。

### 8.2.3.2 弱提示、强提示与进入复核

当前已确认的判断基准是：

- 以是否开始丢失可用性为准

更准确地说：

- `drifted + none`
  - 更接近弱提示
- `drifted + recommended`
  - 更接近较强提醒，但不应打断主流程
- `drifted + required`
  - 更接近强提示
  - 用户应被明确告知继续当前动作存在错误风险
  - 但是否立刻进入复核入口，可由用户决定

典型例子：

- 导出后修改了数量，但旧导出 basis 仍可解释
  - `drifted + recommended`
- shipment 已导入，但当前对象仍可稳定映射
  - `drifted + recommended`
- adjustment target 被删除，或已无法唯一重放
  - `drifted + required`

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
