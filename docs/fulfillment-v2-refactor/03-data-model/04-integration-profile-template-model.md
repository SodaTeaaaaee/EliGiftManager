# 集成配置、Profile 与模板模型

> 配置与集成层数据结构。

## IntegrationProfile

来源业务面的统一配置入口。`profile_key` = `<source_channel>.<source_surface>`。

核心字段分三层：

1. **语义策略**：`demand_kind`、`initial_allocation_strategy`、`identity_strategy`、`entitlement_authority_mode`、`recipient_input_mode`、`reference_strategy`、`tracking_sync_mode`、`closure_policy`
2. **正交能力**：`supports_partial_shipment`、`supports_api_import`、`supports_api_export`、`requires_carrier_mapping`、`requires_external_order_no`、`allows_manual_closure`
3. **外部实现**：`connector_key`、`IntegrationProfileTemplateBinding`

其他：`profile_key`、`source_channel`、`source_surface`、`supported_locales`、`default_locale`、`extra_data`、时间戳。

不保留匿名 `capabilities` blob。复杂平台差异下沉到 connector/service 层。

## DocumentTemplate

文档字段映射配置。字段：`id`、`template_key`、`document_type`（`import_entitlement`/`import_sales_order`/`import_product_catalog`/`export_supplier_order`/`import_supplier_shipment`/`export_source_tracking_update`）、`format`（`csv`/`xlsx`/`json`/`api_payload`）、`mapping_rules`、`extra_data`、时间戳。

## IntegrationProfileTemplateBinding

一个 Profile 可绑定多个不同用途的模板（需求导入、工厂导出、发货回传、物流回填）。

字段：`id`、`integration_profile_id`、`document_type`、`template_id`、`is_default`、`created_at`。
