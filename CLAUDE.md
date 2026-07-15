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
cd frontend && deno task test         # Vitest unit tests
cd frontend && deno task lint:guardrails # UI/import guardrails + generated enum check
cd frontend && deno task gen:enums    # regenerate TS enums from Go domain enums
cd frontend && deno task preview      # preview production build
```

## Architecture

- **Backend**: Go + Wails v2 + GORM + SQLite (`internal/`)
- **Frontend**: Vue 3 SFC + TypeScript + Vite + Pinia + vue-i18n + Naive UI + token/skin CSS (`frontend/`)
- **Tooling**: Deno (exclusive — `package.json` exists only for dependency metadata)
- **Desktop shell**: Wails asset pipeline, native window lifecycle via `main.go`
- **Smart path resolution**: Three-tier data directory detection (dev → portable → system) via `internal/service/path_service.go`
- **Legacy frontend**: `frontend-legacy/` is frozen, retained for one release cycle after cutover, then deleted

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
| `internal/service/`                    | Runtime path resolution and local settings persistence                              |
| `frontend/src/main.ts`                 | Vue bootstrap; installs Pinia, router, i18n, theme, and global styles               |
| `frontend/src/app/`                    | App shell composition and hash-router registration                                 |
| `frontend/src/pages/`                  | Route-level screens and page-local workflow modules                                |
| `frontend/src/entities/`               | TypeScript entity aliases/adapters over generated DTOs                              |
| `frontend/src/shared/api/bridge.ts`     | **Single entry point for all runtime Wails bridge calls**                           |
| `frontend/src/shared/api/generated/`   | TypeScript enums generated from `internal/domain/enums.go`                          |
| `frontend/src/shared/i18n/`             | vue-i18n setup, zh-CN/en-US messages, and domain glossary                           |
| `frontend/src/shared/lib/`              | Framework-agnostic helpers and reusable workflow composables                        |
| `frontend/src/shared/model/`            | Cross-cutting Pinia stores (route progress, operator roster)                        |
| `frontend/src/shared/styles/`           | Global reset and stylesheet entry point                                             |
| `frontend/src/shared/theme/`            | Design tokens, density/theme state, skin loader, and Naive UI adapter               |
| `frontend/src/shared/ui/`               | Token-driven shell, feedback, grid, status, filter, drawer, and wizard components   |
| `frontend/src/skins/`                   | Registered static skin packages (`default`, `dusk`)                                 |
| `frontend/scripts/`                     | Deno guardrail scanner and Go-to-TypeScript enum generator                          |
| `frontend/wailsjs/`                     | Generated Wails bindings (committed)                                                |
| `frontend-legacy/`                      | Frozen pre-cutover frontend; retain one release cycle, then delete                   |

## Key Conventions

1. **Wails bridge boundary**: All runtime Wails calls MUST go through `frontend/src/shared/api/bridge.ts`. Do not import controller functions from `wailsjs` elsewhere. Type-only imports from `frontend/wailsjs/go/models` are allowed for DTO typing.
2. **Deno-only frontend**: Use `deno task` for all frontend commands. `npm`/`yarn`/`pnpm` are not used in this project.
3. **V2 architecture**: Backend uses 4 layers: `domain` (pure structs + interfaces) → `app` (use cases + DTOs) → `infra` (GORM repos) → `controller` (Wails bindings). Frontend uses `app` / `pages` / `entities` / `shared` boundaries, with page-local workflow modules kept under their owning page.
4. **`.cache/`**: Use `.cache/` directories for local build/test caches. Already gitignored.
5. **Generated vs authored**: `frontend/wailsjs/` and `frontend/src/shared/api/generated/enums.ts` are generated but committed. `frontend/dist/`, `frontend/node_modules/`, and `build/bin/` are generated and ignored.
6. **Controller pattern**: All Wails bound methods live in `controller_*.go` files (package main). Each controller is self-contained (constructs its own repos/use cases from `database.GetDB()`). New controllers require `wails generate module` to produce JS/TS bindings.
7. **DB access**: Controllers use `database.GetDB()` singleton (initialized in `main.go`). Do NOT open/close DB per request.
8. **Path resolution**: Use `service.ResolveDataDir()` / `service.ResolveAssetsDir()` for all data paths. Three tiers: dev (`<workdir>/data`), portable (`<exe>/data` with a `.portable` marker), system (`UserConfigDir/EliGiftManager/data`).
9. **DTO convention**: Go DTOs use camelCase `json:"fieldName"` tags and are authoritative. The generated Wails models carry those shapes into TypeScript; `frontend/src/entities/` adds aliases or frontend-only structures.
10. **Enum alignment**: `internal/domain/enums.go` is the source of truth. Run `deno task gen:enums` after changing it; `deno task lint:guardrails` checks generated output and glossary coverage.
11. **Undo/Redo**: Wave undo/redo lives in `frontend/src/pages/waves/workspace/useWaveUndoRedo.ts`, handles Ctrl+Z/Ctrl+Shift+Z/Ctrl+Y with an editable-target focus guard, and refreshes workspace data after success.
12. **Design system**: The new frontend does not use Tailwind. Shared components consume semantic CSS tokens from `frontend/src/shared/theme/`; skins may override token values but must not introduce application code.
13. **UI guardrails**: User-facing strings go through vue-i18n, domain states render through shared status/glossary helpers, and pages use shared layout/feedback components. Run `deno task lint:guardrails` after UI changes.
14. **History scope**: Only `wave` scope is implemented. Profile/template history is explicitly deferred (see `docs/fulfillment-v2-refactor/03-data-model/06a-history-scope-deferral.md`).
15. **Legacy freeze**: Do not add features to `frontend-legacy/`. It exists only as a frozen rollback reference for one release cycle after the 2026-07-13 cutover and is then removed.

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
