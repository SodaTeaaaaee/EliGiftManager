> **⚠️ Some content deprecated (2026-05-10 v3)**
>
> The following requirements are no longer applicable:
> - "Remove all indicator-related concepts from both composable and pages" — **deprecated**. The indicator has been restored as a shrink-0 display-only component.
> - "Delete the decorative indicator block completely" — **deprecated**.
> - "indicator is non-essential / indicator is currently a layout liability" — **deprecated**. The indicator is a useful visual cue in paginated mode.
>
> Current direction: keep string `tableMode`, keep stable pagination architecture, keep indicator (shrink-0 display-only, not participating in height calculation).

# Frontend Table Mode Fix Brief

## Objective

Stabilize the frontend "table mode" feature controlled from settings:

- `scroll` mode: show full data in a scrollable table viewport.
- `paginated` mode: show a stable, viewport-based page slice with explicit pagination.

This work must only address the settings-driven table mode system and the wave workflow tables that consume it. Do not modify the product library `grid/list` mode.

## Scope

Primary files:

- `frontend/src/shared/model/settings.ts`
- `frontend/src/shared/composables/useAdaptiveTable.ts`
- `frontend/src/pages/settings/ui/SettingsPage.vue`
- `frontend/src/pages/waves/ui/WaveImportStep.vue`
- `frontend/src/pages/waves/ui/WaveTagStep.vue`
- `frontend/src/pages/waves/ui/WavePreviewStep.vue`

Relevant layout/container context:

- `frontend/src/app/AppLayout.vue`
- `frontend/src/pages/waves/ui/DispatchTaskShell.vue`

## Non-Goals

- Do not work on `frontend/src/pages/products/ui/ProductLibraryPage.vue`.
- Do not redesign page visuals beyond what is required to fix behavior.
- Do not change backend APIs.
- Do not add virtualization unless strictly necessary; the preferred fix is a simpler and more stable pagination model.

## Current Diagnosis

### 1. Shared table mode state is boolean-based and semantically weak

Current state lives in `frontend/src/shared/model/settings.ts` as:

- `scrollMode = ref(true)`
- persisted under `eligift_scrollMode`

Problems:

- Boolean state obscures meaning and leaks conversion logic into consumers.
- Settings UI converts between string radio values and boolean manually.
- Future extension is harder.

### 2. `useAdaptiveTable` has structural flaws

Current implementation in `frontend/src/shared/composables/useAdaptiveTable.ts` mixes:

- mode selection
- row height measurement
- synthetic page packing
- indicator layout
- resize observation

Key defects:

1. Measurement temporarily renders all rows.
   - `visibleItems` returns full `items` when `needsMeasure` is true.
   - `remeasure()` sets `needsMeasure = true`, waits a tick, reads all rendered row heights, then switches back.
   - This causes visual flashing and unnecessary heavy renders on mode switch and data change.

2. Page size is derived from a brittle pixel-packing algorithm.
   - `packByHeights()` tries to fit rows exactly by measured heights.
   - This couples pagination to transient DOM layout and content wrapping.
   - Small layout changes can reshuffle page boundaries.

3. Available height is mis-modeled.
   - `pages` uses `availableH - paginationH * 2 - 12`.
   - Template structure only has one pagination block.
   - The current math creates avoidable blank space and under-fills the table.

4. Height changes force page reset.
   - In `ResizeObserver`, a container height change sets `currentPage = 1`.
   - This is hostile to users and causes mode/resize/top-content interactions to jump back to page 1.

5. Content changes that do not affect item count can leave stale pagination.
   - Consumers often only watch array length.
   - Row height can change while length stays constant.
   - The current system depends on remeasurement, but remeasurement triggers are incomplete.

6. Indicator UI consumes layout space and is not functionally necessary.
   - Each page renders an indicator area in pagination mode.
   - In templates it is often `flex-1`, so it competes with the table for height.
   - This is one direct cause of large blank regions.

### 3. Consumers are coupled to the flawed API

The following pages depend on the current composable contract:

- `WaveImportStep.vue`
- `WaveTagStep.vue`
- `WavePreviewStep.vue`

Common consumer issues:

- They pass `paginationRef` and `indicatorRef`.
- They render decorative indicator blocks.
- They watch `scrollMode` to reattach indicator observers.
- They often only remeasure when item counts change.

