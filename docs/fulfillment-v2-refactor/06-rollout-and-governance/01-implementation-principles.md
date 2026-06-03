# 实施原则

## 10.1 Greenfield 优先

旧业务代码被清理，V2 以 greenfield 方式重建。保留文档、备份分支、仓库历史和必要工程壳子，不保留旧业务实现的连续演进义务。

## 10.2 命名尽早收敛

新语义从第一天用新名（文档、service、DTO、页面、测试、API）。旧名只在历史文档和备份分支中存在。

### 一次到位清单

| V1 | V2 |
|----|-----|
| `Member` | `CustomerProfile` |
| `MemberNickname` | `CustomerIdentity` |
| `MemberAddress` | `CustomerAddress` |
| `WaveMember` | `WaveParticipantSnapshot` |
| `ProductTag` | `AllocationPolicyRule` |
| `DispatchRecord` | `FulfillmentLine` |
| `TemplateConfig` | `IntegrationProfile` + `DocumentTemplate` |

新增：`DemandDocument`、`DemandLine`、`SupplierOrder`、`Shipment`、`ChannelSyncJob`、`FulfillmentAdjustment`、`HistoryScope`/`HistoryNode`/`HistoryCheckpoint`/`HistoryPin`。

## 10.3 先领域后 UI

优先级：领域语言 → 数据结构 → 状态投影 → 导入导出服务 → 页面 UI。

## 10.4 先保真后自动化

优先保证准确落库、准确追踪来源、准确追踪物流与回填。API 自动化和异步 worker 后续补。

## 10.5 语义边界先定，视图排布可反馈驱动

必须先定清：编辑权归属、调整层边界、重算保留策略、跨步骤跳转行为、basis 偏离提示。分组命名、导航样式、计数位置可反馈迭代。

## 10.6 模式收敛优先于字段堆叠

如果一个对象同时承担领域语义、页面入口、流程状态、偏离提示、外部差异，说明已偏离正确边界。

## 10.7 树状历史以"用户意图"为节点

一个 history node 代表一次用户意图。重算、副作用、overview 刷新不应各自生成独立撤销节点。

## 10.8 本地历史与外部 basis 分离

- 本地 undo/redo 只修改 `HistoryScope` 当前 head
- `SupplierOrder`/`Shipment`/`ChannelSyncJob` 保留自己的 basis 引用
- 偏离进入 `basis_drift_status` + `review_requirement` 双轴提示

## 10.9 快捷键不劫持文本输入

普通工作区上下文 → `Ctrl+Z` 作用于 HistoryScope。input/textarea 焦点内 → 优先尊重原生撤销。

## 10.10 历史持久化但不用每步完整快照

平时 patch/inverse patch，周期性 checkpoint，外部 basis 引用节点 pin，旧分支 GC。
