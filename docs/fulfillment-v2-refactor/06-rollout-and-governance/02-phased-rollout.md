# 分阶段实施计划

> V2 greenfield 重建的推进顺序。

## 阶段 0：冻结语义与基线 ✅ 已完成

冻结 V1 语义，建立 V2 业务语言和重构计划文档，创建备份分支。

## 阶段 0.5：清理边界与保留资产 ✅ 已完成

明确保留/删除内容，标记旧文档为历史参考，停止以旧结构作为设计约束。

## 阶段 1：统一命名与领域边界 ✅ 已完成

补充 V2 词汇表，明确平台多维语义，正式采用 `WaveAllocationStep`，统一命名扫尾。

## 阶段 2：搭建新代码骨架与新 Schema ✅ 已完成

清理旧业务代码，搭建 `internal/domain/`、`internal/app/`、`internal/infra/` 三层架构，建立目标 Schema。

## 阶段 3：重构履约真相层 ✅ 已完成

`FulfillmentLine` 多维状态、来源引用、工厂执行引用、渠道回填引用、`FulfillmentAdjustment` 入口。

## 阶段 4：引入上游需求层 ✅ 已完成

`DemandDocument`/`DemandLine` 支持会员权益和零售订单统一落库，双路径初始分配。

## 阶段 5：引入工厂执行层 ✅ 已完成

`SupplierOrder`/`SupplierOrderLine`，导出时创建，允许后续编辑覆盖。

## 阶段 6：引入发货与物流回传 ✅ 已完成

`Shipment`/`ShipmentLine`，支持部分发货和多包裹，basis 偏离提示。

## 阶段 7：引入来源渠道回填 ✅ 已完成

`ChannelSyncJob`/`ChannelSyncItem`，承运商映射，可区分待回填/已回填/回填失败。

## 阶段 8：重做波次状态投影与 UI 📋 进行中

- 替换 `Wave.Status` 简化展示
- 重做 Dashboard 和波次列表
- 工作流拆出独立页面：`Membership Allocation`、`Demand Mapping`、`Wave Overview`、`Adjustment Review`
- 步骤向导与跨步骤导航
- 非破坏性编辑约束

## 阶段 9：工作区历史与树状撤销重做 📋 待实现

- `HistoryScope`/`HistoryNode`/`HistoryCheckpoint`/`HistoryPin`
- 树状分支、持久化、basis pin、GC
- `Ctrl+Z`/`Ctrl+Shift+Z`，toast 反馈

## 阶段 10：模板系统能力化 📋 待实现

- `IntegrationProfile` 作为配置入口
- `TemplateConfig` → `DocumentTemplate`
- 连接器能力描述
