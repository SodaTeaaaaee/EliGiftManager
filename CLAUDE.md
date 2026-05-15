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
| `main.go`                              | Wails bootstrap, DB singleton init, controller binding                              |
| `app.go`                               | Lifecycle hooks (startup) + PickCSVFile/PickZIPFile + shared types/functions        |
| `controller_*.go`                      | Domain-specific Wails bound methods (Demand/Wave/Export/Shipment/ChannelSync/Adjustment/Product/Profile/Template/AllocationPolicy) |
| `internal/config/`                     | App metadata, window sizing                                                         |
| `internal/db/`                         | SQLite init (WAL mode), auto-migration, DB singleton                                |
| `internal/domain/`                     | Pure business structs, repository interfaces, enums                                 |
| `internal/app/`                        | Use cases, DTO definitions, business orchestration                                  |
| `internal/app/dto/`                    | Data transfer objects (JSON-tagged, camelCase)                                      |
| `internal/infra/`                      | Repository implementations (GORM), one file per aggregate                           |
| `internal/infra/persistence/`          | GORM models, enum mapping, domain↔persistence mappers                              |
| `internal/middleware/`                 | Wails AssetServer middleware for `/local-images/`                                   |
| `internal/service/`                    | Path resolution only (`path_service.go`)                                            |
| `frontend/src/app/`                    | App shell, layout, router                                                           |
| `frontend/src/pages/`                  | Route-level screens                                                                 |
| `frontend/src/entities/`              | TypeScript entity type definitions (mirrors Go DTOs)                                |
| `frontend/src/shared/`                 | Reusable UI, types, Wails wrappers, composables                                     |
| `frontend/src/shared/composables/`     | Vue 3 composables (useUndoRedo, useContextMenu, useAdaptiveTable)                   |
| `frontend/src/shared/model/`           | Reactive singletons (settings — scrollMode, zoom persistence; theme)                |
| `frontend/src/shared/ui/`              | Shared UI components (ContextMenu.vue — floating right-click menu)                  |
| `frontend/src/shared/lib/wails/app.ts` | **Single entry point for all Wails bridge calls** (imports from generated bindings) |
| `frontend/wailsjs/`                    | Generated Wails bindings (committed)                                                |

## Key Conventions

1. **Wails bridge boundary**: Pages and composables MUST call through `frontend/src/shared/lib/wails/app.ts`. Never import from `wailsjs` directly outside that layer. Bridge imports use generated TypeScript bindings from `frontend/wailsjs/go/main/`.
2. **Deno-only frontend**: Use `deno task` for all frontend commands. `npm`/`yarn`/`pnpm` are not used in this project.
3. **V2 architecture**: Backend uses 4-layer architecture: `domain` (pure structs + interfaces) → `app` (use cases + DTOs) → `infra` (GORM repos) → `controller` (Wails bindings). Frontend uses entities layer mirroring Go DTOs.
4. **`.cache/`**: Use `.cache/` directories for local build/test caches. Already gitignored.
5. **Generated vs authored**: `frontend/wailsjs/` is generated but committed. `frontend/dist/`, `frontend/node_modules/`, and `build/bin/` are generated and ignored.
6. **Controller pattern**: All Wails bound methods live in `controller_*.go` files (package main). Each controller is self-contained (constructs its own repos/use cases from `database.GetDB()`). New controllers require `wails generate module` to produce JS/TS bindings.
7. **DB access**: Controllers use `database.GetDB()` singleton (initialized in `main.go`). Do NOT open/close DB per request.
8. **Path resolution**: Use `service.ResolveDataDir()` / `service.ResolveAssetsDir()` for all data paths. Three tiers: dev (Temp→workdir), portable (`.portable` marker), system (`UserConfigDir`).
9. **DTO convention**: Go DTOs use camelCase `json:"fieldName"` tags. Frontend entity types mirror these exactly. Go DTO is the authoritative source for field names.
10. **Enum alignment**: Go `domain/enums.go` and frontend `entities/*.ts` enum values must be identical strings. No code generation — manual sync required.
11. **Undo/Redo**: `useUndoRedo` composable handles Ctrl+Z/Ctrl+Shift+Z/Ctrl+Y with focus guard (skips text inputs). After success, `refreshKey` increments to force child route remount.
12. **Tailwind in h() render functions**: Tailwind JIT does NOT scan Vue `h()` string literals. Use `<style>`-block CSS classes or inline styles instead of Tailwind utilities in render functions.

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
