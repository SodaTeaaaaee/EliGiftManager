<script setup lang="ts">
/**
 * Design-lab showcase for the DataGrid kit (shared/ui/data-grid/**).
 * One grid proves: CJK-aware sorting (mixed 中文/日本語/English participant
 * names — click the "Participant" header), status columns resolved through
 * the glossary, client-side pagination (24 rows > the default page size),
 * a multi-select toolbar, and loading/empty toggles.
 */
import { computed, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NSwitch } from 'naive-ui'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import type { DataGridColumnSpec } from '@/shared/ui/data-grid'
import type { ProductKindValue, ShipmentStatusValue, SupplierStateValue } from '@/shared/i18n/glossary'

interface ParticipantRow {
  id: string
  name: string
  wave: string
  productKind: ProductKindValue
  supplierState: SupplierStateValue
  shipmentStatus: ShipmentStatusValue
  quantity: number
  updatedAt: string
}

const { t } = useI18n()

const productKinds: ProductKindValue[] = ['badge', 'standee', 'charm', 'postcard', 'print', 'bundle', 'other']
const supplierStates: SupplierStateValue[] = [
  'not_submitted',
  'submitted',
  'accepted',
  'producing',
  'partially_shipped',
  'shipped',
  'canceled',
]
const shipmentStatuses: ShipmentStatusValue[] = ['pending', 'shipped', 'in_transit', 'delivered', 'exception', 'returned']
const waves = ['2026-07 会员波 · July Wave', '2026-06 零售波 · June Wave', '2026-05 限定波 · May Wave']

/** Mixed 中文 / 日本語 / English names — proves script-aware (pinyin/kana) sorting. */
const participantNames = [
  '田中さくら',
  '张伟',
  'Ava Whitfield',
  '鈴木ひなた',
  '李娜',
  "Liam O'Connor",
  'アイリ',
  '王芳',
  'Olivia Bennett',
  '佐藤あかり',
  '陈晓明',
  'Noah Fitzgerald',
  'ゆづき',
  '刘洋',
  'Mia Thompson',
  '高橋美咲',
  '赵敏',
  'Ethan Brooks',
  'カレン',
  '孙丽',
  'Grace Sullivan',
  '松本ゆい',
  '周杰',
  'James Whitmore',
]

function buildRows(): ParticipantRow[] {
  const now = Date.now()
  return participantNames.map((name, index) => ({
    id: `participant-${index}`,
    name,
    wave: waves[index % waves.length],
    productKind: productKinds[index % productKinds.length],
    supplierState: supplierStates[index % supplierStates.length],
    shipmentStatus: shipmentStatuses[index % shipmentStatuses.length],
    quantity: ((index * 7) % 6) + 1,
    updatedAt: new Date(now - index * 9 * 60 * 60 * 1000).toISOString(),
  }))
}

const allRows = ref<ParticipantRow[]>(buildRows())
const demoLoading = ref(false)
const demoEmpty = ref(false)
const selectedKeys = ref<Array<string | number>>([])
const lastClickedName = ref<string | null>(null)

const rows = computed(() => (demoEmpty.value ? [] : allRows.value))

const columns = computed(() => {
  const specs: DataGridColumnSpec<ParticipantRow>[] = [
    {
      type: 'text',
      key: 'name',
      title: t('uiKit.dataGridDemo.columns.name'),
      width: 160,
    },
    {
      type: 'text',
      key: 'wave',
      title: t('uiKit.dataGridDemo.columns.wave'),
      minWidth: 200,
    },
    {
      type: 'status',
      key: 'productKind',
      title: t('uiKit.dataGridDemo.columns.productKind'),
      dimension: 'productKind',
      width: 140,
    },
    {
      type: 'status',
      key: 'supplierState',
      title: t('uiKit.dataGridDemo.columns.supplierState'),
      dimension: 'supplierState',
      showDot: true,
      width: 150,
    },
    {
      type: 'status',
      key: 'shipmentStatus',
      title: t('uiKit.dataGridDemo.columns.shipmentStatus'),
      dimension: 'shipmentStatus',
      showDot: true,
      width: 140,
    },
    {
      type: 'number',
      key: 'quantity',
      title: t('uiKit.dataGridDemo.columns.quantity'),
      width: 90,
    },
    {
      type: 'date',
      key: 'updatedAt',
      title: t('uiKit.dataGridDemo.columns.updatedAt'),
      format: 'datetime',
      width: 160,
    },
    {
      type: 'actions',
      key: 'actions',
      title: t('uiKit.dataGridDemo.columns.actions'),
      width: 120,
      render: (row) =>
        h(
          NButton,
          {
            size: 'tiny',
            quaternary: true,
            onClick: () => {
              lastClickedName.value = row.name
            },
          },
          { default: () => t('uiKit.dataGridDemo.actions.viewDetail') },
        ),
    },
  ]
  return createColumns<ParticipantRow>(specs)
})
</script>

<template>
  <section class="data-grid-section">
    <header class="data-grid-section__header">
      <h2 class="data-grid-section__title">{{ t('uiKit.dataGridDemo.title') }}</h2>
      <p class="data-grid-section__subtitle">{{ t('uiKit.dataGridDemo.subtitle') }}</p>
    </header>

    <div class="data-grid-section__controls">
      <label class="data-grid-section__control">
        <NSwitch v-model:value="demoLoading" />
        <span>{{ t('uiKit.dataGridDemo.controls.loadingLabel') }}</span>
      </label>
      <label class="data-grid-section__control">
        <NSwitch v-model:value="demoEmpty" />
        <span>{{ t('uiKit.dataGridDemo.controls.emptyLabel') }}</span>
      </label>
      <p v-if="lastClickedName" class="data-grid-section__last-clicked">
        {{ t('uiKit.dataGridDemo.lastClicked', { name: lastClickedName }) }}
      </p>
    </div>

    <DataGrid
      v-model:selected-keys="selectedKeys"
      :columns="columns"
      :rows="rows"
      row-key="id"
      :loading="demoLoading"
      selectable
      :empty="{
        title: t('uiKit.dataGridDemo.empty.title'),
        description: t('uiKit.dataGridDemo.empty.description'),
      }"
    >
      <template #selection-toolbar="{ selectedKeys: keys, clearSelection }">
        <span class="data-grid-section__selection-count">
          {{ t('uiKit.dataGridDemo.selectionToolbar.countLabel', { n: keys.length }) }}
        </span>
        <NButton size="tiny" @click="clearSelection">
          {{ t('uiKit.dataGrid.selectionToolbar.clear') }}
        </NButton>
        <NButton size="tiny" type="primary" @click="clearSelection">
          {{ t('uiKit.dataGridDemo.selectionToolbar.markShipped') }}
        </NButton>
      </template>
    </DataGrid>
  </section>
</template>

<style scoped>
.data-grid-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.data-grid-section__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.data-grid-section__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.data-grid-section__subtitle {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.data-grid-section__controls {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-4);
}

.data-grid-section__control {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  cursor: pointer;
}

.data-grid-section__last-clicked {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-accent);
}

.data-grid-section__selection-count {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}
</style>
