<script setup lang="ts">
/**
 * BatchStockToWaveDialog — multi-select product masters -> pick a wave ->
 * dedup-aware batch stock (plan §3.7: "批量备货到波次...显示该波已有快照、去重
 * 提示"). Two independent entry points share this one component:
 *
 * - From `ProductsPage.vue`: `selectedMasters` is provided (the grid's
 *   current multi-selection) and `preselectedWaveId` is omitted — the
 *   dialog shows an in-dialog wave picker.
 * - From `WaveAllocationTab.vue`'s reverse entry ("从主档挑选商品"):
 *   `preselectedWaveId` is provided (the current wave) and `selectedMasters`
 *   is omitted — the dialog shows an in-dialog master picker instead, fed by
 *   the same `listProductMasters()` full list (archived masters excluded —
 *   stocking an archived master into a wave is not a supported flow).
 *
 * Either way, once BOTH a wave and a master set are resolved, the dialog
 * fetches `listProductsByWave(waveId)` and cross-references by
 * `(supplierPlatform, factorySku)` to render the dedup hint BEFORE submit,
 * then persists via the Detailed snapshot variant so the created/skipped
 * counts can be reflected back to the caller.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NFormItem, NSelect, NButton } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { DataGrid, createColumns, type DataGridColumnSpec } from '@/shared/ui/data-grid'
import { useFeedback } from '@/shared/ui/feedback'
import { listProductMasters, listProductsByWave, listWaves, snapshotProductsForWaveDetailed } from '@/shared/api/bridge'
import type { ProductMaster, Product } from '@/entities/product'
import type { Wave } from '@/entities/wave'

const props = withDefaults(
  defineProps<{
    show: boolean
    /** Pre-selected masters (batch action from `ProductsPage.vue`'s grid). Omit to pick masters inside this dialog. */
    selectedMasters?: ProductMaster[]
    /** Pre-scoped wave id (reverse entry from `WaveAllocationTab.vue`). Omit to show the in-dialog wave picker. */
    preselectedWaveId?: number
  }>(),
  {
    selectedMasters: undefined,
    preselectedWaveId: undefined,
  },
)

