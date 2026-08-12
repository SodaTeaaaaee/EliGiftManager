/**
 * useInboxGrid — the demand-inbox grid's data/state composable (plan P4).
 * Owns:
 *
 * - The bespoke (non-glossary, per P4 foundations decision #5) `assignment`
 *   3-way toggle (`'all' | 'assigned' | 'unassigned'`) — a plain ref, NOT
 *   part of `useUrlFilters` / FilterBar. Ignored (forced to `'assigned'`)
 *   when `options.waveId` is set. In unscoped mode its changes are written
 *   back to `route.query.assignment` ('all' removes the param) — still
 *   read-only on init, unlike FilterBar fields.
 * - `filters` — `useUrlFilters(INBOX_GRID_FILTER_SCHEMA)`, the two
 *   `demandKind` / `routingDisposition` enum-multi dimensions
 *   (FilterBar-driven, reusing the existing glossary dimensions).
 * - Server-paginated row fetch via `listDemandInboxRowsPage` (bridge.ts).
 * - Selection state (`selectedKeys`, a plain `demandDocumentId[]`), mirroring
 *   `useFulfillmentGrid`'s contract: never auto-cleared on page/filter
 *   change; selected row objects are cached across pages for batch actions.
 * - `refresh()` / `mutationDone()` — re-fetch in place (no ref-identity
 *   replacement, no route remount). The two are currently identical aliases;
 *   `mutationDone()` exists so call sites read the same way
 *   `useFulfillmentGrid`'s do. Unlike `useFulfillmentGrid`, this composable
 *   does NOT itself hold a `useWaveWorkspaceContext()` — `WaveIntakeTab.vue`
 *   (the only wave-scoped caller) is responsible for ALSO calling
 *   `useWaveWorkspaceContext().refresh()` alongside `mutationDone()` when a
 *   mutation should refresh the workspace shell too.
 *
 * Wave-scoped mode sends both `assignment: 'assigned'` and the active
 * `waveId`, then uses the same server pagination and sorting contract.
 */
import { computed, onBeforeMount, onBeforeUnmount, onMounted, reactive, ref, watch, type ComputedRef, type Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listDemandInboxRowsPage } from '@/shared/api/bridge'
import { useUrlFilters, type UseUrlFiltersApi } from '@/shared/ui/filter-bar'
import { registerRefreshTarget } from '@/shared/lib/view-hotkeys'
import type { DemandInboxRow } from '@/entities/demand'
import { INBOX_GRID_FILTER_SCHEMA } from './filter-schema'

export type InboxAssignmentFilter = 'all' | 'assigned' | 'unassigned'

const DEFAULT_PAGE_SIZE = 50

export interface UseInboxGridOptions {
  /**
   * Scopes the grid to a single wave's assigned demand documents (see the
   * WAVE-SCOPED MODE doc above). A `ComputedRef` so it reactively follows
   * `useWaveWorkspaceContext().waveId` — a cross-wave deep link (workspace
   * shell stays mounted, only the route param changes) re-fetches
   * automatically.
   */
  waveId?: ComputedRef<number>
}

export interface UseInboxGridApi {
  /** `'all' | 'assigned' | 'unassigned'` — bind to a plain 3-way segmented control / radio group, NOT FilterBar. Inert (forced to `'assigned'`) in wave-scoped mode. In unscoped mode its changes are mirrored into `route.query.assignment` (write-only; init deep-link read stays one-shot). */
  assignment: Ref<InboxAssignmentFilter>
  /** True when this instance was built with `options.waveId` (wave-scoped mode). */
  isWaveScoped: boolean
  /** Pass straight to `<FilterBar :filters>` / `<SavedViews :filters>`. */
  filters: UseUrlFiltersApi<typeof INBOX_GRID_FILTER_SCHEMA>
  rows: Ref<DemandInboxRow[]>
  loading: Ref<boolean>
  error: Ref<boolean>
  /** 1-based current page. */
  page: Ref<number>
  pageSize: Ref<number>
  /** Total matching rows reported by the page endpoint. */
  totalCount: Ref<number>
  totalPages: Ref<number>
  /** Selected `demandDocumentId`s. Bind as `DataGrid`'s `selectedKeys` (v-model). */
  selectedKeys: Ref<number[]>
  /** Cached selected rows across loaded pages — feed `BatchActionBar`'s `selectedRows` prop. */
  selectedRows: ComputedRef<DemandInboxRow[]>
  /** Re-fetch with the current filter/assignment/page state. Also called internally on filter/page/assignment/waveId change. */
  fetchPage(): Promise<void>
  /** Alias for `fetchPage()`. */
  refresh(): Promise<void>
  /** Alias for `fetchPage()` — see the module doc comment for why this does NOT also refresh a wave-workspace context. */
  mutationDone(): Promise<void>
  /** Wire to `DataGridPagination`'s `server.onChange`. */
  onPageChange(nextPage: number, nextPageSize: number): void
  onSort(sortBy: string | null, sortDir: 'asc' | 'desc' | null): void
}

