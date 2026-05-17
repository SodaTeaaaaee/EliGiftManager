# EliGiftManager

Desktop gift fulfillment management application built with Wails v2.

## Tech Stack

| Layer    | Technology                                              |
| -------- | ------------------------------------------------------- |
| Backend  | Go + Wails v2 + GORM + SQLite (WAL mode)                |
| Frontend | Vue 3 SFC + TypeScript + Vite + Naive UI + Tailwind CSS |
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

### Frontend — entity-based

```
frontend/src/app/                    App shell, layout, router
frontend/src/pages/                  Route-level screens
frontend/src/entities/               TypeScript entity types (mirrors Go DTOs)
frontend/src/shared/composables/     useUndoRedo, useContextMenu, useAdaptiveTable
frontend/src/shared/model/           Reactive singletons (settings, theme)
frontend/src/shared/ui/              Shared UI components
frontend/src/shared/lib/wails/app.ts Single entry point for all Wails bridge calls
frontend/wailsjs/                    Generated Wails bindings (committed)
```

All pages and composables call through `frontend/src/shared/lib/wails/app.ts`. Direct imports from `wailsjs/` outside that layer are not allowed.

### Data directory — three-tier resolution

Path resolution via `internal/service/path_service.go`:

1. **Dev** — temp directory adjacent to workdir
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

## Core Workflow

1. **Demand Intake** — Import demand documents with profile binding
2. **Wave Creation** — Group demands, generate participants
3. **Fulfillment Generation** — Dual-path: demand-driven mapping + policy-driven allocation
4. **Supplier Export** — Grouped by execution boundary (profile + template)
5. **Shipment Tracking** — Manual creation + bulk import with quantity safety
6. **Channel Sync** — Profile-driven closure with carrier mapping enforcement

## Key Design Principles

- Workspace history with undo/redo (wave scope) — `useUndoRedo` composable
- Basis drift detection with review requirement signals
- Bound profile behavior for active waves
- Import failure mode selection (reject-all / skip-invalid)
- DTO convention: Go DTOs are the authoritative source for field names; frontend `entities/*.ts` mirrors them exactly
- Enum values in `domain/enums.go` and `entities/*.ts` must be identical strings — manual sync, no code generation

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
cd frontend && deno task preview      # preview production build
```

## Generated vs. Authored

| Path                                          | Status              |
| --------------------------------------------- | ------------------- |
| `frontend/wailsjs/`                           | Generated, committed|
| `frontend/dist/`, `frontend/node_modules/`    | Generated, ignored  |
| `build/bin/`                                  | Generated, ignored  |
| `.cache/`, `.claude/`, `.agents/`             | Tool caches, ignored|

## Documentation

Detailed V2 design documentation is in `docs/fulfillment-v2-refactor/`.