### 4. Container structure matters

Wave pages sit inside nested flex layouts:

- `AppLayout.vue`: `RouterView` is inside a `h-full`, flex-column container.
- `DispatchTaskShell.vue`: inner `RouterView` is also `flex-1 min-h-0`.

This means the fix should rely on proper flex/min-height behavior and real viewport containers rather than compensating with extra decorative layout blocks.

## Target Behavior

### Scroll Mode

- Render the full list for the page/table instance.
- Use a constrained table viewport with internal scrolling.
- No indicator block.
- No pagination UI for that table.
- No page state mutation while in scroll mode.

### Paginated Mode

- Render a stable page slice only.
- Use explicit `NPagination`.
- Page size is derived from viewport height and a table-specific row height hint.
- Mode switching should not flash full data to the DOM.
- Height changes should preserve the current page when still valid.
- If the current page becomes invalid after resize/data change, clamp to the last valid page.

## Required Design Direction

Do not keep the current measured-row packing system.

Instead, replace it with a simpler model:

1. Separate the mode decision from the rendering strategy.
2. Use viewport height and a configurable `rowHeightHint` to calculate `pageSize`.
3. Reserve only real fixed UI areas outside the table viewport.
4. Remove all indicator-related concepts from both composable and pages.

This tradeoff is intentional:

- slightly less "pixel-perfect" page fill
- much more stable behavior
- less layout thrash
- easier reasoning
- lower maintenance cost

## Implementation Tasks

### Task 1: Replace boolean mode state with explicit string mode

File:

- `frontend/src/shared/model/settings.ts`

Requirements:

1. Replace boolean `scrollMode` state with:
   - `tableMode = ref<'scroll' | 'paginated'>('scroll')`

2. Expose a composable with explicit semantics, for example:
   - `useTableMode()`

3. Persist using a new storage key:
   - `eligift_tableMode`

4. Add backward compatibility:
   - If `eligift_tableMode` exists, use it.
   - Else read legacy `eligift_scrollMode`.
   - Legacy `true` means `'scroll'`.
   - Legacy `false` means `'paginated'`.

5. After migration, persist only the new key.

Acceptance:

- Fresh users default to `'scroll'`.
- Existing users with old storage retain their previous mode behavior.

### Task 2: Rewrite `useAdaptiveTable` into a stable viewport-based composable

File:

- `frontend/src/shared/composables/useAdaptiveTable.ts`

Requirements:

1. Remove these concepts entirely:
   - `packByHeights`
   - `measuredHeights`
   - `needsMeasure`
   - `indicatorRef`
   - `indicatorObserver`
   - indicator text generation
   - `paginationRef` measurement logic

2. Redefine the composable around stable inputs:
   - source items
   - table mode
   - viewport element ref
   - row height hint
   - optional min page size
   - optional extra reserved height if absolutely needed

3. Core outputs should include something equivalent to:
   - `renderItems`
   - `tableMaxHeight`
   - `pageSize`
   - `currentPage`
   - `pageCount`
   - `handlePageChange`
   - `refreshLayout`

4. `tableMaxHeight` must come from the actual viewport container height.

5. In `scroll` mode:
   - `renderItems = items`
   - `pageCount = 1`
   - do not mutate `currentPage`

6. In `paginated` mode:
   - compute `pageSize = max(minPageSize, floor(viewportHeight / rowHeightHint))`
   - compute `pageCount` from items length
   - compute `renderItems` by slicing `items`

7. Resize handling:
   - observe only the viewport container
   - on resize, recompute `pageSize`
   - preserve `currentPage` if still valid
   - clamp only if out of range

8. Data changes:
   - when item count changes, clamp current page if needed
   - do not reset to page 1 unless explicitly requested by a caller-level action

9. Avoid full-data measurement passes entirely.

10. Keep the composable generic enough for all current consumers.

Recommended shape:

- a narrow, deterministic composable is preferred over a "smart" auto-measuring one

Acceptance:

- No code path renders all items just to calculate pagination.
- No decorative layout refs are required by the composable.

### Task 3: Update settings page to use explicit mode semantics

File:

- `frontend/src/pages/settings/ui/SettingsPage.vue`