export function useInboxGrid(options: UseInboxGridOptions = {}): UseInboxGridApi {
  const isWaveScoped = options.waveId != null
  const scopedWaveId = options.waveId

  const assignment = ref<InboxAssignmentFilter>('all') as Ref<InboxAssignmentFilter>

  const route = useRoute()
  const router = useRouter()

  // Deep-link support (unscoped mode only): `WaveIntakeTab.vue`'s "assign more
  // from inbox" affordance links to `/inbox?assignment=unassigned` — read it
  // once on init so the bespoke toggle (read-only init, unlike FilterBar's
  // two-way URL sync) still honors an incoming deep link.
  if (options.waveId == null) {
    const initialAssignment = route.query.assignment
    if (initialAssignment === 'assigned' || initialAssignment === 'unassigned') {
      assignment.value = initialAssignment
    }
  }

  const filters = useUrlFilters(INBOX_GRID_FILTER_SCHEMA)

  const rows = ref<DemandInboxRow[]>([]) as Ref<DemandInboxRow[]>
  const loading = ref(true)
  const error = ref(false)
  const page = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const totalCount = ref(0)
  const totalPages = ref(0)
  const selectedKeys = ref<number[]>([]) as Ref<number[]>
  const sortBy = ref<string | null>(null)
  const sortDir = ref<'asc' | 'desc' | null>(null)
  const selectedRowCache = reactive(new Map<number, DemandInboxRow>())

  /** demandKind 与 routingDisposition 多值直传；空数组即不筛。 */
  function resolveInboxFilterParams(): { demandKinds: string[] | undefined; routingDispositions: string[] | undefined } {
    const kinds = filters.state.demandKind
    const dispositions = filters.state.routingDisposition
    return {
      demandKinds: kinds.length > 0 ? kinds : undefined,
      routingDispositions: dispositions.length > 0 ? dispositions : undefined,
    }
  }

  async function fetchPage(): Promise<void> {
    loading.value = true
    try {
      const { demandKinds, routingDispositions } = resolveInboxFilterParams()

      const result = await listDemandInboxRowsPage({
        assignment: scopedWaveId != null ? 'assigned' : assignment.value === 'all' ? undefined : assignment.value,
        demandKinds,
        routingDispositions,
        waveId: scopedWaveId?.value,
        sortBy: sortBy.value ?? undefined,
        sortDir: sortDir.value ?? undefined,
        limit: pageSize.value,
        offset: (page.value - 1) * pageSize.value,
      })
      totalCount.value = result.totalCount
      totalPages.value = Math.ceil(result.totalCount / pageSize.value)
      if ((page.value - 1) * pageSize.value >= result.totalCount && page.value > 1) {
        page.value = Math.max(1, totalPages.value)
        await fetchPage()
        return
      }
      rows.value = result.items
      for (const row of result.items) {
        if (selectedKeys.value.includes(row.demandDocumentId)) {
          selectedRowCache.set(row.demandDocumentId, row)
        }
      }
      error.value = false
    } catch {
      // `listDemandInboxRowsPage` is soft-fail only for the "no Wails runtime"
      // case (returns an empty page) — a real backend RPC error still
      // rejects, so this must be caught explicitly.
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
    await refresh()
  }

  function onPageChange(nextPage: number, nextPageSize: number): void {
    page.value = nextPageSize === pageSize.value ? nextPage : 1
    pageSize.value = nextPageSize
    void fetchPage()
  }

  function onSort(nextSortBy: string | null, nextSortDir: 'asc' | 'desc' | null): void {
    sortBy.value = nextSortBy
    sortDir.value = nextSortDir
    page.value = 1
    void fetchPage()
  }

  // Any filter-schema field change resets to page 1 and re-fetches (unscoped
  // mode only — page is always 1 in wave-scoped mode anyway).
  watch(
    () => INBOX_GRID_FILTER_SCHEMA.map((field) => filters.state[field.key]),
    () => {
      page.value = 1
      void fetchPage()
    },
    { deep: true },
  )

  // The bespoke assignment toggle also resets to page 1 and re-fetches. In
  // unscoped mode it mirrors its 3-way state into `route.query.assignment`
  // ('all' removes the param) so the current view stays deep-linkable —
  // write-only, the one-shot init read above is the only URL -> state sync.
  watch(assignment, () => {
    if (isWaveScoped) return
    page.value = 1
    const nextQuery = { ...route.query }
    if (assignment.value === 'all') delete nextQuery.assignment
    else nextQuery.assignment = assignment.value
    void router.replace({ query: nextQuery }).catch(() => { /* duplicated navigation */ })
    void fetchPage()
  })

  // Cross-wave deep link in wave-scoped mode.
  if (scopedWaveId != null) {
    watch(scopedWaveId, () => void fetchPage())
  }

  onMounted(() => {
    void fetchPage()
  })

  if (!isWaveScoped) {
    let unregisterRefresh: (() => void) | undefined
    onBeforeMount(() => {
      unregisterRefresh = registerRefreshTarget(mutationDone)
    })
    onBeforeUnmount(() => unregisterRefresh?.())
  }

  watch(selectedKeys, (keys) => {
    const keySet = new Set(keys)
    for (const key of selectedRowCache.keys()) {
      if (!keySet.has(key)) selectedRowCache.delete(key)
    }
    for (const row of rows.value) {
      if (keySet.has(row.demandDocumentId)) selectedRowCache.set(row.demandDocumentId, row)
    }
  }, { deep: true })

  const selectedRows = computed<DemandInboxRow[]>(() =>
    selectedKeys.value
      .map((key) => selectedRowCache.get(key))
      .filter((row): row is DemandInboxRow => row != null),
  )

  return {
    assignment,
    isWaveScoped,
    filters,
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
    onSort,
  }
}
