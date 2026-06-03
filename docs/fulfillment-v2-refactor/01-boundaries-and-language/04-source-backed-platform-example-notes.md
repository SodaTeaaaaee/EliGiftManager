# 基于官方资料的平台例子说明

> 约束 V2 文档中使用平台例子的方式。任何平台例子都应先核对官方资料，落到"供应商+业务面+能力"维度。

## Patreon

- 存在 membership tiers、one-time purchases、merch for membership
- 不能建模为单一"会员平台"
- V2：`patreon.membership`、`patreon.shop_purchase`
- 连续订阅阶段性礼物的"是否 earned"视为上游平台权威结果

参考：[membership tiers](https://support.patreon.com/hc/en-us/articles/218202363)、[one-time purchases](https://support.patreon.com/hc/en-us/articles/4413300654733)、[merch for membership](https://support.patreon.com/hc/en-us/articles/11111747095181)

## Gumroad

- 存在 memberships/subscriptions 和 shipped purchases
- 不能建模为单一"零售订单平台"
- V2：`gumroad.membership`、`gumroad.one_time_order`
- 物流闭环能力应保守建模

参考：[features](https://gumroad.com/features)、[purchase status](https://gumroad.com/help/article/209)

## itch.io

- 创作者店面，可通过 custom fields 收集实体邮寄地址
- 不应视为"完整原生实体履约平台"
- V2：`storefront_order` 或 `physical_reward_order` 业务面
- 物流闭环能力有限或需人工确认

参考：[creators FAQ](https://itch.io/docs/creators/faq)、[payments](https://itch.io/docs/creators/payments)

## pixivFANBOX

- 存在 creator plans / price tiers，实体物料通过 BOOTH secret release
- 默认为 `support_plan` 业务面
- 实体礼物履约更常见语义是 `membership_entitlement`
- 支持者限定购买：`support_plan` 提供资格，BOOTH 订单属于 `retail_order`

参考：[plans](https://fanbox.pixiv.help/hc/en-us/sections/4510559539737)、[physical merchandise](https://fanbox.pixiv.help/hc/en-us/articles/37477452376089)

## Bilibili

- 直播、会员购等不同业务入口
- V2：`bilibili.live_support`、`bilibili.creator_commerce`
- 不同业务面的导入结构、身份策略和闭环能力可能不同

## 新增平台例子规则

1. 核对官方资料，确认有哪些业务面
2. 明确举例的是哪个业务面，不只写品牌名
3. 做不到就用抽象业务语言
