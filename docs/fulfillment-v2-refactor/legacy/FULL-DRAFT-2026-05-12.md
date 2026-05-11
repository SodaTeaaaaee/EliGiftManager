# Fulfillment V2 Refactor Plan

## 0. 文档目的

本文档用于定义 EliGiftManager 从“会员回馈发货工具”演进为“创作者履约系统”的
完整重构计划。

当前系统已经能覆盖：

- 会员名单导入
- 波次内商品分配
- 地址绑定
- 工厂订单 CSV 导出

但新的业务要求已经把系统边界向前和向后同时拉长：

- 向前：
  - 除传统会员权益来源外，还要接入创作者零售订单来源
  - 同一平台未来可能同时提供会员和零售两个业务面
  - 需要处理“有订单号但没有会员身份”的买家
- 向后：
  - 工厂生产后会回传发货状态和快递单号
  - 系统需要把物流信息映射、转换、再同步回来源渠道
  - 该回填不只服务零售订单来源，也可能服务会员权益来源或其他上游渠道

因此，本次重构的重点不再是某一个页面或某一个导入模板，而是：

1. 建立统一、稳定、可扩展的业务语言
2. 建立可承接会员权益需求和零售订单需求的核心数据结构
3. 把“导出工厂订单即结束”的短生命周期，扩展为“需求导入 -> 分配 -> 下工厂 -> 回传物流 -> 回填来源渠道”的长生命周期
4. 让模板系统后续能够围绕通用结构扩展，而不是继续堆叠平台特例

---

## 1. 备份分支

在开始本计划前，已基于当前 `main` HEAD 创建备份分支：

`backup/pre-fulfillment-v2-refactor-2026-05-12`

用途：

- 作为“V2 重构前最后一个稳定基线”
- 当 `main` 后续开始实施重构时，任何人都可以随时切回此分支查看旧实现
- 如果中途需要回退整体设计方向，可以从该分支重新拉出实验分支

说明：

- 当前工作区在创建分支时是干净的，因此该分支完整代表了重构前的代码状态
- 后续建议继续在 `main` 上推进重构，不再把“重构前状态”留在 `main`

---

## 2. 当前系统的真实现状

### 2.1 当前核心模型

当前权威模型定义在：

- `docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md`
- `internal/model/tables.go`

当前业务核心实体是：

- `Member`
- `MemberNickname`
- `MemberAddress`
- `Wave`
- `WaveMember`
- `ProductMaster`
- `Product`
- `ProductTag`
- `DispatchRecord`
- `TemplateConfig`

当前分层是健康的，尤其是这两点：

1. `ProductMaster + Product`
   - 已经实现“全局主档 + 波次快照”分层
2. `Member + WaveMember`
   - 已经实现“全局实体 + 波次身份快照”分层

这两点不应该在重构中被推翻，而应该被泛化和延伸。

### 2.2 当前工作流

当前主链路基本是：

`导入会员 -> 生成 WaveMember -> 导入商品 -> 配置 ProductTag -> ReconcileWave -> 绑定地址 -> 导出工厂订单 CSV`

其中：

- `ProductTag` 定义分配规则
- `ReconcileWave` 负责把波次成员和分配规则收敛成 `DispatchRecord`
- `DispatchRecord` 是“当前版本的最终发货真相”

### 2.2.1 当前 tag 系统的真实定位

当前 `WaveTagStep + ProductTag + ReconcileWave` 这套机制，本质上不是通用订单分发器，而是：

- 以商品为中心的规则编辑器
- 面向会员权益语义的分配系统
- 带有用户级例外覆盖能力的自动重算器

它当前主要由两层组成：

1. 规则层

- `identity tag`
  - 负责表达：
    - `gift_level`
    - `platform_all`
    - `wave_all`
- 它们定义“哪些身份默认拿哪些商品”

2. 覆盖层

- `user tag`
  - 负责表达：
    - 对某个具体波次参与者的加送
    - 减送
    - 例外修正

再通过 `ReconcileWave` 把两层规则收敛成最终的 `DispatchRecord`。

这一点非常重要，因为它解释了为什么当前系统在会员权益场景里很好用：

- 批量效率高
- 规则可读性强
- 例外覆盖清晰
- 可以通过协商、补偿、赠送等方式自由调整

### 2.2.2 当前预览页其实已经承担了“调整层”职责

虽然当前系统表面上是“先配 tag，再导出”，但实际上预览页已经在承担第二层职责。

当前预览页中的以下操作：

- 改数量
- 添加礼物
- 移除礼物

本质上并不是直接编辑最终履约真相，而是：

- 回写成用户级覆盖
- 再触发一次 `ReconcileWave`

这意味着当前系统实际上已经隐含存在：

- 规则生成层
- 人工调整层
- 最终履约真相层

只是这三层目前还没有被显式命名和拆开。

### 2.3 当前状态模型

当前状态模型非常短：

- `DispatchRecord.Status`
  - `pending`
  - `pending_address`
  - `exported`
- `Wave.Status`
  - 通过 `DispatchRecord` 聚合推导

这套模型在“导出即结束”的世界里还能工作，但它天然无法表达：

- 已提交给工厂但未发货
- 部分发货
- 已回传快递单号但未回填来源渠道
- 回填失败需重试
- 买家订单与会员权益在同一波次并存

### 2.4 当前模型的关键缺口

当前最主要的结构性缺口有 6 个：

1. 缺少“上游需求单”抽象

- 会员回馈没有订单号
- 零售买家有订单号
- 当前系统没有一个统一结构去承接这两类来源

2. 缺少“供应商/工厂执行单”抽象

- 当前只有导出 CSV 的动作
- 没有持久化“发给哪个工厂、哪次导出、对应哪些行、外部订单号是什么”

3. 缺少“发货/包裹/物流”抽象

- 当前没有独立的 shipment / tracking 层
- 这会导致快递单号无处稳定落库

4. 缺少“来源渠道回填任务”抽象

- “已经拿到物流单号”不等于“已经成功同步到工坊/Gumroad/Patreon/FANBOX”

5. `platform` 语义过载

当前至少已经有两类平台语义：

- 会员来源平台
- 商品/工厂平台

新需求再加入：

- 来源渠道平台
- 来源渠道的业务面
- 物流承运商平台或承运商编码体系

