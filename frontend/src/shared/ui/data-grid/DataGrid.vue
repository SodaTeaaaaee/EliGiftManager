<script setup lang="ts" generic="T extends object">
/**
 * DataGrid — the house wrapper around NDataTable (plan 4.3). The app's
 * most-used component: a strict, opinionated API on top of naive-ui's very
 * loose one. Columns are built with `createColumns<T>()` (see
 * `./createColumns.ts`) from the `DataGridColumnSpec<T>` house shape —
 * consumers never touch naive-ui's raw `TableColumns` type directly.
 *
 * - Sorting defaults to the revived CJK-aware `compareValues` (pinyin / kana
 *   romaji / hangul collation) for every sortable column.
 * - `loading` renders skeleton rows in-place (never naive-ui's built-in
 *   spinner overlay) so the header/column layout never jumps.
 * - `empty` is slot-driven: pass `empty` for the fallback copy, and/or fill
 *   the `#empty` slot yourself once `EmptyState` is available to drop in.
 * - Row density (`--row-height` / `--row-height-sm`) follows the app's
 *   density store automatically via naive-ui's own `size` prop.
 */
import { computed, h } from 'vue'
import { storeToRefs } from 'pinia'
import { NDataTable } from 'naive-ui'
import type {
  DataTableColumns,
  DataTableProps,
  DataTableRowKey,
  DataTableSize,
  DataTableSortState,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useThemeStore } from '@/shared/theme/theme'
import type { DataGridEmptyConfig, DataGridPagination, DataGridRowKey } from './types'

const props = withDefaults(
  defineProps<{
    columns: DataTableColumns<T>
    rows: T[]
    /** Row field name, or a function — either way must yield a stable unique key. */
    rowKey: DataGridRowKey<T>
    loading?: boolean
    /** Shows a checkbox column + enables `selectedKeys` / the selection toolbar slot. */
    selectable?: boolean
    selectedKeys?: Array<string | number>
    pagination?: DataGridPagination
    /** Fallback copy shown by the built-in empty state (also handed to the `#empty` slot). */
    empty?: DataGridEmptyConfig
  }>(),
  {
    loading: false,
    selectable: false,
    selectedKeys: () => [],
    pagination: 'client',
    empty: undefined,
  },
)

const emit = defineEmits<{
  'update:selectedKeys': [keys: Array<string | number>]
  'row-click': [row: T, index: number]
}>()

const { t } = useI18n()
const themeStore = useThemeStore()
const { density } = storeToRefs(themeStore)

/** `--row-height` (comfortable) / `--row-height-sm` (compact) via naive-ui's own size prop. */
const tableSize = computed<DataTableSize>(() => (density.value === 'compact' ? 'small' : 'medium'))

const rowKeyFn = computed<(row: T) => DataTableRowKey>(() => {
  if (typeof props.rowKey === 'function') return props.rowKey
  const field = props.rowKey
  return (row: T) => (row as Record<string, unknown>)[field] as DataTableRowKey
})

const SKELETON_ROW_COUNT = 6
/** Deterministic, non-uniform bar widths so skeleton rows don't look like a robotic grid. */
const SKELETON_WIDTHS = [42, 68, 55, 80, 60, 74, 48, 66]

interface SkeletonRow {
  __skeletonKey: number
}

const skeletonRows = computed<SkeletonRow[]>(() =>
  Array.from({ length: SKELETON_ROW_COUNT }, (_unused, index) => ({ __skeletonKey: index })),
)

function skeletonWidth(seed: number): string {
  return `${SKELETON_WIDTHS[seed % SKELETON_WIDTHS.length]}%`
}

const displayColumns = computed<DataTableColumns<Record<string, unknown>>>(() => {
  const contentColumns = props.columns as unknown as DataTableColumns<Record<string, unknown>>
  const withSelection = props.selectable
    ? [{ type: 'selection' as const, multiple: true }, ...contentColumns]
    : contentColumns

  if (!props.loading) return withSelection

  // Loading: keep every column's title/width/alignment (so the header never
  // reflows), but replace its render with a skeleton bar and strip
  // sorting/filtering (there's nothing real to sort/filter yet).
  return withSelection.map((column, columnIndex) => {
    if ('type' in column && column.type === 'selection') {
      return { ...column, disabled: () => true }
    }
    return {
      ...column,
      sorter: undefined,
      filter: undefined,
      render: (_row: Record<string, unknown>, rowIndex: number) =>
        h('span', {
          class: 'data-grid__skeleton-bar',
          style: { width: skeletonWidth(rowIndex + columnIndex) },
        }),
    }
  })
})

const effectiveData = computed<Array<Record<string, unknown>>>(() =>
  props.loading
    ? (skeletonRows.value as unknown as Array<Record<string, unknown>>)
    : (props.rows as unknown as Array<Record<string, unknown>>),
)

const effectiveRowKey = computed<(row: Record<string, unknown>) => DataTableRowKey>(() => {
  if (props.loading) return (row) => (row as unknown as SkeletonRow).__skeletonKey
  return rowKeyFn.value as unknown as (row: Record<string, unknown>) => DataTableRowKey
})

const isServerPagination = computed(
  () => typeof props.pagination === 'object' && props.pagination !== null && 'server' in props.pagination,
)

