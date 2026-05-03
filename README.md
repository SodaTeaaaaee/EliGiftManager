# EliGiftManager

多平台桌面发货管理工具。从 Bilibili 等平台导入会员列表，从柔造等工厂平台导入商品数据，通过 Tag 系统做等级-礼物匹配，批量生成工厂规格的发货清单 CSV。

## 技术栈

| 层   | 技术                                                    |
| ---- | ------------------------------------------------------- |
| 后端 | Go 1.26 + Wails v2.12 + GORM + SQLite (WAL)             |
| 前端 | Vue 3.5 + TypeScript + Vite 8 + Naive UI + Tailwind CSS |
| 工具 | Deno 2.7 (前端), Go modules (后端)                      |

## 快速开始

```bash
go mod tidy
cd frontend && deno install
cd .. && wails dev          # 开发模式
wails build                  # 生产构建
deno task typecheck          # 前端类型检查
go test ./...                # 后端测试
```

## 核心流程

1. 导入会员 (Bilibili CSV) → Member 表 + WaveMember 快照 + Wave.LevelTags
2. 导入商品 (柔造 ZIP/CSV) → Product 表 + ProductImage 多图（DynamicTemplateRules 模板驱动）
3. Tag 商品 → Level Tag（按平台+等级）+ User Tag（按具体会员）
4. ReconcileWave → 按 Tag 自动计算 DispatchRecord（SSOT，幂等调和）
5. BindDefaultAddresses → 补全收件地址
6. ExportOrderCSV → 工厂规格 CSV 导出（按商品平台过滤）

## 目录结构

```
├── main.go                       # Wails bootstrap, DB 初始化, Controller DI 注入
├── app.go                        # 生命周期（含 Temp 清理） + PickCSV/ZIP + 共享类型/函数
├── controller_*.go               # 5 个领域 Controller (Member/Product/Wave/System/Template)
├── internal/
│   ├── config/                   # 应用元数据
│   ├── db/                       # SQLite 初始化 (WAL), 自动迁移
│   ├── middleware/               # AssetServer Middleware (/local-images/)
│   ├── model/                    # GORM 模型 + 常量 + DynamicTemplateRules schema + Dispatch 状态
│   └── service/                  # 业务逻辑 (Dynamic Parser, 导入流水线, 波次调和, 导出, 图片存储, 智能路径)
├── frontend/
│   ├── src/pages/                # 路由级 Vue 页面
│   ├── src/shared/lib/wails/app.ts  # Wails 桥接层 (单一入口)
│   └── wailsjs/                  # 生成的 Wails 绑定 (已提交, 按 Controller 拆分)
└── data/                         # 运行时数据 (gitignored)
    ├── eligiftmanager.db
    └── assets/
```

## 数据存储

- **数据库**: 三级智能路径 (Temp dev → `.portable` 便携 → UserConfigDir 系统)
- **图片**: SHA256 Content-Addressable, `data/assets/{hash[:2]}/{hash}.{ext}`
- **代理**: Wails Middleware `/local-images/` → `data/assets/`

## 开发约定

- Wails 桥接层收敛在 `app.ts`，导入自 6 个 Controller 绑定文件
- Deno only — 前端工具链不用 npm/yarn/pnpm
- Controller 按领域拆分，新增业务方法应加入对应 Controller
- Controller 通过 DI 持有 `*gorm.DB`（`main.go` 显式注入），不再使用全局单例
- 路径解析统一使用 `service.ResolveDataDir()` / `ResolveAssetsDir()`
- CSV 模板统一使用 `DynamicTemplateRules` JSON 格式（`internal/model/dynamic_mapping.go`），旧 flat/V2 格式不再兼容
- 领域逻辑（ReconcileWave、导入流水线、导出）在 `internal/service/`，Controller 为 thin wrapper
- 模板不自动写入 DB，用户通过「添加模板」弹窗从预设选择或自定义

## 生成与忽略

- `frontend/wailsjs/` — Wails 生成, 已提交
- `frontend/dist/`, `frontend/node_modules/`, `build/bin/` — 生成, 已忽略
- `data/` — 运行时, 已忽略
- `.cache/`, `.claude/`, `.agents/` — 工具缓存, 已忽略
