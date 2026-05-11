# Fulfillment V2 Refactor

本目录是 EliGiftManager 履约系统 V2 重构的模块化计划入口，用于替代原先单文件的长篇总稿。

本次重构的核心目标不是继续给现有发货流程补字段，而是把系统从“会员回馈导出工具”演进为“支持会员权益、创作者零售、工厂履约、物流回传、来源渠道回填”的长生命周期履约系统。

## 阅读顺序

1. 先看 [00-overview/01-document-purpose-and-baseline.md](./00-overview/01-document-purpose-and-baseline.md) 与 [00-overview/02-current-state-and-gap-analysis.md](./00-overview/02-current-state-and-gap-analysis.md)
2. 再看 [01-boundaries-and-language](./01-boundaries-and-language/) 下的边界、统一业务语言、平台与 profile 讨论
   其中 [01-boundaries-and-language/04-source-backed-platform-example-notes.md](./01-boundaries-and-language/04-source-backed-platform-example-notes.md) 专门记录了当前平台例子的官方资料依据
3. 然后看 [02-allocation-model](./02-allocation-model/) 与 [03-data-model](./03-data-model/)，这两部分定义未来核心语义和数据结构
4. 再看 [04-workflows-and-state](./04-workflows-and-state/)，用于理解波次生命周期、状态与进度展示
   其中 [04-workflows-and-state/03-entitlement-resolution-and-routing.md](./04-workflows-and-state/03-entitlement-resolution-and-routing.md) 专门说明 `membership_entitlement` 的判定权威、会员输入采集与本系统路由决策
5. 最后看 [05-profile-system](./05-profile-system/) 与 [06-rollout-and-governance](./06-rollout-and-governance/)，用于落地实施与迁移治理

## 目录结构

- `00-overview/`
  - 解释文档目标、基线分支、当前现状与主要缺口
- `01-boundaries-and-language/`
  - 定义业务边界、统一名词、平台维度、`IntegrationProfile` / profile 系统定位
- `02-allocation-model/`
  - 定义会员与零售在分配语义上的差异、混合波次的统一方式、`WaveAllocationStep` 的演化方向
- `03-data-model/`
  - 给出 V2 目标数据结构、分层边界、当前模型到目标模型的映射
- `04-workflows-and-state/`
  - 定义长生命周期工作流、行级状态、波次聚合状态与进度展示，并补充会员权益型需求的判定、输入采集与路由模型
- `05-profile-system/`
  - 说明为什么模板系统要升级为 profile 系统，以及 profile / 模板 / service 的分工
- `06-rollout-and-governance/`
  - 包含实施原则、阶段计划、迁移策略、测试验收、风险与待决策问题
- `legacy/`
  - 保留拆分前的完整长稿，便于全文检索和历史对照

## 基线与归档

- 重构前代码基线分支：`backup/pre-fulfillment-v2-refactor-2026-05-12`
- 拆分前完整总稿： [legacy/FULL-DRAFT-2026-05-12.md](./legacy/FULL-DRAFT-2026-05-12.md)
- 顶层入口兼容文件： [../FULFILLMENT-V2-REFACTOR-PLAN.md](../FULFILLMENT-V2-REFACTOR-PLAN.md)

## 当前重点约束

- 会员分配的现有可用性不能因为支持零售而退化
- 不把“会员平台 / 零售平台”当成唯一平台分类方式，而是拆成来源渠道、业务面、能力面来建模
- 物流映射、快递单号转换、来源渠道回填必须纳入主数据结构与生命周期，而不是后补脚本
- 模板系统的升级方向应是 profile 系统，而不是继续堆叠零散模板类型

## 待决策问题

请优先参考 [06-rollout-and-governance/06-open-decisions.md](./06-rollout-and-governance/06-open-decisions.md)。其中列出了几个会直接影响 V2 方案收敛速度的设计决策点。