更关键的是：

- “会员权益型需求”和“零售订单型需求”是需求类型
- 不是平台类别

例如：

- 哔哩哔哩既可能作为会员来源平台出现
- 也可能作为零售订单来源平台出现
- 甚至未来同一个平台名还会在不同业务面下产生不同格式和不同回填能力

如果继续统一叫 `platform`，或者把平台刚性二分成“会员平台 / 零售平台”，模板系统和领域服务会越来越混乱。

6. `DispatchRecord` 承担过多职责

它现在同时承担：

- 分配结果
- 地址准备状态
- 工厂导出状态

后续如果继续把工厂回传和来源渠道回填也叠进去，会让该表变成单表黑洞。

---

## 3. V2 的业务边界

### 3.1 本次重构要支持的业务

V2 必须支持以下两类需求同时存在：

1. 会员权益型需求

- 直播平台会员
- 创作者赞助会员
- 特征：
  - 有“身份”
  - 可能没有外部订单号
  - 商品需求可能需要系统内规则推导

2. 零售订单型需求

- 哔哩哔哩工坊
- Gumroad
- itch.io
- Patreon 实体周边订单
- FANBOX 实体回馈订单
- 其他创作者零售业务面
- 特征：
  - 有“订单”
  - 可能没有会员身份
  - 商品需求通常在外部订单里已明确给出

### 3.1.0 这不是“平台二分法”

这里区分的“会员权益型需求”和“零售订单型需求”，首先是：

- 上游需求类型
- 履约语义差异

而不是：

- 把所有平台刚性分成“会员平台”和“零售平台”两类

原因是同一个平台完全可能同时拥有多个业务面。

典型例子：

- 哔哩哔哩
  - 既可能输出会员权益名单
  - 也可能输出工坊零售订单
- 抖音
  - 既可能存在会员/粉丝权益来源
  - 也可能存在商城或零售来源

因此 V2 应区分的是：

1. `需求类型`

- `membership_entitlement`
- `retail_order`
- `manual_adjustment`

2. `来源渠道`

- 例如 `bilibili`
- `douyin`
- `patreon`
- `gumroad`

3. `来源渠道业务面`

- 例如 `membership`
- `retail`
- `workshop`
- `shop`

4. `来源渠道能力`

- 是否支持物流回填
- 是否支持部分发货
- 是否只支持 CSV
- 是否支持 API 推送

这里要特别强调：

- “物流回填”不是零售订单的专属能力
- 会员权益来源如果平台或业务面支持物流状态回写，也应复用同一套来源渠道回填模型

也就是说：

- “平台是谁”是一回事
- “这次需求属于该平台的哪个业务面”是第二回事
- “这个业务面具有什么导入导出能力”是第三回事

V2 的模型必须允许：

- 同一个 `source_channel` 下面存在多个 `source_surface`
- 且不同 `source_surface` 对应不同模板、不同回填能力、不同业务规则

### 3.1.1 新需求的业务拆解

为了避免后续讨论时把多个问题混成一句“支持更多平台”，这里把新增需求拆成 6 个具体业务点。

1. 新增了第二类消费者

旧模型里，消费者几乎等同于：

- 有平台身份
- 没有外部订单号
- 需要根据会员等级或波次规则推导该发什么

新模型里，又新增了一类消费者：

- 有外部订单
- 未必有会员身份
- 购买内容通常在外部订单里已经明确

也就是说，系统以后要同时处理：

- “有身份但没有订单的会员”
- “有订单但没有会员身份的买家”

2. 新增了第二类上游来源

旧模型的上游来源主要是：

- 会员权益来源导出的名单

新模型的上游来源还包括：

- 创作者零售业务面导出的订单
- 手工补录的零售需求
- 未来可能接入的 API 订单来源

这意味着“导入”不能再默认理解为“导入会员名单”，而必须泛化成：

- 导入权益需求
- 导入零售订单需求
- 导入手工补单需求

3. 工厂导出不再是终点，而是中间节点

旧模型里：

- 只要把工厂 CSV 导出去，流程基本就结束

新模型里：

- 工厂提交只是执行开始
- 还要等待工厂生产和发货
- 还要接收工厂回传的发货结果

因此“导出成功”不再等于“履约完成”。

4. 新增了物流回传和映射问题

工厂回传的数据通常会包含：

- 工厂订单号
- 工厂订单行号
- 发货状态
- 承运商
- 快递单号
- 可能还有部分发货、拆包、取消信息

这些数据不能直接原样塞回上游来源渠道，因为不同渠道或不同业务面通常要求：

- 不同的承运商编码
- 不同的订单号字段
- 不同的订单行匹配方式
- 不同的回填格式或接口

因此 V2 需要显式建模：

- 工厂执行单
- 物流发货实体
- 来源渠道回填任务

5. 新增了“回填结果追踪”问题

以前系统只要“导出文件成功”就算结束。

以后还需要区分：

- 工厂是否已发货
- 物流是否已拿到
- 是否已成功回填上游来源渠道
- 哪些记录回填失败、为何失败、能否重试

所以回填动作必须从“导出器的附带逻辑”升级成可持久化追踪的业务对象。

6. 同一个波次未来可能同时承接多类来源

V2 不应假设：

- 一个波次里只有会员
- 一个波次里只有零售订单
- 一个波次只对应一个平台

同一个波次未来可能同时包含：

- 会员赠礼
- 商城订单
- 人工补发
- 多个工厂平台的商品
- 同一来源渠道下不同业务面的数据

这要求波次内部的每一行数据都必须可追踪来源，而不是只靠波次级别的概念推断。

### 3.1.2 新需求对当前模型的直接冲击

新的业务要求不是“多加一个模板类型”就能承接，它直接冲击了当前模型的几个核心假设。

1. 旧假设：履约对象基本等同于会员

现在不成立，因为买家可能没有会员身份。

2. 旧假设：需求来源只有“名单”

现在不成立，因为零售来源是“订单”，而不是“名单”。

3. 旧假设：`DispatchRecord` 的终点是导出工厂 CSV

现在不成立，因为工厂发货与来源渠道回填发生在导出之后。

4. 旧假设：一个 `status` 字段足以表达阶段

现在不成立，因为后续至少要表达：

- 地址准备
- 工厂提交
- 工厂发货
- 来源渠道回填

