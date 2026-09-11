# Project Structure

当前仓库按 [CONTEXT.md](../CONTEXT.md) 与 [TARGET-DOMAIN-AND-PRODUCT-MODEL.md](./TARGET-DOMAIN-AND-PRODUCT-MODEL.md) 的目标模型实现。

## Top Level

```text
main.go, zoom_config.go          Wails 启动、生命周期与桌面级能力
internal/                         领域、用例、持久化、Wails 绑定
frontend/                         Vue 3 信息架构（待处理、收件箱、波次、资料库、设置）
docs/                             现行设计
Taskfile.yml                      wails3 构建任务驱动（前端命令挂接 deno task）
data/                             开发期运行时数据（生成，不入库）
build/bin/                        Wails 打包产物（生成）
SampleData/, testdata/            示例输入与测试夹具
```

`frontend/` 是 `wails3 dev`、`wails3 build` 和 `main.go` 资源嵌入使用的唯一前端。

## Documentation

- `CONTEXT.md` — 领域术语表
- `docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md` — 工作流、实体基数、画面与操作
- `docs/CURRENT-DESIGN-DECISIONS.md` — 已接受决定摘要
- `docs/adr/` — 单条决定
- `docs/DEVELOPMENT.md` — 开发命令

口径冲突时优先：`CONTEXT.md`，然后 `docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md`，然后代码。

## Desktop Boundary

- `main.go` — 嵌入 `frontend/dist`，初始化 SQLite，注册 `WorkspaceController` 与 `FileSystemController` 为 v3 services，并承担 application.New、窗口创建等窗口生命周期与缩放钩子
- `zoom_config.go` — 缩放读写（启动时应用 `data/zoom.cfg`，关窗写回，全部 Go 侧完成）
- `internal/controller/workspace.go`、`internal/controller/api.go` — Wails 传输边界；业务规则在 `internal/app/`

文件对话框由前端 `@wailsio/runtime` 的 `Dialogs` 承担，Go 侧无对话框绑定。

## Backend

```text
internal/domain/        实体、枚举、Store 端口
internal/app/           用例（inbox、library、wave、factory、home）
internal/infra/         GORM Store 与 persistence 映射
internal/db/            SQLite 初始化与 AutoMigrate
internal/controller/    Wails 绑定
internal/config/        应用元数据
internal/middleware/    本地资源
internal/service/       数据目录与本地设置文件
```

目标命名：InputDocument、InputFact、InputFactLine、Wave、EntitlementRule、FulfillmentResult、SupplierOrderLine、ProductItem、CustomerProfile。Demand 与 FulfillmentLine 不是目标概念。

## Frontend

```text
frontend/src/app/          路由与 App 壳
frontend/src/pages/        待处理、收件箱、波次（规则/结果）、资料库（客户/商品/模板）、设置
frontend/src/entities/     与 Go DTO 对齐的类型
frontend/src/shared/       bridge、UI kit、i18n、theme
frontend/bindings/         wails3 生成的 TypeScript 绑定（已提交）
```

运行时 Wails 调用只走 `frontend/src/shared/api/bridge.ts`，其他模块不得直接 import `frontend/bindings`。
