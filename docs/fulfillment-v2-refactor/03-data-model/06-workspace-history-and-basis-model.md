# 工作区历史与 Basis 模型

> 树状撤销/重做、工作区历史持久化、外部对象的 basis 引用。

## 两层概念

| 层 | 回答 |
|----|------|
| `Workspace History` | 本地工作区改了什么、能否撤销到更早节点 |
| `External Basis` | 导出/物流/回填当时依赖哪个本地结果、当前是否已偏离 |

`Ctrl+Z` 只回退本地 head，不回滚外部世界。偏离进入 `basis_drift_status` + `review_requirement` 双轴提示。

## HistoryScope

表示哪块工作区拥有独立历史树。字段：`id`、`scope_type`（`wave`/`template`/`product_library`）、`scope_key`、`current_head_node_id`、时间戳。

全应用共用 history 基础设施，优先做稳 `wave` scope。

## HistoryNode

一次用户意图级操作。字段：`id`、`history_scope_id`、`parent_node_id`、`preferred_redo_child_id`、`command_kind`、`command_summary`、`patch_payload`、`inverse_patch_payload`、`checkpoint_hint`、`projection_hash`、`created_by`、`created_at`。

一个 node = 一次用户意图（如"批量添加规则"），不等于每条底层派生写入。重算、overview 刷新不各自生成 node。

## 树状分支

每个 node 一个 `parent_node_id`，可有多个 children。撤销到旧节点后编辑 → 新 child，旧未来保留。

## HistoryCheckpoint

周期性保存完整快照。字段：`id`、`history_scope_id`、`history_node_id`、`snapshot_payload`、`schema_version`、`created_at`。

日常 node 保存 patch，每若干步打 checkpoint，平衡重放链长度和存储压力。

## HistoryPin

保护被外部对象引用的 node 不被 GC。字段：`id`、`history_node_id`、`pin_kind`（`supplier_order_basis`/`shipment_basis`/`channel_sync_basis`/`manual_pin`）、`ref_type`、`ref_id`、`created_at`。

## 外部对象 basis 引用

`SupplierOrder`/`Shipment`/`ChannelSyncJob` 各保留：`basis_history_node_id`、`basis_projection_hash`、`basis_payload_snapshot`。

冻结的是 basis 引用和必要投影，不是整波次完整快照。

## BasisComparisonProjection

计算型投影，不硬落进外部对象表。字段：`basis_kind`、`basis_drift_status`、`review_requirement`、`drift_reason_codes`、`last_compared_at`。

## 首版不做的事

不自动回滚外部动作、不每步存完整镜像、不做复杂 history merge、不做所有页面细粒度命令回放。

首版目标：scope 化、树状分支不丢、持久化、basis 可追踪、wave 优先。