Requirements:

1. Replace boolean `scrollMode` usage with explicit `tableMode`.
2. Remove manual conversion function `setScrollMode(v)`.
3. Bind `NRadioGroup` directly to `'scroll' | 'paginated'`.
4. Keep displayed labels:
   - `自适应分页`
   - `滚动模式`

Acceptance:

- Settings page reads and writes the new mode directly.
- No boolean conversion logic remains in the component.

### Task 4: Refactor `WaveImportStep.vue`

File:

- `frontend/src/pages/waves/ui/WaveImportStep.vue`

Requirements:

1. Replace current composable usage with the new API for both:
   - product table
   - member table

2. Remove:
   - `productPaginationRef`
   - `productIndicatorRef`
   - `memberPaginationRef`
   - `memberIndicatorRef`
   - indicator template blocks
   - `watch(scrollMode, ...)` indicator setup logic

3. Keep one viewport ref per table.

4. Add explicit row height hints:
   - product table and member table may use different hints
   - pick conservative values, then tune if needed

5. `NPagination` should render only in paginated mode.

6. `NDataTable :max-height` should use the composable’s `tableMaxHeight`.

7. After these actions, pagination must remain stable:
   - import products
   - import members
   - delete product from wave
   - delete member from wave
   - switch settings mode
   - resize window

8. Existing page behavior outside table mode must remain unchanged.

Acceptance:

- No blank flex filler remains under pagination mode.
- Window resize does not force page 1.

### Task 5: Refactor `WaveTagStep.vue`

File:

- `frontend/src/pages/waves/ui/WaveTagStep.vue`

Requirements:

1. Replace current composable usage with the new API.
2. Remove `paginationRef`, `indicatorRef`, indicator template blocks, and indicator watch logic.
3. Preserve selection features:
   - checkbox selection
   - plain click single selection
   - Ctrl/Cmd toggle
   - Shift range
   - Ctrl/Cmd + Shift additive range
   - page-level select all / invert

4. Because page size can change on resize, ensure selection anchor safety:
   - when page boundaries change materially, reset `lastClickedIndex`
   - do not let stale indices drive incorrect Shift range behavior

5. Do not rely on content-height remeasurement after tag changes.
   - adding/removing tags may change row height
   - pagination must still behave predictably because it is no longer based on exact measured row heights

6. After batch actions and context-menu actions, preserve page if still valid.

Acceptance:

- Switching table mode never flashes all rows to the DOM.
- Tag operations do not produce half-empty layouts or pagination jumps.
- Selection semantics remain correct within the current visible page slice.

### Task 6: Refactor `WavePreviewStep.vue`

File:

- `frontend/src/pages/waves/ui/WavePreviewStep.vue`

Requirements:

1. Replace current composable usage with the new API.
2. Remove `paginationRef`, `indicatorRef`, indicator template blocks, and indicator watch logic.
3. Keep explicit pagination only in paginated mode.
4. Handle dynamic header content safely:
   - export template rows
   - preview alert
   - missing-address status

5. When header content height changes, the table viewport height may change.
   - this should trigger recomputation of page size
   - current page should be preserved if valid
   - otherwise clamp, not reset

6. After popup edits, keep the user on the same page whenever possible:
   - set address
   - add address
   - add gift
   - remove gift
   - update quantity

7. Do not depend solely on `memberGroups.length` changes for pagination updates.

Acceptance:

- Returning from edits does not snap the list back to page 1.
- Top-area height changes no longer destabilize the table.

### Task 7: Remove indicator UI from all consumers

Files:

- `frontend/src/pages/waves/ui/WaveImportStep.vue`
- `frontend/src/pages/waves/ui/WaveTagStep.vue`
- `frontend/src/pages/waves/ui/WavePreviewStep.vue`

Requirements:

1. Delete the decorative indicator block completely.
2. Delete related font-size and content logic from the composable.
3. Ensure pagination area is `shrink-0` and does not participate in flexible height sharing.

Rationale:

- indicator is non-essential
- indicator is currently a layout liability
- the fix should reduce moving parts, not preserve them

### Task 8: Normalize caller-level page reset semantics

Requirement:

There is an important distinction between:

- user-triggered filter/search reset
- incidental layout/data refresh

Caller rules:

