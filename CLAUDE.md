# CLAUDE.md

Use [AGENTS.md](./AGENTS.md) as the primary repository instruction file. This file exists for tools that look for `CLAUDE.md`; keep it short and current.

## Current model

The rewrite uses the model in [CONTEXT.md](./CONTEXT.md) and [docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md](./docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md).

Do not revive old Demand/Fulfillment V2 language. External inputs are facts. Accepted facts enter a Wave. Mappings create source-level fulfillment results. Supplier order lines execute those results and receive internal tracking IDs for factory shipment correlation.

The user-facing workflow is:

1. Inbox parses and triages input facts.
2. Wave owns membership fact resolution, product alignment, entitlement deltas, retail orders, operational grants, and supplier execution.
3. Library owns stable product facts, platform integrations, template configuration, carrier mapping, and semantic alignment.
4. Settings owns local app preferences and duplicate import window defaults.

Membership benefit editing is product-centered. Attach membership identity deltas and per-person deltas to products. The supporter/member view is a grouping of the fulfillment result list, not a separate rule editor.

Template configuration must support testing with real or sample data, but a user may create a template before testing it.

## Commands

```bash
go mod tidy
go test ./...
wails3 dev
wails3 build
wails3 task common:generate:bindings

cd frontend && deno task dev
cd frontend && deno task typecheck
cd frontend && deno task test
cd frontend && deno task build
cd frontend && deno task lint:guardrails
cd frontend && deno task gen:enums
```

Use Deno for frontend work. Do not use npm, yarn, or pnpm. Run `wails3 task common:generate:bindings` after changing bound service methods (the task in `build/Taskfile.yml`; full form `wails3 generate bindings -clean=true -ts -i`). The bare `wails3 generate bindings` without flags deletes the committed `.ts` bindings and emits `.js`.

## Boundaries

- Keep root Go files limited to desktop bootstrap.
- Put business rules in `internal/app/`, not controllers.
- Keep domain entities and repository ports in `internal/domain/`.
- Keep GORM and persistence mapping in `internal/infra/`.
- Route runtime Wails calls through `frontend/src/shared/api/bridge.ts`; no other module may import `frontend/bindings` directly, and types come from the `@/entities` facade.
- Keep generated bindings committed, but do not hand-edit generated files except as part of a deliberate regeneration step.

## Documentation

Project docs should describe the current intended design only. Delete stale plan notes instead of preserving historical alternatives in the repo. Git keeps history.
