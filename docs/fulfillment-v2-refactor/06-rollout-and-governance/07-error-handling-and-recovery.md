# 错误处理与恢复策略

## 导入失败

- **原子性**：导入作为单个 `HistoryNode`，不可恢复错误导致整体回滚
- **部分成功**：弹窗让用户选择"整体拒绝"或"跳过错误行"
- **错误报告**：错误行号、错误列、错误原因、原始行摘要

## ReconcileWave 重算失败

- **原子性**：事务内完成，失败则回滚，不产生 HistoryNode
- **典型原因**：规则引用不存在的商品/参与者、assignment/demand line lookup 失败、DB 写入失败
- **"无可执行项"是业务结果**，"数据读取失败"是系统错误——后者必须整体失败，不得静默降级为"创建 0 条"

## 历史 patch 损坏恢复

1. 从最近 checkpoint 重建到损坏 node 之前
2. 损坏 node 及后续子树标记为 `corrupted`
3. 用户可从其他未损坏分支继续，或从当前状态创建新 root

预防：每 20 node 打 checkpoint，pin 引用的 node 对应 checkpoint 优先保留，写入后校验 hash。

## 文件 I/O 失败

- 不修改工作区状态，不产生 HistoryNode，报告具体错误
- 导出失败：`SupplierOrder` 对象在事务内回滚，不出现"记录已导出但文件不存在"
