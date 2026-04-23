# Project Structure

This repository keeps the Wails entrypoint at the root, pushes backend code into `internal`, and keeps the frontend inside `frontend/`. The important caveat is that the current repository structure is more stable than the product behavior itself: business rules are still evolving, and the current UI layout is only a transitional shell.

## Root

- `main.go`: Wails bootstrap and window configuration
- `app.go`: Wails lifecycle hooks and methods exposed to the frontend
- `wails.json`: desktop build and dev-server wiring
- `docs`: repository conventions and workflow notes
- `build`: packaging assets for Windows and macOS

## Backend

- `internal/config`: static application metadata and default window sizing
- `internal/db`: SQLite initialization, directory creation, and auto-migration
- `internal/model`: database tables, dispatch statuses, validation payloads, and template type constants
- `internal/service`: CSV transformers, address logic, member import support, and batch validation services

The backend is already moving toward a real data model, even though the business workflow around those models is still not final.

## Frontend

- `frontend/package.json`: Vue, Vite, Tailwind, and Naive UI dependency metadata
- `frontend/deno.json`: Deno task runner entrypoint
- `frontend/postcss.config.js` / `frontend/tailwind.config.js`: Tailwind build pipeline
- `frontend/vite.config.ts`: Vite configuration and aliases
- `frontend/src/app`: app shell, Naive UI based layout, and router
- `frontend/src/pages`: route-level pages such as dashboard, orders, members, products, templates, and settings
- `frontend/src/shared`: reusable UI, shared types, and Wails integration wrappers
- `frontend/src/shared/lib/wails/app.ts`: the only place where application code should talk to generated Wails bindings
- `frontend/wailsjs`: generated Wails bridge files imported by the wrapper layer

The current frontend routes intentionally separate `templates` and `settings`, but the visual design and exact interaction model are expected to change substantially. For now the UI stays close to standard Naive UI layout and data-display components so later product rewrites can replace content without first removing a large custom design layer.

## Generated vs Authored

- Authored application code belongs under `frontend/src` and `internal`.
- Generated bridge code stays under `frontend/wailsjs`.
- Vite production output stays under `frontend/dist` and is ignored.
- Local caches stay under `.cache` and are ignored.
- Packaging metadata under `build/windows` and `build/darwin` is committed because it changes desktop output.
- Compiled binaries under `build/bin` are ignored.

## Build Assets

- `build/windows`: Windows icon, manifest, installer assets
- `build/darwin`: macOS plist metadata
- `build/appicon.png`: shared app icon source
