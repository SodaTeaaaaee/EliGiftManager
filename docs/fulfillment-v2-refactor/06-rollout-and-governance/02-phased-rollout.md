# 分阶段实施计划

本文件拆出阶段化落地路线，用于指导 V2 领域重构、数据迁移、状态投影和 profile 系统升级的推进顺序。

## 11. 分阶段实施计划

### 阶段 0：冻结语义与基线

目标：

- 冻结当前系统语义
- 建立 V2 业务语言
- 建立本重构计划文档

产出：

- 本文档
- 重构前备份分支

### 阶段 1：统一命名与领域边界

目标：

- 在 docs、service 注释、DTO 命名层明确新语义

任务：

- 补充“CustomerProfile / Demand / Fulfillment / SupplierOrder / Shipment / ChannelSync”词汇表
- 标记旧术语与新术语映射关系
- 明确 `platform` 的多维语义
- 补充基于官方资料的平台业务面例子说明
- 明确 `IntegrationProfile` 和 `DocumentTemplate` 的边界
- 为现有 `TemplateConfig` 预留向 `DocumentTemplate` 迁移的路径
- 明确 `policy-driven` / `demand-driven` / `hybrid` 三种分配语义
- 明确 `membership_entitlement` 与“会员限定购买”之间的边界
- 明确 `ObligationTriggerKind`、`EntitlementAuthority`、`RecipientInputState`、`RoutingDisposition`
- 明确当前 tag 系统在会员权益场景中的保留价值与边界
- 正式采用 `WaveAllocationStep` 作为新文档语义

验收：

- 文档层不再把会员、买家、订单、工厂单和模板能力混为一谈

### 阶段 2：引入新表结构

目标：

- 以新增为主，引入 V2 所需核心表

建议优先新增：

- `demand_documents`
- `demand_lines`
- `supplier_orders`
- `supplier_order_lines`
- `shipments`
- `shipment_lines`
- `channel_sync_jobs`
- `channel_sync_items`
- `integration_profiles`
- `integration_profile_template_bindings`

并在需求侧补充或预留：

- `obligation_trigger_kind`
- `entitlement_authority`
- `recipient_input_state`
- `routing_disposition`
- `eligibility_context_ref`

同时为旧表补充过渡字段：

- `waves.wave_type`
- `waves.allocation_mode`
- `waves.lifecycle_stage`
- `dispatch_records` 的多维状态字段

验收：

- 不影响现有导入导出链路
- 新表迁移可重复执行
- 新的 profile 层可以先只服务少量试点来源

### 阶段 3：重构履约真相层

目标：

- 让 `DispatchRecord` 向 `FulfillmentLine` 语义演进

任务：

- 拆出多维状态
- 增加来源引用
- 增加工厂执行引用
- 增加来源渠道回填引用
- 为显式调整层预留 `FulfillmentAdjustment` 入口

关键约束：

- `ReconcileWave` 仍可继续生成履约行
- 但履约行不再只服务“导出 CSV 即结束”的短链路

验收：

- 履约行可同时表示会员来源和零售来源
- 履约行可独立追踪后续工厂执行和回填过程

### 阶段 4：引入上游需求层

目标：

- 支持“会员权益需求”和“零售订单需求”的统一落库

任务：

- 新增会员权益导入到 `DemandDocument/DemandLine`
- 新增零售订单导入到 `DemandDocument/DemandLine`
- 支持“平台权威已判定成立的权益”与“本地规则判定权益”并存
- 支持会员输入采集晚于权益成立
- 支持本系统接手范围的显式路由决策
- 建立 demand 到 wave participant / fulfillment line 的映射
- 建立“规则推导生成第一版履约行”和“需求行直入生成第一版履约行”的双路径

验收：

- 会员导入和零售导入都可进入统一履约链路
- 不由本系统处理的需求会被单独统计，而不是伪装成执行状态

### 阶段 5：引入工厂执行层

目标：

- 把“导出工厂 CSV”从一次性动作升级为持久化的工厂执行记录

任务：

- 导出时创建 `SupplierOrder`
- 导出行映射到 `SupplierOrderLine`
- 记录模板、导出时间、外部工厂单号、提交方式

验收：

- 任何一次工厂导出都可追踪、可回查、可关联到履约行

### 阶段 6：引入发货与物流回传

目标：

- 支持工厂回传发货状态和快递单号

任务：

- 新增 `import_supplier_shipment`
- 支持按工厂模板映射 shipment 数据
- 建立 `Shipment` 与 `SupplierOrderLine/FulfillmentLine` 的关联

验收：

- 系统能稳定落库存储发货结果
- 同一波次支持部分发货和多包裹

### 阶段 7：引入来源渠道回填

目标：

- 把物流状态同步回来源渠道

任务：

- 新增 `ChannelSyncJob/Item`
- 建立物流承运商映射
- 建立外部订单号和外部订单行号映射
- 导出或调用 API 回填

验收：

- 可区分“待回填”“已回填”“回填失败”
- 可重试失败项

### 阶段 8：重做波次状态投影与 UI

目标：

- 让波次状态真正反映业务阶段

任务：

- 替换当前 `Wave.Status` 简化展示逻辑
- 重做 Dashboard 和波次列表状态卡片
- 增加分业务面进度、异常计数、回填结果、待人工闭环计数
- 在导入侧或共享总览页增加“未纳入本系统处理”计数与分类
- 将当前 `WaveTagStep` 迁移为正式语义 `WaveAllocationStep`
- 在工作流上拆出独立页面：
  - `Membership Allocation`
  - `Demand Mapping`
  - `Adjustment Review`
- 引入轻量人工闭环决策，而不是允许任意状态改写
- 对会员权益型需求增加领取参数采集与路由总览

验收：

- 页面状态与真实流程一致
- 不再出现“已导出但其实还没发货/还没回填”的误导
- 会员权益型分配体验不因零售语义接入而明显退化

### 阶段 9：模板系统能力化

目标：

- 把模板系统从“字段映射集合”升级为“Profile + 字段模板 + 平台能力”

任务：

- 扩展模板类型
- 加入连接器能力描述
- 直接以 `IntegrationProfile` 作为新的配置入口
- 把 `TemplateConfig` 迁移为 `DocumentTemplate` 语义
- 拆出物流映射、订单号映射、承运商映射逻辑

验收：

- 新来源业务面接入不再需要临时硬编码整套流程

---

