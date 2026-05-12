# 为什么要升级为 Profile 系统

本文件说明为什么不能继续只扩模板系统，以及 `source_profile` / `IntegrationProfile` 应该如何理解。

## 9. Profile / 模板系统升级方向

### 9.0 为什么不能只继续扩模板系统

当前 `TemplateConfig` 更接近“文档字段映射配置”。

如果继续只在模板系统上做加法，会越来越难表达以下问题：

- 这个来源属于哪个业务面
- 它产生哪种需求类型
- 它的义务是由资格成立还是由订单成立
- 它的权益判定权威来自哪里
- 它通常如何收集地址、选项和领取参数
- 它用什么身份策略识别履约对象
- 它支不支持物流回填
- 它需要哪些不同文档模板协同工作
- 它采用什么闭环策略

这说明当前系统真正缺少的，不只是“更多模板类型”，而是：

- 一个比模板更高层的 Profile 系统

### 9.0.1 对 `source_profile` 的判断

本次讨论里的 `source_profile`，本质上可以理解为：

- `IntegrationProfile`

它与当前模板系统的关系是：

- 可以复用模板系统的已有资产
- 但不应被简化成“模板系统换个名字”

更准确的关系是：

- `Profile System` 是能力配置层
- `Template System` 是 Profile 下面的文档映射子层

### 9.0.3 Profile 不是“平台类别标签”

还需要再把一个常见误区提前拆开：

- `IntegrationProfile` 不是给平台贴一个“会员平台 / 零售平台”总标签
- 它更接近“某个平台供应商下的某个具体业务面合同”

例如：

- `patreon.membership`
- `patreon.shop_purchase`
- `fanbox.support_plan`
- `fanbox.supporter_only_purchase`

这些 profile 可能来自同一供应商名下，但在本系统里承担的是不同语义：

- 有的产生 `membership_entitlement`
- 有的产生 `retail_order`
- 有的只提供资格上下文，但真正履约义务仍由后续订单成立

因此 Profile 系统要回答的是：

- “这个来源业务面到底是什么”
- “它在本系统里落成哪种需求语义”
- “它的义务由资格成立还是由订单成立”

而不是只回答：

- “这个平台品牌叫什么”

同时也要明确：

- `IntegrationProfile` 负责来源业务面的稳定语义
- `Adjustment Review` 负责波次内最终履约例外
- 两者不能互相吞并

### 9.0.2 当前决策：直接升级旧模板入口

根据当前讨论，V2 文档层已经采用以下决策：

- 不再把现有模板系统当成一个需要长期并行保留的成熟子系统
- 现有模板入口应直接朝 `IntegrationProfile` 中心化入口升级

原因不是“旧模板没有任何价值”，而是：

- 旧模板系统本身能力边界不清
- 与特定测试数据耦合较深
- 几乎没有真正可编辑、可复用、可扩展的业务语义层

因此更合理的做法是：

- 保留旧模板里仍有价值的字段映射资产
- 但入口、命名、配置分层直接升级为 Profile 体系

这里还要再补一条当前阶段的收敛原则：

- `IntegrationProfile` 不应继续膨胀成万能配置包
- 如果某种差异属于真实外部交互实现，应优先进入 connector / service 层
- 如果某种差异属于稳定流程语义，应优先进入命名清晰的 strategy 字段
- 如果某种差异只是一个正交能力，应优先进入单一职责的 capability flag

### 9.1 当前模板类型不够用

当前只有：

- `import_product`
- `import_dispatch_record`
- `export_order`

这不足以表达 V2 的完整链路。

### 9.2 V2 推荐模板类型

建议至少引入：

- `import_entitlement`
- `import_sales_order`
- `import_product_catalog`
- `export_supplier_order`
- `import_supplier_shipment`
- `export_source_tracking_update`
