# Development Guide

在开始修改代码之前，建议至少先看两份文档：

1. [`docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md`](./PRODUCT-DOMAIN-AND-PAIN-POINTS.md)
2. [`docs/PROJECT-STRUCTURE.md`](./PROJECT-STRUCTURE.md)

第一份解释“软件到底在解决什么问题”，第二份解释“这些业务语义现在落在代码库的哪里”。

## 1. 当前分支的开发前提

本仓库是一个已经可以承载真实业务的 Wails 桌面项目，不应再把核心领域模型视为纯草稿。
当前分支至少有以下边界是明确的：

- `Member` 是全局会员 CRM 主体。
- `WaveMember` 是波次级会员快照，`GiftLevel` 只属于这一层。
- `ProductMaster` 是全局商品主档。
- `Product` 是波次内商品快照。
- `DispatchRecord` 是最终发货明细的单一事实来源。
- `TemplateConfig` 是映射层，不是领域本体。

如果任务与模板无关，不要默认去修改模板系统。

## 2. 技术栈

- 后端：
  - Go
  - Wails
  - GORM
  - SQLite
- 前端：
  - Vue 3
  - TypeScript
  - Vite
  - Naive UI
  - Tailwind CSS
- 前端任务运行器：
  - Deno

## 3. 初始化

在仓库根目录执行：

```powershell
go mod tidy
Set-Location frontend
deno install
Set-Location ..
```

## 4. 常用命令

默认从仓库根目录执行。

- `wails dev`
  - 启动桌面开发模式，同时代理前端开发服务器。
- `cd frontend && deno task dev`
  - 只启动前端 Vite 开发服务器，监听 `127.0.0.1:5173`。
- `cd frontend && deno task typecheck`
  - 运行 `vue-tsc`。
- `cd frontend && deno task build`
  - 构建前端生产包。
- `go test ./...`
  - 运行后端测试。
- `wails build`
  - 生成桌面打包产物。

## 5. 格式化与代码风格

当前仓库约定如下：

- Go：
  - 使用官方工具 `gofmt`
  - 缩进保持 tab
- Vue / TypeScript / JavaScript / CSS / JSON（前端目录内）：
  - 遵循 `frontend/.prettierrc`
  - Vue SFC 默认 2 空格缩进、单引号、无分号
- 其他内容：
  - 优先使用对应生态的默认工具
  - 不要为了“统一”而把 Go、Vue、文档、生成文件强行混用同一格式化器

格式化原则比命令本身更重要：

- Go 代码保持 `gofmt` clean
- 前端源码保持 Prettier 风格
- 生成物不要手工美化

## 6. 目录与职责边界

### 6.1 后端

- `main.go`
  - Wails 启动入口和控制器绑定。
- `app.go`
  - 桌面级公共函数、共享 payload、文件选择等能力。
- `controller_member.go`
  - 全局会员、地址、波次成员移除等接口。
- `controller_product.go`
  - 全局商品库与商品相关接口。
- `controller_wave.go`
  - 波次创建、导入、对账、地址绑定、导出。
- `controller_system.go`
  - 仪表盘、备份恢复、系统级能力。
- `controller_template.go`
  - 模板系统接口。
  - 只有在任务明确要求模板时才应优先修改这里。

`internal/` 内部分层：

- `internal/model`
  - 权威数据结构定义。
- `internal/service`
  - 业务逻辑实现。
  - 控制器不应承载复杂业务规则。
- `internal/db`
  - SQLite 初始化、自动迁移、历史结构升级逻辑。
- `internal/config`
  - 静态配置。
- `internal/middleware`
  - 运行时资源服务等中间件。

### 6.2 前端

- `frontend/src/app`
  - 应用壳、布局、路由。
- `frontend/src/pages`
  - 路由级页面。
- `frontend/src/shared`
  - 通用 UI、共享类型、组合式逻辑和基础设施。
- `frontend/src/shared/lib/wails/app.ts`
  - 前端唯一允许直接包装 `frontend/wailsjs` 的入口。

页面或 composable 不应直接 import `frontend/wailsjs`。
如果需要新增 Wails 调用，优先补到 `frontend/src/shared/lib/wails/app.ts` 再向上暴露。

## 7. 运行时数据与路径

真实运行数据不在源码目录里硬编码，而由
[`internal/service/path_service.go`](../internal/service/path_service.go)
统一解析。

当前路径策略是三选一：

1. Wails 开发模式：
   - 使用工作目录下的 `data/`
2. 可执行文件同级存在 `.portable`：
   - 使用便携模式目录 `exe/data`
3. 其他情况：
   - 使用 `os.UserConfigDir()/EliGiftManager/data`

运行时数据主要包括：

- 数据库：
  - `data/eligiftmanager.db`
- 资源文件：
  - `data/assets/`
- 导入临时文件：
  - `data/tmp/`

## 8. 生成物与应忽略内容

常见生成物包括：

- `frontend/node_modules`
  - Deno 的 npm 兼容层生成
- `frontend/node_modules/.tmp`
  - TypeScript 构建缓存
- `frontend/package.json.md5`
  - Deno 维护文件
- `frontend/dist`
  - Vite 构建产物
- `build/bin`
  - Wails 打包结果
- `.cache`
  - 本地缓存

`frontend/wailsjs` 是一个例外：

- 它也是生成物
- 但当前仓库会提交它，因为前端源码直接消费这层桥接

其他生成输出一般不应提交。

## 9. 测试与验证要求

当前没有全仓覆盖率门槛，但以下约束应视为默认要求：

- 后端逻辑改动：
  - 至少补或更新聚焦的回归测试
  - 执行 `go test ./...`
- 前端改动：
  - 至少执行 `cd frontend && deno task typecheck`
- 涉及导入、导出、状态机、迁移的修改：
  - 优先补 service 级测试，而不是只依赖手点 UI

如果某次变更没有执行这些验证，应在提交说明里明确写出原因。

## 10. 模板系统的维护边界

模板系统是项目的一部分，但它不是默认的改动入口。

除非任务明确要求模板相关工作，否则不要顺手改动下面这些区域：

- `controller_template.go`
- `frontend/src/pages/templates`
- `internal/service/csv_transformer.go`
- `internal/service/*_csv_transformer.go`
- `internal/service/dispatch_import_processor.go`

更直接地说：

- 会员问题，先看 `Member` / `WaveMember`
- 商品问题，先看 `ProductMaster` / `Product`
- 发货问题，先看 `DispatchRecord`
- 波次状态问题，先看波次相关 service
- 只有输入输出格式映射问题，才优先去看模板层

## 11. 开发时的常见判断原则

- 不要把波次快照层和全局主档层混为一谈。
- 不要把导出文件格式问题误当作核心业务问题。
- 不要让控制器承载复杂业务规则。
- 不要绕过 `path_service` 自己拼运行时目录。
- 不要在页面里直接散落 `wailsjs` 调用。
- 不要把 TODO 文档、旧分支思路或临时注释当作当前产品真相。

需要确认当前业务语义时，先回到
[`docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md`](./PRODUCT-DOMAIN-AND-PAIN-POINTS.md)。
