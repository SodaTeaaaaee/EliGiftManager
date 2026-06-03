# 会员权益判定、输入采集与路由模型

> `membership_entitlement` 进入履约链路前的前置判断。

## 三个独立维度

| 维度 | 字段 | 回答 |
|------|------|------|
| 权益判定权威 | `entitlement_authority` | 谁说权益成立了？`local_policy`/`upstream_platform`/`manual_grant` |
| 输入采集状态 | `recipient_input_state` | 能不能转成可执行履约？`not_required`/`waiting_for_input`/`partially_collected`/`ready`/`waived`/`expired` |
| 路由处置 | `routing_disposition` | 本系统接不接手？`pending_intake`/`accepted`/`deferred`/`excluded_manual`/`excluded_duplicate`/`excluded_revoked` |

这三件事不能混成一个状态字段。

## 关键边界

- `excluded_manual` = 本系统不接手 ≠ 系统外已履约完成
- `accepted` = 本系统接手 ≠ 已归属到某个具体 wave
- `waiting_for_input` 的需求可进入波次上下文，但不应过早视为可执行履约
- 输入来源不限于平台 claim，也包括外部表单、私聊协商、手工录入

## 何时进入波次

- 只有 `routing_disposition = accepted` 才进入稳定波次处理
- `deferred` 保留在候选池，不伪装成已进入执行链
- `excluded_manual` 进入统计但单独归类

## 统计呈现

分开显示：`accepted`、`waiting_for_input`、`deferred`、`excluded_manual`、`excluded_duplicate`、`excluded_revoked`。

## 典型判断

- 连续订阅阶段性礼物：`membership_entitlement` + `loyalty_membership` + `upstream_platform`
- 支持者限定购买：`retail_order` + `supporter_only_purchase` + `eligibility_context_ref`

## 与执行层的边界

前置层解决"权益是否成立、输入是否收齐、本系统是否接手"。执行层解决"接手后怎么执行"。两者不互相吞并。
