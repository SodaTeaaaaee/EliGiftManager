# CLAUDE.md

## Build & Test Commands

```bash
# Backend
go mod tidy                           # install Go deps
go test ./...                         # run all tests
go test -v -run TestX ./internal/...  # run specific test
wails dev                             # start desktop dev server
wails build                           # build packaged binary

# Frontend (Deno only — never npm/yarn/pnpm)
cd frontend && deno install           # install deps
cd frontend && deno task dev          # Vite dev server on :5173
cd frontend && deno task build        # typecheck + production build
cd frontend && deno task typecheck    # vue-tsc type checking only
cd frontend && deno task preview      # preview production build
```

## Architecture

- **Backend**: Go + Wails v2 + GORM + SQLite (`internal/`)
- **Frontend**: Vue 3 SFC + TypeScript + Vite + Naive UI + Tailwind CSS (`frontend/`)
- **Tooling**: Deno (exclusive — `package.json` exists only for dependency metadata)
- **Desktop shell**: Wails asset pipeline, native window lifecycle via `main.go`
- **Smart path resolution**: Three-tier data directory detection (dev → portable → system) via `internal/service/path_service.go`

### Directory Map

| Path                                   | Purpose                                                                             |
| -------------------------------------- | ----------------------------------------------------------------------------------- |
| `main.go`                              | Wails bootstrap, DB init, controller DI wiring                                      |
| `app.go`                               | Lifecycle hooks (startup + temp cleanup) + PickCSVFile/PickZIPFile + shared types   |
| `controller_*.go`                      | Domain-specific Wails bound methods (Member/Product/Wave/System/Template)           |
| `internal/config/`                     | App metadata, window sizing                                                         |
| `internal/db/`                         | SQLite init (WAL mode), auto-migration                                              |
| `internal/middleware/`                 | Wails AssetServer middleware for `/local-images/`                                   |
| `internal/model/`                      | DB tables (GORM), constants, enums, payload types, DynamicTemplateRules schema      |
| `internal/service/`                    | Business logic: dynamic CSV parser, import pipeline, wave reconciliation, export, image storage, path resolution |
| `frontend/src/app/`                    | App shell, layout, router                                                           |
| `frontend/src/pages/`                  | Route-level screens                                                                 |
| `frontend/src/shared/`                 | Reusable UI, types, Wails wrappers, composables                                     |
| `frontend/src/shared/composables/`     | Vue 3 composables (useContextMenu, useAdaptiveTable)                                |
| `frontend/src/shared/model/`           | Reactive singletons (settings — scrollMode, zoom persistence; theme)                |
| `frontend/src/shared/ui/`              | Shared UI components (ContextMenu.vue — floating right-click menu)                  |
| `frontend/src/shared/lib/wails/app.ts` | **Single entry point for all Wails bridge calls** (imports from 6 controller files) |
| `frontend/wailsjs/`                    | Generated Wails bindings (committed)                                                |

## Key Conventions

