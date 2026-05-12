# 平台能力模型与边界

本文件定义平台能力模型，并明确哪些内容不应该被放进 profile。

### 9.6 平台能力模型

建议增加“平台能力”概念，而不是让模板自己承担全部职责。

这里的“平台能力”更准确地说，应当按 `IntegrationProfile` 或 `source_surface` 建模，
而不是按平台品牌全局一刀切。

典型能力：

- `supports_partial_shipment`
- `requires_carrier_mapping`
- `requires_external_order_no`
- `entitlement_authority_mode`
- `recipient_input_collection_mode`
- `tracking_sync_mode`
- `closure_policy`
- `allows_manual_closure`
- `supports_api_import`
- `supports_api_export`

说明：

- 模板解决字段问题
- 能力模型解决流程问题
- `tracking_sync_mode` 用来区分自动回填、文档回填、人工确认、完全不支持
- `closure_policy` 用来决定某个业务面在什么条件下才算真正闭环
- `entitlement_authority_mode` 用来说明权益是否应由上游平台判定
- `recipient_input_collection_mode` 用来说明地址、款式、领取确认等参数通常如何补齐
- `allows_manual_closure` 用来说明该业务面是否允许人工闭环决策进入统计

这里还应再加一条收敛约束：

- 不建议同时保留一组抽象 `can_*` 能力、另一组 `supports_*` 布尔值，以及重复表达相同问题的 strategy 字段
- 更稳妥的方式是：
  - strategy 字段表达主流程分支
  - capability flag 表达正交能力
  - connector 负责真实实现差异

### 9.7 不应该放进 Profile 的东西

虽然 Profile 比模板更强，但它不应演变成万能低代码系统。

以下内容不建议完全配置化进 Profile：

- 复杂的波次分配算法
- 复杂的工厂回传合并逻辑
- 所有异常分支的完整 DSL
- 大量条件式脚本
- 需要深入调试的业务规则代码
- 复杂动态 selector 或多身份交集规则语言

Profile 更适合承载：

- 能力声明
- 策略枚举
- 闭环规则
- 模板绑定
- 身份与引用规则
- 权益判定权威模式
- 输入采集模式
- 导入导出入口配置

而真正复杂的流程逻辑仍应保留在 service 层。

---
