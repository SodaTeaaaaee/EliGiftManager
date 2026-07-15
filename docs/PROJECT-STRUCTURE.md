# Project Structure

> 本文件描述 2026-07-13 前端切换后的当前代码结构。
> 业务语义说明见 [`docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md`](./PRODUCT-DOMAIN-AND-PAIN-POINTS.md) 和 [`docs/fulfillment-v2-refactor/`](./fulfillment-v2-refactor/)。

## Top Level

```text
main.go, app.go              Wails 启动、生命周期与桌面级能力
controller_*.go              Wails 绑定与传输边界
internal/                    Go 领域层、应用层与基础设施层
frontend/                    当前 Vue 3 前端
frontend-legacy/             切换前前端的冻结副本
docs/                        当前说明与设计历史
data/                        开发期运行时数据（生成）
build/bin/                   Wails 打包产物（生成）
SampleData/, testdata/       示例输入与测试夹具
```

`frontend/` 是 `wails dev`、`wails build` 和 `main.go` 资源嵌入使用的唯一活动前端。`frontend-legacy/` 不再接收功能开发；它在切换后保留一个发布周期，随后删除。

## Documentation

- `docs/PROJECT-STRUCTURE.md` — 当前仓库结构与模块边界。
- `docs/DEVELOPMENT.md` — 日常开发命令、格式与验证入口。
- `docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md` — 业务域、需求类型、平台维度与核心痛点。
- `docs/FRONTEND-REDESIGN-PLAN.md` — 前端重设计的历史方案和实施阶段。
- `docs/fulfillment-v2-refactor/` — 履约重构设计文档，覆盖业务边界、数据模型、工作流、profile 系统、非功能基础与实施治理。
- `docs/FULFILLMENT-V2-REFACTOR-PLAN.md` — 指向 `fulfillment-v2-refactor/README.md` 的兼容入口。

口径冲突时，优先级为：当前代码实现、`docs/fulfillment-v2-refactor/` 设计文档、其他说明文档。

## Desktop Boundary

- `main.go` — 嵌入 `frontend/dist`，初始化数据库单例，配置窗口与本地资源中间件，并注册 Wails 绑定。
- `app.go` — 应用生命周期、CSV/ZIP 文件选择、缩放持久化和数据库路径解析。
- `context_provider.go` — 共享 Wails context。
- `zoom_config.go` — 缩放配置读写。
- `wails.json` — 指向 `frontend/`，安装、构建和开发命令均通过 Deno task 执行。

根目录 `controller_*.go` 是应用层到 Wails 的传输边界。主要绑定按职责分组如下：

- 任务与查询：`ActionCenterController`、`ListPaginationController`。
- 需求与波次：`DemandController`、`WaveController`，相关 CSV 导入、收件箱查询和生命周期方法拆分在同名前缀文件中。
- 执行链：`ExportController`、`ShipmentController`、`ChannelSyncController`、`AdjustmentController`。
- 配置与主数据：`TemplateController`、`AllocationPolicyController`、`ProductController`、`ProfileController`、`AddressController`。
- 客户与归并：`CustomerProfileController`、`MergeController`、`MergeUndoController`。
- 桌面文件系统：`FileSystemController`。

控制器负责参数转换、依赖组装和调用应用用例；业务规则放在 `internal/app/`，不放在控制器中。

## Backend

后端由三层核心架构和根目录的 Wails 传输边界组成：

```text
internal/domain/       领域实体、枚举和仓库端口
internal/app/          用例、DTO、业务编排、投影与执行器
internal/infra/        GORM 仓库、事务适配器和持久化映射
controller_*.go        Wails 绑定与 DTO 传输边界
```

### `internal/domain/`

- `models.go` — 客户、需求、波次、履约行、供应商订单、发货、渠道同步、接入配置、历史与商品等领域实体。
- `enums.go` — 业务枚举及其稳定 wire value；也是前端生成枚举的源文件。
- `ports.go` 及专用 `*_ports.go` — 仓库接口与按写入/查询场景拆分的端口。

