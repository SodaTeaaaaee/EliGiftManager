# 客户、需求与波次模型

本文件覆盖全局客户层、上游需求层、波次层，以及分配与规则语义层，是 V2 数据模型的前半部分。

### 5.1 全局层

#### CustomerProfile

建议字段：

- `id`
- `display_name`
- `profile_type`
  - `member`
  - `buyer`
  - `mixed`
  - `manual`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 当前 `Member` 可作为其过渡实现
- 不再把“会员”视为唯一客体

#### CustomerIdentity

建议字段：

- `id`
- `customer_profile_id`
- `identity_platform`
- `identity_value`
- `identity_type`
  - `platform_uid`
  - `email`
  - `username`
  - `external_buyer_id`
- `is_primary`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 这是对当前 `(platform, platform_uid)` 单一身份结构的泛化
- 一个客户可能同时有多个身份来源

#### CustomerAddress

可在当前 `MemberAddress` 基础上扩展，建议继续保留：

- 历史
- 默认地址
- 测试地址
- 软删除
- 标签化备注

未来可新增：

- `normalized_region`
- `postal_code`
- `country_code`
- `validation_status`

### 5.2 上游需求层

#### DemandDocument

建议字段：

- `id`
- `kind`
  - `membership_entitlement`
  - `retail_order`
  - `manual_adjustment`
- `source_channel`
- `source_surface`
- `source_document_no`
- `source_customer_ref`
- `customer_profile_id`
- `source_created_at`
- `source_paid_at`
- `currency`
- `authority_snapshot_at`
- `raw_payload`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 会员导入没有订单号时，`source_document_no` 可为空
- 零售渠道订单导入时，`source_document_no` 通常应有值
- 同一个 `source_channel` 下应允许通过 `source_surface` 区分会员业务面、商城业务面等不同来源语义

#### DemandLine

建议字段：

- `id`
- `demand_document_id`
- `source_line_no`
- `line_type`
  - `entitlement_rule`
  - `sku_order`
  - `manual_adjustment`
- `obligation_trigger_kind`
  - `periodic_membership`
  - `loyalty_membership`
  - `supporter_only_purchase`
  - `member_only_discount_purchase`
  - `campaign_reward`
  - `manual_compensation`
- `entitlement_authority`
  - `local_policy`
  - `upstream_platform`
  - `manual_grant`
- `recipient_input_state`
  - `not_required`
  - `waiting_for_input`
  - `partially_collected`
  - `ready`
  - `waived`
  - `expired`
- `routing_disposition`
  - `pending_intake`
  - `accepted`
  - `deferred`
  - `excluded_manual`
  - `excluded_duplicate`
  - `excluded_revoked`
- `routing_reason_code`
- `eligibility_context_ref`
- `product_master_id`
- `external_title`
- `requested_quantity`
- `entitlement_code`
- `gift_level_snapshot`
- `recipient_input_payload`
- `raw_payload`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 会员场景里，这一层用于记录“本次应得权益”
- 零售场景里，这一层用于记录“用户下单了什么”
- `routing_disposition` 记录的是本系统是否接手处理
- 它不应被误用为系统外履约完成状态
- 对会员限定购买一类零售订单，应使用 `eligibility_context_ref` 保留其资格来源，而不是把它改判成 `membership_entitlement`

### 5.2.1 会员权益的判定、输入采集与路由

对 `membership_entitlement` 而言，建议明确拆开三件事：

1. 权益成立的判定权威

- 回答：
  - “谁说这条权益现在真的成立了？”
- 由 `entitlement_authority` 表达

2. 收货对象输入是否收齐

- 回答：
  - “这条权益现在能不能真的转成待执行履约？”
- 由 `recipient_input_state` 表达
- 这里的输入可能包括地址、尺码、款式、颜色、组合、领取确认等

3. 本系统是否接手

- 回答：
  - “这条权益这次是否进入 EliGiftManager 的处理范围？”
- 由 `routing_disposition` 表达

这三件事不能混成一个状态字段。

### 5.3 波次层

#### Wave

建议保留现有表，并新增/演进：

- `wave_type`
  - `membership`
  - `retail`
  - `mixed`
- `allocation_mode`
  - `rule_based`
  - `direct_from_demand`
  - `hybrid`
- `lifecycle_stage`
- `progress_snapshot`
- `notes`

`status` 可在迁移期保留，但长期应降级为兼容字段或投影视图字段。

#### WaveParticipantSnapshot

建议基于当前 `WaveMember` 扩展或替换。

建议字段：

- `id`
- `wave_id`
- `customer_profile_id`
- `snapshot_type`
  - `member`
  - `buyer`
  - `mixed`
- `identity_platform`
- `identity_value`
- `display_name`
- `gift_level`
- `source_channel`
- `source_surface`
- `source_document_refs`
- `extra_data`
- `created_at`

说明：

- 当前 `WaveMember` 不应该继续被理解为“只有会员”
- 它应该变成波次内参与方快照
- 只有 `routing_disposition = accepted` 的需求，才应稳定进入后续波次处理语义

### 5.3.1 分配与调整语义层

V2 的目标不是让所有需求都经过同一种分配引擎，而是让不同来源在不同层级被正确处理后再收敛。

建议明确以下三层：

1. `Base Allocation Source`

表示“初始履约结果从哪里来”。

可能来源：

- 规则推导
- 上游订单行直入
- 手工补单直入

2. `Adjustment Layer`

表示“在初始履约结果之上的修正”。

可能动作：

- 加送
- 减送
- 替换
- 补发
- 取消

3. `Final Fulfillment Result`

表示最终需要执行的履约真相。

这一层最终统一落到 `FulfillmentLine`。

### 5.3.2 AllocationPolicyRule

建议将当前 `ProductTag` 明确定位为：

- `policy-driven` 分配模式下的规则对象

当前 `ProductTag` 不应被要求承担：

- 零售订单原始行真相
- 外部订单义务真相
- 所有来源需求的统一表达

而应主要承担：

- 身份规则
- 波次级批量赠送规则
- 规则驱动分配中的用户例外覆盖

过渡期可继续使用 `ProductTag` 承接此职责。

长期可以考虑把语义进一步重命名为更明确的规则对象，但当前阶段不要求立即改表名。

