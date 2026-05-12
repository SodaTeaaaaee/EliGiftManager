# 统一业务语言

本文件整理 V2 的核心名词、边界关系、旧术语映射与判断规则，用于避免后续实现继续在旧语义上硬补。

## 4. V2 统一业务语言

### 4.0 为什么必须先统一语言

本次重构里，统一业务语言不是文档洁癖，而是工程前提。

如果不先统一语言，后续会不断出现下面这些误解：

- 把“会员”误当成所有消费者
- 把“订单”误当成所有上游来源
- 把“发货记录”误当成“工厂订单”
- 把“导出成功”误当成“履约完成”
- 把“平台”误当成单一维度字段
- 把“需求类型”误当成“平台类别”

这些误解会直接导致：

- 表结构命名错误
- service 职责混乱
- 模板系统为了兼容术语混乱而不断堆特例
- 状态系统无法稳定扩展

因此 V2 的第一原则是：

- 先定义业务语言
- 再定义数据结构
- 最后才是页面、模板和交互

### 4.1 统一名词

后续文档、代码、模板、页面文案，应尽量统一使用以下业务语言。

#### A. CustomerProfile

表示全局履约对象。

它回答的问题是：

- “这个收货对象在系统里是谁？”

它不再只代表“会员”，而是同时覆盖：

- 会员
- 买家
- 手工补录的履约对象

短期可以不立即改表名 `Member`，但文档和 service 层语义应逐步迁移到
`CustomerProfile`。

#### B. CustomerIdentity

表示某个履约对象在某个平台上的身份。

它回答的问题是：

- “这个人在平台 X 上是谁？”

典型例子：

- Bilibili UID
- Patreon member id
- Gumroad buyer email
- 外部 creator-commerce buyer id

#### C. DemandDocument

表示上游需求单，是“为什么系统要履约这件事”的起点。

它不强制等于电商订单。

`kind` 至少包括：

- `membership_entitlement`
- `retail_order`
- `manual_adjustment`

#### D. DemandLine

表示上游需求行。

它回答的问题是：

- “这张需求单里，具体有哪些待履约的行？”

会员场景：

- 行内容可能是身份等级、权益代码、活动资格
- 未必已经对应到最终工厂 SKU

零售场景：

- 行内容通常已经包含明确商品、数量、外部订单行号

在更复杂的会员权益场景里，它还可以承担：

- 平台已判定成立的一条权益
- 仍待会员补充参数的一条权益候选
- 本系统决定暂不接手处理的一条权益候选

#### E. Wave

继续保留，表示一次履约批次。

它回答的问题是：

- “这次履约任务的业务边界是什么？”

V2 不移除 Wave，而是把它从“导出终点容器”升级为“全链路履约容器”。

#### F. WaveParticipantSnapshot

表示某个履约对象在该波次里的快照。

它是当前 `WaveMember` 的泛化版本。

它回答的问题是：

- “这个人在本次波次里以什么身份参与？”

会员场景：

- 可能带 `giftLevel`

零售场景：

- 可能没有 `giftLevel`
- 但仍然需要保留渠道来源、昵称、订单来源信息等快照

#### G. FulfillmentLine

表示实际需要执行的一条履约行。

它是当前 `DispatchRecord` 的 V2 业务语义。

它回答的问题是：

- “最终要给谁发什么、发多少、寄到哪里、目前执行到哪一步？”

#### H. SupplierOrder / SupplierOrderLine

表示发给工厂或供应商的一次提交，以及其行项目。

它回答的问题是：

- “这批工厂单是什么时候提交的？”
- “提交给哪个工厂平台？”
- “外部订单号是什么？”
- “对应了哪些履约行？”

当前阶段还应补一句：

- 它更接近当前工作区里的最近一次导出/提交辅助对象
- 不是默认不可覆盖的历史账本

#### I. Shipment / ShipmentParcel / ShipmentLine

表示工厂回传后的发货实体。

它回答的问题是：

- “工厂是否已经发货？”
- “包裹号是什么？”
- “承运商是什么？”
- “这次发货覆盖了哪些履约行？”

当前阶段也应补一句：

- 它表达的是当前工作区里最近一次已知物流结果
- 不自动等于永久冻结的历史归档

#### J. ChannelSyncJob

