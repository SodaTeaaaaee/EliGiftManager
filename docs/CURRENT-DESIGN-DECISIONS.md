# 现行设计决定

本文是已接受决定的摘要。术语以 [CONTEXT.md](../CONTEXT.md) 为准，工作流以 [TARGET-DOMAIN-AND-PRODUCT-MODEL.md](./TARGET-DOMAIN-AND-PRODUCT-MODEL.md) 为准，取舍理由在 [docs/adr/](./adr/)。

实现前还要补齐实体字段、基数、状态机和画面操作清单。未写清的部分见文末。当前代码不是目标。

## 重做契约

- 主路径按新模型一次性重建，不保留旧数据或旧接口兼容层。[ADR 0031](./adr/0031-no-backward-compatibility-burden.md)、[ADR 0033](./adr/0033-rebuild-end-to-end-primary-path.md)
- 实现命名直接使用 `InputDocument`、`InputFact`、`InputFactLine`、`FulfillmentResult`、`SupplierOrderLine`、`ProductItem`、`ProductAlias`、`CustomerProfile`。Demand、FulfillmentLine、Supporter 作为档案类型名都不是目标概念。[ADR 0015](./adr/0015-retire-demand-as-domain-supertype.md)、[ADR 0032](./adr/0032-rename-demand-and-fulfillment-line-concepts.md)、[ADR 0051](./adr/0051-person-record-is-customer-profile.md)
- 现有 `frontend/` 只作为共享组件来源，不是目标信息架构。`frontend-legacy/` 删除。
- 不带入 command/patch 历史图、legacy 客户迁移表，以及作为流程权威的波次阶段机。[ADR 0044](./adr/0044-rebuild-drops-history-graph-and-legacy-migration.md)、[ADR 0045](./adr/0045-wave-status-is-derived-not-a-stage-machine.md)

## 第一层导航

待处理、收件箱、波次、资料库、设置。[ADR 0034](./adr/0034-top-navigation-and-library.md)

- 待处理首页展示跨波次待处理事项和最近波次。收件箱是其中一个分区，不独占首页。[ADR 0026](./adr/0026-home-centers-work-to-do-not-inbox.md)
- 收件箱是输入事实工作台：解析、归属、重复判定、商品对齐冲突、身份挂靠、输入修订。默认行是输入事实行，文档只是分组。第一期不做两个客户档案合并。[ADR 0027](./adr/0027-inbox-is-input-fact-workbench.md)、[ADR 0053](./adr/0053-v1-identity-attach-not-profile-merge.md)、[ADR 0057](./adr/0057-inbox-rows-are-fact-lines.md)
- 波次有两个主面。规则面写本期权益和运营授予，改规则当场重算未提交的权益结果。履约结果工作台编组、提交工厂、发货和回写。编组只改视图。[ADR 0028](./adr/0028-wave-centers-fulfillment-result-workbench.md)、[ADR 0047](./adr/0047-entitlement-rules-live-on-the-wave.md)、[ADR 0055](./adr/0055-wave-has-two-primary-surfaces.md)、[ADR 0056](./adr/0056-recompute-unsubmitted-entitlement-on-rule-edit.md)
- 履约结果工作态由数据派生，不持久化单一生命周期。[ADR 0058](./adr/0058-fulfillment-work-state-is-derived.md)
- 资料库在第一层之内只有三个平级区：客户、商品、模板。客户不进第一层导航。平台是模板归属，承运商映射挂在平台上。[ADR 0046](./adr/0046-supporter-master-data-lives-in-library.md)、[ADR 0049](./adr/0049-library-has-three-sections.md)、[ADR 0051](./adr/0051-person-record-is-customer-profile.md)、[ADR 0052](./adr/0052-platform-owns-templates-and-carrier-maps.md)
- 设置保存本地偏好和重复导入窗口默认。
- 第一期不做撤销。[ADR 0050](./adr/0050-no-undo-in-v1.md)

## 输入、波次、结果

