# 基于官方资料的平台例子说明

本文件用于约束 V2 文档中使用平台例子的方式。

原则：

- 任何平台例子在进入方案文档前，都应先核对该平台官方资料
- 平台例子只能用于说明“平台供应商 + 业务面 + 能力”这三个维度
- 不允许再把某个平台名直接当成“会员平台”或“零售平台”的同义词

以下记录基于 2026-05-12 核对到的官方资料，用于支撑当前文档中的举例方式。

## Patreon

官方资料显示：

- Patreon 官方帮助中心明确存在 membership tiers
- Patreon 官方帮助中心明确存在 one-time purchases
- Patreon 官方帮助中心明确存在 merch for membership

这意味着：

- `patreon` 不能被建模为单一的“会员平台”
- 在 V2 里至少应允许出现 `patreon.membership`
- 也应允许出现 `patreon.shop_purchase`
- 物理履约需求还可能来自 membership entitlement，而不是传统零售订单
- 对连续订阅阶段性礼物，更合理的做法通常是把“是否 earned”视为上游平台权威结果，而不是由本系统本地重算

参考：

- https://support.patreon.com/hc/en-us/articles/218202363-How-to-edit-your-membership-tiers-a-guide-for-creators
- https://support.patreon.com/hc/en-us/articles/4413300654733-How-to-set-up-one-time-purchases-as-a-creator
- https://support.patreon.com/hc/en-us/articles/11111747095181-How-to-set-up-merch-for-membership
- https://support.patreon.com/hc/en-us/articles/360043651572-When-will-my-members-earn-merch

## Gumroad

官方资料显示：

- Gumroad 官方产品页明确强调 memberships and subscriptions
- Gumroad 官方帮助中心存在 shipped purchase 相关说明

这意味着：

- `gumroad` 不能被建模为单一的“零售订单平台”
- 在 V2 里至少应允许出现 `gumroad.membership`
- 也应允许出现 `gumroad.one_time_order`
- 对 Gumroad 的物流闭环能力应采取保守建模，不能默认它一定存在原生 tracking push

参考：

- https://gumroad.com/features
- https://gumroad.com/help/article/327-incorrect-recurring-charge
- https://gumroad.com/help/article/209-whats-the-status-of-my-purchase

## itch.io

官方资料显示：

- itch.io 官方文档明确把自己描述为 creator storefront / digital creator marketplace
- itch.io 官方支付文档提到可以通过 custom fields 收集 physical rewards 的邮寄地址

这意味着：

- `itchio` 可以产生命令式订单或奖励型需求
- 它不应被直接视作“完整原生实体履约平台”
- 在 V2 中更适合作为 `storefront_order` 或 `physical_reward_order` 一类业务面示例
- 默认应把其物流闭环能力视为有限或需要人工确认，除非后续核对到更具体能力

参考：

- https://itch.io/docs/creators/faq
- https://itch.io/docs/creators/payments

## pixivFANBOX

官方资料显示：

- pixivFANBOX 官方帮助中心明确存在 creator plans / price tiers
- pixivFANBOX 官方帮助中心在 physical merchandise 说明中，会引导创作者通过 BOOTH 的 secret release 处理实体物料

这意味着：

- `fanbox` 默认更应被视为 `support_plan` 业务面
- 不应在文档里把 `fanbox` 直接举例成“原生零售订单平台”
- 如果 FANBOX 场景下出现实体礼物履约，更常见的语义很可能是 `membership_entitlement`
- 如果未来真的接入了与 BOOTH 等外部销售面联动的数据，那也应单独建模为新的 `source_surface`
- 对“支持者限定购买”这类场景，更合理的建模通常是：
  - `support_plan` 只提供资格上下文
  - 真正成交的 BOOTH 或其他销售面订单仍然属于 `retail_order`

参考：

- https://fanbox.pixiv.help/hc/en-us/sections/4510559539737-Creating-Plans-Price-Tiers
- https://fanbox.pixiv.help/hc/en-us/articles/37477452376089-About-physical-merchandise-special-offers
- https://fanbox.pixiv.help/hc/en-us/articles/360015676174-Let-s-provide-limited-goods

## Bilibili

官方资料显示：

- Bilibili 官方首页本身就暴露出直播、会员购等不同业务入口
- Bilibili 官方直播内容中明确存在“大航海”用户语义
- Bilibili 官方工房内容明确围绕创作者商品经营与数据分析展开

这意味着：

- `bilibili` 不能只靠一个 `platform = bilibili` 字段承载全部业务语义
- 在 V2 里至少应允许出现 `bilibili.live_support`
- 也应允许出现 `bilibili.creator_commerce`
- 同一供应商名下的不同业务面，导入结构、身份策略和闭环能力都可能不同

参考：

- https://www.bilibili.com/
- https://www.bilibili.com/opus/1111693890251915289
- https://www.bilibili.com/opus/956778821108785194

## 文档使用规则

后续如果文档需要新增新的平台例子，应先补两件事：

1. 核对官方资料，确认该平台到底有哪些业务面
2. 明确本系统正在举例的是哪个业务面，而不是只写平台品牌名

如果做不到这两点，就应该先用抽象业务语言举例，而不是仓促写平台名。
