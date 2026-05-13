# 工作区历史与 Basis 模型

本文件定义树状撤销/重做、工作区历史持久化，以及外部导出 / 回传对象如何引用本地工作区 basis。

### 5.10 为什么需要独立的工作区历史层

当前讨论已经确认：

- 撤销 / 重做不能只是前端页面里的线性临时栈
- 它需要保留分支，而不是“回退后再编辑就把旧未来直接抹掉”
- 它需要在软件关闭后继续保留
- 它还要和 `SupplierOrder / Shipment / ChannelSyncJob` 的 basis 引用协同工作

因此更稳妥的方向是：

- 把工作区历史建模成独立的持久化层
- 把外部对象依赖的 basis 建模成独立引用
- 明确区分“本地历史”和“外部现实”

### 5.10.1 本地工作区历史不等于外部世界回滚

这里必须先把两个概念拆开：

1. `Workspace History`

- 回答：
  - “本地工作区刚刚改了什么？”
  - “能否撤销到更早的某个节点？”

2. `External Basis`

- 回答：
  - “这次导出 / 物流导入 / 渠道回填，当时依赖的是哪个本地结果？”
  - “当前工作区是否已经偏离了它当时依赖的基础？”
  - “这种偏离现在是否已经需要人工复核？”

这意味着：

- `Ctrl+Z` 只回退本地工作区 head
- 不意味着工厂、物流平台、来源渠道真的跟着回滚
- 外部对象更适合通过 basis 引用进入：
  - `basis_drift_status`
  - `review_requirement`
  这两条独立提示轴

### 5.10.2 `HistoryScope`

建议新增。

建议字段：

- `id`
- `scope_type`
  - `wave`
  - `template`
  - `product_library`
  - 其他未来工作区类型
- `scope_key`
- `current_head_node_id`
- `created_at`
- `updated_at`

说明：

- `HistoryScope` 表示“哪一块工作区拥有自己独立的一棵历史树”
- 全局快捷键可以统一存在
- 但真正撤销 / 重做的，应始终是当前激活 scope

当前方向已经确认：

- 全应用应共用同一套 history 基础设施
- 但首版优先完善 `wave` scope

### 5.10.3 `HistoryNode`

建议新增。

建议字段：

- `id`
- `history_scope_id`
- `parent_node_id`
- `preferred_redo_child_id`
- `command_kind`
- `command_summary`
- `patch_payload`
- `inverse_patch_payload`
- `checkpoint_hint`
- `projection_hash`
- `created_by`
- `created_at`

说明：

- `HistoryNode` 表示一次“用户意图级操作”
- 它不应等于每一条底层派生写入

例如：

- “批量给 12 个商品添加某个身份规则”更接近 1 个 node
- “导入一批会员”更接近 1 个 node
- “确认一次共享调整”更接近 1 个 node

而不应把以下内容再拆成额外 node：

- `ReconcileWave`
- 状态重算
- 导出快照失效
- overview 计数刷新

这样做的原因是：

- 历史节点必须可读
- 历史图必须可导航
- 树状分支必须围绕用户意图，而不是围绕系统副作用

### 5.10.4 历史结构应优先满足“树状分支”

当前需求并不要求复杂合并历史。

更稳妥的最小模型是：

- 每个 node 只有一个 `parent_node_id`
- 但可以有多个 children
- 当用户撤销到旧节点后再次编辑，就创建一个新 child
- 旧未来分支继续保留

这在用户语义上更像一棵树。

实现层即使内部按 DAG 泛化，也应先保证这组树状语义。

### 5.10.5 `HistoryCheckpoint`

建议新增。

建议字段：

- `id`
- `history_scope_id`
- `history_node_id`
- `snapshot_payload`
- `schema_version`
- `created_at`

说明：

- 不建议每一步都持久化整工作区完整快照
- 更稳妥的方式是：
  - 日常 node 保存 patch / inverse patch
  - 每隔若干步保存 checkpoint

这样可以在：

- 减少重放链长度
- 降低恢复成本
- 控制存储压力

之间取得平衡

### 5.10.6 `HistoryPin`

建议新增。

建议字段：

- `id`
- `history_node_id`
- `pin_kind`
  - `supplier_order_basis`
  - `shipment_basis`
  - `channel_sync_basis`
  - `manual_pin`
- `ref_type`
- `ref_id`
- `created_at`

说明：

- 某些历史节点不能被普通 GC 随意裁掉
- 因为外部对象仍然依赖它们作为 basis

典型情况：

- 某个 `SupplierOrder` 是基于 node A 导出的
- 某个 `Shipment` 导入是基于 node B 的 basis 对齐出来的
- 某个 `ChannelSyncJob` 回填是基于 node C 的 payload 生成的

这类节点更适合通过 `HistoryPin` 被保留。

### 5.10.7 外部对象上的 basis 引用

除了 history 表本身，`SupplierOrder / Shipment / ChannelSyncJob` 也应补充 basis 相关字段。

更建议至少保留：

- `basis_history_node_id`
- `basis_projection_hash`
- `basis_payload_snapshot`

其作用分别是：

1. `basis_history_node_id`

- 说明该外部对象创建时，工作区 head 在哪个 node

2. `basis_projection_hash`

- 说明当时真正外发 / 外导入映射所依赖的投影签名

3. `basis_payload_snapshot`

- 说明当时真正使用过的轻量 materialized payload
- 它更像“必要依据快照”
- 不等于整波次完整备份

这也意味着：

- 所谓“冻结旧版本”，更适合冻结 basis 引用和必要投影
- 不必默认复制整个 wave 的全部状态

### 5.10.7.1 Basis 比较结果更适合做成投影对象

除了 basis 引用本身，系统还需要一个稳定的“比较结果”语义。

更稳妥的方式不是把所有结果都硬落进外部对象表，而是引入一个计算型或缓存型 projection，例如：

- `BasisComparisonProjection`

建议至少表达：

- `basis_kind`
  - `supplier_order_basis`
  - `shipment_basis`
  - `channel_sync_basis`
  - `adjustment_basis`
- `basis_drift_status`
  - `in_sync`
  - `drifted`
- `review_requirement`
  - `none`
  - `recommended`
  - `required`
- `drift_reason_codes`
- `last_compared_at`

这里的模式边界应保持清楚：

- basis 引用回答“当时依赖了什么”
- comparison projection 回答“现在还是否 still valid”

这样才能避免把：

- 历史引用
- 事实偏离
- 处理建议

重新压回一个单枚举字段。

### 5.10.8 与 `SupplierOrder / Shipment / ChannelSyncJob` 的关系

当前更稳妥的关系是：

- `HistoryScope / HistoryNode`
  - 负责本地工作区树状历史
- `SupplierOrder`
  - 负责工厂导出 / 提交工作区对象
- `Shipment`
  - 负责物流导入后的当前已知结果
- `ChannelSyncJob`
  - 负责来源渠道回填任务

它们之间通过 basis 引用关联，但不相互替代。

### 5.10.9 首版不建议做的事情

为了控制复杂度，首版不建议：

- 让所有外部动作都支持“自动真正回滚”
- 把每一步历史都存成完整数据库镜像
- 一开始就支持复杂 history merge
- 一开始就把所有页面都做成高度细粒度命令回放编辑器

首版更稳妥的目标是：

- scope 化
- 树状分支不丢
- 持久化
- basis 可追踪
- wave 优先接入

---
