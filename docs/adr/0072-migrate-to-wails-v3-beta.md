# 迁移到 Wails v3 并跟随最新 beta

应用从 Wails v2.14 整体迁移到 Wails v3，首个落地版本为 v3.0.0-beta.20。动机是工程卫生：v2 处于维护模式（官方文档主站已导向 v3）、v3 绑定系统直接生成 TypeScript 类型、services 模式取代上下文传递 hack。v3 的多窗口、托盘、原生菜单等能力当前产品用不到，不为它们做任何提前设计。

版本策略是不钉死版本、跟随最新 beta。每次升级是一次显式事件：`wails3` CLI 与 `go.mod` 里的库版本同步升级、重新生成绑定、跑一遍既定冒烟清单（导入、本地图片、对话框、导出与写回、便携模式、缩放、`wails3 build` 产物）；本机保留一个 last-known-good 的 exe 作为回退。stable 发布后自然收敛，升级节奏届时再定。

前端与 Wails 的缝保持单一但变薄：`frontend/src/shared/api/bridge.ts` 仍是全前端唯一允许 import 绑定模块（v3 生成于 `frontend/bindings/`）的文件，内容收敛为三件事——re-export 生成的调用、runtime 可用性守卫与 health 状态、少量参数整形。`entities/models.ts` 不再手写镜像 Go DTO，改为生成模型的类型门面（re-export 加前端专有类型），guardrails 的 no-restricted-import 规则同步指向新路径。被放弃的两条路：无墙（pages 直接 import 生成绑定，生成物路径与 Go 模块路径焊进每个页面）与厚墙（人工同步的手写类型镜像层）。

迁移顺带采用的原生能力与清理：

- 文件对话框改走前端 `@wailsio/runtime` 的 `Dialogs`，删除 Go 侧四个 `Pick*File` 绑定。
- 窗口缩放做完整：启动时应用 `zoom.cfg` 的值，关闭窗口时把当前缩放写回，全部在 Go 侧完成；删除 `WindowExecJS(__persistZoom)` 调用与无人使用的 `SaveZoom` 绑定。
- 本地图片服务改挂 `AssetOptions.Middleware`，`/local-images/` 前缀拦截语义与目录穿越防护不变。
- dev 模式探测改用 `production` 构建标签，取代对 v2 环境变量 `devserver`/`frontenddevserverurl` 的嗅探；`.portable` 与数据目录三级判定语义不变。
- 控制器注册为 v3 services，删除 `SetAppContext` 包级上下文 hack。

事件系统本次不启用。何时为长耗时导入推送进度是独立的产品决策，届时另行记录。
