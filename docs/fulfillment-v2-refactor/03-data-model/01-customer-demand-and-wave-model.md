# 客户、需求与波次模型

> V2 数据模型前半部分：全局客户层、上游需求层、波次层、分配语义层。
> 当前实现见 `internal/domain/models.go`。

## 全局层

### CustomerProfile

字段：`id`、`display_name`、`profile_type`（`member`/`buyer`/`mixed`/`manual`）、`extra_data`、时间戳。

不再只代表"会员"，同时覆盖买家和手工补录。

### CustomerIdentity

字段：`id`、`customer_profile_id`、`identity_platform`、`identity_value`、`identity_type`（`platform_uid`/`email`/`username`/`external_buyer_id`）、`is_primary`、`extra_data`、时间戳。

一个客户可有多个身份来源。

### CustomerAddress

保留历史/默认/测试地址/软删除/标签化备注。扩展 `normalized_region`、`postal_code`、`country_code`、`validation_status`。

## 上游需求层

### DemandDocument

字段：`id`、`kind`（`membership_entitlement`/`retail_order`）、`capture_mode`（`document_import`/`api_ingest`/`manual_entry`）、`source_channel`、`source_surface`、`integration_profile_id`、`source_document_no`、`customer_profile_id`、时间与元数据字段。

- 手工录入不是第三种 `kind`，通过 `capture_mode` 区分
- `integration_profile_id` 表示按哪个业务面合同解释进系统
- 当前阶段默认不支持跨波次拆分

### DemandLine

字段：`id`、`demand_document_id`、`source_line_no`、`line_type`（`entitlement_rule`/`sku_order`/`manual_entry`）、`obligation_trigger_kind`、`entitlement_authority`、`recipient_input_state`、`routing_disposition`、`product_master_id`、`requested_quantity`、`gift_level_snapshot`、元数据字段。

关键三件事拆开：
- **权益判定权威**（`entitlement_authority`）：谁说权益成立了
- **输入采集状态**（`recipient_input_state`）：能不能转成待执行履约
- **路由处置**（`routing_disposition`）：本系统是否接手

## 波次层

### Wave

保留现有表，新增 `wave_type`（`membership`/`retail`/`mixed`）、`lifecycle_stage`、`progress_snapshot`。混合波次不通过单一 `allocation_mode` 字段表达，由不同 demand/profile 路径各自声明策略，投影层汇总。

### WaveDemandAssignment

回答"这份 demand 这次由哪个 wave 接手"。三层职责分开：`DemandDocument`（来源真相）→ `WaveDemandAssignment`（接手关系）→ `FulfillmentLine`（执行真相）。

### WaveParticipantSnapshot

基于 `WaveMember` 泛化。字段：`id`、`wave_id`、`customer_profile_id`、`snapshot_type`、`identity_platform`、`identity_value`、`display_name`、`gift_level`、`source_document_refs`、`source_profile_refs`、`extra_data`。

只有 `routing_disposition = accepted` 的需求才进入波次处理。

## 分配与调整语义层

| 层 | 说明 |
|----|------|
| `Base Allocation Source` | `policy_driven` 或 `demand_driven` |
| `Allocation Contribution` | 规则贡献，可正可负 |
| `Base Allocation Result` | 非负基础结果 |
| `Adjustment Layer` | 显式修正，delta 可正可负 |
| `Final Fulfillment Result` | `FulfillmentLine`，`max(base + delta, 0)` |

非破坏性编辑：`Membership Allocation`、`Demand Mapping`、`Wave Overview`、`Adjustment Review` 是同一波次数据的不同视角，页面切换不破坏数据。

### AllocationPolicyRule

`policy_driven` 分配模式下的规则对象。字段：`selector_payload`、`product_target_ref`、`contribution_quantity`（可正负）、`rule_kind`、`priority`、`active`。

首版不引入复杂布尔 DSL。`ProductTag` 作短期桥接，长期收敛到新规则对象。
