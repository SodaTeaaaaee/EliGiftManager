/**
 * useFulfillmentGrid — the fulfillment-lines grid's data/state composable
 * (plan 3.3.2/3.3.3, P3 grid core). Owns:
 *
 * - Server-paginated + server-filtered row fetch via
 *   `listWaveFulfillmentRowsFiltered` (bridge.ts), driven by
 *   `useUrlFilters(FULFILLMENT_GRID_FILTER_SCHEMA)` (URL-synced filter
 *   state) + local `page`/`pageSize` refs (NOT part of the URL-synced
 *   filter schema — `useUrlFilters` only owns the 6 enum dims + keyword).
 * - The tracking-number CLIENT-SIDE JOIN: `listShipmentsByWave(waveId)` is
 *   called once per distinct `waveId` (cached — never re-fetched just
 *   because the grid page/filter changed) and reduced into a
 *   `fulfillmentLineId -> {trackingNo, shipmentTrackingStatus}` map, merged
 *   onto the current page's ~50 rows. No Go change; the join is bounded to
 *   whatever page is currently loaded, per the data-layer contract's
 *   sensei-approved decision.
 * - Selection state (`selectedKeys`, a plain `fulfillmentLineId[]`) — NOT
 *   auto-cleared on page/filter change (this allows a cross-page batch
 *   selection workflow); `selectedRows` only ever reflects rows present on
 *   the CURRENTLY LOADED page (batch actions operate on what's visibly
 *   selected right now, never on hidden/other-page rows sharing an id that
 *   happens to still be in `selectedKeys`).
 * - `refresh()` / `mutationDone()` — re-fetch the current page IN PLACE
 *   (no ref-identity replacement, no route remount) and, for
 *   `mutationDone()`, also call `useWaveWorkspaceContext().refresh()` so
 *   the workspace shell (overview/six-bucket/drift/undo-boundary) stays in
 *   sync. Callers (`BatchActionBar.vue` on `'done'`, `RowDetailDrawer.vue`
 *   on `'changed'`) must call `mutationDone()`, never `refresh()` alone,
 *   after any batch-adjust / address-bind mutation.
 *
 * Filter/selection state is intentionally never reset by `refresh()`/
 * `mutationDone()` — only an actual filter-schema state change (or a
 * `waveId` change) resets `page` back to 1 and re-fetches.
 */
import { computed, onBeforeMount, onBeforeUnmount, onMounted, ref, watch, type ComputedRef, type Ref } from 'vue'
import { listShipmentsByWave, listWaveFulfillmentRowsFiltered } from '@/shared/api/bridge'
import { useUrlFilters, type UseUrlFiltersApi } from '@/shared/ui/filter-bar'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { registerRefreshTarget } from '@/shared/lib/view-hotkeys'
import type { WaveFulfillmentRow } from '@/entities/fulfillment'
import { FULFILLMENT_GRID_FILTER_SCHEMA, FULFILLMENT_GRID_PRESET_SNAPSHOTS, type FulfillmentGridPresetId } from './filter-schema'

/** The grid row view: the backend DTO plus the client-joined tracking-number fields. */
export type FulfillmentGridRow = WaveFulfillmentRow & {
  /** From the matching `ShipmentLineDTO`'s parent `ShipmentDTO.trackingNo` (current wave, current page only). `undefined` when no shipment covers this line yet. */
  trackingNo?: string
  /** The matching shipment's `status` (raw enum — NOT rendered directly; a future status column must route this through a glossary dimension if one is added for it). */
  shipmentTrackingStatus?: string
}

export interface UseFulfillmentGridOptions {
  /** Pre-applies this preset's snapshot on mount — ONLY when no filter is already active (preserves deep-linked URLs). Used by the readiness route to start on `'blocked'`. */
  initialPreset?: FulfillmentGridPresetId
}

