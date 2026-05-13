# 当前模型到目标模型的映射

本文件用于帮助实施时识别哪些结构可以保留、哪些结构需要泛化、哪些旧术语需要停用。

在当前 greenfield 路线下，这份文件的主要作用是：

- 历史对照
- 语义映射
- 考古参考

它不应被理解为：

- 新代码必须逐层桥接旧结构的实施义务

## 6. 当前结构到目标结构的映射

### 6.1 当前实体与目标实体映射

| 当前实体 | 目标语义 | 处理策略 |
| --- | --- | --- |
| `Member` | `CustomerProfile` 的桥接实现 | 语义应尽快一次到位；若物理表暂不改名，也只作为存储层别名 |
| `MemberNickname` | `CustomerProfile` 的昵称历史 | 保留，后续可并入 profile/identity 辅助表 |
| `MemberAddress` | `CustomerAddress` | 保留并增强 |
| `Wave` | `Wave` | 保留并扩充生命周期语义 |
| `Wave.Status` | `Wave.lifecycle_stage` + `progress_snapshot` + 辅助提示信号 | 若旧实现路径尚未切断，可短期保留兼容投影；新实现应直接收敛到目标字段 |
| `WaveMember` | `WaveParticipantSnapshot` | 泛化，不再只承载会员 |
| `ProductMaster` | `ProductMaster` | 直接保留 |
| `Product` | `Wave Product Snapshot` | 直接保留 |
| `ProductTag` | `AllocationPolicyRule` / `AllocationContribution` 的桥接实现 | 会员权益波次可短期继续接线；新设计不应再把它当主语义 |
| 当前负数 tag + `ReconcileWave` 中间求和 | `AllocationContribution -> Base Allocation Result` | 负号保留在贡献层；基础结果与最终执行结果不应为负 |
| 当前显式用户覆盖 / 手工修正 | `FulfillmentAdjustment` | 逐步从隐式覆盖演进为显式共享调整层对象 |
| `DispatchRecord` | `FulfillmentLine` | 语义应直接收敛到最终名；若历史代码尚未清理，仅保留兼容别名 |
| 当前工厂导出文件过程 | `SupplierOrder` + `SupplierOrderLine` | 从瞬时导出动作升级为可追踪对象，并记录 basis |
| 当前工厂发货导入过程 | `Shipment` + `ShipmentLine` | 从临时回传数据升级为物流对象，并记录 basis |
| 当前来源渠道回填脚本 / 手动操作 | `ChannelSyncJob` + `ChannelSyncItem` | 升级为可追踪、可重试、可失配提示的回填对象 |
| `TemplateConfig` | `IntegrationProfile` + `DocumentTemplate` + `IntegrationProfileTemplateBinding` | 语义应直接升级到 profile 体系；若实现分步，别名只做短期桥接 |
| 当前缺失的全局撤销 / 重做持久层 | `HistoryScope` + `HistoryNode` + `HistoryCheckpoint` + `HistoryPin` | 新增；先接入 `wave`，再复用到模板 / 商品等工作区 |

补充收敛说明：

- 旧的“手工修正需求”如果表达的是上游事实，应迁移为 `DemandDocument(kind = membership_entitlement|retail_order, capture_mode = manual_entry)`
- 旧的“波次内最终修正”应迁移为 `FulfillmentAdjustment`
- 旧的波次级 `allocation_mode` 不建议保留为目标真相字段，更适合改为只读投影

### 6.2 关键保留原则

以下设计原则必须保持：

1. 历史波次必须是快照，不受全局实体后续变动污染
2. 全局商品主档和波次商品快照必须继续分层
3. 履约真相必须有单一归宿
4. 工厂执行和物流回填不能只靠导入导出瞬时脚本
5. 本地工作区历史与外部执行现实必须分层，不能把 undo/redo 伪装成外部世界回滚
6. 会员规则层与共享调整层必须分层，不能把 `Adjustment Review` 重新长成第二套规则引擎

### 6.3 旧概念需要拆开的地方

实施时最容易出错的，不是“表要不要改名”，而是继续拿旧概念承担过多语义。

当前至少要明确拆开以下几组旧概念：

1. “会员”与“履约对象”

- 旧模型里，`Member` 很容易被误当成唯一主角
- 目标模型里，应拆成：
  - `CustomerProfile`
  - `CustomerIdentity`
  - `WaveParticipantSnapshot`

2. “需求真相”与“执行真相”

- 旧模型里，很多内容会被压进 `DispatchRecord`
- 目标模型里，应拆成：
  - `DemandDocument / DemandLine`
  - `FulfillmentLine`
  - `FulfillmentAdjustment`

3. “导出成功”与“履约闭环”

- 旧模型里，导出工厂文件几乎等于流程结束
- 目标模型里，应拆成：
  - `SupplierOrder`
  - `Shipment`
  - `ChannelSyncJob`
  - 人工闭环决策记录

4. “模板配置”与“来源业务面语义”

- 旧模型里，`TemplateConfig` 同时承担字段映射和部分业务解释
- 目标模型里，应拆成：
  - `IntegrationProfile`
  - `DocumentTemplate`
  - `Service / Connector`

5. “页面编辑结果”与“工作区历史”

- 旧模型里，很多编辑状态更接近页面临时态
- 目标模型里，应拆成：
  - 当前工作区 head
  - 树状 `HistoryNode`
  - 外部对象 basis 引用

### 6.4 History 与 Basis 迁移补充

当前阶段虽然不需要为旧生产数据背复杂包袱，但仍应在目标模型里先把以下迁移方向写清：

1. 新增 history 相关表

- `history_scopes`
- `history_nodes`
- `history_checkpoints`
- `history_pins`

2. 新增外部对象 basis 字段

- `SupplierOrder.basis_history_node_id`
- `SupplierOrder.basis_projection_hash`
- `SupplierOrder.basis_payload_snapshot`
- `Shipment.basis_history_node_id`
- `Shipment.basis_projection_hash`
- `Shipment.basis_payload_snapshot`
- `ChannelSyncJob.basis_history_node_id`
- `ChannelSyncJob.basis_projection_hash`
- `ChannelSyncJob.basis_payload_snapshot`

3. 明确迁移边界

- 这些 basis 字段回答的是“当时依赖了哪个本地结果”
- 它们不等于“永久冻结整个旧 wave 数据库快照”
- 它们的核心价值是支持：
  - 偏离提示
  - 复核判断
  - 外部对象与当前工作区的可解释关联

4. 当前阶段不背旧兼容包袱

- 由于项目仍在早期阶段
- 若某些过渡问题在删库后自然消失，可不为其设计额外长期兼容层
- 迁移重点应放在目标语义清晰，而不是为旧数据形状补复杂绕路
- 这里的分阶段仅指实现顺序，不是指长期保留旧名旧结构作为主语义

---
