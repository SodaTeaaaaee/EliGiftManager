# 统一业务语言

> V2 核心名词、边界关系、旧术语映射与判断规则。
> 当前代码已按这些术语实现（见 `internal/domain/models.go` 和 `internal/domain/enums.go`）。

## 4. V2 统一业务语言

### 4.1 统一名词

| 术语 | 说明 | V1 对应 |
|------|------|---------|
| `CustomerProfile` | 全局履约对象（会员/买家/手工补录） | `Member` |
| `CustomerIdentity` | 某个履约对象在某个平台上的身份 | `(platform, platform_uid)` |
| `DemandDocument` | 上游需求单（`membership_entitlement` 或 `retail_order`） | 无 |
| `DemandLine` | 上游需求行 | 无 |
| `Wave` | 一次履约批次（V2 升级为全链路容器） | `Wave` |
| `WaveParticipantSnapshot` | 某个履约对象在该波次里的快照 | `WaveMember` |
| `FulfillmentLine` | 实际需要执行的一条履约行 | `DispatchRecord` |
| `SupplierOrder` / `SupplierOrderLine` | 发给工厂的执行单及行项目 | 无 |
| `Shipment` / `ShipmentLine` | 工厂回传后的发货实体 | 无 |
| `ChannelSyncJob` / `ChannelSyncItem` | 物流信息回填到来源渠道的同步任务 | 无 |
| `IntegrationProfile` | 来源渠道+业务面的统一配置入口 | `TemplateConfig`（部分） |
| `FulfillmentAdjustment` | 初始履约结果之上的人工/系统调整 | `ProductTag.user`（部分） |
| `AllocationPolicyRule` | `policy_driven` 分配模式下的规则对象 | `ProductTag` |
| `ProductMaster` / `Product` | 全局商品主档 / 波次商品快照 | 同名 |
| `CarrierMapping` | 内部承运商编码到外部编码的映射 | 无 |

### 4.1.1 关键边界关系

| 区分 | 说明 |
|------|------|
| `CustomerProfile` ≠ `CustomerIdentity` | 一个 Profile 可有多个 Identity |
| `DemandDocument` ≠ `Wave` | 需求单是上游真相，波次是履约容器。通过 `WaveDemandAssignment` 关联 |
| `DemandLine` ≠ `FulfillmentLine` | 需求行是"应得/下单"，履约行是"最终执行" |
| `FulfillmentLine` ≠ `SupplierOrderLine` | 履约行是内部真相，工厂单行是一次提交。一行可多次提交 |
| `SupplierOrder` ≠ `Shipment` | 工厂接单 ≠ 工厂发货 |
| `Shipment` ≠ `ChannelSyncJob` | 物流存在 ≠ 外部渠道已知道 |
| `RoutingDisposition` ≠ 履约完成状态 | "接不接手" ≠ "执行到哪了" |

### 4.1.2 旧术语映射

| 旧说法 | V2 说法 |
|--------|---------|
| 会员 | `CustomerProfile` / `WaveParticipantSnapshot` |
| 平台 UID | `CustomerIdentity` |
| 导入名单 | Demand import |
| 订单 | `DemandDocument`（上游）或 `SupplierOrder`（工厂） |
| 发货记录 | `FulfillmentLine` |
| 工厂导出 | `SupplierOrder export` |
| 快递信息回填 | `ChannelSync` |

### 4.1.3 命名收敛原则

1. 新对象直接使用 V2 目标名，不先用旧名占位
2. 旧名可短暂作为 compatibility alias，但不应长期并行
3. 当前没有历史包袱，应尽早统一

### 4.1.4 判断规则

遇到新字段/模板/页面，不确定落在哪层时：

| 问 | 答 |
|----|----|
| "这个人是谁" vs "在哪个平台上是谁"？ | `CustomerProfile` vs `CustomerIdentity` |
| "上游原始需求" vs "最终执行"？ | `DemandDocument/DemandLine` vs `FulfillmentLine` |
| "提交给工厂了没" vs "真的发货了没"？ | `SupplierOrder` vs `Shipment` |
| "物流存在" vs "物流已同步回渠道"？ | `Shipment` vs `ChannelSyncJob` |
| 身份来源 / 来源渠道 / 工厂来源 / 承运商？ | `identity_platform` / `source_channel` / `supplier_platform` / `carrier_code` |
| "支持资格本身" vs "后续订单"成立？ | `membership_entitlement` vs `retail_order` |
| 本系统判定 vs 上游平台判定？ | `EntitlementAuthority = local_policy` vs `upstream_platform` |
| "未被接手" vs "已接手但未完成"？ | `RoutingDisposition` vs `FulfillmentLine`/`Shipment`/`ChannelSyncJob` |
| 改上游真相 / 改默认逻辑 / 改最终例外？ | 前两者回前置层，后者进 `Adjustment Review` |
| 动态 selector 规则 vs 具体对象结果？ | `AllocationSelector` vs `FulfillmentAdjustment` |
| 本地历史 vs 回滚外部动作？ | `HistoryScope/HistoryNode` vs 不可 undo |

### 4.2 统一平台词汇

V2 必须把"平台"拆开：

| 维度 | 说明 | 示例 |
|------|------|------|
| `identity_platform` | 用户身份来源 | Bilibili、Patreon |
| `source_channel` | 平台供应商 | patreon、gumroad、bilibili |
| `source_surface` | 供应商下的业务面 | membership、shop_purchase |
| `supplier_platform` | 工厂/供应商平台 | 柔造 |
| `carrier_code` | 物流承运商编码 | SF、YT |

关键：`source_channel` + `source_surface` 必须拆开。同一平台可有多个业务面（如 `patreon.membership` vs `patreon.shop_purchase`）。

### 4.2.1 平台差异的归属层

- **核心履约层尽量平台无关**：`Wave`、`FulfillmentLine`、`SupplierOrder`、`Shipment`
- **平台差异落在配置与边界层**：`DemandDocument`、`IntegrationProfile`、`DocumentTemplate`、`ChannelSyncJob`
- **Service 层负责翻译**：把平台差异翻译成统一履约动作

### 4.2.2 `IntegrationProfile` 定位

`profile_key` 格式：`<source_channel>.<source_surface>`，如 `patreon.membership`、`bilibili.creator_commerce`。

它回答的不是"CSV 长什么样"，而是：
- 这个来源是哪个业务面
- 归类为哪种需求语义（`membership_entitlement` / `retail_order`）
- 义务由资格成立还是订单成立
- 如何识别履约对象
- 权益判定权威来自哪里
- 是否支持物流回填
- 绑定哪些文档模板与连接器

### 4.2.3 平台权威判定的阶段性会员礼物

`EntitlementAuthority = upstream_platform` 时，本系统不自行重算资格，只负责输入采集、路由与履约执行。"本系统不接手"（`excluded_manual`）≠"系统确认外部已履约"。