export interface UseFulfillmentGridApi {
  /** Pass straight to `<FilterBar :filters>` / `<SavedViews :filters>`. */
  filters: UseUrlFiltersApi<typeof FULFILLMENT_GRID_FILTER_SCHEMA>
  /** The wave this grid is scoped to (from `useWaveWorkspaceContext()`). */
  waveId: ComputedRef<number>
  /** The current page's rows, already merged with `trackingNo`/`shipmentTrackingStatus`. */
  rows: Ref<FulfillmentGridRow[]>
  /** True during the initial load and every subsequent `fetchPage()`. */
  loading: Ref<boolean>
  /** True after `listWaveFulfillmentRowsFiltered` throws — `rows` is `[]` in this state. */
  error: Ref<boolean>
  /** 1-based current page (server pagination — feeds `DataGridPagination`'s `server.page`). */
  page: Ref<number>
  pageSize: Ref<number>
  /** `PaginationResult.totalCount` — feeds `server.total`. */
  totalCount: Ref<number>
  totalPages: Ref<number>
  /** Selected `fulfillmentLineId`s. Bind as `DataGrid`'s `selectedKeys` (v-model). */
  selectedKeys: Ref<number[]>
  /** `rows` filtered down to the currently-selected keys — feed `BatchActionBar`'s `selectedRows` prop. */
  selectedRows: ComputedRef<FulfillmentGridRow[]>
  /** Re-fetch the current page with the current filter state. Also called internally on filter/page/waveId change. */
  fetchPage(): Promise<void>
  /** Alias for `fetchPage()` — re-fetch in place, no state reset. Use after a mutation when the workspace shell does NOT need refreshing (rare — prefer `mutationDone()`). */
  refresh(): Promise<void>
  /** `refresh()` (this grid's current page) + `useWaveWorkspaceContext().refresh()` (the workspace shell). Call this after every batch-adjust / address-bind mutation. */
  mutationDone(): Promise<void>
  /** Wire to `DataGridPagination`'s `server.onChange`. */
  onPageChange(nextPage: number, nextPageSize: number): void
}

const DEFAULT_PAGE_SIZE = 50

export interface ShipmentJoinEntry {
  trackingNo?: string
  shipmentTrackingStatus?: string
}

/** Minimal shipment fields used to rank competing joins for one fulfillment line. */
export interface ShipmentJoinSource {
  id: number
  status: string
  trackingNo?: string
  createdAt?: string
  lines?: Array<{ fulfillmentLineId: number }> | null
}

function hasTrackingNo(trackingNo: string | undefined): boolean {
  return (trackingNo ?? '').trim() !== ''
}

/**
 * Lexicographic join preference: non-voided, then non-empty trackingNo,
 * then later createdAt (ISO string), then higher id. Array order is ignored.
 */
export function pickPreferredShipmentJoin(
  current: ShipmentJoinSource,
  candidate: ShipmentJoinSource,
): ShipmentJoinSource {
  const currentActive = current.status !== 'voided' ? 1 : 0
  const candidateActive = candidate.status !== 'voided' ? 1 : 0
  if (candidateActive !== currentActive) {
    return candidateActive > currentActive ? candidate : current
  }
  const currentTracked = hasTrackingNo(current.trackingNo) ? 1 : 0
  const candidateTracked = hasTrackingNo(candidate.trackingNo) ? 1 : 0
  if (candidateTracked !== currentTracked) {
    return candidateTracked > currentTracked ? candidate : current
  }
  const currentCreated = current.createdAt ?? ''
  const candidateCreated = candidate.createdAt ?? ''
  if (candidateCreated !== currentCreated) {
    return candidateCreated > currentCreated ? candidate : current
  }
  return candidate.id > current.id ? candidate : current
}

export function buildShipmentTrackingMap(
  shipments: readonly ShipmentJoinSource[],
): Map<number, ShipmentJoinEntry> {
  const preferred = new Map<number, ShipmentJoinSource>()
  for (const shipment of shipments) {
    for (const line of shipment.lines ?? []) {
      const existing = preferred.get(line.fulfillmentLineId)
      preferred.set(
        line.fulfillmentLineId,
        existing == null ? shipment : pickPreferredShipmentJoin(existing, shipment),
      )
    }
  }
  const next = new Map<number, ShipmentJoinEntry>()
  for (const [fulfillmentLineId, shipment] of preferred) {
    next.set(fulfillmentLineId, {
      trackingNo: shipment.trackingNo,
      shipmentTrackingStatus: shipment.status,
    })
  }
  return next
}

