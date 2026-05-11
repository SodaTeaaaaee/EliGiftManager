# 商品、履约与工厂执行模型

本文件覆盖商品层、履约层与工厂执行层，用于定义系统内部履约真相如何逐步转译为工厂侧订单。

### 5.4 商品层

#### ProductMaster

建议继续保留为全局商品主档。

未来可补充：

- `product_kind`
- `supplier_platform`
- `supplier_product_ref`
- `archived`

#### Product

继续保留为波次级商品快照。

未来建议加强：

- 明确其是“波次里的履约商品版本”
- 保持与 `ProductMaster` 生命周期分离

### 5.5 履约层

#### FulfillmentLine

建议在当前 `DispatchRecord` 基础上演进。

建议核心字段：

- `id`
- `wave_id`
- `customer_profile_id`
- `wave_participant_snapshot_id`
- `product_id`
- `demand_document_id`
- `demand_line_id`
- `customer_address_id`
- `quantity`
- `allocation_state`
- `address_state`
- `supplier_state`
- `channel_sync_state`
- `line_reason`
  - `entitlement`
  - `retail_order`
  - `manual_adjustment`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 该层只负责“履约执行真相”
- 不再把所有外部流程状态粗暴压到一个 `status` 字段里
- 未被本系统接手的需求，不应伪装成 `FulfillmentLine`
- `FulfillmentLine` 是 `RoutingDisposition = accepted` 之后的执行对象，而不是所有上游需求的总账本

#### FulfillmentAdjustment

建议新增，作为长期目标中的显式调整层对象。

建议字段：

- `id`
- `wave_id`
- `fulfillment_line_id`
- `wave_participant_snapshot_id`
- `from_product_id`
- `to_product_id`
- `adjustment_kind`
  - `add`
  - `reduce`
  - `replace`
  - `compensation`
  - `remove`
- `quantity_delta`
- `reason_code`
- `note`
- `created_by`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 这层的目标是把“人工调整”从隐式规则覆盖逐步演进为显式履约修正对象
- 在过渡期内，部分调整仍可继续通过 `user tag` 实现
- 但从长期语义上讲，`user tag` 更接近调整实现，而不是最终理想形态

### 5.6 工厂执行层

#### SupplierOrder

建议新增。

建议字段：

- `id`
- `wave_id`
- `supplier_platform`
- `template_id`
- `batch_no`
- `external_order_no`
- `submission_mode`
  - `csv`
  - `manual`
  - `api`
- `submitted_at`
- `status`
  - `draft`
  - `submitted`
  - `accepted`
  - `partially_shipped`
  - `shipped`
  - `canceled`
- `request_payload`
- `response_payload`
- `extra_data`
- `created_at`
- `updated_at`

#### SupplierOrderLine

建议新增。

建议字段：

- `id`
- `supplier_order_id`
- `fulfillment_line_id`
- `supplier_line_no`
- `supplier_sku`
- `submitted_quantity`
- `accepted_quantity`
- `status`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 这层负责把系统内履约行映射到工厂提交和工厂回传

