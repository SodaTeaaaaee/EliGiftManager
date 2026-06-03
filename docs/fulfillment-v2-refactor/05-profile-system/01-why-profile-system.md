# 为什么要升级为 Profile 系统

## 不能只扩模板系统

当前 `TemplateConfig` 只是"文档字段映射配置"，无法表达：来源业务面、需求类型、义务触发方式、权益判定权威、身份策略、物流回填能力、闭环策略。

## `IntegrationProfile` 定位

比模板更高层的能力配置层。`profile_key` = `<source_channel>.<source_surface>`。

- 不是"平台类别标签"，而是"某个平台供应商下的某个具体业务面合同"
- 负责来源业务面的稳定语义
- `Adjustment Review` 负责波次内最终履约例外，两者不互相吞并

### 结构收敛

- strategy 字段 → 主流程语义
- capability flag → 正交能力（不与 strategy 重复）
- connector binding → 外部交互实现入口
- 不膨胀成万能配置包

## 直接升级旧模板入口

V2 不再长期并行保留旧模板系统。保留字段映射资产，入口/命名/配置分层直接升级为 Profile 体系。

## V2 推荐模板类型

`import_entitlement`、`import_sales_order`、`import_product_catalog`、`export_supplier_order`、`import_supplier_shipment`、`export_source_tracking_update`。

## Profile 与活跃波次

活跃波次继续使用创建时绑定的 profile 行为，直到用户显式刷新。首版不引入版本号。已关闭波次不受 profile 变更影响。
