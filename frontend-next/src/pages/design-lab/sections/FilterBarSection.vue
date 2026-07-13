<script setup lang="ts">
/**
 * Design-lab showcase for the FilterBar kit (shared/ui/filter-bar/**): a
 * combination filter bar over sample fulfillment lines (address state ∧
 * supplier state, plus a keyword field), presets + saved views, and a live
 * URL-query preview so the useUrlFilters() sync is visibly demonstrated.
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { FilterBar, SavedViews, useUrlFilters, type FilterSchema, type FilterViewPreset } from '@/shared/ui/filter-bar'
import { StatusBadge } from '@/shared/ui/status'

const { t } = useI18n()
const route = useRoute()

const schema = [
  { key: 'addressState', type: 'enum-multi', dimension: 'addressState' },
  { key: 'supplierState', type: 'enum-multi', dimension: 'supplierState' },
  { key: 'keyword', type: 'keyword' },
] as const satisfies FilterSchema

const filters = useUrlFilters(schema)

const presets = computed<FilterViewPreset[]>(() => [
  {
    id: 'blocked',
    label: t('filterBar.demo.presets.blocked'),
    snapshot: { addressState: ['missing', 'invalid'] },
  },
  {
    id: 'ready-to-submit',
    label: t('filterBar.demo.presets.readyToSubmit'),
    snapshot: { addressState: ['ready'], supplierState: ['not_submitted'] },
  },
  {
    id: 'producing',
    label: t('filterBar.demo.presets.producing'),
    snapshot: { supplierState: ['producing'] },
  },
])

interface SampleRow {
  id: string
  participant: string
  product: string
  addressState: 'missing' | 'ready' | 'invalid'
  supplierState:
    | 'not_submitted'
    | 'submitted'
    | 'accepted'
    | 'producing'
    | 'partially_shipped'
    | 'shipped'
    | 'canceled'
}

/** Realistic fulfillment-domain sample data — CJK + Latin names mixed, per the design-lab convention. */
const sampleRows: SampleRow[] = [
  { id: 'L-001', participant: '星野・アイ（Ai Hoshino）', product: '限定徽章套装', addressState: 'ready', supplierState: 'not_submitted' },
  { id: 'L-002', participant: '有村架純 Arimura Kasumi', product: '亚克力立牌', addressState: 'missing', supplierState: 'not_submitted' },
  { id: 'L-003', participant: '佐藤あかり Sato Akari', product: '应援手幅', addressState: 'invalid', supplierState: 'not_submitted' },
  { id: 'L-004', participant: '宮子 Miyako', product: '明信片套组', addressState: 'ready', supplierState: 'submitted' },
  { id: 'L-005', participant: '類想 Ayaka', product: '挂件钥匙扣', addressState: 'ready', supplierState: 'producing' },
  { id: 'L-006', participant: '铃木ひなた Suzuki Hinata', product: 'Elissia 限定徽章', addressState: 'missing', supplierState: 'accepted' },
  { id: 'L-007', participant: '高橋洋子 Takahashi Yoko', product: '印刷海报', addressState: 'ready', supplierState: 'producing' },
  { id: 'L-008', participant: '中村悠斗 Nakamura Yuto', product: '立牌套装', addressState: 'invalid', supplierState: 'accepted' },
  { id: 'L-009', participant: '田中愛子 Aiko Tanaka', product: '挂件', addressState: 'ready', supplierState: 'partially_shipped' },
  { id: 'L-010', participant: '渡辺さくら Watanabe Sakura', product: '明信片', addressState: 'ready', supplierState: 'shipped' },
  { id: 'L-011', participant: '小林大地 Kobayashi Daichi', product: '徽章套装', addressState: 'missing', supplierState: 'canceled' },
  { id: 'L-012', participant: '山本ひまり Yamamoto Himari', product: '立牌', addressState: 'ready', supplierState: 'not_submitted' },
  { id: 'L-013', participant: '陈薇 Chen Wei', product: '应援色纸', addressState: 'ready', supplierState: 'submitted' },
  { id: 'L-014', participant: '木村拓也 Kimura Takuya', product: '徽章', addressState: 'invalid', supplierState: 'producing' },
]