表示把物流信息回填到来源渠道的一次同步任务。

它回答的问题是：

- “物流是否已成功回填？”
- “失败了吗？”
- “是否需要重试？”

#### K. AllocationMode

表示某类需求或某个波次默认采用哪种“初始分配生成方式”。

推荐语义：

- `rule_based`
  - 由规则推导初始履约结果
- `direct_from_demand`
  - 由上游需求行直接生成初始履约结果
- `hybrid`
  - 同时允许规则推导和需求直入并存

说明：

- 会员权益型需求通常更接近 `rule_based`
- 零售订单型需求通常更接近 `direct_from_demand`
- 混合波次通常更接近 `hybrid`

#### L. FulfillmentAdjustment

表示在“初始履约结果”之上的人工或系统调整层。

它回答的问题是：

- “在原始规则或原始订单之外，又对最终履约做了什么修正？”

典型例子：

- 加送
- 减送
- 替换
- 补发
- 取消

说明：

- 在当前实现中，部分调整是通过 `user tag` 间接表达的
- 在 V2 长期目标中，这类调整应逐步演进为显式的履约调整对象

#### M. ObligationTriggerKind

表示“履约义务究竟因什么事件成立”。

它回答的问题是：

- “系统为什么现在欠这位用户一条待履约义务？”

建议语义：

- `periodic_membership`
  - 周期性会员/支持关系本身触发权益
- `loyalty_membership`
  - 连续订阅、阶段性成就、earned benefit 触发权益
- `supporter_only_purchase`
  - 支持者资格只提供购买资格，义务由后续订单触发
- `member_only_discount_purchase`
  - 会员专属折扣或专属购买入口触发订单
- `campaign_reward`
  - 活动或企划规则触发
- `manual_compensation`
  - 人工补偿或人工授予触发

它的关键价值在于：

- 不新增顶层 `demand_kind`
- 但仍能区分“会员权益怎么成立”和“订单为什么成立”

#### N. EntitlementAuthority

表示“谁有权判定这条会员权益已经成立”。

它回答的问题是：

- “系统为什么相信这条权益现在是真的应当履约？”

建议语义：

- `local_policy`
  - 由本系统或创作者本地规则判定
- `upstream_platform`
  - 由上游会员平台的权威结果判定
- `manual_grant`
  - 由人工显式授予

这个概念尤其重要，因为：

- 像 Patreon merch for membership 这类连续订阅成就
- 很多时候应由平台自己给出权威达成结果
- 而不应由本系统在缺乏完整历史的情况下自行重算

#### O. RecipientInputState

表示“会员或收货对象是否已经补齐本次履约所需输入”。

它回答的问题是：

- “本次权益现在能不能真正转成可执行履约？”

这里的“输入”可能包括：

- 地址
- 款式
- 尺码
- 颜色
- 组合选项
- 领取确认
- 通过表单或协商补充的其他参数

建议语义：

- `not_required`
- `waiting_for_input`
- `partially_collected`
- `ready`
- `waived`
- `expired`

这里刻意不用“claim button”语义，是因为：

- 会员输入未必通过平台原生 claim 按钮完成
- 也可能通过表单、私聊协商、人工登记等方式完成

#### P. RoutingDisposition

表示“本系统是否接手处理这条需求，以及为什么”。

它回答的问题是：

- “这条需求是否进入本系统流程？”

建议语义：

- `pending_intake`
- `accepted`
- `deferred`
- `excluded_manual`
- `excluded_duplicate`
- `excluded_revoked`

这里必须明确：

- 它记录的是本系统的路由与处理范围决策
- 不是系统外履约完成事实
- `excluded_manual` 的含义是“本系统这次不接手”
- 不是“系统确认外部已经履约完毕”

#### Q. AllocationSelector

表示“某条分配规则打算命中哪些参与者”。

它回答的问题是：

- “这条规则到底想作用于谁？”

当前阶段更稳妥的理解是：

- 它属于 `Membership Allocation` 或更早的规则层语言
- 不属于 `Adjustment Review` 的最终例外语义

首版更建议只覆盖：

- 单平台身份
- `platform_all`
- `wave_all`
- 显式用户覆盖

如果未来真的出现交集、子集或更复杂的多条件 selector：

