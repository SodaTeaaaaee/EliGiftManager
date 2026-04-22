# Project Structure

This repository keeps the Wails entrypoint at the root, while pushing reusable backend code into `internal` and frontend code into a feature-oriented tree.

## Backend

- `main.go`: Wails bootstrap and window configuration
- `app.go`: Wails lifecycle hooks and bound methods exposed to the frontend
- `internal/config`: static application metadata and window sizing

## Frontend

- `frontend/package.json`: standard Vue + Vite dependency and script metadata
- `frontend/deno.json`: Deno npm compatibility and task runner entrypoint
- `frontend/vite.config.ts`: Vite configuration and path aliases
- `frontend/src/app`: app shell and router setup
- `frontend/src/pages`: route-level screens
- `frontend/src/features`: feature-level components and model constants
- `frontend/src/shared`: shared UI primitives, types, and Wails wrappers
- `frontend/wailsjs`: generated Wails bridge files that are imported by the frontend
- `frontend/node_modules`: Deno-managed npm compatibility directory, ignored
- `frontend/dist`: Vite production build output, ignored

## Generated vs Authored

- Authored application code should live under `frontend/src` and `internal`.
- Generated bridge code should stay under `frontend/wailsjs`.
- Packaging metadata under `build/windows` and `build/darwin` is committed because it affects desktop output.
- Compiled binaries under `build/bin` are ignored.

## Build Assets

- `build/windows`: Windows icon, manifest, installer assets
- `build/darwin`: macOS plist metadata
- `build/appicon.png`: shared app icon source