领域层不依赖 GORM、Wails 或前端类型。

### `internal/app/`

- `*_usecase.go` — 需求、波次、分配、履约调整、发货、接入、归并与关闭流程的应用用例。
- `dto/` — Wails 边界使用的输入、输出、分页和投影 DTO。
- `selector/` — 分配规则的选择器匹配逻辑。
- `*_service.go`、`*_query_usecase.go`、`*_projection_usecase.go` — 跨仓库编排、历史、快照、漂移检测和只读投影。
- `*_executor.go`、`executor*.go` — CSV/文档导出、渠道执行与补丁执行。
- `*_test.go` — 与用例并置的聚焦测试和集成测试。

### `internal/infra/`

- `*_repo.go` — GORM 仓库实现，包括分页查询和专用写入适配器。
- `persistence/models.go` — GORM 持久化模型。
- `persistence/mapper.go`、`persistence/enums.go` — 领域模型与持久化模型之间的转换。
- `tx_repos.go` — 事务内仓库组合。

### Supporting Packages

- `internal/db/sqlite.go` — SQLite/GORM 初始化、WAL 与外键配置、AutoMigrate、数据库单例。
- `internal/config/config.go` — 应用元数据和窗口尺寸。
- `internal/service/path_service.go` — 开发、便携、系统安装三种数据路径解析。
- `internal/service/settings_service.go` — 本地系统设置持久化。
- `internal/middleware/local_assets.go` — `/local-images/` 资源服务中间件。

## Frontend

当前前端位于 `frontend/`，使用 Vue 3、TypeScript、Vite、Pinia、vue-i18n、Naive UI 和 CSS design tokens。Deno 是唯一前端任务运行器；`package.json` 只保存依赖元数据。

```text
frontend/
  deno.json                 Deno tasks
  vite.config.ts            Vite 配置和 `@` 路径别名
  scripts/                  guardrails 与枚举生成脚本
  src/
    main.ts                 Vue 启动入口
    app/                    根组件和 hash router
    pages/                  路由页面与页面内工作流模块
    entities/               前端实体别名和 DTO 适配类型
    shared/
      api/                  Wails 桥接、连接状态、生成契约
      assets/               共享字体等静态资源
      i18n/                 locale、消息与领域 glossary
      lib/                  通用算法和工作流 composable
      model/                跨页面 Pinia store
      styles/               reset 与全局样式入口
      theme/                tokens、主题、密度、skin 加载和 Naive 适配
      ui/                   共享 UI 组件
    skins/                  静态 skin 注册表和资源包
  wailsjs/                  Wails 生成绑定（提交到仓库）
  dist/                     Vite 构建产物（忽略）
```

### Bootstrap And Routing

- `src/main.ts` 安装 Pinia、router、vue-i18n 和主题系统，再挂载应用。
- `src/app/App.vue` 组合 Naive UI 主题适配、全局反馈、断连提示、应用壳和路由出口。
- `src/app/router/index.ts` 使用 `createWebHashHistory()` 以适配 Wails；注册任务中心、波次工作区、收件箱、客户、商品、接入、设置和开发期 design lab。

### Pages And Entities

- `src/pages/home/` — 任务中心与波次行动摘要。
- `src/pages/waves/` — 波次列表、工作区 shell，以及总览、需求、分配、履约、工厂、发货和收尾 tabs。
- `src/pages/inbox/` — 需求收件箱、文件导入、手工录入和行详情。
- `src/pages/customers/` — 客户列表、统一详情、地址、履约历史与归并预览。
- `src/pages/products/` — 商品管理与批量备货到波次。
- `src/pages/integrations/` — 接入配置卡、详情和样本映射向导。
- `src/pages/settings/` — 外观、语言、操作员和本地设置。
- `src/pages/design-lab/` — 共享组件的开发期展示页。
- `src/entities/` — 从生成 DTO 和生成枚举派生的前端类型；只保留前端特有的补充结构。

### Shared Modules

