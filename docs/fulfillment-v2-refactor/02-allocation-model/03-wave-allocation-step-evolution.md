# WaveAllocationStep 演进方向

> `WaveAllocationStep` 的正式命名、工作流定位和页面拆分原则。

## 正式命名

文档和代码采用 `WaveAllocationStep`。`WaveTagStep` 仅在讨论旧实现时出现。

## 工作流分区优先于视觉分区

波次编辑拆成有顺序的大阶段，每阶段拆成职责明确的步骤。会员和零售在"初始分配生成"阶段走不同页面。

### 大阶段

1. `Demand Intake` — 导入标准化上游需求
2. `Initial Allocation` — 生成第一版履约结果
   - `Membership Allocation` — `policy_driven`，保留商品级批量规则编辑
   - `Demand Mapping` — `demand_driven`，展示需求行与商品映射
3. `Wave Overview` — 只读聚合总览（阶段导航、异常汇总、偏离提示）
4. `Adjustment Review` — 共享修正与审查
5. `Execution Readiness` — 地址/缺项/异常校验
6. `Supplier Execution` — 工厂导出与执行跟踪
7. `Shipment Intake` — 工厂发货回传
8. `Channel Sync / Closure` — 回填或人工闭环

### 混合波次含义

混合的是"履约容器"和"后链路"，不是"初始编辑心智"。

## 共享调整层边界

### 三问分流

1. 改"上游原始事实"？ → 回前置页面
2. 改"默认生成逻辑"？ → 回 `Membership Allocation` 或 `Demand Mapping`
3. 改"本波次最终发什么"？ → 进 `Adjustment Review`
4. 改"具体对象" vs "某类集合以后都这样"？ → 前者进调整层，后者回规则层

### 首版允许

加送、减送、替换、移除、补发/补偿、一次性数量修正。target 是具体参与者或履约行。

### 首版禁止

改写 DemandLine/RoutingDisposition/RecipientInputState/EntitlementAuthority、改写规则/映射、新增全新参与者、selector 级动态批量改动。

### 灰区判断

- 修正当前少量对象 → `Adjustment Review`
- 大范围同类变化 → 回前置页面
- 未来波次也生效 → 回前置页面
- selector 动态跟随 → 回前置页面

## 会员分配 UX 保留原则

1. 会员权益型波次默认以 `Membership Allocation` 为主入口
2. 商品中心批量配置体验不得退化
3. "先规则，再覆盖，再预览"认知路径保留
4. 零售订单不应迫使会员用户理解订单映射
