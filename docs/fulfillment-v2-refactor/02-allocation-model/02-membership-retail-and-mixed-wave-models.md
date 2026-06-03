# 会员、零售与混合波次模型

> 三类波次在统一工作流中的不同入口方式。

## 统一三层结构

| 层 | 会员权益型 | 零售订单型 |
|----|-----------|-----------|
| 初始分配层 | `policy_driven`（规则推导） | `demand_driven`（需求行直入） |
| 调整层 | 人工协商/赠送/补发/减送 | 同左 |
| 最终履约层 | `FulfillmentLine` | 同左 |

两者可进入同一波次，但不应被强迫使用同一种初始分配引擎。

### 边界

1. 动态集合规则属于 `Membership Allocation`，不属于共享调整层
2. 有符号数量可存在于规则贡献和调整 delta，但最终执行结果不为负

## 会员权益型波次

导入权益数据 → `DemandDocument/DemandLine` → 判断 `EntitlementAuthority` + `ObligationTriggerKind` → 采集 `RecipientInputState` → 记录 `RoutingDisposition` → 生成 `WaveParticipantSnapshot` → 导入商品 → 配置 `AllocationPolicyRule` → `ReconcileWave` 生成 `FulfillmentLine` → `Wave Overview` → `Adjustment Review` → 地址校验 → 导出 `SupplierOrder` → 导入 `Shipment` → `ChannelSyncJob`/人工闭环 → 关闭

`excluded_manual` = 本系统不接手 ≠ 系统拥有外部履约完成真相。

## 零售订单型波次

导入订单 → `DemandDocument/DemandLine` → 记录 `eligibility_context_ref`（如有） → `RoutingDisposition` → 归并 `CustomerProfile` → `Demand Mapping` 生成 `FulfillmentLine` → `Wave Overview` → `Adjustment Review` → 地址校验 → 导出 → 发货回传 → 渠道回填 → 关闭

## 混合波次

- `Membership Allocation` 与 `Demand Mapping` 分别开放，写入同一套 `FulfillmentLine`
- 共享 `Wave Overview` → `Adjustment Review` → 后续执行链路
- 行级保持来源可追踪（哪条来自哪类需求、是否需要回填）
- 单独统计未纳入项（`deferred`/`excluded_manual`/`excluded_duplicate`/`excluded_revoked`）