5. 旧假设：波次列表里显示“已导出”就能代表结束

现在不成立，因为“已导出”只是流程中段，而不是业务闭环。

### 3.1.3 新需求的典型业务场景

为了统一后续判断标准，下面给出 4 个 V2 必须能解释清楚的典型场景。

#### 场景 A：纯会员回馈

输入：

- 一份会员名单
- 一套波次商品
- 一组会员等级分配规则

过程：

- 系统推导出履约行
- 绑定地址
- 导出工厂单
- 工厂发货回传物流
- 系统做物流存档

特点：

- 上游没有外部订单号
- 但下游仍然可能有工厂订单号和快递单号
- 如果会员来源渠道支持物流状态回写，也可能在发货后触发 `ChannelSyncJob`

#### 场景 B：纯零售订单

输入：

- 一份来源渠道订单表

过程：

- 系统解析外部订单和订单行
- 生成履约行
- 导出工厂单
- 工厂回传发货
- 系统将物流单号回填到来源渠道

特点：

- 上游有外部订单号
- 履约需求通常不需要会员规则推导
- 回填来源渠道通常是闭环中的必要步骤

#### 场景 C：同平台双业务面

输入：

- 哔哩哔哩会员权益名单
- 哔哩哔哩工坊订单

过程：

- 系统把它们识别为同一个 `source_channel`
- 但区分不同 `source_surface`
- 会员权益走规则推导
- 工坊订单走订单行直入
- 两者都可以进入同一波次履约
- 后续是否回填、如何回填，由各自业务面能力决定

特点：

- 平台名相同不代表业务语义相同
- 业务面不同可能导致模板、字段、回填方式都不同
- V2 模型不能只靠 `platform='bilibili'` 就做全部推断

#### 场景 D：混合波次

输入：

- 一批会员赠礼需求
- 一批商城零售订单
- 若干人工补发项

过程：

- 系统把它们收敛到同一个波次
- 共用商品和工厂导出流程
- 但在履约行级别保留来源、工厂执行、物流、回填信息

特点：

- 波次是统一容器
- 履约行必须保留差异
- 不能再用“波次类型推断每条记录语义”

### 3.1.4 分配语义的核心判断

V2 必须接受一个现实：

- 会员权益型需求和零售订单型需求，可以在最终履约层收敛
- 但不应该被强迫使用同一种“初始分配生成方式”

推荐明确区分两种基础语义：

1. `policy-driven`

特征：

- 上游给出的通常不是最终 SKU 履约行，而是身份、权益或资格
- 系统需要通过规则把“谁符合条件”推导成“该发什么”
- 允许在推导结果之上做协商式、补偿式、赠送式修正

适用：

- 会员权益型需求
- 活动回馈型需求

2. `demand-driven`

特征：

- 上游给出的通常已经是明确需求行
- 系统优先尊重原始订单或原始需求行
- 后续可以调整，但调整应被视为对原始需求的显式修正，而不是重新推导原始需求

适用：

- 零售订单型需求
- 手工补单中带有明确商品行的需求

### 3.1.5 当前 tag 系统在 V2 中的定位

V2 不应否定当前 tag 系统的价值。

相反，应明确承认：

- 当前 tag 系统是一个优秀的 `policy-driven` 分配系统
- 它在会员权益场景下已经证明了高效、清晰、自由、易用

因此 V2 的原则不是“用一种全新的统一系统替换 tag”，而是：

1. 保留它在会员权益场景中的主导地位
2. 不强迫零售订单型需求先被翻译成 tag 再进入履约
3. 允许两种初始分配方式最终收敛到同一套 `FulfillmentLine`

### 3.1.6 会员体验不得退化

这是 V2 的强约束，而不是软建议。

当前 `policy-driven` 会员分配逻辑在以下方面已经具备明显优势：

- 商品级批量配置效率高
- 身份规则表达直接
- 例外覆盖灵活
- 创作者视角下非常易于理解

因此后续重构必须满足：

1. 会员权益型波次仍然可以使用规则驱动分配
2. 会员权益型波次的批量加赠、减赠、覆盖能力不得明显退化
3. 不能为了统一零售语义而破坏当前会员分配体验

换句话说：

- V2 应当扩展当前系统
- 而不是让会员分配 UX 变得比现在更笨重

### 3.2 本次重构要支持的完整生命周期

V2 的完整生命周期定义为：

`上游需求导入 -> 波次汇入 -> 分配与整理 -> 地址确认 -> 提交工厂 -> 工厂回传发货 -> 物流映射 -> 回填来源渠道 -> 波次关闭`

其中最关键的新增阶段是：

- 工厂回传发货
- 物流映射
- 来源渠道回填

### 3.3 本次重构不打算解决的问题

V2 不是要把 EliGiftManager 做成完整电商平台。

本次明确不做：

- 购物车
- 在线支付
- 税费计算
- 站内商品展示
- 库存精细承诺和仓配算法
- 复杂财务对账
- ERP / WMS 全套中台能力

V2 的目标是：

- 成为创作者周边和会员回馈场景下的“履约中台”
- 在必要处参考专业电商/OMS 的成熟抽象
- 不把系统拉成一个难以维护的伪全能商城

---

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
- 工坊 buyer id

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

#### I. Shipment / ShipmentParcel / ShipmentLine

表示工厂回传后的发货实体。

它回答的问题是：

- “工厂是否已经发货？”
- “包裹号是什么？”
- “承运商是什么？”
- “这次发货覆盖了哪些履约行？”

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

- 同一个人既可能是 Bilibili 会员
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

- 迁移期允许旧名和新名并存
- 但任何新功能设计都应优先按新名思考

### 4.1.3 业务语言的判断规则

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

### 4.1.4 两个完整例子

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

- 用户 B 在 Gumroad 下单购买挂件 1 个
- 工厂完成生产后回传了物流单号
- 系统需要把物流回填到 Gumroad 或其对应运营流程

V2 语言下可解释为：

- `CustomerProfile`
  - 用户 B 这个全局履约对象
- `CustomerIdentity`
  - buyer email / external buyer id
- `DemandDocument`
  - Gumroad 订单 #12345
- `DemandLine`
  - 挂件 x1
- `WaveParticipantSnapshot`
  - 用户 B 在本次波次中的快照
