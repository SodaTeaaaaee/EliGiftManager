# Development Guide

> 当前代码的开发指南。
> 业务设计见 [`docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md`](./TARGET-DOMAIN-AND-PRODUCT-MODEL.md)。
> 代码结构见 [`docs/PROJECT-STRUCTURE.md`](./PROJECT-STRUCTURE.md)。

## 1. 技术栈

- **后端**：Go + Wails v3（beta，跟随最新，见 [ADR 0072](./adr/0072-migrate-to-wails-v3-beta.md)）+ GORM + SQLite (WAL mode)
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
| `wails3 dev` | 桌面开发模式（含前端 dev server） |
| `cd frontend && deno task dev` | 仅前端 Vite dev server (`127.0.0.1:5173`) |
| `cd frontend && deno task typecheck` | vue-tsc 类型检查 |
| `cd frontend && deno task test` | Vitest 单元测试 |
| `cd frontend && deno task build` | 前端生产构建 |
| `cd frontend && deno task gen:enums` | 从 Go domain 枚举生成 TypeScript 契约 |
| `cd frontend && deno task lint:guardrails` | UI/import guardrails 与生成枚举一致性检查 |
| `go test ./...` | 后端测试 |
| `wails3 build` | 桌面打包 |
| `wails3 task common:generate:bindings` | 变更绑定的服务方法后重新生成前端绑定（`frontend/bindings/`，已提交）。等价全参数形式为 `wails3 generate bindings -clean=true -ts -i`；不带参数的裸命令会删光已提交的 `.ts` 并改产出 `.js`，不要运行 |

版本号升级需同步四处：`internal/config/config.go`、`build/config.yml`、`build/windows/info.json`、`build/windows/wails.exe.manifest`。

## 4. 代码风格

- Go：`gofmt` clean，tab 缩进，domain 命名用业务语言（`CustomerProfile`、`FulfillmentResult`、`SupplierOrderLine` 等）
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

所有运行时 Wails 调用必须通过 `frontend/src/shared/api/bridge.ts`。其他模块不得直接导入 `frontend/bindings`；类型经 `frontend/src/entities/models.ts` 获取。bridge 在 v3 下是薄墙：re-export 生成调用、runtime 可用性守卫与少量参数整形；`entities/models.ts` 是生成模型的类型门面（re-export 加前端专有类型）。

## 6. 运行时数据路径

由 `internal/service/path_service.go` 统一解析：

1. **开发模式** — 工作目录下 `data/`
2. **便携模式** — `.portable` 标记文件存在时，`exe/data`
3. **系统模式** — `os.UserConfigDir()/EliGiftManager/data`

使用 `service.ResolveDataDir()` / `service.ResolveAssetsDir()` 获取路径。

注意：便携模式（`.portable`）的验证必须用 `wails3 build` 的 production 产物——非 production 标签的二进制一律按开发模式落工作目录 `data/`，`.portable` 对其不生效。

## 7. 测试与验证

- 后端改动：补聚焦的回归测试，执行 `go test ./...`
- 前端改动：执行 `cd frontend && deno task typecheck`、`deno task test` 和 `deno task lint:guardrails`
- 导入/导出/状态机/迁移相关：优先补 service 级测试

## 8. 生成物

| 路径 | 状态 |
|------|------|
| `frontend/bindings/` | wails3 生成的前端绑定（已提交） |
| `frontend/src/shared/api/generated/enums.ts` | Go domain 枚举生成文件（已提交） |
| `frontend/dist/`、`build/bin/` | 构建产物（已忽略） |
| `frontend/node_modules/` | Deno npm 兼容层（已忽略） |

## 9. 模板与资料库维护边界

模板配置在资料库模板区，绑定平台、文档类型与方向。相关代码：

- `internal/app/library.go`
- `frontend/src/pages/library/templates/`
- `frontend/src/shared/api/bridge.ts`
- `frontend/bindings/`（wails3 生成的 `WorkspaceController` 绑定）

## 10. 开发判断原则

- 领域实体用当前业务语言命名（`CustomerProfile`、`FulfillmentResult`、`SupplierOrderLine`），不要使用旧术语（`Demand`、`FulfillmentLine` 等）
- 业务逻辑在 `internal/app/` 用例层，不要堆在控制器
- 不要绕过 `path_service` 自己拼运行时目录
- 不要在页面里直接 import `frontend/bindings`，运行时调用一律经 `bridge.ts`
- 不要把 TODO 文档或旧分支思路当作当前产品真相
- 问题在删库从零开始后仍然存在，视为真实问题；仅在旧库升级中出现的，默认不作为高优先级

需要确认业务语义时，先看 [`CONTEXT.md`](../CONTEXT.md) 和 [`docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md`](./TARGET-DOMAIN-AND-PRODUCT-MODEL.md)。