export function useFulfillmentGrid(options: UseFulfillmentGridOptions = {}): UseFulfillmentGridApi {
  const ctx = useWaveWorkspaceContext()
  const filters = useUrlFilters(FULFILLMENT_GRID_FILTER_SCHEMA)

  // Apply the initial preset ONCE, synchronously, before any watcher/fetch
  // is wired up — only when the URL didn't already bring in filter state
  // (a deep link always wins over the route's default preset).
  if (options.initialPreset && !filters.isActive.value) {
    filters.applySnapshot(FULFILLMENT_GRID_PRESET_SNAPSHOTS[options.initialPreset])
  }

  const rows = ref<FulfillmentGridRow[]>([]) as Ref<FulfillmentGridRow[]>
  const loading = ref(true)
  const error = ref(false)
  const page = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const totalCount = ref(0)
  const totalPages = ref(0)
  const selectedKeys = ref<number[]>([]) as Ref<number[]>

  // Per-wave shipment-join cache. Keyed by waveId so a cross-wave deep link
  // (workspace shell stays mounted, only the route param changes) doesn't
  // serve a stale map from the previous wave.
  let shipmentCacheWaveId: number | null = null
  let shipmentCache = new Map<number, ShipmentJoinEntry>()

  async function loadShipmentMap(waveId: number): Promise<Map<number, ShipmentJoinEntry>> {
    if (shipmentCacheWaveId === waveId) return shipmentCache
    const shipments = await listShipmentsByWave(waveId) // soft-fail bridge call -> [] on no runtime
    const next = buildShipmentTrackingMap(shipments)
    shipmentCacheWaveId = waveId
    shipmentCache = next
    return next
  }

  function buildFilterInput(waveId: number) {
    return {
      waveId,
      allocationStates: filters.state.allocationState,
      addressStates: filters.state.addressState,
      supplierStates: filters.state.supplierState,
      channelSyncStates: filters.state.channelSyncState,
      reviewRequirements: filters.state.reviewRequirement,
      driftStatuses: filters.state.driftStatus,
      keyword: filters.state.keyword,
      pagination: { page: page.value, pageSize: pageSize.value, sortBy: '', sortDesc: false },
    }
  }

  async function fetchPage(): Promise<void> {
    const waveId = ctx.waveId.value
    loading.value = true
    try {
      const [result, shipmentMap] = await Promise.all([
        listWaveFulfillmentRowsFiltered(buildFilterInput(waveId)),
        loadShipmentMap(waveId),
      ])
      rows.value = result.items.map((item) => {
        const join = shipmentMap.get(item.fulfillmentLineId)
        return { ...item, trackingNo: join?.trackingNo, shipmentTrackingStatus: join?.shipmentTrackingStatus }
      })
      page.value = result.pagination.page
      pageSize.value = result.pagination.pageSize
      totalCount.value = result.pagination.totalCount
      totalPages.value = result.pagination.totalPages
      error.value = false
    } catch {
      // Hard-fail bridge call (per bridge.ts contract) — mirror
      // useWaveWorkspace.ts's loadSnapshot pattern: never throw into the
      // template, surface via `error` instead.
      rows.value = []
      error.value = true
    } finally {
      loading.value = false
    }
  }

  async function refresh(): Promise<void> {
    await fetchPage()
  }

  async function mutationDone(): Promise<void> {
    await Promise.all([refresh(), ctx.refresh()])
  }

  function onPageChange(nextPage: number, nextPageSize: number): void {
    page.value = nextPage
    pageSize.value = nextPageSize
    void fetchPage()
  }

  // Any filter-schema field change resets to page 1 and re-fetches. Wired
  // AFTER the initial preset application above, so that synchronous
  // `applySnapshot` call does not itself trigger a redundant extra fetch
  // (the single `onMounted` fetch below already reflects it).
  watch(
    () => FULFILLMENT_GRID_FILTER_SCHEMA.map((field) => filters.state[field.key]),
    () => {
      page.value = 1
      void fetchPage()
    },
    { deep: true },
  )

  // Cross-wave deep link (workspace shell stays mounted, only `route.params.id`
  // changes) — reset to page 1 and re-fetch under the new wave.
  watch(ctx.waveId, () => {
    page.value = 1
    void fetchPage()
  })

  onMounted(() => {
    void fetchPage()
  })

  let unregisterRefresh: (() => void) | undefined
  onBeforeMount(() => {
    unregisterRefresh = registerRefreshTarget(mutationDone)
  })
  onBeforeUnmount(() => unregisterRefresh?.())

  const selectedRows = computed<FulfillmentGridRow[]>(() =>
    rows.value.filter((row) => selectedKeys.value.includes(row.fulfillmentLineId)),
  )

  return {
    filters,
    waveId: ctx.waveId,
    rows,
    loading,
    error,
    page,
    pageSize,
    totalCount,
    totalPages,
    selectedKeys,
    selectedRows,
    fetchPage,
    refresh,
    mutationDone,
    onPageChange,
  }
}
