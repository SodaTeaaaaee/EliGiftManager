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

- 会员权益型需求主要走 `policy_driven`
- 零售订单型需求主要走 `demand_driven`

两者都可以进入同一波次，也都可以进入同一套最终履约真相，但不应被强迫使用同一种初始分配引擎。

### 7.0.2 波次编辑应按工作流阶段组织，而不是按单页视觉分栏组织

当前更合理的产品结构是：

1. `Demand Intake`
   - 导入和标准化上游需求
2. `Initial Allocation`
   - 生成第一版履约结果
3. `Wave Overview`
   - 波次内独立总览页
   - 采用只读优先原则
   - 承担阶段导航、聚合诊断、异常分桶与进入下一步判断
   - 还应汇总当前工作区相对最近一次导出、物流回传或渠道回填基础的偏离提示
4. `Adjustment Review`
   - 共享的修正与审查页面
5. `Execution Readiness`
   - 地址、缺项、异常校验
6. `Supplier Execution`
   - 工厂导出与执行跟踪
7. `Shipment Intake`
   - 工厂发货回传
8. `Channel Sync / Closure`
   - 回填来源渠道或人工闭环确认

在 `Initial Allocation` 阶段内部，再按语义拆成独立步骤和独立页面：

- `Membership Allocation`
  - 面向会员权益型需求
- `Demand Mapping`
  - 面向零售订单型需求

然后由所有波次统一进入：

- `Wave Overview`
  - 作为波次内独立页面
  - 不与复杂编辑层合并
  - 以只读优先的方式统一查看聚合结果、待处理项和是否具备进入下一步的条件
  - 汇总显示当前工作区相对最近一次导出、物流回传或渠道回填基础的偏离提示
  - 可作为进入 `Adjustment Review` 的显式关口，也可在无需编辑时直接放行到后续阶段

- `Adjustment Review`

这样做的关键点不是“页面更多”，而是“不同领域能力拥有不同主入口”。

在这之前，还应补一个前置判断：

- 哪些需求已经被本系统 `accepted`
- 哪些仍在等待输入
- 哪些被 `deferred`
- 哪些被明确标记为 `excluded_manual`

### 7.0.3 波次编辑必须支持非破坏性跨步骤往返

当前更合理的实现原则是：

- 不同步骤首先是同一波次数据的不同视角
- 不是一串会相互覆盖、相互抹平的孤立临时页面

因此：

- 用户应能通过更强的步骤向导，在波次任意步骤间快速跳转
- 跳回前置页面时，不应大幅破坏后续步骤已经形成的数据
- 页面切换本身不应隐式触发不可解释的大重写

更具体地说：

- `AllocationPolicyService`
  - 主要承载规则驱动初始分配能力
  - 同时承载 selector、规则贡献和动态集合规则能力
- `DemandMappingService`
  - 主要承载需求行到内部商品的映射能力
- `FulfillmentAdjustmentService`
  - 主要承载显式履约修正能力
  - 不承载动态 selector 规则语义
- `WaveOverviewProjection`
  - 主要承载聚合观察、诊断和路由能力

对应页面只是这些能力的主要 UX 入口：

- `Membership Allocation` -> `AllocationPolicyService`
- `Demand Mapping` -> `DemandMappingService`
- `Adjustment Review` -> `FulfillmentAdjustmentService`
- `Wave Overview` -> `WaveOverviewProjection`

因此真正需要稳定的，是领域能力边界，而不是页面命名本身。

还要再补一条共享调整层边界：

- `Adjustment Review` 可以修改已接手对象的最终履约结果
- 但不应直接新增一个本来不在当前波次里的全新参与者
- 也不应基于身份 / 平台 / 交集 selector 创建会继续随重算自动扩张的动态批量规则

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
11. 进入 `Wave Overview` 页面统一查看聚合结果、异常分桶、未纳入项统计和下一步阻塞
12. 如有必要，返回 `Membership Allocation` 补充规则或等待输入收集完成
13. 如有必要，进入共享的 `Adjustment Review` 页面做例外修正
14. 如无需进一步编辑，可直接进入后续执行准备阶段
15. 地址与缺项校验
16. 生成并导出 `SupplierOrder`
17. 导入工厂发货回传，生成 `Shipment`
18. 根据 `IntegrationProfile` 的能力与闭环策略，决定执行 `ChannelSyncJob` 还是人工确认闭环
19. 波次关闭

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
8. 进入 `Wave Overview` 页面统一查看聚合结果、异常分桶、未纳入项统计和下一步阻塞
9. 如有必要，返回 `Demand Mapping` 修正需求映射或等待缺失信息补齐
10. 如有必要，进入共享的 `Adjustment Review` 页面，仅在必要处做显式修正
11. 如无需进一步编辑，可直接进入后续执行准备阶段
12. 地址校验
13. 生成并导出 `SupplierOrder`
14. 导入工厂发货回传，生成 `Shipment`
15. 根据 `IntegrationProfile` 的能力与闭环策略，执行来源渠道回填或人工闭环确认
16. 波次关闭

### 7.3 混合波次

系统应允许同一个波次同时承接：

- 会员权益履约行
- 零售订单履约行
- 人工补发或履约修正项

但混合波次不应要求用户在单一页面里混用两套初始分配心智。

建议结构是：

1. 先按波次内真实需求，分别开放 `Membership Allocation` 与 `Demand Mapping`
2. 两个入口都写入同一套 `FulfillmentLine`
3. 再进入波次内独立的 `Wave Overview`
4. 在共享总览页统一查看聚合结果、风险、未纳入项分类与阶段阻塞
5. 必要时再进入共享的 `Adjustment Review`
6. 无需进一步编辑时可直接进入后续执行准备阶段
7. 后续共享同一套执行、发货、回填与闭环流程

如果确实需要新增一个当前波次本来不存在的对象：

- 应回到导入侧、手工接手侧或前置参与者整理阶段
- 不应绕过前置结构直接在共享调整层生成

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

- 一个围绕“初始履约生成 + 波次内共享总览 + 调整审查”组织起来的工作流阶段

在这个阶段内：

- `Membership Allocation` 保留当前 tag 系统的高效规则驱动体验
- 它是未来动态 selector、规则贡献和更复杂会员集合语义的唯一正确承接层
- `Demand Mapping` 提供零售订单到内部商品的映射入口
- `Wave Overview` 提供波次级的只读优先聚合诊断、阶段导航与后续分流
- `Adjustment Review` 作为两条链路的共享收敛层

这里还要再明确：

- 如果某个动作想表达“所有某类身份以后都这样改”，那它属于规则层
- 如果某个动作想表达“这次波次里这个具体对象最终怎么发”，那它才属于共享调整层

### 7.5 会员分配 UX 保留原则

当前 tag 系统在会员权益场景下已经证明是高效方案。

因此 `WaveAllocationStep` 的设计必须满足：

1. 会员权益型波次默认仍以 `Membership Allocation` 为主入口
2. 现有商品中心批量配置体验不得明显退化
3. “先规则，再覆盖，再预览”的认知路径应尽量保留
4. 零售订单型需求的引入不应迫使会员用户先理解订单映射语义
5. 共享的 `Adjustment Review` 只能作为收敛层，不能反向变成会员分配主入口
6. 波次内跨步骤导航不应破坏会员分配已经形成的规则结果和例外覆盖

---
