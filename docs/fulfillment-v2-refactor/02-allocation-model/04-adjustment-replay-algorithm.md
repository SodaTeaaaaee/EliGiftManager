# Adjustment 重放算法

本文件定义 `FulfillmentAdjustment` 在基础分配结果重建后的精确重放语义，包括重放顺序、合并规则和失败处理。

## 1. 重放触发时机

当以下情况发生时，系统需要重放调整层：

- 用户跳回前置步骤修改了规则或映射，导致基础分配结果重建
- 用户在 `Membership Allocation` 或 `Demand Mapping` 中做了变更后触发 `ReconcileWave`
- 显式请求重算

## 2. 重放顺序

### 2.1 跨层级：按步骤层级顺序

重建过程严格按波次编辑步骤的层级顺序执行：

1. 先重建基础分配结果（来自 `AllocationPolicyRule` 或 `DemandMapping`）
2. 再重放调整层（`FulfillmentAdjustment`）

这保证了"用户跳回更早步骤修改数据"的语义正确性——前置步骤的变更先生效，调整层再基于新的基础结果重放。

### 2.2 同层级内：按 `created_at` 升序

同属调整层的多条 `FulfillmentAdjustment` 之间，按 `created_at` 升序逐条重放。

理由：保持用户操作的时间因果关系。用户先做的调整先生效，后做的调整基于前面的累积结果。

## 3. 合并规则

### 3.1 不做隐式合并

多条 adjustment 作用于同一 target 时，不做隐式合并，逐条独立应用：

- 每条 adjustment 的 delta 独立作用于当前累积结果
- 例如：`base=2, adj1=+1, adj2=-1 → resolved = max(2+1-1, 0) = 2`

### 3.2 replace 类型的精确语义

- `replace` 表示"把 from_product 换成 to_product"
- 替换的是商品，不是数量
- 如果 from_product 存在但数量已变化，replace 仍然生效
- 后续 delta 类 adjustment 基于替换后的结果继续计算

### 3.3 最终结果约束

- `resolved = max(base + sum(deltas), 0)`
- 最终执行结果非负
- 结果被压到 0 本身不是错误，只是"不产生可执行履约行"

## 4. 重放失败处理

### 4.1 失败类型

- **orphaned**：target 已被删除（例如重算后某个参与者不再存在于基础结果中）
- **ambiguous**：target 不再唯一（例如重算后产生了多条同 product 的基础行，无法确定 adjustment 应作用于哪一条）

### 4.2 默认处理模式：整体暂停

默认行为：

- 遇到第一条无法重放的 adjustment 时，整体暂停重放
- 向用户报告失败原因和涉及的 adjustment
- 等待用户手动处理（修正、删除或重新指定 target）后继续

### 4.3 可选处理模式：标记并继续

用户可在设置界面切换为此模式：

- 失败的 adjustment 标记为 `orphaned` 或 `ambiguous`，进入 `review_requirement = required`
- 其余 adjustment 继续正常重放，不因一条失败而整体中断
- 失败的 adjustment 在 `Wave Overview` 中显示为待复核项

### 4.4 失败后的用户动作

无论哪种模式，用户都可以：

- 删除失效的 adjustment
- 修改 adjustment 的 target 指向新的有效对象
- 重新创建一条新的 adjustment 替代失效的

## 5. 与 HistoryNode 的关系

- 重放本身不产生新的 `HistoryNode`（它是派生副作用，不是用户意图）
- 用户手动处理失效 adjustment 的操作才产生新的 `HistoryNode`
- 如果重放导致最终结果与重放前不同，这个变化归属于触发重放的那个 `HistoryNode`（即前置步骤的修改操作）

---
