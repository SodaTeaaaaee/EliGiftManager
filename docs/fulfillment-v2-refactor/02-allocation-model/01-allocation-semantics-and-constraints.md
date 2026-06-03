# 分配语义与约束

> 会员权益与零售订单在分配语义上的根本差异。

## 两种基础语义

| 语义 | 适用 | 特征 |
|------|------|------|
| `policy_driven` | 会员权益、活动回馈 | 规则推导初始履约结果，允许协商/补偿/赠送修正 |
| `demand_driven` | 零售订单、手工补单 | 上游已是明确需求行，优先尊重原始订单 |

判断规则：不主动下单也欠用户应发物 → `policy_driven`；不主动下单就不出现待履约行 → `demand_driven`。

## Tag 系统在 V2 中的定位

当前 tag 系统是优秀的 `policy_driven` 分配系统。V2 保留其主导地位，不强迫零售订单翻译成 tag，两种方式收敛到同一套 `FulfillmentLine`。

## 会员体验不得退化（强约束）

1. 会员权益型波次仍可使用规则驱动分配
2. 批量加赠/减赠/覆盖能力不得退化
3. 不能为统一零售语义破坏会员分配体验

## 四层分配结构

| 层 | 说明 | 约束 |
|----|------|------|
| `Allocation Contribution` | 规则对某参与者/某商品贡献的数量 | 可正可负 |
| `Base Allocation Result` | 初始分配层结算后的第一版履约结果 | 非负（≤0 则不产生行） |
| `FulfillmentAdjustment` | 当前波次最终履约例外 | delta 可正可负，面向具体对象 |
| `Resolved Fulfillment Result` | 最终执行结果 | `max(base + delta, 0)`，不为负 |

## 动态集合语义归属

- 动态 selector 规则属于 `AllocationPolicyService`（`Membership Allocation` 入口）
- 具体对象例外属于 `FulfillmentAdjustmentService`（`Adjustment Review` 入口）
- 首版覆盖：单平台身份等级、`platform_all`、`wave_all`、显式用户覆盖
- 页面不是领域所有者，只是工作流入口和观察视图