- 也应优先扩展这一层
- 而不是把共享调整层演化成第二套规则引擎

#### R. AllocationContribution

表示“某条规则在命中某个参与者后，为某件商品贡献了多少数量”。

它回答的问题是：

- “这条规则命中后，对基础分配结果加了多少、减了多少？”

重要边界：

- 这一层可以出现正数或负数
- 它不是最终履约行
- 它更接近规则求和时的中间语义

#### S. HistoryScope / HistoryNode

表示“本地工作区撤销/重做历史的作用域”和“其中一次用户意图操作”。

它回答的问题是：

- “这次撤销/重做到底属于哪个工作区？”
- “刚才用户执行的那一步操作是什么？”

重要边界：

- 它记录的是本地工作区编辑历史
- 不等于外部世界已经回滚
- 也不等于工厂、物流、来源渠道会自动跟着撤销

#### T. Basis Reference / Basis Projection

表示“某次导出、物流导入或渠道回填，当时依赖的是哪一个本地工作区结果”。

它回答的问题是：

- “这个外部对象当时是基于哪一个工作区 head 生成的？”
- “当前工作区是否已经偏离了它当时依赖的基础？”

重要边界：

- 它服务于偏离提示和 review 语义
- 不等于把整个系统做成不可覆盖的历史账本
- 更不等于普通 undo/redo 的本地节点本身

### 4.1.1 这些词之间的边界关系

后续讨论时，最容易混淆的不是某个词本身，而是相邻几个词之间的边界。

下面把最重要的几组边界明确下来。

#### A. CustomerProfile 不是 CustomerIdentity

- `CustomerProfile` 回答：
  - “系统里这个履约对象是谁？”
- `CustomerIdentity` 回答：
  - “这个履约对象在某个平台上是谁？”

一个 `CustomerProfile` 可以有多个 `CustomerIdentity`。

典型例子：

- 同一个人既可能以 Bilibili 直播支持身份出现
- 又可能用邮箱在 Gumroad 下单

在 V2 里，这两种身份应能归并到同一个全局履约对象，而不是被系统视为两个互不相关的人。

#### B. DemandDocument 不是 Wave

- `DemandDocument` 回答：
  - “上游为什么产生了这次履约需求？”
- `Wave` 回答：
  - “这些需求在本次履约中如何被组织和执行？”

上游需求单和波次不是一回事。

一个波次可以包含多张需求单。
一张需求单在极端情况下也可能被拆到多个波次处理。

#### C. DemandLine 不是 FulfillmentLine

- `DemandLine` 回答：
  - “用户原始应得/下单的是什么？”
- `FulfillmentLine` 回答：
  - “最终要执行发出的是什么？”

这两者在会员场景里尤其不能混：

- 会员需求行可能只是“提督权益”
- 最终履约行才是“立牌 x2、徽章 x1”

在零售场景里，两者可能非常接近，但语义仍不同。

#### D. FulfillmentLine 不是 SupplierOrderLine

- `FulfillmentLine` 是系统内部要执行的履约真相
- `SupplierOrderLine` 是发给工厂的一次具体提交行

一个履约行未来可能：

- 一次提交给工厂
- 多次补发给工厂
- 被不同工厂拆分处理

因此不能把工厂单直接等同于系统内部履约行。

#### E. SupplierOrder 不是 Shipment

- `SupplierOrder` 表示提交给工厂的执行单
- `Shipment` 表示工厂已经发货后的物流实体

工厂接单不等于工厂发货。
工厂发货也不等于来源渠道已经收到物流更新。

#### F. Shipment 不是 ChannelSyncJob

- `Shipment` 表示物流真相
- `ChannelSyncJob` 表示把物流真相同步给外部来源渠道的任务

物流已经存在，不代表外部来源渠道已经知道这件事。

#### G. RoutingDisposition 不是履约完成状态

- `RoutingDisposition` 回答：
  - “本系统接不接手这条需求？”
- `FulfillmentLine` / `Shipment` / `ChannelSyncJob` 回答：
  - “一旦接手后，执行到哪里了？”

如果把这两层混在一起，就会把：

- 未纳入本系统处理
- 还没进入波次
- 已进入波次但未发货
- 已发货但未回填

这些完全不同的问题混成一个状态字段。