const filteredRows = computed(() => {
  const addressSelected = filters.state.addressState
  const supplierSelected = filters.state.supplierState
  const keyword = filters.state.keyword.trim().toLowerCase()

  return sampleRows.filter((row) => {
    if (addressSelected.length > 0 && !addressSelected.includes(row.addressState)) return false
    if (supplierSelected.length > 0 && !supplierSelected.includes(row.supplierState)) return false
    if (keyword.length > 0 && !`${row.participant} ${row.product}`.toLowerCase().includes(keyword)) return false
    return true
  })
})
</script>

<template>
  <section class="filter-bar-section">
    <header class="filter-bar-section__header">
      <h2 class="filter-bar-section__title">{{ t('filterBar.demo.title') }}</h2>
      <p class="filter-bar-section__subtitle">{{ t('filterBar.demo.subtitle') }}</p>
    </header>

    <div class="filter-bar-section__toolbar">
      <FilterBar class="filter-bar-section__filter-bar" :filters="filters">
        <template #result-count>
          {{ t('filterBar.demo.resultCount', { n: filteredRows.length }) }}
        </template>
      </FilterBar>
      <SavedViews :filters="filters" scope-id="design-lab-fulfillment-demo" :presets="presets" />
    </div>

    <p class="filter-bar-section__url">
      <span class="filter-bar-section__url-label">{{ t('filterBar.demo.urlPreviewLabel') }}</span>
      <code class="filter-bar-section__url-value tabular-nums">{{ route.fullPath }}</code>
    </p>

    <div class="filter-bar-section__table-wrap">
      <table class="filter-bar-section__table">
        <thead>
          <tr>
            <th>{{ t('filterBar.demo.tableHeaders.participant') }}</th>
            <th>{{ t('filterBar.demo.tableHeaders.product') }}</th>
            <th>{{ t('filterBar.demo.tableHeaders.addressState') }}</th>
            <th>{{ t('filterBar.demo.tableHeaders.supplierState') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in filteredRows" :key="row.id">
            <td>{{ row.participant }}</td>
            <td>{{ row.product }}</td>
            <td><StatusBadge dimension="addressState" :value="row.addressState" size="sm" show-dot /></td>
            <td><StatusBadge dimension="supplierState" :value="row.supplierState" size="sm" show-dot /></td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<style scoped>
.filter-bar-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.filter-bar-section__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.filter-bar-section__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.filter-bar-section__subtitle {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.filter-bar-section__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--card-padding);
  background: var(--card-bg);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  box-shadow: var(--card-shadow);
}

.filter-bar-section__filter-bar {
  flex: 1;
  min-width: 320px;
}

.filter-bar-section__url {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin: 0;
  padding: var(--space-2) var(--space-3);
  background: var(--color-inset);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
}

.filter-bar-section__url-label {
  color: var(--color-text-muted);
  white-space: nowrap;
}

.filter-bar-section__url-value {
  color: var(--color-accent);
  font-family: var(--font-mono);
  word-break: break-all;
}

.filter-bar-section__table-wrap {
  overflow-x: auto;
  background: var(--card-bg);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  box-shadow: var(--card-shadow);
}

.filter-bar-section__table {
  width: 100%;
  min-width: 560px;
  border-collapse: collapse;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
}

.filter-bar-section__table th {
  text-align: left;
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
}

.filter-bar-section__table td {
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text-primary);
  vertical-align: middle;
}

.filter-bar-section__table tbody tr:last-child td {
  border-bottom: none;
}

.filter-bar-section__table tbody tr:hover {
  background: var(--row-bg-hover);
}
</style>
