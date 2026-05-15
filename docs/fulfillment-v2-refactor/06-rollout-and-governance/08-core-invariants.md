# 核心不变量

本文件显式列举 V2 的核心领域逻辑不变量。这些不变量是测试的基础，也是实现时的硬约束。

## 1. 数据完整性不变量

1. 每条 `FulfillmentLine` 必须关联一个有效的 `WaveParticipantSnapshot`
2. 每条 `FulfillmentLine` 的 `quantity` ≥ 0（最终执行结果非负）
3. 每条 `SupplierOrderLine` 必须关联一条有效的 `FulfillmentLine`
4. 每个 `HistoryNode` 必须属于且仅属于一个 `HistoryScope`
5. 每个 `HistoryNode`（除 root）必须有且仅有一个 `parent_node_id`
6. 被 `HistoryPin` 引用的 node 不可被 GC 删除

## 2. 状态转换不变量

7. `routing_disposition` 只有从 `pending_intake` 才能转到 `accepted`
8. 只有 `routing_disposition = accepted` 的需求才能产生 `FulfillmentLine`
9. 外部事实状态不可通过本地 undo/redo 回退（例如 `channel_sync_state` 从 `synced` 不可通过 Ctrl+Z 回退到 `pending`）。但用户可以通过显式的正向操作（重新同步、重新导出、标记需重做）创建新的执行对象来替代旧的
10. `basis_drift_status = in_sync` 时，`review_requirement` 不应为 `required`

### 2.1 第 9 条的精确边界

"外部事实不可本地撤销"的核心理念是防止软件以为自己撤销了外部事实，而不是对用户的限制。

以下场景不违反此不变量：

- **工厂回传了错误数据**：用户重新导入正确的回传文件 → 生成新的 `Shipment` 替代旧的。这是"录入新的外部事实"，不是"假装旧事实没发生过"
- **导出时用了错误配置**：用户修改配置后重新导出 → 生成新的 `SupplierOrder`。旧的标记为 `superseded`
- **导出了文件但物理文件丢失**：用户重新导出同一批数据。`SupplierOrder` 对象仍在系统里，只是重新生成物理文件

区分：

- undo = "假装没发生过" → 不允许
- 重新操作 = "承认发生过，但创建新的来替代" → 允许

## 3. 业务规则不变量

11. 同一 `Wave` 内，同一 `CustomerProfile` 只产生一个 `WaveParticipantSnapshot`
12. `FulfillmentAdjustment` 的 target 必须是当前波次内已存在的对象
13. `AllocationPolicyRule` 的 selector 语义不能出现在 `FulfillmentAdjustment` 中（动态集合规则属于规则层，不属于调整层）

## 4. 不变量的验证时机

- 数据完整性不变量：在每次事务提交前由 service 层校验
- 状态转换不变量：在状态变更方法内由领域逻辑保证
- 业务规则不变量：在对应 service 的写入路径中校验

如果不变量被违反：

- 事务回滚
- 向用户报告具体违反的约束
- 记录到应用日志用于排查

---
