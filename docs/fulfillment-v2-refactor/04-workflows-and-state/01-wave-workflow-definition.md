# 波次工作流定义

本文件完整整理 V2 的波次工作流，包括统一分配结构、会员 / 零售 / 混合波次，以及 `WaveAllocationStep` 在工作流中的位置。

## 7. V2 工作流定义

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

### 7.0.2 波次编辑应按工作流阶段组织，而不是按单页视觉分栏组织

当前更合理的产品结构是：

1. `Demand Intake`
   - 导入和标准化上游需求
2. `Initial Allocation`
   - 生成第一版履约结果
3. `Adjustment Review`
   - 共享的修正与审查页面
4. `Execution Readiness`
   - 地址、缺项、异常校验
5. `Supplier Execution`
   - 工厂导出与执行跟踪
6. `Shipment Intake`
   - 工厂发货回传
7. `Channel Sync / Closure`
   - 回填来源渠道或人工闭环确认

在 `Initial Allocation` 阶段内部，再按语义拆成独立步骤和独立页面：

- `Membership Allocation`
  - 面向会员权益型需求
- `Demand Mapping`
  - 面向零售订单型需求

然后由所有波次统一进入：

- `Adjustment Review`

这样做的关键点不是“页面更多”，而是“不同语义拥有不同主入口”。

在这之前，还应补一个前置判断：

- 哪些需求已经被本系统 `accepted`
- 哪些仍在等待输入
- 哪些被 `deferred`
- 哪些被明确标记为 `excluded_manual`

### 7.1 会员权益型波次

建议链路：

1. 导入会员权益来源数据
2. 生成 `DemandDocument/DemandLine`
3. 判断 `EntitlementAuthority`
4. 收集或等待 `RecipientInputState`
5. 记录 `RoutingDisposition`
6. 对 `accepted` 的项归并或生成 `CustomerProfile/Identity`
7. 对 `accepted` 的项生成 `WaveParticipantSnapshot`
8. 导入商品
9. 进入 `Membership Allocation` 页面配置 `AllocationPolicyRule`
10. 仅对已具备执行条件的项，通过 `ReconcileWave` 生成第一版 `FulfillmentLine`
11. 进入共享的 `Adjustment Review` 页面做例外修正
12. 地址与缺项校验
13. 生成并导出 `SupplierOrder`
14. 导入工厂发货回传，生成 `Shipment`
15. 根据 `IntegrationProfile` 的能力与闭环策略，决定执行 `ChannelSyncJob` 还是人工确认闭环
16. 波次关闭

其中：

- `excluded_manual` 只表示“本系统这次不接手”
- 不表示“系统确认外部已经完成履约”

### 7.2 零售订单型波次

建议链路：

1. 导入零售订单
2. 生成 `DemandDocument/DemandLine`
3. 如有会员限定购买，补充 `eligibility_context_ref`
4. 记录 `RoutingDisposition`
5. 对 `accepted` 的项归并买家资料到 `CustomerProfile/Identity`
6. 生成 `WaveParticipantSnapshot`
7. 进入 `Demand Mapping` 页面，从 `DemandLine` 直接或半自动生成第一版 `FulfillmentLine`
8. 进入共享的 `Adjustment Review` 页面，仅在必要处做显式修正
9. 地址校验
10. 生成并导出 `SupplierOrder`
11. 导入工厂发货回传，生成 `Shipment`
12. 根据 `IntegrationProfile` 的能力与闭环策略，执行来源渠道回填或人工闭环确认
13. 波次关闭

### 7.3 混合波次

系统应允许同一个波次同时承接：

- 会员权益履约行
- 零售订单履约行
- 人工补发或履约修正项

但混合波次不应要求用户在单一页面里混用两套初始分配心智。

建议结构是：

1. 先按波次内真实需求，分别开放 `Membership Allocation` 与 `Demand Mapping`
2. 两个入口都写入同一套 `FulfillmentLine`
3. 再进入共享的 `Adjustment Review`
4. 后续共享同一套执行、发货、回填与闭环流程

必须在行级保持来源可追踪：

- 哪条履约行来自哪类上游需求
- 哪条履约行来自哪个 `source_surface`
- 哪条履约行需要自动回填、文档回填、人工确认，还是无需回填

同时在履约行之外，还应有单独统计说明：

- 哪些需求未纳入本系统处理
- 这些未纳入项是 `deferred`、`excluded_manual`、`excluded_duplicate` 还是 `excluded_revoked`

### 7.4 `WaveAllocationStep` 的产品含义

在 V2 中，`WaveAllocationStep` 不再等于一个旧的 tag 页面。

它更准确地表示：

- 一个围绕“初始履约生成 + 调整审查”组织起来的工作流阶段

在这个阶段内：

- `Membership Allocation` 保留当前 tag 系统的高效规则驱动体验
- `Demand Mapping` 提供零售订单到内部商品的映射入口
- `Adjustment Review` 作为两条链路的共享收敛层

### 7.5 会员分配 UX 保留原则

当前 tag 系统在会员权益场景下已经证明是高效方案。

因此 `WaveAllocationStep` 的设计必须满足：

1. 会员权益型波次默认仍以 `Membership Allocation` 为主入口
2. 现有商品中心批量配置体验不得明显退化
3. “先规则，再覆盖，再预览”的认知路径应尽量保留
4. 零售订单型需求的引入不应迫使会员用户先理解订单映射语义
5. 共享的 `Adjustment Review` 只能作为收敛层，不能反向变成会员分配主入口

---
