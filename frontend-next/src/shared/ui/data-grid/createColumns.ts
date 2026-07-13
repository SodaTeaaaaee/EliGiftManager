import { h } from 'vue'
import type { DataTableBaseColumn, DataTableColumns } from 'naive-ui'
import { compareValues } from '@/shared/lib/table/compareSortValues'
import { useGlossary } from '@/shared/i18n/glossary'
import StatusBadge from '@/shared/ui/status/StatusBadge.vue'
import { formatGridDate } from './formatDate'
import type { DataGridColumnSpec } from './types'

/** What every empty/missing cell value renders as — never a blank, silent gap. */
const EMPTY_PLACEHOLDER = '—' // em dash

function defaultGetValue<T, V>(key: string): (row: T) => V {
  return (row: T) => (row as Record<string, unknown>)[key] as V
}

/**
 * Translates the house `DataGridColumnSpec<T>[]` API into a real naive-ui
 * `DataTableColumns<T>`. This is the ONLY place that knows how each column
 * `type` renders and sorts — `DataGrid.vue` just forwards the result (plus
 * an optional selection column) straight to `<NDataTable>`.
 *
 * String/date/status sorting goes through `compareValues` (the revived
 * script-aware sort lib), so pinyin/kana/hangul ordering is the default,
 * not an opt-in.
 */
export function createColumns<T extends object>(specs: DataGridColumnSpec<T>[]): DataTableColumns<T> {
  const { label: glossaryLabel } = useGlossary()

  return specs.map((spec): DataTableBaseColumn<T> => {
    const common = {
      key: spec.key,
      title: spec.title,
      width: spec.width,
      minWidth: spec.minWidth,
      maxWidth: spec.maxWidth,
      fixed: spec.fixed,
    }

    switch (spec.type) {
      case 'text': {
        const getValue = spec.getValue ?? defaultGetValue<T, string | null | undefined>(spec.key)
        const sortable = spec.sortable ?? true
        return {
          ...common,
          align: spec.align ?? 'left',
          ellipsis: spec.ellipsis ?? true,
          sorter: sortable ? (a: T, b: T) => compareValues(getValue(a), getValue(b)) : undefined,
          render: (row: T, index: number) => {
            if (spec.render) return spec.render(row, index)
            const value = getValue(row)
            return value == null || value === '' ? EMPTY_PLACEHOLDER : value
          },
        }
      }

      case 'number': {
        const getValue = spec.getValue ?? defaultGetValue<T, number | null | undefined>(spec.key)
        const sortable = spec.sortable ?? true
        const format = spec.formatter ?? ((value: number) => value.toLocaleString())
        return {
          ...common,
          align: spec.align ?? 'right',
          sorter: sortable ? (a: T, b: T) => compareValues(getValue(a), getValue(b)) : undefined,
          render: (row: T) => {
            const value = getValue(row)
            if (value == null || Number.isNaN(value)) return EMPTY_PLACEHOLDER
            return h('span', { class: 'tabular-nums' }, format(value))
          },
        }
      }

      case 'status': {
        const getValue = spec.getValue ?? defaultGetValue<T, string | null | undefined>(spec.key)
        const sortable = spec.sortable ?? true
        return {
          ...common,
          align: spec.align ?? 'left',
          sorter: sortable
            ? (a: T, b: T) => {
                const va = getValue(a)
                const vb = getValue(b)
                const la = va == null ? '' : glossaryLabel(spec.dimension, va)
                const lb = vb == null ? '' : glossaryLabel(spec.dimension, vb)
                return compareValues(la, lb)
              }
            : undefined,
          render: (row: T) => {
            const value = getValue(row)
            if (value == null || value === '') return EMPTY_PLACEHOLDER
            return h(StatusBadge, {
              dimension: spec.dimension,
              value,
              size: spec.size ?? 'sm',
              showDot: spec.showDot ?? false,
            })
          },
        }
      }

      case 'date': {
        const getValue = spec.getValue ?? defaultGetValue<T, Date | string | number | null | undefined>(spec.key)
        const sortable = spec.sortable ?? true
        const format = spec.format ?? 'date'
        return {
          ...common,
          align: spec.align ?? 'left',
          sorter: sortable
            ? (a: T, b: T) => {
                const va = getValue(a)
                const vb = getValue(b)
                const ta = va == null || va === '' ? null : new Date(va).getTime()
                const tb = vb == null || vb === '' ? null : new Date(vb).getTime()
                return compareValues(ta, tb)
              }
            : undefined,
          render: (row: T) => {
            const formatted = formatGridDate(getValue(row), format)
            return formatted === '' ? EMPTY_PLACEHOLDER : h('span', { class: 'tabular-nums' }, formatted)
          },
        }
      }

      case 'actions': {
        return {
          ...common,
          align: spec.align ?? 'right',
          render: (row: T, index: number) => spec.render(row, index),
        }
      }

      default: {
        // Exhaustiveness guard: if a new column `type` is added to
        // `DataGridColumnSpec<T>` without a case here, this is a compile error.
        const exhaustive: never = spec
        throw new Error(`DataGrid: unhandled column type ${JSON.stringify(exhaustive)}`)
      }
    }
  })
}