- `FulfillmentLine`
  - 最终需要发出的挂件 x1
- `SupplierOrder`
  - 提交给工厂的执行单
- `Shipment`
  - 工厂回传的物流单号
- `ChannelSyncJob`
  - 将物流同步回来源渠道的任务

### 4.2 统一平台词汇

V2 必须把“平台”拆开说，禁止继续在关键领域对象里只写一个含糊的 `platform`
而不说明语义。

推荐至少区分：

- `identity_platform`
  - 用户身份来源平台
- `source_channel`
  - 上游来源渠道平台或平台供应商
- `source_surface`
  - 该来源渠道下的具体业务面
- `supplier_platform`
  - 工厂/供应商平台
- `carrier_code`
  - 物流承运商编码

说明：

- 同一个平台名可能同时出现在多个角色里
- 同一个 `source_channel` 下也可能存在多个 `source_surface`

例如：

- `source_channel = bilibili`
- `source_surface = membership`

和

- `source_channel = bilibili`
- `source_surface = workshop`

虽然平台供应商相同，但业务语义、模板结构和回填能力都可能不同。

如果确实保留 `platform` 字段，也必须在对象语义上说明它是哪一类平台、哪一个业务面。

### 4.2.1 分平台的必要性到底有多大

这个问题的答案不是简单的“要”或“不要”。

更准确的说法是：

- 在边界层，分平台或分业务面非常必要
- 在核心履约层，不应该让平台差异无限渗透

原因如下。

#### A. 在单个波次内部，共性很强

一旦进入履约核心层，会员权益型需求和零售订单型需求通常都会收敛成：

- 给谁发
- 发什么
- 发多少
- 地址是什么
- 提交给哪个工厂
- 是否已发货
- 是否已回填来源渠道

这说明：

- `FulfillmentLine` 不应按平台拆成两套模型
- `Wave` 也不应因为来源不同而裂变成两套完全不同的容器

#### B. 在波次前后两端，差异很大

真正的差异主要发生在：

- 上游需求导入
- 外部编号体系
- 身份识别方式
- 是否需要规则推导
- 物流回填协议
- 平台能力与失败处理

因此平台差异最应该被封装在：

- 需求导入层
- Profile / 连接器层
- 回填层

而不是直接压进核心履约真相表里。

#### C. 真正该分的不是“会员平台 / 零售平台”二元类别

更应该区分的是：

1. `demand_kind`
   - 这次需求是什么语义
2. `source_channel`
   - 这次需求来自哪个平台供应商
3. `source_surface`
   - 这次需求来自该平台的哪个业务面
4. `capabilities`
   - 这个业务面支持哪些导入导出与回填能力

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

3. Service 层负责把“平台差异”翻译成“统一履约动作”

也就是说：

- 平台差异需要被保留
- 但不应直接污染核心履约模型

### 4.2.3 `source_profile` / `IntegrationProfile` 的定位

本次讨论里提到的 `source_profile`，可以理解为：

- 某个来源渠道的某个业务面的统一配置入口

例如：

- `bilibili.membership`
- `bilibili.workshop`
- `douyin.membership`
- `douyin.shop`
- `gumroad.order`

它回答的问题不是：

- “这个 CSV 长什么样”

而是：

- “这个来源是什么业务面”
- “它产生哪种需求类型”
- “它怎么识别用户”
- “它支不支持物流回填”
- “它应绑定哪些文档模板”

因此它比模板更高一层。

---

## 5. V2 目标数据结构

本节描述目标数据结构，不要求一步到位落地成最终表名，但要求所有后续改动都向该结构对齐。

### 5.1 全局层

#### CustomerProfile

建议字段：

- `id`
- `display_name`
- `profile_type`
  - `member`
  - `buyer`
  - `mixed`
  - `manual`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 当前 `Member` 可作为其过渡实现
- 不再把“会员”视为唯一客体

#### CustomerIdentity

建议字段：

- `id`
- `customer_profile_id`
- `identity_platform`
- `identity_value`
- `identity_type`
  - `platform_uid`
  - `email`
  - `username`
  - `external_buyer_id`
- `is_primary`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 这是对当前 `(platform, platform_uid)` 单一身份结构的泛化
- 一个客户可能同时有多个身份来源

#### CustomerAddress

可在当前 `MemberAddress` 基础上扩展，建议继续保留：

- 历史
- 默认地址
- 测试地址
- 软删除
- 标签化备注

未来可新增：

- `normalized_region`
- `postal_code`
- `country_code`
- `validation_status`

### 5.2 上游需求层

#### DemandDocument

建议字段：

- `id`
- `kind`
  - `membership_entitlement`
  - `retail_order`
  - `manual_adjustment`
- `source_channel`
- `source_surface`
- `source_document_no`
- `source_customer_ref`
- `customer_profile_id`
- `source_created_at`
- `source_paid_at`
- `currency`
- `raw_payload`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 会员导入没有订单号时，`source_document_no` 可为空
- 零售渠道订单导入时，`source_document_no` 通常应有值
- 同一个 `source_channel` 下应允许通过 `source_surface` 区分会员业务面、商城业务面等不同来源语义

#### DemandLine

建议字段：

- `id`
- `demand_document_id`
- `source_line_no`
- `line_type`
  - `entitlement_rule`
  - `sku_order`
  - `manual_adjustment`
- `product_master_id`
- `external_title`
- `requested_quantity`
- `entitlement_code`
- `gift_level_snapshot`
- `raw_payload`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- 会员场景里，这一层用于记录“本次应得权益”
- 零售场景里，这一层用于记录“用户下单了什么”

### 5.3 波次层

#### Wave

建议保留现有表，并新增/演进：

- `wave_type`
  - `membership`
  - `retail`
  - `mixed`
- `allocation_mode`
  - `rule_based`
  - `direct_from_demand`
  - `hybrid`
- `lifecycle_stage`
- `progress_snapshot`
- `notes`

`status` 可在迁移期保留，但长期应降级为兼容字段或投影视图字段。

#### WaveParticipantSnapshot

建议基于当前 `WaveMember` 扩展或替换。

建议字段：

- `id`
- `wave_id`
- `customer_profile_id`
- `snapshot_type`
  - `member`
  - `buyer`
  - `mixed`
