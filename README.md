# EliGiftManager

Desktop gift fulfillment management application built with Wails v2.

## Tech Stack

| Layer    | Technology                                              |
| -------- | ------------------------------------------------------- |
| Backend  | Go + Wails v2 + GORM + SQLite (WAL mode)                |
| Frontend | Vue 3 SFC + TypeScript + Vite + Pinia + vue-i18n + Naive UI + token/skin CSS |
| Tooling  | Deno (exclusive frontend toolchain)                     |
| Desktop  | Wails native window lifecycle via `main.go`             |

## Architecture

### Backend — 4-layer

```
internal/domain/        Pure business structs, repository interfaces, enums
internal/app/           Use cases, DTOs, business orchestration
internal/infra/         Repository implementations (GORM), one file per aggregate
controller_*.go         Wails-bound methods (one file per domain, package main)
```

Each controller is self-contained: it constructs its own repos and use cases from the `database.GetDB()` singleton. Adding a new controller requires `wails generate module` to produce JS/TS bindings.

### Frontend — feature-sliced

```
frontend/src/app/                 App root and hash router
frontend/src/pages/               Route screens and page-local workflows
frontend/src/entities/            DTO aliases and frontend-specific entity types
frontend/src/shared/api/          Wails bridge, health state, generated contracts
frontend/src/shared/i18n/         zh-CN/en-US messages and domain glossary
frontend/src/shared/lib/          Reusable algorithms and workflow composables
frontend/src/shared/model/        Cross-page Pinia stores
frontend/src/shared/theme/        Tokens, theme/density state, skins, Naive adapter
frontend/src/shared/ui/           Shared shell, feedback, grid, status, and form UI
frontend/src/skins/               Static skin packages
frontend/scripts/                 Guardrails and Go-to-TypeScript enum generation
frontend/wailsjs/                 Generated Wails bindings (committed)
```

All runtime Wails calls go through `frontend/src/shared/api/bridge.ts`. Direct controller imports from `wailsjs/` outside that layer are not allowed; type-only imports from `wailsjs/go/models` are permitted.

`frontend-legacy/` is the frozen pre-cutover frontend. It is retained for one release cycle after the 2026-07-13 cutover and then removed.

### Data directory — three-tier resolution

Path resolution via `internal/service/path_service.go`:

1. **Dev** — `data/` under the working directory
2. **Portable** — `.portable` marker file present next to the binary
3. **System** — `UserConfigDir` (OS default)

Use `service.ResolveDataDir()` / `service.ResolveAssetsDir()` for all data paths.

## Domain Controllers

| Controller                  | Responsibility                                              |
| --------------------------- | ----------------------------------------------------------- |
| `DemandController`          | Demand document intake and routing                          |
| `WaveController`            | Wave lifecycle, participants, overview                      |
| `ExportController`          | Supplier order export with execution grouping               |
| `ShipmentController`        | Shipment creation and bulk import                           |
| `ChannelSyncController`     | Channel sync planning and execution                         |
| `AdjustmentController`      | Fulfillment adjustments and replay                          |
| `ProductController`         | Product catalog management                                  |
| `ProfileController`         | Integration profile configuration                           |
| `TemplateController`        | Document template and binding management                    |
| `AllocationPolicyController`| Policy-driven allocation rules                              |
| `AddressController`         | Customer address management                                 |
| `CustomerProfileController` | Customer profile CRUD                                       |
| `MergeController`           | Customer profile merge suggestions                          |

## Core Workflow

1. **Demand Intake** — Import demand documents with profile binding
2. **Wave Creation** — Group demands, generate participants
3. **Fulfillment Generation** — Dual-path: demand-driven mapping + policy-driven allocation
4. **Supplier Export** — Grouped by execution boundary (profile + template)
5. **Shipment Tracking** — Manual creation + bulk import with quantity safety
6. **Channel Sync** — Profile-driven closure with carrier mapping enforcement

## Key Design Principles

- Workspace history with undo/redo (wave scope) — `pages/waves/workspace/useWaveUndoRedo.ts`
- Basis drift detection with review requirement signals
- Bound profile behavior for active waves
- Import failure mode selection (reject-all / skip-invalid)
- DTO convention: generated Wails models are authoritative; `frontend/src/entities/` adds aliases or frontend-only shapes
- Enum convention: `internal/domain/enums.go` is authoritative; `deno task gen:enums` updates `shared/api/generated/enums.ts`

## Development

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
cd frontend && deno task lint:guardrails # UI/import guardrails + enum consistency
cd frontend && deno task gen:enums    # regenerate TS enums from Go enums
cd frontend && deno task preview      # preview production build
```

## Generated vs. Authored

| Path                                          | Status              |
| --------------------------------------------- | ------------------- |
| `frontend/wailsjs/`                           | Generated, committed|
| `frontend/src/shared/api/generated/enums.ts`  | Generated, committed|
| `frontend/dist/`, `frontend/node_modules/`    | Generated, ignored  |
| `build/bin/`                                  | Generated, ignored  |
| `frontend-legacy/`                            | Frozen for one release cycle |
| `.cache/`, `.claude/`, `.agents/`             | Tool caches, ignored|

## Documentation

- [`docs/PROJECT-STRUCTURE.md`](docs/PROJECT-STRUCTURE.md) — Code structure, layering, architecture principles
- [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) — Dev commands, code style, testing
- [`docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md`](docs/PRODUCT-DOMAIN-AND-PAIN-POINTS.md) — Business domain, demand types, platform model, pain points
- [`docs/fulfillment-v2-refactor/`](docs/fulfillment-v2-refactor/) — Fulfillment redesign docs (boundaries, data model, workflows, profile system, non-functional foundations)