### 4.1.2 旧术语与新术语的映射关系

为了降低迁移期沟通成本，下面给出旧术语和新术语的参考映射。

| 旧说法 | V2 建议说法 | 说明 |
| --- | --- | --- |
| 会员 | `CustomerProfile` / `WaveParticipantSnapshot` | 视上下文决定是全局对象还是波次快照 |
| 买家 | `CustomerProfile` | 买家和会员同属履约对象 |
| 平台 UID | `CustomerIdentity` | 不再把单一 UID 当作全局客体本身 |
| 导入名单 | `Demand import` / `participant import` | 不再默认所有导入都是会员名单 |
| 订单 | `DemandDocument` 或 `SupplierOrder` | 需要先分清是上游订单还是工厂订单 |
| 发货记录 | `FulfillmentLine` | 表达系统内部履约行语义 |
| 工厂导出 | `SupplierOrder export` | 导出是工厂执行层动作 |
| 快递信息回填 | `ChannelSync` | 明确它是回填来源渠道的任务，而不是物流真相本身 |

说明：

- 迁移期允许旧名作为桥接别名短暂存在
- 但任何新功能设计、文档写作和 service 命名都应优先按新名思考
- 不应为了“以后可能兼容”而让旧名长期和新名并列存在

### 4.1.3 命名收敛原则

当前没有历史包袱，因此 V2 应当尽早收敛命名。

推荐原则：

1. 新对象、新字段、新页面优先使用目标名

- 不要先用旧名占位，再找机会改
- 只要语义已经明确，就直接使用新名

2. 旧名只作为过渡桥接

- 旧表名、旧控制器名、旧 DTO 字段名可以在迁移期短暂保留
- 但它们应明确被视为 compatibility alias，而不是并行主语义

3. 不要把“没有历史包袱”误解成“可以慢慢想”

- 正因为没有复杂旧账，才更应该尽快统一
- 命名迟疑会直接制造新的技术债

4. 真正需要谨慎的是物理迁移成本，不是语义决策本身

- 语义一旦定下，应尽早在新代码里稳定使用
- 物理重命名是否分阶段，取决于具体迁移风险和成本

### 4.1.4 业务语言的判断规则

后续如果遇到一个新字段、新模板或新页面，不确定应该落在哪一层，可以先问下面这些问题。

1. 这个信息回答的是“这个人是谁”还是“这个人在哪个平台上是谁”？

- 前者更接近 `CustomerProfile`
- 后者更接近 `CustomerIdentity`

2. 这个信息回答的是“上游原始需求是什么”还是“最终要执行发什么”？

- 前者更接近 `DemandDocument / DemandLine`
- 后者更接近 `FulfillmentLine`

3. 这个信息回答的是“提交给工厂了没”还是“真的发货了没”？

- 前者更接近 `SupplierOrder`
- 后者更接近 `Shipment`

4. 这个信息回答的是“物流是否存在”还是“物流是否已同步回外部来源渠道”？

- 前者更接近 `Shipment`
- 后者更接近 `ChannelSyncJob`

5. 这个字段是身份来源、来源渠道、工厂来源，还是物流承运商？

- 应分别落到：
  - `identity_platform`
  - `source_channel`
  - `source_surface`
  - `supplier_platform`
  - `carrier_code`

6. 这条义务是由“支持资格本身”成立，还是由“后续订单”成立？

- 前者更接近 `membership_entitlement`
- 后者更接近 `retail_order`
- 进一步可由 `ObligationTriggerKind` 补充说明

7. 这条权益是本系统自己判定的，还是上游平台已经权威判定过的？

- 前者更接近 `EntitlementAuthority = local_policy`
- 后者更接近 `EntitlementAuthority = upstream_platform`

8. 这条需求是“未被本系统接手”，还是“已接手但执行未完成”？

- 前者更接近 `RoutingDisposition`
- 后者才应该继续看 `FulfillmentLine`、`Shipment`、`ChannelSyncJob`

9. 这次修改是在改“上游真相”“默认生成逻辑”，还是“本波次最终履约例外”？

- 如果在改上游真相，更接近 `DemandDocument / DemandLine / RecipientInputState / RoutingDisposition`
- 如果在改默认生成逻辑，更接近 `AllocationPolicyRule` 或 demand mapping 规则
- 如果在改本波次最终履约例外，更接近 `FulfillmentAdjustment`