- `identity_platform`
- `identity_value`
- `display_name`
- `gift_level`
- `source_channel`
- `source_surface`
- `source_document_refs`
- `extra_data`
- `created_at`

说明：

- 当前 `WaveMember` 不应该继续被理解为“只有会员”
- 它应该变成波次内参与方快照

### 5.3.1 分配与调整语义层

V2 的目标不是让所有需求都经过同一种分配引擎，而是让不同来源在不同层级被正确处理后再收敛。

建议明确以下三层：

1. `Base Allocation Source`

表示“初始履约结果从哪里来”。

可能来源：

- 规则推导
- 上游订单行直入
- 手工补单直入

2. `Adjustment Layer`

表示“在初始履约结果之上的修正”。

可能动作：

- 加送
- 减送
- 替换
- 补发
- 取消

3. `Final Fulfillment Result`

表示最终需要执行的履约真相。

这一层最终统一落到 `FulfillmentLine`。

### 5.3.2 AllocationPolicyRule

建议将当前 `ProductTag` 明确定位为：

- `policy-driven` 分配模式下的规则对象

当前 `ProductTag` 不应被要求承担：

- 零售订单原始行真相
- 外部订单义务真相
- 所有来源需求的统一表达

而应主要承担：

- 身份规则
- 波次级批量赠送规则
- 规则驱动分配中的用户例外覆盖

过渡期可继续使用 `ProductTag` 承接此职责。

长期可以考虑把语义进一步重命名为更明确的规则对象，但当前阶段不要求立即改表名。

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

### 5.9 配置与集成层

这一层是 V2 相比当前系统新增的重要抽象，用来承接“来源渠道 / 业务面 / 能力 / 模板绑定”的配置。

#### IntegrationProfile

建议新增。

它是当前模板系统的上位概念，不是模板本身。

建议字段：

- `id`
- `profile_key`
  - 例如 `bilibili.membership`
  - 例如 `bilibili.workshop`
  - 例如 `gumroad.order`
- `source_channel`
- `source_surface`
- `demand_kind`
  - `membership_entitlement`
  - `retail_order`
  - `manual_adjustment`
- `default_allocation_mode`
  - `rule_based`
  - `direct_from_demand`
  - `hybrid`
- `identity_strategy`
  - `platform_uid`
  - `email`
  - `external_buyer_id`
- `reference_strategy`
  - `member_level`
  - `order_level`
  - `order_line_level`
- `supports_tracking_push`
- `supports_partial_shipment`
- `supports_api_import`
- `supports_api_export`
- `capabilities`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- `IntegrationProfile` 回答的是“这个来源业务面整体怎么工作”
- 它不是某一个导入 CSV 的字段映射
- 同一个 `source_channel` 可以挂多个 `IntegrationProfile`

#### DocumentTemplate

建议作为 `TemplateConfig` 的未来语义方向保留。

建议字段：

- `id`
- `template_key`
- `document_type`
  - `import_entitlement`
  - `import_sales_order`
  - `import_product_catalog`
  - `export_supplier_order`
  - `import_supplier_shipment`
  - `export_source_tracking_update`
- `format`
  - `csv`
  - `xlsx`
  - `json`
  - `api_payload`
- `mapping_rules`
- `extra_data`
- `created_at`
- `updated_at`

说明：

- `DocumentTemplate` 只负责文档字段结构
- 它不负责决定需求类型、业务面和平台能力

#### IntegrationProfileTemplateBinding

建议新增。

建议字段：

- `id`
- `integration_profile_id`
- `document_type`
- `template_id`
- `is_default`
- `created_at`

说明：

- 一个 `IntegrationProfile` 应允许绑定多个不同用途的模板
- 例如同一个 profile 同时绑定：
  - 需求导入模板
  - 工厂导出模板
  - 工厂发货回传模板
  - 来源渠道物流回填模板

---

## 6. 当前结构到目标结构的映射

### 6.1 当前实体与目标实体映射

| 当前实体 | 目标语义 | 处理策略 |
| --- | --- | --- |
| `Member` | `CustomerProfile` 的过渡实现 | 短期保留表名，先扩展语义 |
| `MemberNickname` | `CustomerProfile` 的昵称历史 | 保留，后续可并入 profile/identity 辅助表 |
| `MemberAddress` | `CustomerAddress` | 保留并增强 |
| `Wave` | `Wave` | 保留并扩充生命周期语义 |
| `WaveMember` | `WaveParticipantSnapshot` | 泛化，不再只承载会员 |
| `ProductMaster` | `ProductMaster` | 直接保留 |
| `Product` | `Wave Product Snapshot` | 直接保留 |
| `ProductTag` | 分配规则层 | 会员权益波次继续使用；零售订单波次可弱化 |
| `DispatchRecord` | `FulfillmentLine` | 演进，不再继续承担全部外部状态 |
| `TemplateConfig` | 模板配置层 | 保留，但需升级模板类型和能力模型 |

### 6.2 关键保留原则

以下设计原则必须保持：

1. 历史波次必须是快照，不受全局实体后续变动污染
2. 全局商品主档和波次商品快照必须继续分层
3. 履约真相必须有单一归宿
4. 工厂执行和物流回填不能只靠导入导出瞬时脚本

---

## 7. V2 工作流定义

### 7.0 分配工作流的统一结构

无论来源是会员权益还是零售订单，V2 都建议把“生成最终履约结果”的过程拆成三层。

1. 初始分配层

回答：

- “第一版履约结果从哪里来？”

会员权益型需求：

- 主要来自规则推导

零售订单型需求：

- 主要来自上游需求行直入

2. 调整层

回答：

- “在第一版履约结果之上，还做了哪些修正？”

这层负责承接：

- 人工协商
- 赠送
- 补发
- 减送
- 替换

3. 最终履约层

回答：

- “最终到底发什么？”

这层统一收敛到 `FulfillmentLine`。

### 7.0.1 会员与零售不应该共用同一种初始分配方式

V2 的目标是“最终收敛”，不是“前置步骤强统一”。

因此：

- 会员权益型需求主要走 `policy-driven`
- 零售订单型需求主要走 `demand-driven`

两者都可以进入同一波次，也都可以进入同一套最终履约真相，但不应被强迫使用同一种初始分配引擎。

### 7.1 会员权益型波次

