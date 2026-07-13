import type { VNodeChild } from 'vue'
import type { GlossaryDimension, GlossaryDimensionValueMap } from '@/shared/i18n/glossary'

/**
 * The house column-spec API for `createColumns<T>()` (see `./createColumns.ts`).
 * Callers describe *what a column means*, not how naive-ui renders it — the
 * translation to a real `DataTableColumns<T>` (render fns, sorters, alignment
 * defaults) lives entirely in `createColumns.ts`.
 *
 * `title` is an already-resolved display string (call `t(...)` before
 * building the spec), matching the rest of the shared/ui kit's convention
 * (see `EmptyState`'s `title`/`description` props).
 */
export interface DataGridColumnBase<T> {
  /** Row field name (used as the naive-ui column key, and as the default `getValue` source). */
  key: (keyof T & string) | (string & {})
  /** Already-resolved column header text. */
  title: string
  width?: number | string
  minWidth?: number | string
  maxWidth?: number | string
  align?: 'left' | 'center' | 'right'
  fixed?: 'left' | 'right'
  /**
   * Whether this column can be sorted. Defaults to `true` for every column
   * type except `actions` — sorting goes through `compareValues` from the
   * revived CJK-aware sort library (`@/shared/lib/table/compareSortValues`),
   * so pinyin/kana/hangul ordering works out of the box.
   */
  sortable?: boolean
}

export interface DataGridTextColumn<T> extends DataGridColumnBase<T> {
  type: 'text'
  /** Defaults to `(row) => row[key]`. */
  getValue?: (row: T) => string | null | undefined
  /** Full cell-content override (icons, links, multi-line — anything beyond plain text). */
  render?: (row: T, index: number) => VNodeChild
  /** Truncate with an ellipsis + hover tooltip when the cell overflows. Defaults to `true`. */
  ellipsis?: boolean
}

export interface DataGridNumberColumn<T> extends DataGridColumnBase<T> {
  type: 'number'
  /** Defaults to `(row) => row[key]`. */
  getValue?: (row: T) => number | null | undefined
  /** Defaults to `value.toLocaleString()` in the active app locale. */
  formatter?: (value: number) => string
}

export interface DataGridStatusColumn<T> extends DataGridColumnBase<T> {
  type: 'status'
  /** The glossary dimension this column's raw values belong to. */
  dimension: GlossaryDimension
  /** Defaults to `(row) => row[key]`. */
  getValue?: (row: T) => GlossaryDimensionValueMap[GlossaryDimension] | (string & {}) | null | undefined
  showDot?: boolean
  size?: 'sm' | 'md'
}

export type DataGridDateFormat = 'date' | 'datetime' | 'relative'

export interface DataGridDateColumn<T> extends DataGridColumnBase<T> {
  type: 'date'
  /** Defaults to `(row) => row[key]`. */
  getValue?: (row: T) => Date | string | number | null | undefined
  /** Defaults to `'date'`. */
  format?: DataGridDateFormat
}

export interface DataGridActionsColumn<T> extends Omit<DataGridColumnBase<T>, 'sortable'> {
  type: 'actions'
  /** Required — renders the row's action buttons/menu. */
  render: (row: T, index: number) => VNodeChild
}

export type DataGridColumnSpec<T> =
  | DataGridTextColumn<T>
  | DataGridNumberColumn<T>
  | DataGridStatusColumn<T>
  | DataGridDateColumn<T>
  | DataGridActionsColumn<T>

/**
 * `'client'` — DataGrid paginates the full `rows` array itself (naive-ui's
 * built-in client-side pagination).
 * `'none'` — no pagination UI, all rows render.
 * `{ server: {...} }` — the caller owns the page/pageSize/total state and
 * fetches accordingly; DataGrid only renders the pager and forwards changes
 * via `onChange`.
 */
export type DataGridPagination =
  | 'client'
  | 'none'
  | {
      server: {
        total: number
        page: number
        pageSize: number
        pageSizes?: number[]
        onChange: (page: number, pageSize: number) => void
        onSort?: (
          sortBy: string | null,
          sortDir: 'asc' | 'desc' | null,
        ) => void
      }
    }

/** Already-resolved strings, mirroring `EmptyState`'s own `title`/`description` props. */
export interface DataGridEmptyConfig {
  title: string
  description?: string
}

export type DataGridRowKey<T> = (keyof T & string) | (string & {}) | ((row: T) => string | number)
