# Adjustment 重放算法

> `FulfillmentAdjustment` 在基础分配结果重建后的重放语义。

## 触发时机

用户跳回前置步骤修改规则/映射导致基础重建、显式请求重算。

## 重放顺序

1. **跨层级**：先重建基础分配结果，再重放调整层
2. **同层级内**：按 `created_at` 升序逐条重放

## 合并规则

- 不做隐式合并，逐条独立应用
- `replace`：替换商品（不是数量），后续 delta 基于替换后结果计算
- 最终结果：`resolved = max(base + sum(deltas), 0)`

## 失败处理

### 失败类型

- **orphaned**：target 已被删除
- **ambiguous**：target 不再唯一

### 默认模式：整体暂停

遇到第一条无法重放的 adjustment 时暂停，报告失败原因，等待用户手动处理后继续。

### 可选模式：标记并继续

用户可在设置切换：失败的标记为 `review_requirement = required`，其余继续正常重放。

### 失败后用户动作

删除失效 adjustment / 修改 target 指向新对象 / 重新创建替代。

## 与 HistoryNode 的关系

重放不产生新 `HistoryNode`（是派生副作用，不是用户意图）。用户手动处理失效 adjustment 才产生新 node。重放导致的变化归属于触发重放的那个 node。
