# 物流与渠道回填模型

> Shipment、ShipmentLine、ChannelSyncJob、ChannelSyncItem 等后链路结构。

## Shipment

字段：`id`、`supplier_order_id`、`supplier_platform`、`shipment_no`、`external_shipment_no`、`carrier_code`、`carrier_name`、`tracking_no`、`status`（`pending`/`shipped`/`in_transit`/`delivered`/`exception`/`returned`）、`shipped_at`、`delivered_at`、`basis_history_node_id`、`basis_projection_hash`、`basis_payload_snapshot`、`raw_payload`、`extra_data`、时间戳。

## ShipmentLine

字段：`id`、`shipment_id`、`fulfillment_line_id`、`supplier_order_line_id`、`quantity`、`created_at`。

处理拆包/合包场景。保留 basis 引用便于偏离判断。

## ChannelSyncJob

字段：`id`、`wave_id`、`integration_profile_id`、`direction`（`push_tracking`）、`status`（`pending`/`running`/`success`/`partial_success`/`failed`）、`basis_history_node_id`、`basis_projection_hash`、`basis_payload_snapshot`、`request_payload`、`response_payload`、`error_message`、`started_at`、`finished_at`、时间戳。

## ChannelSyncItem

字段：`id`、`channel_sync_job_id`、`fulfillment_line_id`、`shipment_id`、`external_document_no`、`external_line_no`、`tracking_no`、`carrier_code`、`status`、`error_message`、时间戳。

### 约束

- 回填成功/失败必须可追踪
- 不是所有需求都生成 `ChannelSyncJob`（`routing_disposition != accepted` 的不生成）
- 使用 `integration_profile_id` 比重复落 `source_channel/source_surface` 更稳妥
- 保留 basis 引用，便于回填后再修改时给出失配提示
