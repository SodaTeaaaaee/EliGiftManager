# 集成配置、Profile 与模板模型

本文件整理配置与集成层的数据结构，解释为什么 `IntegrationProfile` 是后续模板系统升级的宿主。

### 5.9 配置与集成层

这一层是 V2 相比当前系统新增的重要抽象，用来承接“来源渠道 / 业务面 / 能力 / 模板绑定”的配置。

#### IntegrationProfile

建议新增。

它是当前模板系统的上位概念，不是模板本身。

建议字段：

- `id`
- `profile_key`
  - 例如 `patreon.membership`
  - 例如 `patreon.shop_purchase`
  - 例如 `gumroad.one_time_order`
  - 例如 `fanbox.support_plan`
  - 例如 `fanbox.supporter_only_purchase`
  - 例如 `bilibili.live_support`
  - 例如 `bilibili.creator_commerce`
- `source_channel`
- `source_surface`
- `demand_kind`
  - `membership_entitlement`
  - `retail_order`
  - `manual_adjustment`
- `default_allocation_mode`
  - `rule_based`
  - `direct_from_demand`
  - `hybrid`
- `identity_strategy`
  - `platform_uid`
  - `email`
  - `external_buyer_id`
- `entitlement_authority_mode`
  - `local_policy`
  - `upstream_platform`
  - `manual_grant_only`
- `recipient_input_mode`
  - `none`
  - `platform_claim`
  - `external_form`
  - `manual_collection`
- `reference_strategy`
  - `member_level`
  - `order_level`
  - `order_line_level`
- `tracking_sync_mode`
  - `api_push`
  - `document_export`
  - `manual_confirmation`
  - `unsupported`
- `closure_policy`
  - `close_after_sync`
  - `close_after_manual_confirmation`
  - `close_after_shipment`
- `supports_tracking_push`
- `supports_partial_shipment`
- `supports_api_import`
- `supports_api_export`
- `capabilities`
- `supported_locales`
- `default_locale`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- `IntegrationProfile` 回答的是“这个来源业务面整体怎么工作”
- 它不是某一个导入 CSV 的字段映射
- 同一个 `source_channel` 可以挂多个 `IntegrationProfile`
- `profile_key` 是系统内部稳定业务语言，不要求照抄平台官方命名，但必须与官方业务面语义一致
- `tracking_sync_mode` 和 `closure_policy` 用来避免系统继续靠平台印象推断闭环逻辑
- `entitlement_authority_mode` 用来约束本系统是否应自行判定权益成立
- `recipient_input_mode` 用来说明该业务面通常通过什么方式补齐领取参数
- `supported_locales` 和 `default_locale` 用来描述该业务面在 UI / 模板 / 导出展示上支持哪些语言
- 这些语言字段不应影响核心业务 code 的稳定性

#### DocumentTemplate

建议作为 `TemplateConfig` 的未来语义方向保留。

建议字段：

- `id`
- `template_key`
- `document_type`
  - `import_entitlement`
  - `import_sales_order`
  - `import_product_catalog`
  - `export_supplier_order`
  - `import_supplier_shipment`
  - `export_source_tracking_update`
- `format`
  - `csv`
  - `xlsx`
  - `json`
  - `api_payload`
- `mapping_rules`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- `DocumentTemplate` 只负责文档字段结构
- 它不负责决定需求类型、业务面和平台能力
- 如果同一业务面需要中英两套模板，推荐通过不同 `locale` 版本或不同 `template_key` 表达，而不是把翻译文本塞进业务判断

#### IntegrationProfileTemplateBinding

建议新增。

建议字段：

- `id`
- `integration_profile_id`
- `document_type`
- `template_id`
- `is_default`
- `created_at`

说明：

- 一个 `IntegrationProfile` 应允许绑定多个不同用途的模板
- 例如同一个 profile 同时绑定：
  - 需求导入模板
  - 工厂导出模板
  - 工厂发货回传模板
  - 来源渠道物流回填模板

---
