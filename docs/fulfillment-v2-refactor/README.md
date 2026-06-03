# Fulfillment V2 Refactor

本目录是 EliGiftManager 履约系统 V2 重构的设计文档。

> **实现状态**：V2 核心已通过 greenfield 重建落地（阶段 0–7 完成），当前代码架构为 `internal/domain/` + `internal/app/` + `internal/infra/`。阶段 8（UI 重构）进行中。

## 阅读顺序

1. [00-overview/](./00-overview/) — 文档目标、基线、当前缺口
2. [01-boundaries-and-language/](./01-boundaries-and-language/) — 业务边界、统一业务语言、平台维度、Profile 定位
3. [02-allocation-model/](./02-allocation-model/) — 分配语义、会员/零售/混合波次、WaveAllocationStep、调整重放算法
4. [03-data-model/](./03-data-model/) — 目标数据结构、当前→目标映射、工作区历史与 basis 模型、身份归并
5. [04-workflows-and-state/](./04-workflows-and-state/) — 波次工作流、状态与进度模型、权益判定与路由、undo/redo
6. [05-profile-system/](./05-profile-system/) — Profile 系统、模板与 service 分层、平台能力模型
7. [06-rollout-and-governance/](./06-rollout-and-governance/) — 实施原则、阶段计划、greenfield 策略、测试验收、决策记录、风险、错误处理、核心不变量
8. [07-non-functional-foundations/](./07-non-functional-foundations/) — 大数据查询、i18n、并发与性能

## 目录结构

| 目录 | 内容 |
|------|------|
| `00-overview/` | 文档目标、基线分支、当前缺口分析 |
| `01-boundaries-and-language/` | 业务边界、统一名词、平台维度、Profile 定位、平台官方资料 |
| `02-allocation-model/` | 分配语义、波次模型、WaveAllocationStep、调整重放算法 |
| `03-data-model/` | 目标数据结构、当前→目标映射、工作区历史与 basis、身份归并 |
| `04-workflows-and-state/` | 波次工作流、状态模型、权益判定与路由、undo/redo |
| `05-profile-system/` | Profile 系统、模板与 service 分层、平台能力 |
| `06-rollout-and-governance/` | 实施原则、阶段计划、迁移策略、测试验收、决策记录、风险、错误处理、核心不变量 |
| `07-non-functional-foundations/` | 大数据查询、i18n、并发假设与性能约束 |
| `legacy/` | 拆分前完整长稿归档 |

## 基线

- 重构前代码基线：`backup/pre-fulfillment-v2-refactor-2026-05-12`
- 拆分前完整总稿：[legacy/FULL-DRAFT-2026-05-12.md](./legacy/FULL-DRAFT-2026-05-12.md)

## 核心设计约束

- 会员分配体验不得退化
- 平台拆为 `source_channel` + `source_surface`，不刚性二分
- 物流映射、快递单号、来源渠道回纳入主数据结构
- Profile 收敛为"策略声明 + 正交能力标记 + 连接器绑定"
- 工作区历史为树状分支、持久化、basis 引用协同
- 大数据查询与 i18n 属于 V2 基础能力
- greenfield 重建，旧代码只保留在备份分支

## 待决策问题

见 [06-rollout-and-governance/06-open-decisions.md](./06-rollout-and-governance/06-open-decisions.md)。
