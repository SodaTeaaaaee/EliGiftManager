# 波次工作流定义

> V2 波次工作流：统一分配结构、会员/零售/混合波次。

## 统一三层结构

| 层 | 回答 | 说明 |
|----|------|------|
| 初始分配层 | "第一版履约结果从哪来" | 会员走 `policy_driven`，零售走 `demand_driven` |
| 调整层 | "做了哪些修正" | 人工协商/赠送/补发/减送/替换 |
| 最终履约层 | "最终发什么" | 统一收敛到 `FulfillmentLine` |

## 工作流阶段

1. `Demand Intake` — 导入和标准化上游需求
2. `Initial Allocation` — 生成第一版履约结果
   - `Membership Allocation` — 面向会员权益（规则驱动）
   - `Demand Mapping` — 面向零售订单（需求行直入）
3. `Wave Overview` — 只读聚合总览，阶段导航、异常分桶、偏离提示
4. `Adjustment Review` — 共享修正与审查
5. `Execution Readiness` — 地址、缺项、异常校验
6. `Supplier Execution` — 工厂导出与执行跟踪
7. `Shipment Intake` — 工厂发货回传
8. `Channel Sync / Closure` — 回填来源渠道或人工闭环

## 核心设计原则

- **非破坏性跨步骤往返**：页面切换不破坏其他步骤已确认的数据
- **领域能力边界稳定**：`AllocationPolicyService`、`DemandMappingService`、`FulfillmentAdjustmentService`、`WaveOverviewProjection` 各司其职
- **页面只是 UX 入口**：不是语义所有权的最终落点
- **`Adjustment Review` 边界**：可修改已接手对象的最终履约结果，不可新增全新参与者，不可创建动态 selector 规则

## 会员权益型波次链路

导入权益数据 → 生成 `DemandDocument/DemandLine` → 判断 `EntitlementAuthority` → 收集 `RecipientInputState` → 记录 `RoutingDisposition` → 归并 `CustomerProfile` → 生成 `WaveParticipantSnapshot` → 导入商品 → `Membership Allocation` 配置规则 → 生成 `FulfillmentLine` → `Wave Overview` → `Adjustment Review` → 地址校验 → 导出 `SupplierOrder` → 导入 `Shipment` → `ChannelSyncJob` 或人工闭环 → 关闭

## 零售订单型波次链路

导入订单 → 生成 `DemandDocument/DemandLine` → 记录 `RoutingDisposition` → 归并 `CustomerProfile` → `Demand Mapping` 生成 `FulfillmentLine` → `Wave Overview` → `Adjustment Review` → 地址校验 → 导出 → 发货回传 → 渠道回填 → 关闭

## 混合波次

- `Membership Allocation` 与 `Demand Mapping` 分别开放，写入同一套 `FulfillmentLine`
- 共享 `Wave Overview` → `Adjustment Review` → 后续执行链路
- 行级保持来源可追踪

## 会员分配 UX 保留原则

1. 会员权益型波次默认以 `Membership Allocation` 为主入口
2. 商品中心批量配置体验不得退化
3. "先规则，再覆盖，再预览"的认知路径保留
4. 零售订单不应迫使会员用户理解订单映射语义