目标链路：

1. 导入会员名单
2. 生成 `CustomerProfile/Identity`
3. 生成 `WaveParticipantSnapshot`
4. 导入商品
5. 在规则编辑器中配置 `AllocationPolicyRule`
6. 通过 `ReconcileWave` 生成第一版 `FulfillmentLine`
7. 通过调整层做必要的例外修正
8. 绑定地址
9. 生成并导出 `SupplierOrder`
10. 导入工厂发货回传，生成 `Shipment`
11. 将 `Shipment` 转换为来源渠道回填任务
12. 按来源渠道能力决定是否执行 `ChannelSyncJob`
13. 波次关闭

### 7.2 零售订单型波次

目标链路：

1. 导入零售订单
2. 生成 `DemandDocument/DemandLine`
3. 归并买家资料到 `CustomerProfile/Identity`
4. 生成 `WaveParticipantSnapshot`
5. 从 `DemandLine` 直接或半自动生成第一版 `FulfillmentLine`
6. 只在必要处通过调整层做显式修正
7. 地址校验
8. 生成并导出 `SupplierOrder`
9. 导入工厂发货回传，生成 `Shipment`
10. 转换并回填来源渠道
11. 波次关闭

### 7.3 混合波次

系统应允许同一个波次同时承接：

- 会员权益履约行
- 零售订单履约行

但必须在行级保持来源可追踪：

- 哪条履约行来自哪类上游需求
- 哪条履约行是否需要回填来源渠道
- 哪条履约行来自同一来源渠道的哪个业务面

### 7.4 `WaveTagStep` 的未来定位

当前 `WaveTagStep` 不应继续被理解为“所有波次统一分配入口”。

V2 更推荐把它逐步演进为：

- `WaveAllocationStep`

并在内部拆成三个能力区：

1. `Rule Allocation`

- 面向 `policy-driven` 需求
- 主要服务会员权益型波次
- 核心交互继续保留当前的商品级批量规则编辑体验

2. `Demand Mapping`

- 面向 `demand-driven` 需求
- 主要服务零售订单型波次
- 核心目标是清晰展示：
  - 原始需求行
  - 商品映射关系
  - 是否存在异常或缺失映射

3. `Manual Adjustments`

- 面向所有波次
- 统一承接：
  - 加送
  - 减送
  - 替换
  - 补发

### 7.5 会员分配 UX 保留原则

当前 tag 系统在会员权益场景下已经证明是高效方案。

因此 `WaveAllocationStep` 的设计必须满足：

1. 会员权益型波次默认仍以 `Rule Allocation` 为主入口
2. 现有商品中心批量配置体验不得明显退化
3. “先规则，再覆盖，再预览”的认知路径应尽量保留
4. 零售订单型需求的引入不应迫使会员用户先理解订单映射语义

---

## 8. 状态与进度模型重构

### 8.1 现有状态模型的问题

现有 `Wave.Status` 有以下问题：

1. 只有单维状态
2. 只反映“是否缺地址”和“是否已导出”
3. 无法表达工厂执行阶段
4. 无法表达物流回填阶段
5. 无法表达部分完成
6. 前端进度条是固定映射，不是业务真实进度

### 8.2 V2 行级状态模型

建议把 `FulfillmentLine` 的生命周期拆成多维状态。

#### allocation_state

- `draft`
- `ready`
- `changed_after_submit`
- `locked`

#### address_state

- `missing`
- `ready`
- `invalid`

#### supplier_state

- `not_submitted`
- `submitted`
- `accepted`
- `producing`
- `partially_shipped`
- `shipped`
- `canceled`

#### channel_sync_state

- `not_required`
- `pending`
- `synced`
- `failed`

### 8.3 V2 波次聚合状态

`Wave` 不再直接表达底层细节，而是表达聚合阶段：

- `draft`
- `allocating`
- `address_blocked`
- `ready_to_submit`
- `submitted_to_supplier`
- `partially_shipped`
- `shipped`
- `syncing_back`
- `closed`

### 8.4 V2 波次进度展示

前端应放弃“伪百分比硬编码”，改成可解释的漏斗指标：

- 总履约行数
- 地址就绪行数
- 已提交工厂行数
- 已回传快递行数
- 已同步来源渠道行数
- 失败回填行数

波次首页和 Dashboard 最终应显示：

- 阶段标签
- 分平台执行摘要
- 风险计数
- 回填失败计数

而不是只显示一个含糊的“已导出/待补全”。

---

## 9. Profile / 模板系统升级方向

### 9.0 为什么不能只继续扩模板系统

当前 `TemplateConfig` 更接近“文档字段映射配置”。

如果继续只在模板系统上做加法，会越来越难表达以下问题：

- 这个来源属于哪个业务面
- 它产生哪种需求类型
- 它用什么身份策略识别履约对象
- 它支不支持物流回填
- 它需要哪些不同文档模板协同工作

这说明当前系统真正缺少的，不只是“更多模板类型”，而是：

- 一个比模板更高层的 Profile 系统

### 9.0.1 对 `source_profile` 的判断

本次讨论里的 `source_profile`，本质上可以理解为：

- `IntegrationProfile`

它与当前模板系统的关系是：

- 可以复用模板系统的已有资产
- 但不应被简化成“模板系统换个名字”

更准确的关系是：

- `Profile System` 是能力配置层
- `Template System` 是 Profile 下面的文档映射子层

### 9.1 当前模板类型不够用

当前只有：

- `import_product`
- `import_dispatch_record`
- `export_order`

这不足以表达 V2 的完整链路。

### 9.2 V2 推荐模板类型

建议至少引入：

- `import_entitlement`
- `import_sales_order`
- `import_product_catalog`
- `export_supplier_order`
- `import_supplier_shipment`
- `export_source_tracking_update`

### 9.3 Profile、模板、Service 的三层分工

V2 推荐明确三层结构：

1. `IntegrationProfile`

- 定义来源业务面
- 定义需求类型
- 定义能力边界
- 选择应绑定的模板集合

2. `DocumentTemplate`

- 定义具体文档字段映射
- 定义列顺序
- 定义 CSV / Excel / JSON 结构

3. `Service / Handler`

- 执行真实业务逻辑
- 处理导入、导出、回填、重试、异常分支

这三层的关系应当是：

