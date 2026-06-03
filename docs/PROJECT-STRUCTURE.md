# Project Structure

> 本文件描述当前代码结构。
> 业务语义说明见 [`docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md`](./PRODUCT-DOMAIN-AND-PAIN-POINTS.md)（需求类型、平台维度、核心痛点）和 [`docs/fulfillment-v2-refactor/`](./fulfillment-v2-refactor/)（设计文档）。

## Docs

- `docs/fulfillment-v2-refactor/` — 履约重构设计文档，覆盖业务边界、数据模型、工作流、profile 系统、非功能基础与实施治理。
- `docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md` — 业务域、需求类型、平台维度、核心痛点。
- `docs/DEVELOPMENT.md` — 日常开发命令、格式化规则、测试入口。
- `docs/FULFILLMENT-V2-REFACTOR-PLAN.md` — 兼容入口页，指向 `fulfillment-v2-refactor/README.md`。

文档优先级（口径冲突时）：

1. 当前代码中的 domain / app / infra 实现
2. `docs/fulfillment-v2-refactor/` 设计文档
3. 其他文档

## Root

- `main.go` — Wails 应用入口、窗口配置、控制器绑定。
- `app.go` — 桌面级公共函数（文件选择、zoom 持久化等）。
- `context_provider.go` — 全局 context 注入。
- `zoom_config.go` — zoom 级别读写。
- `controller_demand.go` — 需求文档导入与路由。
- `controller_wave.go` — 波次生命周期、参与者、总览。
- `controller_export.go` — 供应商订单导出。
- `controller_shipment.go` — 发货创建与批量导入。
- `controller_channel_sync.go` — 渠道同步规划与执行。
- `controller_adjustment.go` — 履约调整与重放。
- `controller_product.go` — 商品目录管理。
- `controller_profile.go` — IntegrationProfile 配置。
- `controller_template.go` — 文档模板与绑定管理。
- `controller_allocation_policy.go` — 分配规则策略。
- `controller_address.go` — 地址管理。
- `controller_customer_profile.go` — 客户资料管理。
- `controller_merge.go` — 客户资料归并建议。

## Backend — 三层架构

```
internal/domain/       领域实体、仓库接口、枚举、策略逻辑
internal/app/          用例、DTO、业务编排、投影查询
internal/infra/        仓库实现（GORM）、迁移、外部 API 客户端
internal/db/           SQLite 初始化、默认连接管理
internal/config/       静态应用配置
internal/service/      路径解析、设置等基础设施服务
internal/middleware/   运行时中间件（本地资源服务）
```

### `internal/domain`

- `models.go` — 核心领域实体：`CustomerProfile`、`CustomerIdentity`、`CustomerAddress`、`DemandDocument`、`DemandLine`、`Wave`、`WaveParticipantSnapshot`、`FulfillmentLine`、`AllocationPolicyRule`、`SupplierOrder`、`SupplierOrderLine`、`Shipment`、`ShipmentLine`、`ChannelSyncJob`、`ChannelSyncItem`、`IntegrationProfile`、`FulfillmentAdjustment`、`ProductMaster`、`Product`、`CarrierMapping`、`HistoryScope`/`HistoryNode`/`HistoryCheckpoint`/`HistoryPin` 等。
- `ports.go` — 约 21 个仓库接口定义。
- `enums.go` — 业务枚举：`ProfileType`、`DemandKind`、`CaptureMode`、`DemandLineType`、`ObligationTriggerKind`、`EntitlementAuthority`、`RecipientInputState`、`RoutingDisposition`、`WaveType`、`LifecycleStage`、`AllocationState`、`AddressState`、`SupplierState`、`ChannelSyncState`、`ProductKind` 等。

### `internal/app`

- 用例文件（`use_cases.go` 等）— 业务编排逻辑。
- DTO（`dto/`）— 数据传输对象。
- 选择器（`selector/`）— 查询参数。
- 投影查询（`*_projection_*`、`*_query_*`）— 只读聚合查询。
- 波次快照服务（`wave_snapshot_service.go`）— 波次参与者快照管理。
- 工作区守卫（`workspace_guard_service.go`）— 工作区编辑守卫。
- 重放逻辑（`replay.go`）— 调整重放。

### `internal/infra`

- `*_repo.go` — 约 14 个 GORM 仓库实现。
- `persistence/` — GORM 模型定义与领域模型映射。
- `tx_repos.go` — 事务管理。
- 外部 API 客户端：`taobao_client.go`、`pdd_client.go`、`jd_client.go`、`douyin_client.go`。
- `migrator.go` — AutoMigration。
- `scheduler.go` — gocron 后台调度。

## Frontend

前端在 `frontend/`，技术栈是 Vue 3 + Pinia + Vite + Deno task runner + Naive UI + Tailwind CSS。

### `frontend/src/app`

应用壳、全局布局、路由注册。

### `frontend/src/pages`

路由级页面。

### `frontend/src/entities`

TypeScript 实体类型（镜像 Go DTO）。

### `frontend/src/shared`

- `composables/` — 组合式逻辑。
- `model/` — 响应式单例（设置、主题）。
- `ui/` — 通用 UI 组件。
- `lib/wails/app.ts` — 前端唯一允许直接对接 `frontend/wailsjs` 的包装层。

页面和 composable 不应直接 import `frontend/wailsjs`。

### `frontend/wailsjs`

Wails 生成的桥接文件。当前仓库提交它们，因为前端代码直接消费这层。

## Runtime Data

运行时数据在 `data/` 下，由 `internal/service/path_service.go` 解析：

1. **开发模式** — 工作目录下 `data/`
2. **便携模式** — `.portable` 标记文件存在时，使用 `exe/data`
3. **系统模式** — `os.UserConfigDir()/EliGiftManager/data`

- 数据库：`data/eligiftmanager.db`
- 资源：`data/assets/`
- 导出：`data/exports/`
- 临时：`data/tmp/`

## Table Architecture Principles

表格数据查询遵循以下原则：

1. **查询真相由后端决定**：排序、过滤、分页的最终结果由后端产出，前端不再对全量数据做最终排序或分页。
2. **DTO 轻量化**：列表接口返回最小字段集，详情信息按需加载。
3. **统一查询状态模型**：所有表格共享同一套 `TableQueryState`（keyword / filters / sorts / page / pageSize / mode），不因页面不同而发明不同协议。
4. **大小数据共存**：小数据页面可保留本地前端方案，大数据页面切远程查询，底层查询状态抽象共享。
5. **排序字段白名单**：后端维护每个接口允许的排序字段映射，不接受前端任意字符串直拼 SQL。
6. **稳定尾排序键**：后端自动追加 `updated_at DESC, id DESC` 作为尾键，保证主排序值相同时结果可预测。

当前项目中小数据集仍可使用前端全量方案。当单表结果集常态达到数千行以上、或离屏测量明显拖慢首屏时，应切向远程查询模式。会员/买家列表是最可能优先切换的页面。

## Generated vs Authored

| 路径 | 状态 |
|------|------|
| `internal/`、`frontend/src/` | 业务源码 |
| `frontend/wailsjs/` | 生成桥接（已提交） |
| `frontend/dist/`、`build/bin/` | 构建产物（已忽略） |
| `.cache/`、`.claude/`、`.agents/` | 工具缓存（已忽略） |