这个判断规则很重要，因为它直接决定：

- 问题该回前置页面处理
- 还是进入共享的 `Adjustment Review`

10. 这次变更想改的是“动态 selector 规则”，还是“已经展开后的具体对象结果”？

- 如果想改的是身份、平台、集合未来如何继续生效，更接近 `AllocationSelector / AllocationContribution`
- 如果想改的是当前已经展开的具体参与者或具体履约行，更接近 `FulfillmentAdjustment`

11. 这次回退是在改“本地工作区历史”，还是想“回滚外部动作”？

- 前者更接近 `HistoryScope / HistoryNode`
- 后者并不应被普通 undo/redo 自动承担
- 外部对象更适合通过 `Basis Reference / Basis Projection` 表达其与当前工作区的偏离关系

### 4.1.5 两个完整例子

为了让上面的术语不只停留在抽象层，这里给出两个完整例子。

#### 例子 A：会员回馈

原始现实：

- 某创作者导出 5 月会员名单
- 用户 A 是 Bilibili 提督
- 按波次规则应获得 2 个立牌

V2 语言下可解释为：

- `CustomerProfile`
  - 用户 A 这个全局履约对象
- `CustomerIdentity`
  - Bilibili UID = xxx
- `DemandDocument`
  - 5 月会员权益导入单
- `DemandLine`
  - 用户 A 的“提督权益”
- `ObligationTriggerKind`
  - `periodic_membership`
- `EntitlementAuthority`
  - `local_policy`
- `RecipientInputState`
  - `waiting_for_input` 或 `ready`
- `RoutingDisposition`
  - `accepted`
- `WaveParticipantSnapshot`
  - 用户 A 在 5 月波次中的身份快照
- `FulfillmentLine`
  - 立牌 x2
- `SupplierOrder`
  - 本次提交给工厂的平台订单
- `Shipment`
  - 工厂发货后回传的快递单
- `ChannelSyncJob`
  - 如果该会员来源支持物流回写，则将物流同步回来源渠道的任务

#### 例子 B：商城零售

原始现实：

- 用户 B 在 Gumroad 的 one-time order surface 下单购买 1 个需要寄送的商品
- 工厂完成生产后回传了物流单号
- 系统需要依据该来源业务面的能力，决定是生成来源渠道回填任务，还是只做人工闭环确认

V2 语言下可解释为：

- `CustomerProfile`
  - 用户 B 这个全局履约对象
- `CustomerIdentity`
  - buyer email / external buyer id
- `DemandDocument`
  - Gumroad one-time order #12345
- `DemandLine`
  - 挂件 x1
- `ObligationTriggerKind`
  - `supporter_only_purchase` 或普通零售订单触发语义
- `RoutingDisposition`
  - `accepted`
- `WaveParticipantSnapshot`
  - 用户 B 在本次波次中的快照
- `FulfillmentLine`
  - 最终需要发出的挂件 x1
- `SupplierOrder`
  - 提交给工厂的执行单
- `Shipment`
  - 工厂回传的物流单号
- `ChannelSyncJob`
  - 如果该来源业务面支持回填，则执行同步；否则记录 `unsupported`、`skipped` 或 `manual_confirmed`

#### 例子 C：平台权威判定的阶段性会员礼物

原始现实：

- Patreon 或类似平台已经判定用户 C earned 某个阶段性 merch benefit
- 创作者这次决定将其中一部分导入系统处理
- 另一部分因数量很少而由创作者自己线下处理

V2 语言下可解释为：

- `DemandDocument`
  - 某次导入的平台权威 earned benefit 快照
- `DemandLine`
  - 用户 C 的某条阶段性会员权益
- `ObligationTriggerKind`
  - `loyalty_membership`
- `EntitlementAuthority`
  - `upstream_platform`
- `RecipientInputState`
  - 可能先是 `waiting_for_input`
- `RoutingDisposition`
  - 对进入系统的项是 `accepted`
  - 对不进入系统的项是 `excluded_manual`

这个例子强调：

- 平台权威判定与本系统是否接手履约，是两件不同的事
- 本系统不接手，不等于本系统拥有系统外履约完成真相