- 输入事实按类型区分：会员身份事实、零售订单、运营授予。它们不是「需求」。运营授予是无外部来源的指定应发。丢件重发由工厂处理，系统不新增履约结果。[ADR 0015](./adr/0015-retire-demand-as-domain-supertype.md)、[ADR 0048](./adr/0048-operator-grant-covers-reissue.md)、[ADR 0054](./adr/0054-factory-handles-lost-package-resend.md)
- 输入永远是文档、事实、行三层。会员身份事实恰好一行，零售订单一个事实多行，运营授予一个事实一行。归属在行上，回写按事实分组。[ADR 0008](./adr/0008-wave-assignment-and-reopening.md)、[ADR 0059](./adr/0059-input-is-document-fact-line.md)
- 波次保存编号、名称、备注和关闭结果（进行中、完整关闭、带残留关闭），以及关闭说明、关闭时间、重开时间。类型和待处理阶段都是投影。[ADR 0009](./adr/0009-wave-type-is-derived.md)、[ADR 0045](./adr/0045-wave-status-is-derived-not-a-stage-machine.md)
- 波次可显式重开；已提交履约继续冻结。[ADR 0014](./adr/0014-distinguish-clean-and-residual-closure.md)
- 来源级履约结果是叶子。会员权益实例、零售订单行和运营授予分别产生结果，界面汇总不能替代它们。[ADR 0001](./adr/0001-preserve-result-source-details.md)、[ADR 0021](./adr/0021-source-level-fulfillment-results.md)、[ADR 0048](./adr/0048-operator-grant-covers-reissue.md)
- 工厂订单行通过带数量的关系承接履约结果。每个波次每个工厂一份订单草稿。生成工厂生产订单时，每行随机得到内部追踪标识，每行每波次唯一，不来自履约结果 ID。[ADR 0013](./adr/0013-record-quantities-on-execution-links.md)、[ADR 0017](./adr/0017-execution-correlation-id-for-merged-lines.md)、[ADR 0062](./adr/0062-tracking-id-is-random-per-supplier-line.md)
- 履约结果工作态由数据派生。第一期阻塞原因闭集：商品未对齐、收件信息不可用、身份未挂靠、数量拆分未凑满来源数量。[ADR 0058](./adr/0058-fulfillment-work-state-is-derived.md)、[ADR 0061](./adr/0061-block-reasons-are-a-closed-set.md)
- 先生成工厂订单（追踪标识、冻结履约）再导出文件。导出失败重试仍用原标识。尚未导出的订单可以作废并解冻履约；已导出的不能作废。[ADR 0002](./adr/0002-freeze-fulfillment-after-supplier-submission.md)、[ADR 0063](./adr/0063-generate-factory-order-then-export.md)、[ADR 0067](./adr/0067-void-unexported-factory-order.md)
- 收件信息快照来自客户档案地址，生成工厂订单前可换，生成后冻结。[ADR 0023](./adr/0023-freeze-address-snapshot-on-submission.md)、[ADR 0064](./adr/0064-address-snapshot-from-customer-profile.md)
- 待处理首页工作桶：未归属、待决策重复、商品对齐冲突、身份未挂靠、阻塞的履约结果、回写失败、可带残留关闭，外加最近波次。[ADR 0066](./adr/0066-home-work-buckets.md)

## 商品与规则

- 统一商品事实绑定一个负责工厂和一个有效工厂 SKU。外部 ID、名称、规格是别名。[ADR 0004](./adr/0004-canonical-product-and-external-alignment.md)、[ADR 0005](./adr/0005-canonical-product-is-factory-bound.md)
- 别名默认全局对齐，波次可覆盖一次。冲突阻断对应输入。[ADR 0012](./adr/0012-global-product-aliases-with-wave-overrides.md)
- 组合装走商品组合映射。数量拆分映射是高级例外，默认只解释当前波次或当前来源行。[ADR 0022](./adr/0022-product-bundle-mapping.md)、[ADR 0024](./adr/0024-quantity-split-mapping-is-advanced-exception.md)
- 会员权益规则只存在于波次，是波次编辑的核心数据。资料库商品不保存应发数量。编辑器仍用商品视角。选择器只有三种：平台加等级、本波次全部会员事实、点名权益实例。规则面显示每个商品尚未进入工厂订单的应发合计。[ADR 0029](./adr/0029-membership-rules-use-product-centered-editor.md)、[ADR 0047](./adr/0047-entitlement-rules-live-on-the-wave.md)、[ADR 0060](./adr/0060-entitlement-selectors-are-level-all-or-instance.md)、[ADR 0068](./adr/0068-rule-surface-shows-product-totals.md)
- 每条已接受的会员身份事实在波次内形成独立权益实例。客户合并不减少独立会员关系。[ADR 0011](./adr/0011-preserve-multiple-membership-facts.md)、[ADR 0018](./adr/0018-membership-fact-is-entitlement-instance.md)

## 模板

模板绑定「平台、文档类型、输入或输出方向」。承运商映射挂在平台上。映射目标来自第一期封闭语义字典和转换器清单，见目标模型文档。[ADR 0035](./adr/0035-template-bound-to-integration-document-direction.md) 至 [ADR 0043](./adr/0043-builtin-templates-are-read-only-and-copyable.md)、[ADR 0052](./adr/0052-platform-owns-templates-and-carrier-maps.md)、[ADR 0065](./adr/0065-closed-semantic-dictionary-v1.md)

## 实现边界

实体与基数、语义字典、画面与操作已经写在 [TARGET-DOMAIN-AND-PRODUCT-MODEL.md](./TARGET-DOMAIN-AND-PRODUCT-MODEL.md)。实现时其余字段（时间戳、备注、扩展 JSON）跟实体走，不再单独拍板。当前代码不是目标。不按旧 Demand / FulfillmentLine 模型开工。
