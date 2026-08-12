/**
 * useUrlFilters — declarative filter schema <-> route.query two-way sync
 * (plan 3.3.2: "筛选状态写入 URL query，深链可分享、任务中心可直达").
 *
 * The query <-> value codec (`parseEnumMultiQuery` / `serializeKeywordQuery` /
 * etc.) is exported as plain functions operating on strings — no vue-router
 * import needed to reason about or unit-test the encoding rules. Only
 * `useUrlFilters` itself reaches for `useRoute`/`useRouter`, and only to wire
 * that codec to the live route.
 *
 * Debouncing keyword input is FilterBar.vue's job, not this composable's:
 * by the time `setKeyword` is called here, the value is final and gets
 * written straight through.
 */
import { computed, reactive, watch, onBeforeUnmount, type ComputedRef } from 'vue'
import { useRoute, useRouter, type LocationQuery, type LocationQueryValue } from 'vue-router'
import type {
  ActiveFilterChip,
  FilterField,
  FilterKey,
  FilterSchema,
  FilterSnapshot,
  FilterState,
} from './types'

/** Collapses vue-router's `string | null | (string|null)[]` query value to one string or null. */
function normalizeQueryValue(raw: LocationQueryValue | LocationQueryValue[] | undefined): string | null {
  if (Array.isArray(raw)) return raw.length ? raw[0] : null
  return raw ?? null
}

/** `"a,b,c"` -> `['a', 'b', 'c']`; blank/missing -> `[]`. Empty tokens are dropped. */
export function parseEnumMultiQuery(raw: string | null): string[] {
  if (!raw) return []
  return raw
    .split(',')
    .map((token) => token.trim())
    .filter((token) => token.length > 0)
}

/** `['a', 'b']` -> `"a,b"`; `[]` -> `undefined` (meaning: remove the query param). */
export function serializeEnumMultiQuery(values: readonly string[]): string | undefined {
  return values.length ? values.join(',') : undefined
}

/** Missing/null -> `''`. */
export function parseKeywordQuery(raw: string | null): string {
  return raw ?? ''
}

/** Blank (after trim) -> `undefined` (remove the query param), else the raw value. */
export function serializeKeywordQuery(value: string): string | undefined {
  return value.trim().length ? value : undefined
}

function fieldInitialValue(field: FilterField, raw: string | null): string[] | string {
  return field.type === 'enum-multi' ? parseEnumMultiQuery(raw) : parseKeywordQuery(raw)
}

function buildStateFromQuery<S extends FilterSchema>(schema: S, query: LocationQuery): FilterState<S> {
  const result: Record<string, string[] | string> = {}
  for (const field of schema) {
    result[field.key] = fieldInitialValue(field, normalizeQueryValue(query[field.key]))
  }
  return result as FilterState<S>
}

export interface UseUrlFiltersApi<S extends FilterSchema> {
  /** The schema this instance was built from — handy for `FilterBar :schema="filters.schema"`. */
  schema: S
  /** The live, URL-synced filter state. One property per field key, typed per field kind. */
  state: FilterState<S>
  /** Replace the full value set of an enum-multi field. */
  setEnumValues(key: FilterKey<S>, values: readonly string[]): void
  /** Add/remove a single value from an enum-multi field. */
  toggleEnumValue(key: FilterKey<S>, value: string): void
  /** Set a keyword field's committed value (call this post-debounce). */
  setKeyword(key: FilterKey<S>, value: string): void
  /** Reset a single field to its empty value. */
  clearField(key: FilterKey<S>): void
  /** Reset every field in the schema. */
  clearAll(): void
  /** Total count of active values across every field (chips + keyword-present). */
  activeCount: ComputedRef<number>
  /** `activeCount.value > 0`. */
  isActive: ComputedRef<boolean>
  /** Removable-chip view of the current state, ready for `FilterBar`'s chip row. */
  activeChips: ComputedRef<ActiveFilterChip[]>
  /** A plain, JSON-serializable copy of `state` — feed to `SavedViews`' "save current". */
  getSnapshot(): FilterSnapshot
  /** Apply a saved/preset snapshot on top of the current state (missing keys are left untouched). */
  applySnapshot(snapshot: FilterSnapshot): void
}

export interface UseUrlFiltersOptions {
  /**
   * When false, filter state is fully local: neither the write-back watcher
   * (state → route.query) nor the reverse watcher (route.query → state) is
   * registered, and the initial state is NOT built from route.query. Used by
   * dialog-hosted grids that share a route with another grid instance and
   * must not cross-talk through the URL (e.g. the wave intake's pull dialog).
   * Defaults to true.
   */
  syncToUrl?: boolean
}

/**
 * Builds URL-synced reactive filter state from a declarative schema.
 * Must be called during a component's `setup()` (it uses `useRoute`/`useRouter`
 * and registers an `onBeforeUnmount` cleanup).
 */
