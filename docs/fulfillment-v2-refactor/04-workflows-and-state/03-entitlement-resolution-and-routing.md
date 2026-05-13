# 会员权益判定、输入采集与路由模型

本文件专门说明 `membership_entitlement` 在进入本系统履约链路之前，还需要经过哪些前置判断与处理。

## 为什么需要这一层

`membership_entitlement` 与普通 `retail_order` 的关键差异，不只是“有没有订单号”，而是：

- 权益是否成立，往往不是由下单事件直接表达
- 地址、尺码、款式、组合等参数，往往晚于权益成立才出现
- 本系统有时只接手其中一部分，另一部分由创作者线下处理

因此在 `DemandLine -> FulfillmentLine` 之间，必须增加一层更精确的语义。

## 1. 权益成立的判定权威

建议使用：

- `entitlement_authority`

它回答的是：

- “谁有权说这条权益已经成立？”

推荐值：

- `local_policy`
- `upstream_platform`
- `manual_grant`

判断原则：

- 如果平台自己定义达成规则，且掌握完整账本，应优先相信 `upstream_platform`
- 如果只是创作者本地规则、活动规则或导入名单规则，则更接近 `local_policy`
- 如果是补偿、手工授予、特殊照顾，则更接近 `manual_grant`

## 2. 会员输入采集不等于平台 claim 按钮

建议使用：

- `recipient_input_state`

它回答的是：

- “把这条权益真正转成可执行履约，还缺哪些收货对象输入？”

输入来源可能包括：

- 平台原生 claim
- 外部表单
- 私聊协商
- 手工录入

因此这里不应把产品语义写死成“平台 claim”。

推荐值：

- `not_required`
- `waiting_for_input`
- `partially_collected`
- `ready`
- `waived`
- `expired`

## 3. 本系统是否接手，不等于外部履约是否完成

建议使用：

- `routing_disposition`

它回答的是：

- “这条需求是否进入 EliGiftManager 的处理范围？”

推荐值：

- `pending_intake`
- `accepted`
- `deferred`
- `excluded_manual`
- `excluded_duplicate`
- `excluded_revoked`

必须明确：

- `excluded_manual` 只表示“本系统这次不接手”
- 它不是“系统外已经履约完成”的真相声明

这条边界决定了 EliGiftManager 仍然是：

- 履约辅助与路由系统

而不是：

- 所有系统外履约事实的总账本

## 4. 何时进入波次

建议规则：

- 只有 `routing_disposition = accepted` 的需求，才进入稳定的波次处理语义
- `deferred` 的需求可以保留在导入侧或候选池中，但不应伪装成已进入执行链
- `excluded_manual` 的需求应进入统计，但单独归类

这里还要再补一条：

- `accepted` 只表示“本系统接手”
- 不自动等于“已经归属到某个具体 wave”

更准确地说：

- 本系统接手，是 routing 层语义
- 归属到哪个 wave，是 `WaveDemandAssignment` 的关系语义

当前阶段的默认约束是：

- 同一份 `DemandDocument`
- 通常不跨多个活跃 wave 拆分处理

但仍保留显式 assignment relation，而不是把它并回 `DemandDocument` 本体。

对 `accepted` 的需求，又应再区分：

- `recipient_input_state = ready`
  - 可以较稳定地进入履约生成
- `recipient_input_state = waiting_for_input / partially_collected`
  - 可以进入波次上下文，但不应被过早视为完全可执行履约

## 5. 统计应如何呈现

当前更合理的统计方式不是把所有导入项都混成“待处理”，而是分开显示：

- `accepted`
- `waiting_for_input`
- `deferred`
- `excluded_manual`
- `excluded_duplicate`
- `excluded_revoked`

其中：

- `excluded_manual` 应该进入统计
- 但必须单独归类为“未纳入本系统处理”

这样用户才能分清：

- 是漏了
- 还是本来就决定不由本系统处理

## 6. 两条典型判断规则

### A. 连续订阅阶段性礼物

例如：

- Patreon merch for membership

建议判断：

- `demand_kind = membership_entitlement`
- `obligation_trigger_kind = loyalty_membership`
- `entitlement_authority = upstream_platform`

如果平台已权威判定 earned，就不要由本系统自行重算。

### B. 支持者限定购买

例如：

- FANBOX 支持者限定的 BOOTH 或其他外部销售面订单

建议判断：

- `demand_kind = retail_order`
- `obligation_trigger_kind = supporter_only_purchase`
- 保留 `eligibility_context_ref`

也就是说：

- 支持资格只提供购买资格
- 真正的履约义务仍由订单成立触发

## 7. 这层和后续执行层的边界

这一层解决的是：

- 权益是否成立
- 输入是否收齐
- 本系统是否接手

后面的 `FulfillmentLine / SupplierOrder / Shipment / ChannelSync` 解决的是：

- 一旦接手后，怎么执行
- 发到哪里
- 工厂怎么做
- 物流怎么回填

不要让前置判定层和后续执行层互相吞并。
