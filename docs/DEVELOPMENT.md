# Development Guide

> 当前代码的开发指南。
> 业务设计见 [`docs/fulfillment-v2-refactor/README.md`](./fulfillment-v2-refactor/README.md)。
> 代码结构见 [`docs/PROJECT-STRUCTURE.md`](./PROJECT-STRUCTURE.md)。

## 1. 技术栈

- **后端**：Go + Wails v2 + GORM + SQLite (WAL mode)
- **前端**：Vue 3 + TypeScript + Vite + Pinia + vue-i18n + Naive UI + token/skin CSS
- **前端工具链**：Deno（唯一允许的包管理器，禁止 npm/yarn/pnpm）
- **桌面**：Wails 原生窗口生命周期

## 2. 初始化

```powershell
go mod tidy
cd frontend && deno install && cd ..
```

## 3. 常用命令

默认从仓库根目录执行。

| 命令 | 用途 |
|------|------|
| `wails dev` | 桌面开发模式（含前端 dev server） |
| `cd frontend && deno task dev` | 仅前端 Vite dev server (`127.0.0.1:5173`) |
| `cd frontend && deno task typecheck` | vue-tsc 类型检查 |
| `cd frontend && deno task test` | Vitest 单元测试 |
| `cd frontend && deno task build` | 前端生产构建 |
| `cd frontend && deno task gen:enums` | 从 Go domain 枚举生成 TypeScript 契约 |
| `cd frontend && deno task lint:guardrails` | UI/import guardrails 与生成枚举一致性检查 |
| `go test ./...` | 后端测试 |
| `wails build` | 桌面打包 |

## 4. 代码风格

- Go：`gofmt` clean，tab 缩进，domain 命名用业务语言（`CustomerProfile`、`FulfillmentLine` 等）
- Vue/TS/CSS/JSON：2 空格缩进，TypeScript 使用单引号且不写分号
- 生成物不要手工美化

## 5. 架构分层

### 后端三层

```
internal/domain/       领域实体 + 仓库接口 + 枚举 + 策略
internal/app/          用例 + DTO + 业务编排 + 投影查询
internal/infra/        GORM 仓库实现 + 迁移 + 外部 API 客户端
internal/controller/   Wails 绑定（controller_*.go）
```

控制器在 `internal/controller/`（package `controller`），每个控制器构建自己的 repo 和 use case。

### 前端

```
frontend/src/app/          根组件与 hash router
frontend/src/pages/        路由页面与页面内工作流模块
frontend/src/entities/     生成 DTO 的别名、适配类型与前端特有结构
frontend/src/shared/api/   Wails 桥接、连接状态与生成契约
frontend/src/shared/i18n/  locale、双语消息与领域 glossary
frontend/src/shared/lib/   通用算法与工作流 composable
frontend/src/shared/model/ 跨页面 Pinia store
frontend/src/shared/theme/ tokens、主题、密度、skin 与 Naive UI 适配
frontend/src/shared/ui/    共享 shell、feedback、grid、status 与表单组件
frontend/src/skins/        静态 skin 包
frontend/scripts/          guardrails 与枚举生成器
```

所有运行时 Wails 调用必须通过 `frontend/src/shared/api/bridge.ts`。其他模块不得直接导入 `wailsjs/go/controller/*` 或 `wailsjs/go/main/*`；仅用于类型的 `wailsjs/go/models` import 可以保留。

`frontend-legacy/` 是 2026-07-13 切换前前端的冻结副本，只保留一个发布周期，期间不再增加功能，随后删除。

## 6. 运行时数据路径

由 `internal/service/path_service.go` 统一解析：

1. **开发模式** — 工作目录下 `data/`
2. **便携模式** — `.portable` 标记文件存在时，`exe/data`
3. **系统模式** — `os.UserConfigDir()/EliGiftManager/data`

使用 `service.ResolveDataDir()` / `service.ResolveAssetsDir()` 获取路径。

## 7. 测试与验证

- 后端改动：补聚焦的回归测试，执行 `go test ./...`
- 前端改动：执行 `cd frontend && deno task typecheck`、`deno task test` 和 `deno task lint:guardrails`
- 导入/导出/状态机/迁移相关：优先补 service 级测试

## 8. 生成物

| 路径 | 状态 |
|------|------|
| `frontend/wailsjs/` | 生成桥接（已提交） |
| `frontend/src/shared/api/generated/enums.ts` | Go domain 枚举生成文件（已提交） |
| `frontend/dist/`、`build/bin/` | 构建产物（已忽略） |
| `frontend/node_modules/` | Deno npm 兼容层（已忽略） |
| `frontend-legacy/` | 冻结副本（保留一个发布周期后删除） |

## 9. 模板与接入维护边界

模板与接入契约跨越以下当前模块，改动时应一并核对，不要在无关任务中顺带修改：

- `internal/controller/controller_template.go`
- `internal/app/template_*.go`
- `frontend/src/pages/integrations/`
- `frontend/src/shared/api/bridge.ts`
- `frontend/wailsjs/`

## 10. 开发判断原则

- 领域实体用当前业务语言命名（`CustomerProfile`、`FulfillmentLine`），不要使用旧术语（`Member`、`DispatchRecord` 等）
- 业务逻辑在 `internal/app/` 用例层，不要堆在控制器
- 不要绕过 `path_service` 自己拼运行时目录
- 不要在页面里直接散落 `wailsjs` 调用
- 不要在 `frontend-legacy/` 增加功能
- 不要把 TODO 文档或旧分支思路当作当前产品真相
- 问题在删库从零开始后仍然存在，视为真实问题；仅在旧库升级中出现的，默认不作为高优先级

需要确认业务语义时，先看 [`docs/fulfillment-v2-refactor/`](./fulfillment-v2-refactor/)。