const naivePagination = computed<false | NonNullable<DataTableProps['pagination']>>(() => {
  const config = props.pagination
  if (config === 'none') return false
  if (config === 'client') {
    return {
      pageSize: 20,
      showSizePicker: true,
      pageSizes: [10, 20, 50, 100],
    }
  }
  const { server } = config
  return {
    page: server.page,
    pageSize: server.pageSize,
    itemCount: server.total,
    showSizePicker: true,
    pageSizes: server.pageSizes ?? [10, 20, 50],
    onUpdatePage: (page: number) => server.onChange(page, server.pageSize),
    onUpdatePageSize: (pageSize: number) => server.onChange(server.page, pageSize),
  }
})

const selectedRows = computed(() => props.rows.filter((row) => props.selectedKeys.includes(rowKeyFn.value(row))))

const emptyTitle = computed(() => props.empty?.title ?? t('uiKit.dataGrid.emptyFallback.title'))
const emptyDescription = computed(() => props.empty?.description)

/** Local, structural-only theming — reads live tokens via `var(--x)`, no
 * getComputedStyle needed since these are plain CSS custom property
 * references that cascade/re-resolve automatically on theme/skin swap. */
const localThemeOverrides: NonNullable<DataTableProps['themeOverrides']> = {
  borderColor: 'var(--row-border-color)',
  tdColorHover: 'var(--row-bg-hover)',
  thColor: 'var(--color-surface)',
  thTextColor: 'var(--color-text-secondary)',
  tdTextColor: 'var(--color-text-primary)',
  tdColor: 'var(--color-surface)',
}

function handleUpdateCheckedRowKeys(keys: DataTableRowKey[]) {
  emit('update:selectedKeys', keys as Array<string | number>)
}

function handleUpdateSorter(sorter: DataTableSortState | DataTableSortState[] | null): void {
  if (!isServerPagination.value || typeof props.pagination !== 'object') return
  const state = Array.isArray(sorter) ? sorter[0] : sorter
  const sortBy = state?.columnKey == null ? null : String(state.columnKey)
  const sortDir = state?.order === 'ascend' ? 'asc' : state?.order === 'descend' ? 'desc' : null
  props.pagination.server.onSort?.(sortDir == null ? null : sortBy, sortDir)
}

function clearSelection() {
  emit('update:selectedKeys', [])
}

function rowProps(row: Record<string, unknown>, index: number) {
  return {
    onClick: () => emit('row-click', row as T, index),
  }
}
</script>

<template>
  <div class="data-grid" :class="{ 'data-grid--loading': loading }">
    <div v-if="selectable && selectedKeys.length > 0" class="data-grid__selection-toolbar">
      <slot
        name="selection-toolbar"
        :selectedKeys="selectedKeys"
        :selectedRows="selectedRows"
        :clearSelection="clearSelection"
      >
        <span class="data-grid__selection-count">
          {{ t('uiKit.dataGrid.selectionToolbar.countLabel', { n: selectedKeys.length }) }}
        </span>
        <button type="button" class="data-grid__selection-clear" @click="clearSelection">
          {{ t('uiKit.dataGrid.selectionToolbar.clear') }}
        </button>
      </slot>
    </div>

    <NDataTable
      :columns="displayColumns"
      :data="effectiveData"
      :row-key="effectiveRowKey"
      :size="tableSize"
      :loading="false"
      :remote="isServerPagination"
      :pagination="naivePagination"
      :checked-row-keys="selectable ? selectedKeys : undefined"
      :row-props="rowProps"
      :bordered="false"
      :single-line="false"
      :theme-overrides="localThemeOverrides"
      @update:checked-row-keys="handleUpdateCheckedRowKeys"
      @update:sorter="handleUpdateSorter"
    >
      <template #empty>
        <slot name="empty" :title="emptyTitle" :description="emptyDescription">
          <div class="data-grid__empty-fallback">
            <p class="data-grid__empty-fallback-title">{{ emptyTitle }}</p>
            <p v-if="emptyDescription" class="data-grid__empty-fallback-description">{{ emptyDescription }}</p>
          </div>
        </slot>
      </template>
    </NDataTable>
  </div>
</template>

<style scoped>
.data-grid {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  overflow: hidden;
  background: var(--color-surface);
}

.data-grid__selection-toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-4);
  background: var(--color-accent-subtle);
  border-bottom: 1px solid var(--card-border-color);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.data-grid__selection-count {
  font-weight: var(--font-weight-medium);
}

.data-grid__selection-clear {
  margin-left: auto;
  border: none;
  background: transparent;
  color: var(--color-accent);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  padding: 0;
}

.data-grid__selection-clear:hover {
  color: var(--color-accent-hover);
  text-decoration: underline;
}

.data-grid__selection-clear:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.data-grid__empty-fallback {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-6) var(--space-4);
  text-align: center;
}

.data-grid__empty-fallback-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.data-grid__empty-fallback-description {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>

<style>
/* Skeleton bars: NDataTable's `render` is called per-cell, so the bar itself
   must be an un-scoped class (rendered outside this SFC's scoped subtree). */
.data-grid__skeleton-bar {
  display: inline-block;
  height: 12px;
  border-radius: var(--radius-full);
  background: var(--color-inset);
  animation: data-grid-skeleton-pulse var(--duration-slower) var(--ease-in-out) infinite alternate;
}

@keyframes data-grid-skeleton-pulse {
  from {
    opacity: 0.55;
  }
  to {
    opacity: 1;
  }
}
</style>
