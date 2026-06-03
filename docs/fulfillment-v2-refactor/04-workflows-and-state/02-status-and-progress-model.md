# 状态与进度模型重构

> 行级状态、波次聚合状态和进度展示。

## 前置状态（DemandLine 级别，先于 FulfillmentLine）

- `recipient_input_state`：`not_required` / `waiting_for_input` / `partially_collected` / `ready` / `waived` / `expired`
- `routing_disposition`：`pending_intake` / `accepted` / `deferred` / `excluded_manual` / `excluded_duplicate` / `excluded_revoked`

## FulfillmentLine 多维状态

| 维度 | 值 |
|------|-----|
| `allocation_state` | `draft` / `ready` |
| `address_state` | `missing` / `ready` / `invalid` |
| `supplier_state` | `not_submitted` / `submitted` / `accepted` / `producing` / `partially_shipped` / `shipped` / `canceled` |
| `channel_sync_state` | `not_required` / `unsupported` / `pending` / `synced` / `manual_confirmed` / `skipped` / `failed` |

状态表达当前工作区最新已知投影，不禁止后续编辑。

### 轻量手动进度

底层事实状态由真实对象自动推导，只在闭环边界允许人工决策（`mark_sync_unsupported`、`mark_sync_completed_manually` 等）。每条决策带 `reason_code`、`operator_id`、`note`。

## Basis 偏离提示（双轴独立）

| 轴 | 值 | 回答 |
|----|-----|------|
| `basis_drift_status` | `in_sync` / `drifted` | 工作区是否偏离 basis |
| `review_requirement` | `none` / `recommended` / `required` | 是否需要人工复核 |

底层双轴，UI 可投影成单一总结状态。

### 什么触发 required

- adjustment target 被删除或不再唯一
- 最近导出/物流/回填 basis 无法无歧义映射到当前对象
- 负数中间量本身不触发 required

### 提示强度

| 组合 | 强度 |
|------|------|
| `in_sync + none` | 正常 |
| `drifted + none` | 弱提示 |
| `drifted + recommended` | 较强提醒 |
| `drifted + required` | 强提示，建议复核 |

## 波次聚合状态

`draft` → `allocating` → `address_blocked` → `ready_to_submit` → `submitted_to_supplier` → `partially_shipped` → `shipped` → `syncing_back` → `awaiting_manual_closure` → `closed`

## 进度展示

放弃伪百分比，改用可解释漏斗：总行数 → 地址就绪 → 已提交工厂 → 已回传快递 → 已完成回填 → 已人工闭环 → 失败回填。

## Wave Overview

只读聚合总览，不承担编辑职责。首版最小分组：可继续 / 需回分配页 / 需回映射页 / 建议进调整层 / 等待输入/地址/资格 / 未纳入处理。
