# 商品、履约与工厂执行模型

> V2 数据模型后半部分：商品层、履约层、工厂执行层。
> 当前实现见 `internal/domain/models.go`。

## 商品层

### ProductMaster
全局商品主档。扩展 `product_kind`、`supplier_platform`、`supplier_product_ref`、`archived`。

### Product
波次级商品快照，与 `ProductMaster` 生命周期分离。

## 履约层

### FulfillmentLine

核心字段：`id`、`wave_id`、`customer_profile_id`、`wave_participant_snapshot_id`、`product_id`、`demand_document_id`、`demand_line_id`、`customer_address_id`、`quantity`、`allocation_state`、`address_state`、`supplier_state`、`channel_sync_state`、`line_reason`（`entitlement`/`retail_order`/`wave_adjustment`）、`extra_data`、时间戳。

约束：
- `quantity` ≥ 0（最终执行结果非负）
- 未被接手的需求不产生 `FulfillmentLine`
- 多维状态而非单一 `status` 字段

### FulfillmentAdjustment

显式调整层对象。字段：`id`、`wave_id`、`fulfillment_line_id`、`wave_participant_snapshot_id`、`from_product_id`、`to_product_id`、`adjustment_kind`（`add`/`reduce`/`replace`/`compensation`/`remove`）、`quantity_delta`、`reason_code`、`note`、`created_by`、`extra_data`、时间戳。

约束：
- 只用于"当前波次最终履约例外"
- `quantity_delta` 可正可负
- target 是具体对象，不是动态 selector
- 基础重算时先重建基础 → 再重放 adjustment

## 工厂执行层

### SupplierOrder

字段：`id`、`wave_id`、`supplier_platform`、`template_id`、`batch_no`、`external_order_no`、`submission_mode`（`csv`/`manual`/`api`）、`submitted_at`、`status`（`draft`→`submitted`→`accepted`→`partially_shipped`→`shipped`→`canceled`）、`basis_history_node_id`、`basis_projection_hash`、`basis_payload_snapshot`、`extra_data`、时间戳。

当前阶段理解为"最近一次工厂导出工作区对象"，保留 basis 快照便于偏离判断。

### SupplierOrderLine

字段：`id`、`supplier_order_id`、`fulfillment_line_id`、`supplier_line_no`、`supplier_sku`、`submitted_quantity`、`accepted_quantity`、`status`、`extra_data`、时间戳。

### Shipment / ShipmentLine

工厂回传后的发货实体。字段：`id`、`wave_id`、`supplier_order_id`、`carrier_code`、`tracking_no`、`shipped_at`、`status`、`basis_history_node_id`、`extra_data`、时间戳。

### CarrierMapping

内部承运商编码到平台外部编码的映射。
