# Project Structure

> 注：本文件当前主要描述 **greenfield 重建前的旧代码结构**。
> 当 `main` 上的旧业务代码被清理后，这份文档应被视为历史参考，
> 不再作为新版本代码结构的权威说明。

在修改任何业务逻辑之前，先读
[`docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md`](./PRODUCT-DOMAIN-AND-PAIN-POINTS.md)。
那份文档是当前分支的业务语义说明；本文件只解释代码和目录如何承载这些语义。

## Docs

- `docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md`
  - 当前最重要的业务域说明。
  - 明确软件目的、核心痛点、术语、数据模型边界和常见误解。
- `docs/DEVELOPMENT.md`
  - 日常开发命令、格式化规则、测试入口和维护约束。
- `docs/BACKEND-TABLE-ARCHITECTURE-TODO.md`
  - 面向未来大数据表格架构的设计备忘录。
- `docs/fulfillment-v2-refactor/README.md`
  - V2 履约重构主计划。
  - 覆盖业务边界、数据模型、工作流、profile、非功能基础（大数据与 i18n）以及实施治理。
  - 这份计划是当前重构工作的主线之一，但仍不是当前运行时代码的权威来源。

`BACKEND-TABLE-ARCHITECTURE-TODO.md` 现在应与 V2 重构中的非功能基础文档一起理解，而不是单独看成一份独立性能备忘录。

如果多个文档口径冲突，在阅读旧代码时，优先级应理解为：

1. `PRODUCT-DOMAIN-AND-PAIN-POINTS`
2. 当前代码中的 model / service 实现
3. `PROJECT-STRUCTURE`
4. 其他 TODO、备忘录和历史说明

## Root

- `main.go`
  - Wails 应用入口、窗口配置和启动流程。
- `app.go`
  - 通用 payload 结构、辅助函数，以及部分桌面级公共逻辑。
- `controller_member.go`
  - 全局会员 CRM、地址、波次成员移除等接口。
- `controller_product.go`
  - 全局商品库和波次商品标签相关接口。
- `controller_wave.go`
  - 波次创建、导入、对账、导出、地址绑定和分配变更。
- `controller_system.go`
  - 仪表盘、数据库备份/恢复、运行时系统能力。
- `controller_template.go`
  - 模板系统相关接口。
  - 除非任务明确要求模板改动，否则不要顺手修改这里。
- `wails.json`
  - Wails 构建和开发期前端代理配置。
- `README.md`
  - 对外展示的仓库入口说明。

## Backend

后端核心代码在 `internal/`。

### `internal/config`

- 静态应用元数据。
- 默认窗口尺寸与桌面配置。

### `internal/db`

- SQLite 初始化、分阶段迁移、默认连接单例管理。
- 这里承载了较多“历史版本升级到当前结构”的兼容逻辑。
- 当前分支最关键的迁移之一是：
  - `ProductMaster` 全局商品库
  - `Product` 波次快照
  的拆分与去重升级。

### `internal/model`

这里是最接近业务域的结构定义。

- `Member`
  - 全局唯一会员实体。
  - 通过 `(platform, platform_uid)` 唯一标识。
- `MemberNickname`
  - 昵称历史。
- `MemberAddress`
  - 地址历史与默认地址状态。
- `ProductMaster`
  - 全局商品库。
  - 唯一键是 `(platform, factory_sku)`。
- `Product`
  - 波次内商品快照。
  - 唯一键是 `(wave_id, platform, factory_sku)`。
- `ProductImage`
  - 波次快照级图片。
- `ProductMasterImage`
  - 全局商品级图片。
- `WaveMember`
  - 波次内会员快照。
  - `GiftLevel`、昵称、平台 UID 等波次相关身份都在这里，不在全局 `Member` 上持久化。
- `ProductTag`
  - 分配规则引擎。
  - `level` 标签按档位分配。
  - `user` 标签按某个 `WaveMember` 做覆盖。
- `DispatchRecord`
  - 当前系统的单一事实来源（SSOT）。
  - 任何“这个波次最终该给谁发什么”的问题，应该落到这里回答。
- `TemplateConfig`
  - 导入/导出/解析模板配置。
  - 它是映射层，不是核心业务对象本身。

### `internal/service`

主要承载控制器不该直接写的业务逻辑。

- CSV / ZIP 导入解析。
- 地址绑定、地址校验、测试地址处理。
- 导出处理和导出预检。
- 图片存储和资源去重。
- 波次状态重算。
- 路径解析：
  - `data/`
  - `data/assets/`
  - `data/tmp/`

## Frontend

前端在 `frontend/`，技术栈是 Vue 3 + Vite + Deno task runner + Naive UI。

### `frontend/src/app`

- 应用壳、全局布局、路由注册。

### `frontend/src/pages`

当前主要页面目录：

- `dashboard`
  - 仪表盘。
- `members`
  - 全局会员 CRM。
- `products`
  - 全局商品库页面。
  - 这里现在应理解为 `ProductMaster` 视图，不是波次快照视图。
- `waves`
  - 波次工作流页面。
  - 这里处理的是 `WaveMember`、`Product` 快照、`DispatchRecord`。
- `templates`
  - 模板系统页面。
  - 当前维护中应谨慎修改。
- `settings`
  - 应用设置页。

### `frontend/src/shared`

- 公共 UI。
- 共享类型。
- Wails 包装层。
- 表格和交互通用能力。

### `frontend/src/shared/lib/wails/app.ts`

- 前端唯一允许直接对接 `frontend/wailsjs` 的包装层。
- 页面和组合式函数不应直接 import `wailsjs`。

### `frontend/wailsjs`

- Wails 生成的桥接文件。
- 这些文件是生成物，但当前仓库会提交它们，因为前端代码直接消费这层桥接。

## Runtime Data

运行时真实数据不在源码里，而在 `data/` 体系下。

- SQLite 数据库：
  - `data/eligiftmanager.db`
- 资源文件：
  - `data/assets/`
- 导入 ZIP 解压和临时中间文件：
  - `data/tmp/`

路径解析逻辑在
[`internal/service/path_service.go`](../internal/service/path_service.go)。

## Generated vs Authored

- 业务源码：
  - `internal/`
  - `frontend/src/`
- 生成桥接：
  - `frontend/wailsjs/`
- 构建产物：
  - `frontend/dist/`
  - `build/bin/`
- 本地缓存：
  - `.cache/`

## Current Business Boundaries in Code

为了避免后续 agent 在错误层级上改逻辑，当前分支的几个边界必须明确：

- 全局会员库问题，优先看 `Member` / `MemberAddress` / `MemberController`。
- 全局商品库问题，优先看 `ProductMaster` / `ProductMasterImage` / `ProductController`。
- 波次内商品问题，优先看 `Product` 快照 / `WaveController`。
- 发货真相问题，优先看 `DispatchRecord` 和相关 service。
- 导入导出格式问题，才去看 `TemplateConfig` 和模板相关代码。
- 如果任务不是模板专项，不要为了“顺手统一”去碰模板系统。
