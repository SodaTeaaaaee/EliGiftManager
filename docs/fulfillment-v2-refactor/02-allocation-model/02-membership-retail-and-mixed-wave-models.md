# 会员、零售与混合波次模型

本文件定义三类波次在统一工作流中的不同入口方式，重点说明为什么会员与零售不能共享同一种初始分配方式。

### 7.0 分配工作流的统一结构

无论来源是会员权益还是零售订单，V2 都建议把“生成最终履约结果”的过程拆成三层。

1. 初始分配层

回答：

- “第一版履约结果从哪里来？”

会员权益型需求：

- 主要来自规则推导

零售订单型需求：

- 主要来自上游需求行直入

2. 调整层

回答：

- “在第一版履约结果之上，还做了哪些修正？”

这层负责承接：

- 人工协商
- 赠送
- 补发
- 减送
- 替换

3. 最终履约层

回答：

- “最终到底发什么？”

这层统一收敛到 `FulfillmentLine`。

### 7.0.1 会员与零售不应该共用同一种初始分配方式

V2 的目标是“最终收敛”，不是“前置步骤强统一”。

因此：

- 会员权益型需求主要走 `policy-driven`
- 零售订单型需求主要走 `demand-driven`

两者都可以进入同一波次，也都可以进入同一套最终履约真相，但不应被强迫使用同一种初始分配引擎。

### 7.1 会员权益型波次

目标链路：

1. 导入权益来源数据，生成 `DemandDocument/DemandLine`
2. 判断 `EntitlementAuthority`
3. 判断 `ObligationTriggerKind`
4. 采集或等待 `RecipientInputState`
5. 记录 `RoutingDisposition`
6. 对 `accepted` 的项生成 `WaveParticipantSnapshot`
7. 导入商品
8. 在规则编辑器中配置 `AllocationPolicyRule`
9. 仅对已具备执行条件的项，通过 `ReconcileWave` 生成第一版 `FulfillmentLine`
10. 通过调整层做必要的例外修正
11. 绑定地址或补全最终执行参数
12. 生成并导出 `SupplierOrder`
13. 导入工厂发货回传，生成 `Shipment`
14. 将 `Shipment` 转换为来源渠道回填任务
15. 按来源渠道能力决定是否执行 `ChannelSyncJob`
16. 波次关闭

这里需要特别强调：

- `excluded_manual` 表示本系统这次不接手处理
- 不表示本系统拥有系统外履约完成真相

### 7.2 零售订单型波次

目标链路：

1. 导入零售订单
2. 生成 `DemandDocument/DemandLine`
3. 如有会员限定购买，记录 `eligibility_context_ref`
4. 判断 `ObligationTriggerKind`
5. 记录 `RoutingDisposition`
6. 对 `accepted` 的项进入本系统流程
7. 归并买家资料到 `CustomerProfile/Identity`
8. 生成 `WaveParticipantSnapshot`
9. 从 `DemandLine` 直接或半自动生成第一版 `FulfillmentLine`
10. 只在必要处通过调整层做显式修正
11. 地址校验
12. 生成并导出 `SupplierOrder`
13. 导入工厂发货回传，生成 `Shipment`
14. 转换并回填来源渠道
15. 波次关闭

### 7.3 混合波次

系统应允许同一个波次同时承接：

- 会员权益履约行
- 零售订单履约行

同时系统也应允许同一次导入或同一份需求快照里存在：

- 已进入本系统的项
- 暂缓进入本系统的项
- 明确不由本系统处理的项

但必须在行级保持来源可追踪：

- 哪条履约行来自哪类上游需求
- 哪条履约行是否需要回填来源渠道
- 哪条履约行来自同一来源渠道的哪个业务面

而在履约行之外，还应有单独统计说明：

- 哪些需求未纳入本系统处理
- 这些未纳入项是 `deferred`、`excluded_manual`、`excluded_duplicate` 还是 `excluded_revoked`