- Profile 决定“怎么理解这个来源”
- Template 决定“怎么读写这个来源的文档”
- Service 决定“实际怎么执行这套流程”

### 9.4 为什么 Profile 是模板系统的上层

同一个来源业务面往往不只需要一个模板。

例如：

- 一个 `bilibili.workshop` profile 可能需要：
  - 订单导入模板
  - 工厂导出模板
  - 工厂发货回传模板
  - 物流回填模板

这意味着：

- 模板是文档级对象
- Profile 是来源业务面级对象

因此更合理的结构是：

- 一个 Profile 绑定多个 Template
- 而不是让一个 Template 自己承担全部业务职责

### 9.5 模板与连接器分离

后续架构里必须明确：

- 模板：
  - 负责字段映射
  - 负责列顺序
  - 负责 CSV/Excel 结构解释
- 连接器：
  - 负责平台能力
  - 负责导入导出方式
  - 负责 API / CSV / 手工上传差异

### 9.6 平台能力模型

建议增加“平台能力”概念，而不是让模板自己承担全部职责。

典型能力：

- `can_import_csv`
- `can_export_csv`
- `can_import_api`
- `can_push_tracking`
- `supports_partial_shipment`
- `requires_carrier_mapping`
- `requires_external_order_no`

说明：

- 模板解决字段问题
- 能力模型解决流程问题

### 9.7 不应该放进 Profile 的东西

虽然 Profile 比模板更强，但它不应演变成万能低代码系统。

以下内容不建议完全配置化进 Profile：

- 复杂的波次分配算法
- 复杂的工厂回传合并逻辑
- 所有异常分支的完整 DSL
- 大量条件式脚本
- 需要深入调试的业务规则代码

Profile 更适合承载：

- 能力声明
- 策略枚举
- 模板绑定
- 身份与引用规则
- 导入导出入口配置

而真正复杂的流程逻辑仍应保留在 service 层。

---

## 10. 实施原则

### 10.1 加法优先，兼容优先

V2 初期尽量采用“新增表/新增字段/新增服务”的方式推进，而不是立刻重命名或直接替换旧表。

原因：

- 当前系统仍在承载业务
- 会员回馈链路必须持续可用
- 模板系统仍需兼容旧流程

### 10.2 表名与业务名可暂时不完全同步

短期可以允许：

- 代码注释、service、DTO 先采用新业务语言
- 数据库表名保留旧名

例如：

- `Member` 表短期可继续存在
- 但业务语义开始按 `CustomerProfile` 理解

### 10.3 先重构领域，再重构 UI

优先级顺序应是：

1. 领域语言
2. 数据结构
3. 状态投影
4. 导入导出服务
5. 页面 UI

不应反过来先修页面进度条，再回头推翻状态模型。

### 10.4 先保真，再自动化

优先保证：

- 能准确落库
- 能准确追踪来源
- 能准确追踪物流与回填结果

API 自动化、批量任务优化、异步 worker 可以在后续阶段再补。

---

## 11. 分阶段实施计划

### 阶段 0：冻结语义与基线

目标：

- 冻结当前系统语义
- 建立 V2 业务语言
- 建立本重构计划文档

产出：

- 本文档
- 重构前备份分支

### 阶段 1：统一命名与领域边界

目标：

- 在 docs、service 注释、DTO 命名层明确新语义

任务：

- 补充“CustomerProfile / Demand / Fulfillment / SupplierOrder / Shipment / ChannelSync”词汇表
- 标记旧术语与新术语映射关系
- 明确 `platform` 的多维语义
- 明确 `IntegrationProfile` 和 `DocumentTemplate` 的边界
- 为现有 `TemplateConfig` 预留向 `DocumentTemplate` 迁移的路径
- 明确 `policy-driven` / `demand-driven` / `hybrid` 三种分配语义
- 明确当前 tag 系统在会员权益场景中的保留价值与边界

验收：

- 文档层不再把会员、买家、订单、工厂单和模板能力混为一谈

### 阶段 2：引入新表结构

目标：

- 以新增为主，引入 V2 所需核心表

建议优先新增：

- `demand_documents`
- `demand_lines`
- `supplier_orders`
- `supplier_order_lines`
- `shipments`
- `shipment_lines`
- `channel_sync_jobs`
- `channel_sync_items`
- `integration_profiles`
- `integration_profile_template_bindings`

同时为旧表补充过渡字段：

- `waves.wave_type`
- `waves.allocation_mode`
- `waves.lifecycle_stage`
- `dispatch_records` 的多维状态字段

验收：

- 不影响现有导入导出链路
- 新表迁移可重复执行
- 新的 profile 层可以先只服务少量试点来源

### 阶段 3：重构履约真相层

目标：

- 让 `DispatchRecord` 向 `FulfillmentLine` 语义演进

任务：

- 拆出多维状态
- 增加来源引用
- 增加工厂执行引用
- 增加来源渠道回填引用
- 为显式调整层预留 `FulfillmentAdjustment` 入口

关键约束：

- `ReconcileWave` 仍可继续生成履约行
- 但履约行不再只服务“导出 CSV 即结束”的短链路

验收：

- 履约行可同时表示会员来源和零售来源
- 履约行可独立追踪后续工厂执行和回填过程

### 阶段 4：引入上游需求层

目标：

- 支持“会员权益需求”和“零售订单需求”的统一落库

任务：

- 新增会员权益导入到 `DemandDocument/DemandLine`
- 新增零售订单导入到 `DemandDocument/DemandLine`
- 建立 demand 到 wave participant / fulfillment line 的映射
- 建立“规则推导生成第一版履约行”和“需求行直入生成第一版履约行”的双路径

验收：

- 会员导入和零售导入都可进入统一履约链路

### 阶段 5：引入工厂执行层

目标：

- 把“导出工厂 CSV”从一次性动作升级为持久化的工厂执行记录

任务：

- 导出时创建 `SupplierOrder`
- 导出行映射到 `SupplierOrderLine`
- 记录模板、导出时间、外部工厂单号、提交方式

验收：

- 任何一次工厂导出都可追踪、可回查、可关联到履约行

### 阶段 6：引入发货与物流回传

目标：

- 支持工厂回传发货状态和快递单号

任务：

