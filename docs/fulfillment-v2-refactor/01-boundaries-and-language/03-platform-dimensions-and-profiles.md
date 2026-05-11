# 平台维度与 Profile 定位

本文件专门整理平台相关讨论，包括是否需要分平台、真正应该分的维度，以及 `IntegrationProfile` 的未来定位。

### 4.2 统一平台词汇

V2 必须把“平台”拆开说，禁止继续在关键领域对象里只写一个含糊的 `platform`。

推荐至少区分：

- `identity_platform`
  - 用户身份来源平台
- `source_channel`
  - 平台供应商或来源渠道供应商
- `source_surface`
  - 该供应商下的具体业务面
- `supplier_platform`
  - 工厂/供应商平台
- `carrier_code`
  - 物流承运商编码体系

这里最重要的不是“平台名”，而是“平台供应商”和“业务面”必须拆开。

例如，官方资料已经足以说明：

- Patreon 不能被简单等同于“会员平台”
- Gumroad 不能被简单等同于“零售平台”
- pixivFANBOX 不能被简单等同于“零售订单平台”
- Bilibili 不能被简单等同于某一个单一业务面

因此文档中的平台例子都应先落到“供应商 + 业务面”这一层。详见
[04-source-backed-platform-example-notes.md](./04-source-backed-platform-example-notes.md)。

### 4.2.1 分平台的必要性到底有多大

这个问题的答案不是简单的“要”或“不要”。

更准确的说法是：

- 在边界层，分平台供应商和业务面非常必要
- 在核心履约层，不应该让平台差异无限渗透

#### A. 在单个波次内部，共性很强

一旦进入履约核心层，会员权益型需求和零售订单型需求通常都会收敛成：

- 给谁发
- 发什么
- 发多少
- 地址是什么
- 提交给哪个工厂
- 是否已发货
- 是否已完成来源渠道同步或人工闭环确认

这说明：

- `FulfillmentLine` 不应按平台供应商拆成多套模型
- `Wave` 也不应因为来源不同而裂变成完全不同的容器

#### B. 在波次前后两端，差异很大

真正的差异主要发生在：

- 上游需求导入
- 外部编号体系
- 身份识别方式
- 是否需要规则推导
- 物流回填或人工闭环方式
- 平台能力与失败处理

因此平台差异最应该被封装在：

- 需求导入层
- `IntegrationProfile` / 连接器层
- 回填与闭环层

而不是直接压进核心履约真相表里。

#### C. 真正该分的不是“会员平台 / 零售平台”二元类别

更应该区分的是：

1. `source_channel`
   - 这次需求来自哪个平台供应商
2. `source_surface`
   - 这次需求来自该平台供应商的哪个业务面
3. `demand_kind`
   - 这个业务面在本系统里产生什么语义的需求
4. `identity_strategy`
   - 该业务面用什么方式识别履约对象
5. `capabilities`
   - 该业务面支持哪些导入导出、回填与闭环能力

也就是说，分平台的必要性很大，但应该分在正确维度上。

### 4.2.2 平台差异应该落在哪一层

V2 推荐这样处理平台差异：

1. 核心履约层尽量平台无关

- `Wave`
- `WaveParticipantSnapshot`
- `FulfillmentLine`
- `SupplierOrder`
- `Shipment`

这些对象应优先表达履约事实，而不是优先表达平台差异。

2. 平台差异优先落在配置与边界层

- `DemandDocument/DemandLine`
- `IntegrationProfile`
- `DocumentTemplate`
- `ChannelSyncJob`

这些层更适合承接：

- 编号规则
- 字段映射
- 业务面能力
- 回填协议
- 闭环策略

3. Service 层负责把“平台差异”翻译成“统一履约动作”

也就是说：

- 平台差异需要被保留
- 但不应直接污染核心履约模型

### 4.2.3 `IntegrationProfile` 的定位

本次讨论里提到的 `source_profile`，在 V2 中更准确地说就是：

- 某个来源渠道供应商的某个业务面的统一配置入口

示例 `profile_key` 可以写成：

- `patreon.membership`
- `patreon.shop_purchase`
- `gumroad.membership`
- `gumroad.one_time_order`
- `fanbox.support_plan`
- `fanbox.supporter_only_purchase`
- `bilibili.live_support`
- `bilibili.creator_commerce`

这里的 key 是系统内部语言，不要求与平台官方命名完全一致，但必须与该平台官方业务面语义相匹配。

`IntegrationProfile` 回答的问题不是：

- “这个 CSV 长什么样”

而是：

- “这个来源到底是哪一个业务面”
- “这个业务面在本系统里应归类为哪种需求语义”
- “这个业务面的义务是由资格成立，还是由订单成立”
- “它怎么识别履约对象”
- “它的会员权益判定权威来自哪里”
- “它的领取/选项/表单输入通常如何补齐”
- “它是否支持物流回填”
- “它是自动闭环、文档闭环，还是人工确认闭环”
- “它应绑定哪些文档模板与连接器能力”

因此它比模板更高一层。

---
