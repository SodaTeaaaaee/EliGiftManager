# 物流与渠道回填模型

本文件覆盖 Shipment、ShipmentLine、ChannelSyncJob、ChannelSyncItem 等后链路结构，解决物流回传与来源渠道回填问题。

### 5.7 物流层

#### Shipment

建议新增。

建议字段：

- `id`
- `supplier_order_id`
- `supplier_platform`
- `shipment_no`
- `external_shipment_no`
- `carrier_code`
- `carrier_name`
- `tracking_no`
- `status`
  - `pending`
  - `shipped`
  - `in_transit`
  - `delivered`
  - `exception`
  - `returned`
- `shipped_at`
- `delivered_at`
- `raw_payload`
- `extra_data`
- `created_at`
- `updated_at`

#### ShipmentLine

建议新增。

建议字段：

- `id`
- `shipment_id`
- `fulfillment_line_id`
- `supplier_order_line_id`
- `quantity`
- `created_at`

说明：

- 如果未来存在一个包裹覆盖多条履约行，或一条履约行被拆包发货，该层是必要的

### 5.8 回填层

#### ChannelSyncJob

建议新增。

建议字段：

- `id`
- `wave_id`
- `source_channel`
- `source_surface`
- `direction`
  - `push_tracking`
- `status`
  - `pending`
  - `running`
  - `success`
  - `partial_success`
  - `failed`
- `request_payload`
- `response_payload`
- `error_message`
- `started_at`
- `finished_at`
- `created_at`
- `updated_at`

#### ChannelSyncItem

建议新增。

建议字段：

- `id`
- `channel_sync_job_id`
- `fulfillment_line_id`
- `shipment_id`
- `external_document_no`
- `external_line_no`
- `tracking_no`
- `carrier_code`
- `status`
- `error_message`
- `created_at`
- `updated_at`

说明：

- 回填的成功与失败必须可追踪
- 不能只在导入导出过程中瞬时处理
- 但并不是所有需求都会生成 `ChannelSyncJob`
- 对 `routing_disposition != accepted` 的需求，本系统不应生成后续执行记录