export function useUrlFilters<S extends FilterSchema>(schema: S, options?: UseUrlFiltersOptions): UseUrlFiltersApi<S> {
  const route = useRoute()
  const router = useRouter()
  const syncToUrl = options?.syncToUrl !== false

  // Localized instances start empty instead of picking up whatever the
  // current route.query happens to carry for these keys.
  const state = reactive(buildStateFromQuery(schema, syncToUrl ? route.query : {})) as FilterState<S>

  // Guards against the write-back watcher reacting to the very update it just
  // pushed to the router (self-triggered route.query change).
  let suppressNextRouteSync = false
  let pendingReplace: ReturnType<typeof setTimeout> | undefined

  function writeQueryFromState(): void {
    const nextQuery: LocationQuery = { ...route.query }
    for (const field of schema) {
      const value = (state as Record<string, string[] | string>)[field.key]
      const serialized =
        field.type === 'enum-multi'
          ? serializeEnumMultiQuery(value as string[])
          : serializeKeywordQuery(value as string)
      if (serialized === undefined) delete nextQuery[field.key]
      else nextQuery[field.key] = serialized
    }
    suppressNextRouteSync = true
    void router.replace({ query: nextQuery }).catch(() => {
      /* navigation duplicated/aborted — state already reflects the intended values */
    })
  }

  // Debounce the router.replace call itself by a tick so several sync field
  // writes in the same render (e.g. clearAll touching N fields) collapse into
  // a single history entry instead of N.
  function scheduleQueryWrite(): void {
    if (pendingReplace !== undefined) clearTimeout(pendingReplace)
    pendingReplace = setTimeout(() => {
      pendingReplace = undefined
      writeQueryFromState()
    }, 0)
  }

  if (syncToUrl) {
    watch(
      () => schema.map((field) => (state as Record<string, string[] | string>)[field.key]),
      scheduleQueryWrite,
      { deep: true },
    )

    // Keep state in sync with external route.query changes (back/forward nav,
    // a task-center deep link, another component editing the same query).
    watch(
      () => route.query,
      (query) => {
        if (suppressNextRouteSync) {
          suppressNextRouteSync = false
          return
        }
        const next = buildStateFromQuery(schema, query)
        for (const field of schema) {
          const target = state as Record<string, string[] | string>
          const incoming = (next as Record<string, string[] | string>)[field.key]
          target[field.key] = incoming
        }
      },
    )
  }

  function setEnumValues(key: FilterKey<S>, values: readonly string[]): void {
    const field = schema.find((f) => f.key === key)
    if (!field || field.type !== 'enum-multi') return
    ;(state as Record<string, string[] | string>)[key] = Array.from(new Set(values))
  }

  function toggleEnumValue(key: FilterKey<S>, value: string): void {
    const current = (state as Record<string, string[] | string>)[key]
    const list = Array.isArray(current) ? current : []
    const next = list.includes(value) ? list.filter((v) => v !== value) : [...list, value]
    setEnumValues(key, next)
  }

  function setKeyword(key: FilterKey<S>, value: string): void {
    const field = schema.find((f) => f.key === key)
    if (!field || field.type !== 'keyword') return
    ;(state as Record<string, string[] | string>)[key] = value
  }

  function clearField(key: FilterKey<S>): void {
    const field = schema.find((f) => f.key === key)
    if (!field) return
    ;(state as Record<string, string[] | string>)[key] = field.type === 'enum-multi' ? [] : ''
  }

  function clearAll(): void {
    for (const field of schema) clearField(field.key as FilterKey<S>)
  }

  const activeCount = computed<number>(() =>
    schema.reduce((total, field) => {
      const value = (state as Record<string, string[] | string>)[field.key]
      if (field.type === 'enum-multi') return total + (Array.isArray(value) ? value.length : 0)
      return total + (value ? 1 : 0)
    }, 0),
  )

  const isActive = computed<boolean>(() => activeCount.value > 0)

  const activeChips = computed<ActiveFilterChip[]>(() => {
    const chips: ActiveFilterChip[] = []
    for (const field of schema) {
      const value = (state as Record<string, string[] | string>)[field.key]
      if (field.type === 'enum-multi') {
        for (const v of value as string[]) {
          chips.push({ id: `${field.key}:${v}`, fieldKey: field.key, type: field.type, dimension: field.dimension, value: v })
        }
      } else if (typeof value === 'string' && value.trim().length > 0) {
        chips.push({ id: field.key, fieldKey: field.key, type: field.type, value })
      }
    }
    return chips
  })

  function getSnapshot(): FilterSnapshot {
    const snapshot: FilterSnapshot = {}
    for (const field of schema) {
      const value = (state as Record<string, string[] | string>)[field.key]
      snapshot[field.key] = Array.isArray(value) ? [...value] : value
    }
    return snapshot
  }

  function applySnapshot(snapshot: FilterSnapshot): void {
    for (const field of schema) {
      if (!(field.key in snapshot)) continue
      const value = snapshot[field.key]
      if (field.type === 'enum-multi') {
        setEnumValues(field.key as FilterKey<S>, Array.isArray(value) ? value : [])
      } else {
        setKeyword(field.key as FilterKey<S>, typeof value === 'string' ? value : '')
      }
    }
  }

  onBeforeUnmount(() => {
    if (pendingReplace !== undefined) clearTimeout(pendingReplace)
  })

  return {
    schema,
    state,
    setEnumValues,
    toggleEnumValue,
    setKeyword,
    clearField,
    clearAll,
    activeCount,
    isActive,
    activeChips,
    getSnapshot,
    applySnapshot,
  }
}
