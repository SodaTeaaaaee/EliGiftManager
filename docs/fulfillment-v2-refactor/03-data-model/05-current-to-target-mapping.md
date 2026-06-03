# 当前模型到目标模型的映射

> 历史对照和语义映射。新代码不再桥接旧结构。

## 实体映射

| V1 实体 | V2 语义 | 策略 |
|---------|---------|------|
| `Member` | `CustomerProfile` | 语义一次到位 |
| `MemberNickname` | `CustomerIdentity` 辅助 | 并入 profile/identity |
| `MemberAddress` | `CustomerAddress` | 保留并增强 |
| `Wave` | `Wave` | 扩充生命周期 |
| `Wave.Status` | `lifecycle_stage` + `progress_snapshot` | 直接收敛到目标字段 |
| `WaveMember` | `WaveParticipantSnapshot` | 泛化 |
| `ProductMaster` | `ProductMaster` | 保留 |
| `Product` | `Product` | 保留 |
| `ProductTag` | `AllocationPolicyRule` | 语义升级 |
| `DispatchRecord` | `FulfillmentLine` | 直接收敛 |
| `TemplateConfig` | `IntegrationProfile` + `DocumentTemplate` | 语义升级 |
| 导出过程 | `SupplierOrder` + `SupplierOrderLine` | 升级为可追踪对象 |
| 发货导入 | `Shipment` + `ShipmentLine` | 升级为物流对象 |
| 回填脚本 | `ChannelSyncJob` + `ChannelSyncItem` | 升级为可追踪/可重试对象 |
| 缺失的 undo/redo | `HistoryScope` + `HistoryNode` + `Checkpoint` + `Pin` | 新增 |

## 关键保留原则

1. 历史波次是快照，不受全局实体后续变动污染
2. 全局商品主档和波次商品快照继续分层
3. 履约真相有单一归宿
4. 工厂执行和物流回填不能只靠瞬时脚本
5. 本地历史与外部执行现实分层
6. 会员规则层与共享调整层分层

## 旧概念需要拆开

1. "会员" → `CustomerProfile` + `CustomerIdentity` + `WaveParticipantSnapshot`
2. "需求真相" vs "执行真相" → `DemandDocument/DemandLine` vs `FulfillmentLine` vs `FulfillmentAdjustment`
3. "导出成功" vs "履约闭环" → `SupplierOrder` + `Shipment` + `ChannelSyncJob` + 人工闭环
4. "模板配置" vs "来源业务面语义" → `IntegrationProfile` + `DocumentTemplate` + `Service/Connector`
5. "页面编辑结果" vs "工作区历史" → 当前 head + 树状 `HistoryNode` + 外部 basis 引用