1. Search/filter actions may explicitly reset page to 1 if desired.
2. Resize/layout changes must not reset page to 1.
3. Data mutations that do not invalidate the current page must preserve it.
4. Only clamp when item count shrinkage invalidates the current page.

Apply this principle consistently across all three consumers.

## Suggested Composable Contract

This is not mandatory, but the implementation should converge to something close to this shape:

```ts
type TableMode = 'scroll' | 'paginated'

interface StableTableOptions {
  viewportRef: Ref<HTMLElement | null>
  rowHeightHint: number
  minPageSize?: number
}

function useAdaptiveTable<T>(
  items: Ref<T[]>,
  mode: Ref<TableMode>,
  options: StableTableOptions,
) {
  // returns:
  // tableMaxHeight
  // currentPage
  // pageSize
  // pageCount
  // renderItems
  // handlePageChange
  // refreshLayout
  // clampCurrentPage
}
```

Important:

- The contract should be easy to understand from the call site.
- Avoid hidden behavior.

## Testing and Verification

### Required automated verification

Run:

```powershell
cd frontend
deno task typecheck
```

### Manual verification matrix

#### Settings

1. Open settings.
2. Switch from scroll to paginated.
3. Navigate to wave import/tag/preview pages.
4. Confirm mode is applied immediately.
5. Reload app and confirm persistence.

#### Wave Import Step

1. In scroll mode:
   - product/member tables scroll internally
   - no pagination visible
2. In paginated mode:
   - pagination visible
   - no large blank filler area
   - page changes work
3. Resize the window:
   - remain on same page if valid
   - otherwise clamp gracefully
4. Import data and delete rows:
   - page remains stable

#### Wave Tag Step

1. In scroll mode:
   - full list visible via scroll
2. In paginated mode:
   - pagination visible
   - no indicator
   - no half-empty table area
3. Perform:
   - batch add level tag
   - batch remove level tag
   - batch add user tag
   - batch remove user tag
   - clear tags from context menu
4. Confirm:
   - table does not flash
   - current page is preserved if valid
5. Verify selection:
   - single click
   - Ctrl/Cmd click
   - Shift range
   - page select all
   - page invert

#### Wave Preview Step

1. In paginated mode:
   - pagination visible
   - top content changes do not force page reset
2. Open member popup, then:
   - set address
   - add address
   - add gift
   - remove gift
   - modify quantity
3. Close popup and confirm current page remains stable.
4. Resize window and verify graceful clamp behavior.

## Acceptance Criteria

The fix is complete only if all of the following are true:

1. Settings use explicit string-based table mode state.
2. Legacy local storage is migrated compatibly.
3. `useAdaptiveTable` no longer renders all rows for measurement.
4. Indicator UI and indicator logic are fully removed.
5. Pagination mode no longer creates large blank regions below the table.
6. Resize and top-content height changes no longer force page 1.
7. Data mutations preserve current page whenever still valid.
8. All three wave pages compile and behave consistently.
9. `deno task typecheck` passes.

## Risks and Mitigations

### Risk 1: Row height hints may underfit or overfit

Mitigation:

- choose conservative row height hints
- prefer slight underfill over overflow or instability
- tune per table, not globally

### Risk 2: Selection logic in `WaveTagStep` may assume old page structure

Mitigation:

- explicitly review all usages of `visibleItems`
- reset `lastClickedIndex` when page geometry changes
- verify range logic after mode switch and resize

### Risk 3: Header changes in `WavePreviewStep` may not propagate viewport resize

Mitigation:

- ensure the observed ref is the actual table viewport container
- verify the container’s height changes when header grows/shrinks

## Review Checklist

Before finishing, confirm:

- no `indicatorRef` remains
- no `setupIndicatorObserver()` remains
- no `packByHeights()` remains
- no `needsMeasure` remains
- no resize path sets `currentPage = 1` unconditionally
- no page relies only on item-length watches for layout correctness
- pagination blocks are `shrink-0`
- table viewport containers are `flex-1 min-h-0`

## Deliverables

1. Refactored frontend code implementing the new stable table mode behavior.
2. Passing frontend typecheck.
3. Short implementation summary noting:
   - chosen row height hints per table
   - any page-specific caveats
   - any residual known limitations