const emit = defineEmits<{
  'update:show': [boolean]
  success: [{ createdCount: number; skippedCount: number }]
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const needsMasterPicker = computed(() => !props.selectedMasters || props.selectedMasters.length === 0)
const needsWavePicker = computed(() => props.preselectedWaveId == null)

// ── Master picker (mode B — no `selectedMasters` prop) ──
const allMasters = ref<ProductMaster[]>([])
const loadingMasters = ref(false)
const checkedMasterKeys = ref<Array<string | number>>([])

const pickableMasters = computed(() => allMasters.value.filter((master) => !master.archived))

async function loadMasters(): Promise<void> {
  loadingMasters.value = true
  try {
    allMasters.value = await listProductMasters()
  } finally {
    loadingMasters.value = false
  }
}

const masterColumns = computed(() => {
  const specs: DataGridColumnSpec<ProductMaster>[] = [
    { type: 'text', key: 'name', title: t('products.columns.name'), minWidth: 160 },
    { type: 'text', key: 'supplierPlatform', title: t('products.columns.supplierPlatform'), width: 130 },
    { type: 'text', key: 'factorySku', title: t('products.columns.factorySku'), width: 140 },
    { type: 'status', key: 'productKind', title: t('products.columns.productKind'), dimension: 'productKind', width: 120 },
  ]
  return createColumns<ProductMaster>(specs)
})

// ── Wave picker (mode B — no `preselectedWaveId` prop) ──
const waves = ref<Wave[]>([])
const loadingWaves = ref(false)
const selectedWaveId = ref<number | null>(null)

async function loadWaves(): Promise<void> {
  loadingWaves.value = true
  try {
    waves.value = await listWaves()
  } finally {
    loadingWaves.value = false
  }
}

const waveOptions = computed<SelectOption[]>(() =>
  waves.value
    .filter((wave) => wave.lifecycleStage !== 'closed')
    .map((wave) => ({ label: `${wave.waveNo} · ${wave.name}`, value: wave.id })),
)

// ── Existing-snapshot / dedup ──
const existingProducts = ref<Product[]>([])
const loadingExisting = ref(false)

function productKey(supplierPlatform: string, factorySku: string): string {
  return `${supplierPlatform}::${factorySku}`
}

async function loadExisting(waveId: number): Promise<void> {
  loadingExisting.value = true
  try {
    existingProducts.value = await listProductsByWave(waveId)
  } finally {
    loadingExisting.value = false
  }
}

watch(selectedWaveId, async (waveId) => {
  if (waveId == null) {
    existingProducts.value = []
    return
  }
  await loadExisting(waveId)
})

const effectiveMasters = computed<ProductMaster[]>(() => {
  if (props.selectedMasters && props.selectedMasters.length > 0) return props.selectedMasters
  const checked = new Set(checkedMasterKeys.value.map((key) => Number(key)))
  return allMasters.value.filter((master) => checked.has(Number(master.id)))
})

const existingKeys = computed(() => new Set(existingProducts.value.map((p) => productKey(p.supplierPlatform, p.factorySku))))

const duplicateMasters = computed<ProductMaster[]>(() =>
  effectiveMasters.value.filter((master) => existingKeys.value.has(productKey(master.supplierPlatform, master.factorySku))),
)

const newMasterCount = computed(() => effectiveMasters.value.length - duplicateMasters.value.length)

// ── Lifecycle ──
const submitting = ref(false)

function resetState(): void {
  checkedMasterKeys.value = []
  selectedWaveId.value = props.preselectedWaveId ?? null
  existingProducts.value = []
  submitting.value = false
}

// This dialog stays mounted (no `v-if` at the call site) — reset + (re)load on every open.
watch(
  () => props.show,
  async (visible) => {
    if (!visible) return
    resetState()
    const loads: Promise<void>[] = []
    if (needsMasterPicker.value) loads.push(loadMasters())
    if (needsWavePicker.value) loads.push(loadWaves())
    await Promise.all(loads)
    if (selectedWaveId.value != null) await loadExisting(selectedWaveId.value)
  },
)

const canSubmit = computed(
  () => !submitting.value && selectedWaveId.value != null && effectiveMasters.value.length > 0,
)

function close(): void {
  emit('update:show', false)
}

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value || selectedWaveId.value == null) return
  submitting.value = true
  try {
    const result = await snapshotProductsForWaveDetailed({
      waveId: selectedWaveId.value,
      masterIds: effectiveMasters.value.map((master) => master.id),
    })
    feedback.success(t('products.batchStock.resultSummary', { created: result.createdCount, skipped: result.skippedCount }))
    emit('success', { createdCount: result.createdCount, skippedCount: result.skippedCount })
    close()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('products.batchStock.dialogTitle')"
    :style="{ width: 'min(680px, 94vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <div class="batch-stock-dialog__body">
      <template v-if="needsWavePicker">
        <NFormItem :label="t('products.batchStock.waveLabel')" label-placement="top">
          <NSelect
            v-model:value="selectedWaveId"
            :options="waveOptions"
            :loading="loadingWaves"
            filterable
            :placeholder="t('products.batchStock.wavePlaceholder')"
            :disabled="submitting"
          />
        </NFormItem>
      </template>
      <p v-else class="batch-stock-dialog__hint">{{ t('products.batchStock.fromWaveWorkspaceHint') }}</p>

      <template v-if="needsMasterPicker">
        <p class="batch-stock-dialog__section-title">{{ t('products.pickFromMasterAction') }}</p>
        <DataGrid
          :columns="masterColumns"
          :rows="pickableMasters"
          row-key="id"
          :loading="loadingMasters"
          selectable
          :selected-keys="checkedMasterKeys"
          pagination="client"
          :empty="{ title: t('products.empty.title') }"
          @update:selected-keys="(keys) => (checkedMasterKeys = keys)"
        />
      </template>

      <div v-if="selectedWaveId != null" class="batch-stock-dialog__dedup">
        <p class="batch-stock-dialog__section-title">{{ t('products.batchStock.existingSnapshotTitle') }}</p>
        <p class="batch-stock-dialog__hint">
          {{ t('products.batchStock.existingSnapshotCount', { count: existingProducts.length }) }}
        </p>

        <template v-if="effectiveMasters.length > 0">
          <p v-if="duplicateMasters.length > 0" class="batch-stock-dialog__dedup-warning">
            {{ t('products.batchStock.dedupHint', { count: duplicateMasters.length }) }}
          </p>
          <ul v-if="duplicateMasters.length > 0" class="batch-stock-dialog__dedup-list">
            <li v-for="master in duplicateMasters" :key="master.id">{{ master.name }} ({{ master.factorySku }})</li>
          </ul>
          <p v-else class="batch-stock-dialog__hint">{{ t('products.batchStock.noDedup') }}</p>
          <p v-if="newMasterCount > 0 && duplicateMasters.length > 0" class="batch-stock-dialog__hint">
            {{ t('products.batchStock.resultSummary', { created: newMasterCount, skipped: duplicateMasters.length }) }}
          </p>
        </template>
      </div>
    </div>

    <template #footer>
      <div class="batch-stock-dialog__footer">
        <NButton :disabled="submitting" @click="close">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('products.batchStock.confirmAction') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.batch-stock-dialog__body {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.batch-stock-dialog__section-title {
  margin: var(--space-2) 0 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.batch-stock-dialog__hint {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.batch-stock-dialog__dedup {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-3);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  background: var(--color-inset);
}

.batch-stock-dialog__dedup-warning {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--status-warning-fg);
}

.batch-stock-dialog__dedup-list {
  margin: 0;
  padding-left: var(--space-4);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  max-height: 140px;
  overflow-y: auto;
}

.batch-stock-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