- `src/shared/api/bridge.ts` — 所有运行时 Wails 调用的唯一入口；统一 DTO 构造、断连行为和薄封装。
- `src/shared/api/health.ts` — Wails bridge 连接状态。
- `src/shared/api/generated/enums.ts` — 从 `internal/domain/enums.go` 生成的 TypeScript 枚举契约。
- `src/shared/i18n/` — zh-CN/en-US 消息、locale store 和领域 glossary。
- `src/shared/lib/` — 表格排序、CSV 导出、快捷键、波次链接与工作区 context 等复用逻辑。
- `src/shared/model/` — 路由进度和操作员名单等跨页面 store。
- `src/shared/theme/`、`src/shared/styles/` — 三层 tokens、亮暗主题、密度、skin 加载、Naive UI 适配与全局 CSS。
- `src/shared/ui/` — shell、feedback、status、data-grid、filter-bar、drawer、wizard、field-mapping 等共享组件族。
- `src/skins/` — `default` 和 `dusk` 静态 skin 包；skin 只提供 token 与可选静态资源，不包含应用逻辑。

### Bridge And Generated Contracts

页面、store 和 composable 的运行时后端调用必须经过 `src/shared/api/bridge.ts`。其他模块不得直接导入 `wailsjs/go/main/*`；仅用于静态类型的 `import type { dto } from '.../wailsjs/go/models'` 可以直接引用生成模型。

`internal/domain/enums.go` 是枚举 wire value 的唯一真相源：

```powershell
cd frontend
deno task gen:enums
deno task lint:guardrails
```

生成器写入 `src/shared/api/generated/enums.ts`。guardrails 同时检查生成文件是否最新、glossary 覆盖、用户可见硬编码文案、裸状态值渲染，以及页面绕过 bridge 或共享 UI 边界的导入。

### Frontend Tasks

| Task | Purpose |
|------|---------|
| `deno task dev` | 在 `127.0.0.1:5173` 启动 Vite |
| `deno task typecheck` | 运行 `vue-tsc -b` |
| `deno task test` | 运行 Vitest |
| `deno task build` | 类型检查并生成 `dist/` |
| `deno task gen:enums` | 从 Go domain 枚举生成 TypeScript |
| `deno task lint:guardrails` | 运行 UI/import guardrails 和枚举一致性检查 |

## Runtime Data

运行时数据路径由 `internal/service/path_service.go` 解析：

1. Wails 开发模式使用工作目录下的 `data/`。
2. 可执行文件同目录存在 `.portable` 时使用 `<exe>/data/`。
3. 其他情况使用 `os.UserConfigDir()/EliGiftManager/data/`。

主要子路径为数据库 `eligiftmanager.db`、资源 `assets/`、导出 `exports/` 和临时文件 `tmp/`。

## Generated And Frozen Paths

| Path | Status |
|------|--------|
| `internal/`, `frontend/src/`, `frontend/scripts/` | 业务与工具源码 |
| `frontend/wailsjs/` | Wails 生成绑定，已提交 |
| `frontend/src/shared/api/generated/enums.ts` | Go 枚举生成文件，已提交 |
| `frontend/dist/`, `frontend/node_modules/`, `build/bin/` | 构建或依赖产物，已忽略 |
| `data/` | 运行时数据，不作为源码提交 |
| `frontend-legacy/` | 冻结的切换前前端；保留一个发布周期后删除 |
| `.cache/`, `.claude/`, `.agents/`, `.gocache/` | 本地工具缓存，已忽略 |

## Table Query Principles

1. 排序、过滤和分页的最终结果由后端查询决定。
2. 列表 DTO 保持轻量，详情按需加载。
3. 表格查询状态统一表达 keyword、filters、sort、page/pageSize 或 limit/offset。
4. 小数据页面可以使用本地方案；大数据页面使用服务端查询。
5. 后端维护排序字段白名单，不接受前端字段直接拼接 SQL。
6. 后端追加稳定尾排序键，避免主排序值相同时分页结果漂移。
