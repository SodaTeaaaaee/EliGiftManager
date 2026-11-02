# 现行设计决定

领域背景见 [CONTEXT.md](../CONTEXT.md)，专有名词见 [GLOSSARY.md](../GLOSSARY.md)，目标模型见 [TARGET-DOMAIN-AND-PRODUCT-MODEL.md](./TARGET-DOMAIN-AND-PRODUCT-MODEL.md)。本文只索引已在代码中核实的硬架构；目标模型中的规划能力不能据此视为已实现。历史 ADR 的核实依据与归宿见 [ADR 审查记录](./ADR-AUDIT.md)。

## 责任与归属

- [ADR 0010](./adr/0010-input-is-document-fact-line.md)：来源证据、事实与波次归属使用不同层级。
- [ADR 0012](./adr/0012-import-auto-creates-customer-profile.md)：客户档案独立于平台身份，导入即建立归属。
- [ADR 0003](./adr/0003-membership-fact-is-entitlement-instance.md)：会员身份事实是独立权益实例的边界。
- [ADR 0008](./adr/0008-entitlement-rules-live-on-the-wave.md)：权益规则的作用域是波次。
- [ADR 0004](./adr/0004-source-level-fulfillment-results.md)：履约责任持久化为来源级叶子。
- [ADR 0007](./adr/0007-wave-status-is-derived-not-a-stage-machine.md)：波次关闭结果与执行进展相互独立。
- [ADR 0002](./adr/0002-configurable-duplicate-import-windows.md)：稳定来源标识与疑似重复采用不同的责任判定。

## 商品与工厂执行

- [ADR 0001](./adr/0001-canonical-product-is-factory-bound.md)：统一商品事实以单一工厂 SKU 为身份锚点。
- [ADR 0011](./adr/0011-generate-factory-order-then-export.md)：工厂订单生成就是责任冻结与追踪身份的建立点。
- [ADR 0009](./adr/0009-factory-handles-lost-package-resend.md)：丢件重发仍属于工厂原执行责任。
- [ADR 0005](./adr/0005-source-order-may-write-back-multiple-parcels.md)：渠道回写以来源事实与包裹为单位。

## 外部协议与桌面边界

- [ADR 0006](./adr/0006-template-bound-to-integration-document-direction.md)：外部协议通过声明式模板对齐且仅保留版本引用。
- [ADR 0013](./adr/0013-migrate-to-wails-v3-beta.md)：桌面宿主采用 Wails v3，前端通过单一边界调用 Go。

## 已核实的实现限制

- 模板版本只是 ID 与版本号引用，旧配置内容不保存；工厂订单再次导出覆盖既有导出载荷与模板引用。
- 执行数量关系只覆盖履约结果到工厂订单行。渠道回写项复制包裹数量，没有多来源分摊或平台多包裹能力校验。
- 全局商品别名与组合展开已实现，通用波次别名覆盖、对齐版本历史与组合版本历史未实现。数量拆分例外的作用域是波次、平台与商品键。
- 后端校验转换器名称和文档类型，尚未强制映射字段键属于语义字典。
- 波次关闭保存当前结果与最近时间，没有关闭事件历史或完整关闭的自动清空校验。

这些差异在保留 ADR 正文和归档说明中明确记录，没有通过改写文档替代码补齐能力。