1. **Wails bridge boundary**: Pages and composables MUST call through `frontend/src/shared/lib/wails/app.ts`. Never import from `wailsjs` directly outside that layer. Bridge imports are split across App.js + 5 Controller.js files.
2. **Deno-only frontend**: Use `deno task` for all frontend commands. `npm`/`yarn`/`pnpm` are not used in this project.
3. **Temporary UI shell**: The current frontend is a prototype. Business logic is not finalized. Keep UI close to Naive UI stock patterns.
4. **`.cache/`**: Use `.cache/` directories for local build/test caches. Already gitignored.
5. **Generated vs authored**: `frontend/wailsjs/` is generated but committed. `frontend/dist/`, `frontend/node_modules/`, and `build/bin/` are generated and ignored.
6. **Controller pattern**: All Wails bound methods live in `controller_*.go` files (package main). Each controller gets its own generated JS binding file. New business methods should be added to the appropriate controller.
7. **DB access**: Controllers hold `db *gorm.DB` via constructor injection (wired explicitly in `main.go`). Do NOT open/close DB per request. The `database.GetDB()` singleton no longer exists.
8. **Path resolution**: Use `service.ResolveDataDir()` / `service.ResolveAssetsDir()` for all data paths. Three tiers: dev (Temp→workdir), portable (`.portable` marker), system (`UserConfigDir`).
9. **Context menu**: Use `useContextMenu` composable (`frontend/src/shared/composables/useContextMenu.ts`) — singleton with `register(key, handler)` for DOM-level right-click. Add `data-contextmenu="key"` to target elements, call `register('key', handler)` in `onMounted`. Global `contextmenu` listener in `App.vue` always calls `preventDefault()` — browser menu never appears.
10. **Adaptive paging pattern**: Table panels use a flex-column parent (with `ref` for `ResizeObserver`), table wrapper (content height, no `flex-1`), indicator div (`flex-1` with dynamic `<`/`>` arrow chars), scaled pagination, and `-12` in the `packByHeights` formula for indicator margin. Three pages (WaveImport/WaveTag/WavePreview) share this pattern.
11. **Tailwind in h() render functions**: Tailwind JIT does NOT scan Vue `h()` string literals. Use `<style>`-block CSS classes or inline styles instead of Tailwind utilities in render functions.
12. **User tag display name**: `wmNicknameMap` (computed from `waveMembers`) maps `waveMemberId → latestNickname` for user tag chip rendering.
13. **Tag color model**: `tagColors(tag)` in `WaveTagStep.vue` returns `{bg, text, accent, number, border}`. Three color roles: `bg` (platform@20%), `text` (var(--text)), `accent` (platform solid — colon + positive number). Negative tags add 2px red border + red number. Drawer tags share the same function.
14. **Adaptive table composable**: `useAdaptiveTable<T>(items, {tableParentRef,tableWrapperRef,paginationRef,indicatorRef})` in `frontend/src/shared/composables/useAdaptiveTable.ts`. Encapsulates ResizeObserver, DOM row-height measurement, packByHeights, indicator arrow chars. Returns `{visibleItems, currentPage, totalPages, scrollMode, indicatorFontSize/Left/Right, init, remeasure, teardown, ...}`. `scrollMode` (Ref<boolean>) is a global singleton from `useScrollMode()` in `settings.ts`.
15. **Scroll mode toggle**: SettingsPage radio group switches `scrollMode` global ref (persisted to localStorage). When true, table wrapper uses `overflow-y-auto flex-1 min-h-0`, pagination + indicator hidden via `v-if="!scrollMode"`.
16. **Zoom persistence**: App startup — `main.go` reads `zoom.cfg` from data dir → `windows.Options{ZoomFactor}`. App shutdown — Go `OnBeforeClose` → `WindowExecJS("window.__persistZoom()")` → JS reads `devicePixelRatio`, computes `currentDPR/baseDPR` ratio, calls `saveZoom` Go binding → writes `zoom.cfg`. `localStorage` as backup. `IsZoomControlEnabled: true` must be explicit in Windows options (Go bool zero-value is false).
17. **Font**: Noto Sans SC — local WOFF2 segments (101 files, unicode-range) served from `public/fonts/` via `index.html` `<link>`. Font stack: `'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', 'Hiragino Sans GB', sans-serif`. Base weight 500. `font-display: block`. Anti-aliasing: `-webkit-font-smoothing` is macOS-only (no effect on Windows); on Windows WebView2, text rendering is determined by Skia + ClearType, not CSS.
18. **Text selection**: `body { user-select: none }` in `main.css`. Form elements exempted.
19. **Formatting**: Deno + Prettier 3.x. Vue files: `--parser vue`. TypeScript files: `--parser typescript`. Config in `frontend/.prettierrc`. **Never run Prettier Vue parser on .ts files** — it compresses them to single lines.
20. **CSV template format**: All templates use `DynamicTemplateRules` JSON schema (`internal/model/dynamic_mapping.go`). Header-based CSVs use `sourceColumn`; headless CSVs use `columnIndex`. Extra columns are captured via `extraData.strategy: "catch_all"`. Old flat/V2 JSON formats are no longer supported.
21. **Service-layer domain logic**: Heavy business logic (reconciliation, import pipelines, export) lives in `internal/service/`. Controllers are thin Wails-binding wrappers that delegate to service functions. Service functions accept `db *gorm.DB` as first parameter.

## Code Style

- **Go**: standard `gofmt`
- **Frontend**: TypeScript + Vue 3 `<script setup lang="ts">`, 2-space indent, LF line endings
- **`.editorconfig`** and **`.gitattributes`** enforce line endings and whitespace

