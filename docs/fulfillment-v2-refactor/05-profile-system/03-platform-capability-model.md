# 平台能力模型与边界

> 平台能力按 `IntegrationProfile` 或 `source_surface` 建模，不按品牌全局一刀切。

## 典型能力

`supports_partial_shipment`、`requires_carrier_mapping`、`requires_external_order_no`、`entitlement_authority_mode`、`recipient_input_collection_mode`、`tracking_sync_mode`、`closure_policy`、`allows_manual_closure`、`supports_api_import`、`supports_api_export`。

## 收敛约束

- strategy 字段 → 主流程分支
- capability flag → 正交能力（不与 strategy 重复）
- connector → 真实实现差异
- 不保留匿名 `capabilities` blob

## 不应放进 Profile

复杂波次分配算法、工厂回传合并逻辑、异常分支 DSL、条件式脚本、复杂业务规则代码、动态 selector 规则语言。

Profile 承载：能力声明、策略枚举、闭环规则、模板绑定、身份与引用规则、权益判定权威模式、输入采集模式。复杂流程逻辑留在 service 层。
