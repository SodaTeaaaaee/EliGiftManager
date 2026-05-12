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
- 补充 `HistoryScope / HistoryNode / Basis Projection` 词汇表
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
- 当前阶段不把旧数据库兼容作为硬约束

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
- 不要求为未投产旧数据补复杂兼容逻辑

### 阶段 3：重构履约真相层

目标：

- 让 `DispatchRecord` 向 `FulfillmentLine` 语义演进

任务：

- 拆出多维状态
- 增加来源引用
- 增加工厂执行引用
- 增加来源渠道回填引用
- 为显式调整层预留 `FulfillmentAdjustment` 入口
- 明确负数量只存在于规则贡献层或调整 delta 层，而不是最终执行履约行
- 定义“基础履约结果重算后，再显式重放调整层”的顺序约束

关键约束：

- `ReconcileWave` 仍可继续生成履约行
- 但履约行不再只服务“导出 CSV 即结束”的短链路

验收：

- 履约行可同时表示会员来源和零售来源
- 履约行可独立追踪后续工厂执行和回填过程
- 前置层重算不会悄悄吃掉已确认的共享调整层例外

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

- 把“导出工厂 CSV”从一次性动作升级为可持久化的工厂工作区对象

任务：

- 导出时创建 `SupplierOrder`
- 导出行映射到 `SupplierOrderLine`
- 记录模板、导出时间、外部工厂单号、提交方式
- 允许后续编辑覆盖当前工作区结果，而不是把提交后状态做成硬锁

验收：

- 当前工作区可稳定保留最近一次导出关联
- 后续重新导出时可重建与更新当前关联，而不强制走历史锁定模式

### 阶段 6：引入发货与物流回传

目标：

- 支持工厂回传发货状态和快递单号

任务：

- 新增 `import_supplier_shipment`
- 支持按工厂模板映射 shipment 数据
- 建立 `Shipment` 与 `SupplierOrderLine/FulfillmentLine` 的关联
- 建立当前工作区与最近一次物流基础之间的偏离提示能力
- 为偏离提示预留进入 `Wave Overview` 与复核入口的路径
- 让后续编辑偏离已导入物流基础时只产生辅助提示，不形成硬锁

验收：

- 系统能稳定落库存储发货结果
- 同一波次支持部分发货和多包裹
- 物流对象可作为当前工作区辅助状态存在，而不被误解为不可覆盖历史

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
- 在导入侧或波次内独立 `Wave Overview` 页面增加“未纳入本系统处理”计数与分类
- 在 `Wave Overview` 增加当前工作区相对导出、回传和回填基础的偏离提示
- 将当前 `WaveTagStep` 迁移为正式语义 `WaveAllocationStep`
- 在工作流上拆出独立页面：
  - `Membership Allocation`
  - `Demand Mapping`
  - `Wave Overview`
  - `Adjustment Review`
- 明确 `Wave Overview` 与 `Adjustment Review` 的页面边界，避免在小窗口中把聚合浏览与复杂编辑挤进同一页
- 将 `Wave Overview` 按只读优先方式落地，并允许其作为“进入共享调整层 / 直接进入后续阶段”的分流关口
- 引入更强的步骤向导，支持波次任意步骤间的快速跳转
- 为跨步骤往返编辑建立非破坏性数据约束
- 引入轻量人工闭环决策，而不是允许任意状态改写
- 对会员权益型需求增加领取参数采集与路由总览
- 保持 `Adjustment Review` 只处理具体对象例外，不承担动态 selector 规则编辑
- 以假设驱动方式试做首版 `Wave Overview` 聚合视角，再根据使用反馈迭代

验收：

- 页面状态与真实流程一致
- 不再出现“已导出但其实还没发货/还没回填”的误导
- 混合波次可先在 `Wave Overview` 统一查看聚合结果，再决定回前置页面还是进入共享调整层
- 无需进一步编辑的波次，可从 `Wave Overview` 直接进入后续执行准备阶段
- 跨步骤跳转不会大幅破坏前后步骤已有数据
- 会员权益型分配体验不因零售语义接入而明显退化

### 阶段 9：引入工作区历史与树状撤销重做

目标：

- 为全应用建立统一的工作区历史基础设施
- 首版优先让 `wave` scope 可用

任务：

- 新增 `HistoryScope / HistoryNode / HistoryCheckpoint / HistoryPin`
- 明确 history node 以用户意图为粒度，而不是以系统副作用为粒度
- 让 scope 支持树状分支，而不是线性未来覆盖
- 让历史在软件关闭后继续保留
- 为 `SupplierOrder / Shipment / ChannelSyncJob` 增加 basis history 引用与 projection hash
- 为被外部对象引用的历史节点建立 pin 语义
- 设计并实现老旧未 pin 分支的压缩 / GC 策略
- 为 `wave` 工作区接入 `Ctrl+Z / Ctrl+Shift+Z`
- 让全局快捷键尊重文本输入自身的原生 undo / redo
- 增加撤销 / 重做 toast 与短期回执托盘
- 提供按需打开的历史图或分支切换入口
- 在 `wave` 跑稳后，再逐步接入 `templates / products` 等其他工作区

验收：

- 撤销到旧节点后重新编辑，旧未来不会消失，而是形成新分支
- 重新打开软件后，历史 scope 与当前 head 仍可恢复
- undo / redo 不会谎称外部动作已被真实回滚
- 外部 basis 与当前 head 脱节时，系统仍能正确给出偏离 / 复核提示
- 撤销 / 重做后，用户会收到显眼但不碍事的即时反馈，并能短时间内回看最近动作

### 阶段 10：模板系统能力化

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

