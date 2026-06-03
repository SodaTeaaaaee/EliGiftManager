# Development Guide

> 当前代码的开发指南。
> 业务设计见 [`docs/fulfillment-v2-refactor/README.md`](./fulfillment-v2-refactor/README.md)。
> 代码结构见 [`docs/PROJECT-STRUCTURE.md`](./PROJECT-STRUCTURE.md)。

## 1. 技术栈

- **后端**：Go + Wails v2 + GORM + SQLite (WAL mode)
- **前端**：Vue 3 + TypeScript + Vite + Naive UI + Tailwind CSS
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
| `cd frontend && deno task build` | 前端生产构建 |
| `go test ./...` | 后端测试 |
| `wails build` | 桌面打包 |

## 4. 代码风格

- Go：`gofmt` clean，tab 缩进，domain 命名用业务语言（`CustomerProfile`、`FulfillmentLine` 等）
- Vue/TS/CSS/JSON：遵循 `frontend/.prettierrc`，2 空格缩进，单引号，无分号
- 生成物不要手工美化

## 5. 架构分层

### 后端三层

```
internal/domain/       领域实体 + 仓库接口 + 枚举 + 策略
internal/app/          用例 + DTO + 业务编排 + 投影查询
internal/infra/        GORM 仓库实现 + 迁移 + 外部 API 客户端
```

控制器在根目录 `controller_*.go`，每个控制器构建自己的 repo 和 use case。

### 前端

```
frontend/src/app/          应用壳、布局、路由
frontend/src/pages/        路由级页面
frontend/src/entities/     TS 实体类型（镜像 Go DTO）
frontend/src/shared/       通用 UI、composables、Wails 包装
```

所有 Wails 调用必须通过 `frontend/src/shared/lib/wails/app.ts`。

## 6. 运行时数据路径

由 `internal/service/path_service.go` 统一解析：

1. **开发模式** — 工作目录下 `data/`
2. **便携模式** — `.portable` 标记文件存在时，`exe/data`
3. **系统模式** — `os.UserConfigDir()/EliGiftManager/data`

使用 `service.ResolveDataDir()` / `service.ResolveAssetsDir()` 获取路径。

## 7. 测试与验证

- 后端改动：补聚焦的回归测试，执行 `go test ./...`
- 前端改动：执行 `cd frontend && deno task typecheck`
- 导入/导出/状态机/迁移相关：优先补 service 级测试

## 8. 生成物

| 路径 | 状态 |
|------|------|
| `frontend/wailsjs/` | 生成桥接（已提交） |
| `frontend/dist/`、`build/bin/` | 构建产物（已忽略） |
| `frontend/node_modules/` | Deno npm 兼容层（已忽略） |

## 9. 模板系统维护边界

除非任务明确要求模板相关工作，否则不要顺手修改：

- `controller_template.go`
- `frontend/src/pages/templates`
- `internal/service/*_csv_transformer.go`

## 10. 开发判断原则

- 领域实体用当前业务语言命名（`CustomerProfile`、`FulfillmentLine`），不要使用旧术语（`Member`、`DispatchRecord` 等）
- 业务逻辑在 `internal/app/` 用例层，不要堆在控制器
- 不要绕过 `path_service` 自己拼运行时目录
- 不要在页面里直接散落 `wailsjs` 调用
- 不要把 TODO 文档或旧分支思路当作当前产品真相
- 问题在删库从零开始后仍然存在，视为真实问题；仅在旧库升级中出现的，默认不作为高优先级

需要确认业务语义时，先看 [`docs/fulfillment-v2-refactor/`](./fulfillment-v2-refactor/)。
