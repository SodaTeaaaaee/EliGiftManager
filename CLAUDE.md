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

| Path | Purpose |
|------|---------|
| `main.go` | Wails bootstrap, DB singleton init, controller binding |
| `app.go` | Lifecycle hooks (startup) + PickCSVFile/PickZIPFile + shared types/functions |
| `controller_*.go` | Domain-specific Wails bound methods (Member/Product/Wave/System/Template) |
| `internal/config/` | App metadata, window sizing |
| `internal/controller/` | *(deprecated — controllers now live at project root as package main)* |
| `internal/db/` | SQLite init (WAL mode), auto-migration, DB singleton |
| `internal/middleware/` | Wails AssetServer middleware for `/local-images/` |
| `internal/model/` | DB tables (GORM), enums, payload types |
| `internal/service/` | Business logic: CSV transformers, import pipeline, image storage, path resolution |
| `frontend/src/app/` | App shell, layout, router |
| `frontend/src/pages/` | Route-level screens |
| `frontend/src/shared/` | Reusable UI, types, Wails wrappers |
| `frontend/src/shared/lib/wails/app.ts` | **Single entry point for all Wails bridge calls** (imports from 6 controller files) |
| `frontend/wailsjs/` | Generated Wails bindings (committed) |

## Key Conventions

1. **Wails bridge boundary**: Pages and composables MUST call through `frontend/src/shared/lib/wails/app.ts`. Never import from `wailsjs` directly outside that layer. Bridge imports are split across App.js + 5 Controller.js files.
2. **Deno-only frontend**: Use `deno task` for all frontend commands. `npm`/`yarn`/`pnpm` are not used in this project.
3. **Temporary UI shell**: The current frontend is a prototype. Business logic is not finalized. Keep UI close to Naive UI stock patterns.
4. **`.cache/`**: Use `.cache/` directories for local build/test caches. Already gitignored.
5. **Generated vs authored**: `frontend/wailsjs/` is generated but committed. `frontend/dist/`, `frontend/node_modules/`, and `build/bin/` are generated and ignored.
6. **Controller pattern**: All Wails bound methods live in `controller_*.go` files (package main). Each controller gets its own generated JS binding file. New business methods should be added to the appropriate controller.
7. **DB access**: Controllers use `database.GetDB()` singleton (initialized in `main.go`). Do NOT open/close DB per request.
8. **Path resolution**: Use `service.ResolveDataDir()` / `service.ResolveAssetsDir()` for all data paths. Three tiers: dev (Temp→workdir), portable (`.portable` marker), system (`UserConfigDir`).

## Code Style

- **Go**: standard `gofmt`
- **Frontend**: TypeScript + Vue 3 `<script setup lang="ts">`, 2-space indent, LF line endings
- **`.editorconfig`** and **`.gitattributes`** enforce line endings and whitespace
