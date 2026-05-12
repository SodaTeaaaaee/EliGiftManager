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
- `quantity` 应被理解为最终进入执行链的非负结果
- 规则贡献层里的负数、共享调整层里的负 delta，都不应直接以负数 `FulfillmentLine` 长期存在
- 如果最终结果被压到 0，更合理的解释是“本次无可执行履约行”，而不是保留一条负数执行记录

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
- 只有在实现路径尚未切换完成时，部分调整才允许暂时继续通过 `user tag` 实现
- 但从长期语义上讲，`user tag` 更接近调整实现，而不是最终理想形态
- 这层只应用于“当前波次最终履约例外”
- 不应用于记录上游需求真相修正、接手边界修正或默认生成逻辑修正
- 不应用于凭空新增一个尚未进入当前波次处理范围的全新参与者
- 它的 target 应是具体 `FulfillmentLine`、具体参与者，或明确展开后的具体对象集合
- 首版不应让它直接持有“身份 / 平台 / 交集 selector”这类动态目标

还应明确：

- `quantity_delta` 可以是正数或负数
- 负 delta 本身不是错误
- 只有当 target 失效、引用歧义，或与外部 basis 出现结构性错配时，才应进入 review 语义

建议补充一条实现约束：

- 当基础分配结果需要重算时，应先重建基础履约结果，再显式重放 `FulfillmentAdjustment`

这样才能保证：

- 前置页面修正规则或映射时，不会悄悄吃掉共享调整层已经确认的例外
- 跨步骤跳转和重新计算仍然可解释、可审计

但这里还要再补一条现实边界：

- 当前软件是辅助履约工作区，而不是不可更改的历史账本
- 因此“可审计”更偏向“当前变换链条可解释”
- 不强制等于“每一次历史提交都永久冻结为不可覆盖事实”

如果未来真有“波次内动态 selector 继续随重算自动生效”的需求：

- 更稳妥的做法是新增独立的 `WaveScopedOverlayRule`
- 它应被视为规则层扩展，而不是把 `FulfillmentAdjustment` 演化成第二套动态规则引擎
- 当前阶段不建议首版直接实现

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
- `basis_history_node_id`
- `basis_projection_hash`
- `basis_payload_snapshot`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- `SupplierOrder` 在当前阶段更适合被理解为“最近一次工厂导出 / 提交工作区对象”
- 它用于承接模板映射、导出参数、回传关联和当前辅助状态
- 建议同时保留本次导出所依据的履约基础快照引用，便于后续判断当前工作区是否已经偏离最近一次导出基础
- 不应被默认理解为不可覆盖的权威历史账本

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
- 当用户在导出后继续修改波次内容时，这层仍然可以被后续导出结果覆盖或重建
- 这层也应保留与基础履约结果的关联，便于重算后显式重建，而不是隐式继承旧提交结果