## Codex Integration

Codex (GPT-5.5, 1M context) is wired in as an extension — NOT a replacement for the sub-agent team.

### Codex scope — three roles only

Codex 仅在以下三个场景介入，其他所有工作（实现编码、仓库内调研、常规审查）一律使用 Claude Code 自带 sub-agent（general-purpose / Explore / Plan）：

| 场景 | 触发条件 | 方式 |
|---|---|---|
| **疑难 bug 修复** | 同一 bug 连续 2 次修复失败 | `codex:rescue` 独立诊断 |
| **Plan 审查** | `/work-plan` 产出设计草案后 | `codex:rescue` 做 devil's advocate 结构化 critique |
| **网络资料调研** | 需要查外部 API 文档、官方最佳实践、过时检查 | `codex:rescue` 做 WebSearch/WebFetch |

Codex **不参与**：实现编码、仓库内代码调研（Grep/Glob/Read）、OCP 审查、一般性代码修改。

### Core constraints

1. **Verify everything**: GPT-5.X has weak attention over long contexts. Every Codex output — analysis, suggestion, code — must be independently verified before acting on it. Do NOT trust a Codex claim just because it sounds plausible.
2. **Exhaustive prompts, zero guesswork**: Codex excels at detailed, sharply-bounded instructions. When writing prompts: spell out all requirements, constraints, file paths, and expected outputs explicitly. Leave nothing for it to infer. Ambiguity is where its attention drifts.
3. **No PR contribution**: This repo is not contributed to upstream projects. Codex is for internal quality and decision support only.

### Model selection

Codex offers two models — choose per task. Manual override always available.

| Model       | Strengths                           | Weaknesses                                  | Cost |
| ----------- | ----------------------------------- | ------------------------------------------- | ---- |
| **GPT-5.5** | Strongest reasoning, deep analysis  | Attention decays severely with long context | Full |
| **GPT-5.4** | Slightly better attention retention | Weaker reasoning than 5.5                   | Half |

**Default strategy (no override):**

| Scenario                           | Model   | Rationale                                                   |
| ---------------------------------- | ------- | ----------------------------------------------------------- |
| Rescue after 2 failures            | GPT-5.5 | Cost of wrong answer too high; needs best reasoning         |
| Devil's advocate / design critique | GPT-5.4 | Output is verified anyway; 5.4's thoroughness is sufficient |
| Refactoring audit (3–10 files)     | GPT-5.4 | Broader scope, moderate stakes — 5.4's attention wins       |
| Broad design review (multi-module) | GPT-5.4 | Context breadth > reasoning depth; 5.5 would drift          |
| Quick sanity check                 | GPT-5.4 | Low stakes, half cost                                       |

**Override:** Manual override always available. Upgrade devil's advocate to 5.5 for high-stakes irreversible decisions (architecture overhaul, data migration, breaking API changes).

### Rescue on repeated failures

Same bug/error after 2 consecutive fix attempts → STOP. Do NOT attempt a 3rd fix. Call `codex:rescue` for independent diagnosis.

### Devil's advocate on design decisions

During `/work` plan phase, after the main architecture direction is drafted:

- Call `codex:rescue` with the design summary and ask for a structured critique: what breaks, what doesn't scale, what edge cases were missed.
- Treat the critique as a checklist of risks to evaluate, NOT as a voted decision. Each point must be verified against the actual codebase.
- Weight: Codex's signal is "what to double-check", not "what to do".

### Refactoring verification

After a significant refactor (3+ files touched or architectural change):

- Call `codex:rescue` with the diff summary + the refactor's stated goal.
- Ask: "Did the refactor achieve its goal? Did it introduce any inconsistency with the surrounding codebase?"
- Cross-check every inconsistency flag against the code yourself. Codex may flag false positives due to missing context.

### Task decomposition for Codex

GPT-5.5 has strong reasoning but weak attention over long contexts. When assigning work to Codex:

- Break tasks into small, well-scoped pieces — single responsibilities with clear boundaries
- Keep each assignment short enough to complete in one pass without drifting
- Do NOT hand Codex large monolithic tasks; the attention window won't hold
- Codex is an extension of the workflow, not a substitute — main work still goes through the sub-agent team