- 新增 `import_supplier_shipment`
- 支持按工厂模板映射 shipment 数据
- 建立 `Shipment` 与 `SupplierOrderLine/FulfillmentLine` 的关联

验收：

- 系统能稳定落库存储发货结果
- 同一波次支持部分发货和多包裹

### 阶段 7：引入来源渠道回填

目标：

- 把物流状态同步回来源渠道

任务：

- 新增 `ChannelSyncJob/Item`
- 建立物流承运商映射
- 建立外部订单号和外部订单行号映射
- 导出或调用 API 回填

验收：

- 可区分“待回填”“已回填”“回填失败”
- 可重试失败项

### 阶段 8：重做波次状态投影与 UI

目标：

- 让波次状态真正反映业务阶段

任务：

- 替换当前 `Wave.Status` 简化展示逻辑
- 重做 Dashboard 和波次列表状态卡片
- 增加分平台进度、异常计数、回填结果
- 将当前 `WaveTagStep` 逐步演进为 `WaveAllocationStep`
- 在 UI 上拆出：
  - `Rule Allocation`
  - `Demand Mapping`
  - `Manual Adjustments`

验收：

- 页面状态与真实流程一致
- 不再出现“已导出但其实还没发货/还没回填”的误导
- 会员权益型分配体验不因零售语义接入而明显退化

### 阶段 9：模板系统能力化

目标：

- 把模板系统从“字段映射集合”升级为“Profile + 字段模板 + 平台能力”

任务：

- 扩展模板类型
- 加入连接器能力描述
- 引入 `IntegrationProfile`
- 把 `TemplateConfig` 迁移为 `DocumentTemplate` 语义
- 拆出物流映射、订单号映射、承运商映射逻辑

验收：

- 新来源业务面接入不再需要临时硬编码整套流程

---

## 12. 数据迁移策略

### 12.1 总原则

迁移必须做到：

- 可重复
- 可回滚
- 不破坏现有有效数据
- 允许一段时间内新旧结构并存

### 12.2 建议策略

1. 先新增表，不删旧表
2. 先新增字段，不强制重命名旧字段
3. 先让新功能写新表，再考虑是否回写旧表
4. 旧页面继续读旧投影，新页面或新功能读新投影

### 12.3 兼容阶段建议

兼容阶段内建议：

- `DispatchRecord.Status` 继续保留
- 但逐步退化为兼容输出，不再作为唯一流程真相

最终目标：

- 新状态由多维字段和聚合投影驱动

---

## 13. 测试与验证计划

### 13.1 必测业务场景

1. 纯会员波次
2. 纯零售订单波次
3. 混合波次
4. 多工厂平台同波次
5. 工厂部分发货
6. 多包裹
7. 物流回填成功
8. 物流回填失败后重试
9. 会员无订单号场景
10. 买家无会员身份场景
11. `policy-driven` 规则推导场景
12. `demand-driven` 订单直入场景
13. `hybrid` 混合分配场景

### 13.2 回归重点

重点回归以下老能力：

- 地址绑定
- 波次重算
- 商品快照与主档关系
- 用户补发/减发覆盖
- 工厂导出模板
- 当前会员规则分配与批量编辑体验

### 13.3 验收标准

V2 至少应满足：

1. 能同时承接会员和买家两类来源
2. 能把履约来源追踪到具体需求单和需求行
3. 能持久化记录工厂提交结果
4. 能持久化记录物流回传
5. 能把物流回填状态作为独立流程追踪
6. 波次状态能真实反映完整生命周期
7. 模板系统能为新增平台提供明确扩展入口
8. 会员权益型分配体验不明显劣化于当前版本

---

## 14. 风险与注意事项

### 14.1 最大风险：在旧术语上继续硬补

如果继续把：

- `Member` 当成唯一客体
- `DispatchRecord` 当成唯一流程真相
- `exported` 当成流程终点

那么后续每接一个平台都会继续堆特例，最终很难维护。

### 14.2 第二大风险：过早重命名数据库表

如果一开始就强行把旧表重命名成新语义，迁移复杂度和风险都会显著提升。

建议：

- 先重构语义和读写路径
- 再决定是否物理重命名表

### 14.3 第三大风险：用单一进度条伪装复杂流程

V2 后，单一百分比会越来越不真实。

建议：

- 前端以阶段 + 计数 + 风险提示为主
- 百分比只作为辅助投影，不作为唯一信息来源

### 14.4 第四大风险：强迫所有需求都走 tag 规则系统

当前 tag 系统在会员权益场景下表现优秀，但如果强迫零售订单型需求也全部先转换成 tag，再生成履约行，会带来以下风险：

- 冲淡原始订单行语义
- 让零售真相来源变得难以审计
- 为零售场景引入不必要的规则复杂度
- 反过来污染会员场景的原有优秀体验

建议：

- 保留 tag 作为 `policy-driven` 分配主引擎
- 不把它当作所有需求类型的统一真相入口
- 让零售订单主要走 `demand-driven` 直入路径

### 14.5 第五大风险：为适配零售语义而让会员分配 UX 退化

V2 最大的产品风险之一，不是“支持不了零售”，而是“支持零售之后把会员分配做笨了”。

建议：

- 会员分配体验作为独立约束保留
- `WaveAllocationStep` 中默认保留规则驱动主入口
- 新增的 demand / adjustment 能力不应迫使会员用户改用更复杂的认知路径

---

## 15. 当前结论

本次重构的核心不是“加一个快递单号字段”，也不是“把波次状态多加几个枚举”。

真正要做的是：

- 把 EliGiftManager 从“短链路导出工具”
- 演进成“支持会员回馈与创作者零售的履约系统”

其主线应该是：

`Demand -> Wave -> FulfillmentLine -> SupplierOrder -> Shipment -> ChannelSync`

现有系统里最值得保留的资产是：

- `ProductMaster + Product` 双层商品结构
- `Wave` 作为业务批次边界
- `ReconcileWave` 的履约收敛思路

现有系统里最需要尽快停止扩张的部分是：

- 把所有流程状态塞进 `DispatchRecord.Status`
- 把所有平台语义都叫 `platform`
- 把“导出”继续当作波次终点

后续任何实施细节，只要与本计划冲突，都应优先以本计划中的业务边界、数据结构和迁移原则为准，再重新评估代码改动方向。
