# EliGiftManager

EliGiftManager is a Wails desktop app for creator merchandise fulfillment. It turns external facts into mapped fulfillment responsibilities, exports supplier production orders, imports shipment results, and writes tracking data back to the source platforms.

The current product model is documented in [CONTEXT.md](./CONTEXT.md) and [docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md](./docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md). Older Demand/Fulfillment V2 language is not a compatibility target.

## Product model

The smallest workflow is input -> mapping -> output.

Input enters the Inbox as input documents and input fact lines. Operators assign accepted facts into a Wave. A Wave resolves membership facts, retail orders, operational grants, product alignment, and addresses into source-level fulfillment results.

The Wave workbench is a large editable responsibility list. Users can sort it or group it by work state, customer, product, or source. Grouping changes the view only. The leaf fulfillment results keep their source facts and quantities.

Supplier order export creates supplier order lines and internal tracking IDs. Factory shipment files map back through those tracking IDs. Source platforms receive writeback items, including multiple parcels for one source order when the platform supports it.

## Core domain terms

| Term | Meaning |
|------|---------|
| InputDocument | Raw file, API response, or manual batch submitted to the system. |
| InputFact / InputFactLine | Parsed business fact. Membership identity, retail order, and operational grant are different fact types. |
| Wave | A bounded fulfillment work scope. A membership identity fact must be resolved inside one Wave. |
| ProductItem | Internal physical product fact, anchored to one responsible factory SKU. |
| ProductAlias | External platform product ID, name, or spec mapped to a ProductItem. |
| Entitlement rule | Product-centered membership benefit rule made from membership identity deltas and per-person deltas. |
| FulfillmentResult | Source-level responsibility to deliver a product and quantity to a recipient. |
| SupplierOrderLine | Factory execution line. It is not a fulfillment result. |
| Internal tracking ID | Stable ID generated per supplier order line for supplier export and shipment import correlation. |
| ChannelWritebackItem | Source-platform update record, usually carrying shipment and parcel data. |
| TemplateConfig | Versioned mapping between external document fields and internal semantics, in either input or output direction. |

## Architecture

```text
internal/domain/       domain entities, enums, repository ports
internal/app/          use cases, DTOs, orchestration, projections, executors
internal/infra/        GORM repos, migrations, persistence mapping
internal/controller/   Wails bindings
frontend/src/app/      Vue bootstrap and hash router
frontend/src/pages/    route-level screens and workflow modules
frontend/src/entities/ frontend types derived from Go DTOs
frontend/src/shared/   API bridge, UI, composables, theme, i18n
```

Runtime Wails calls go through `frontend/src/shared/api/bridge.ts`. Type-only imports from `frontend/wailsjs/go/models` are allowed.

## Development

```bash
go mod tidy
go test ./...
wails dev
wails build

cd frontend && deno task dev
cd frontend && deno task typecheck
cd frontend && deno task test
cd frontend && deno task build
cd frontend && deno task lint:guardrails
cd frontend && deno task gen:enums
```

Deno is the frontend task runner. Do not use npm, yarn, or pnpm for project tasks.

## Generated and runtime paths

| Path | Status |
|------|--------|
| `frontend/wailsjs/` | Generated Wails bindings, committed. |
| `frontend/src/shared/api/generated/enums.ts` | Generated from Go domain enums, committed. |
| `frontend/dist/`, `frontend/node_modules/`, `build/bin/` | Generated output, ignored. |
| `data/` | Runtime data, ignored. |
| `.cache/`, `.claude/`, `.agents/` | Local tool caches, ignored. |

## Documentation

- [CONTEXT.md](./CONTEXT.md) is the domain glossary.
- [docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md](./docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md) is the product and data-model target.
- [docs/CURRENT-DESIGN-DECISIONS.md](./docs/CURRENT-DESIGN-DECISIONS.md) is the current decision summary.
- [docs/PROJECT-STRUCTURE.md](./docs/PROJECT-STRUCTURE.md) describes source layout and ownership.
- [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) describes commands, validation, and implementation guardrails.
