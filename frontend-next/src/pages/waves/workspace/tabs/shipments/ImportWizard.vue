<script setup lang="ts">
/**
 * ImportWizard — the CSV import half of the 发货回传 tab (plan 3.3.4 second
 * bullet). Fixes the old tree's two confirmed defects
 * (`frontend/src/pages/wave-workspace/WaveShipmentStep.vue`):
 *
 * 1. The column-mapping step only ever asks the operator to bind a
 *    reconciliation key (`lineId`, or `batchNo`+`supplierLineNo`) — never the
 *    internal `supplierOrderLineId`/`fulfillmentLineId` by name. Those
 *    mandatory wire-contract ids are resolved client-side via
 *    `useReconciliationIndex()` before `importShipments` is ever called.
 * 2. The import result is a SEPARATE, STICKY view (`ImportResultView`,
 *    rendered instead of the wizard once `importResult` is set) that is only
 *    cleared when the operator explicitly clicks "start a new import" — the
 *    old tree's `resetImportWizard()` wiped `importResult` immediately on
 *    partial success, discarding the very errors it had just collected.
 *
 * A row that cannot be resolved to a supplier order line (bad/missing
 * reconciliation key) is never sent to the backend as a garbage entry —
 * it's folded into the persisted result as a client-side error row,
 * side-by-side with the backend's own per-row `errors[]`, so the operator
 * sees every row's outcome in one place regardless of which side rejected it.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NSelect, NRadioGroup, NRadioButton } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { WizardFrame, type WizardStep } from '@/shared/ui/wizard'
import { FieldMappingEditor, applyMapping, type FieldMappingDestField, type FieldMappingValue } from '@/shared/ui/field-mapping'
import { CalloutBar } from '@/shared/ui/guidance'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import { useFeedback } from '@/shared/ui/feedback'
import { useGlossary } from '@/shared/i18n/glossary'
import { pickCsvFile, parseCSVFile, importShipments } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { useReconciliationIndex, batchLineNoKey, type ReconciliationLine } from './useReconciliationIndex'
import ImportResultView from './ImportResultView.vue'
import type { ImportResultViewData } from './ImportResultView.vue'
import type { dto } from '@/../wailsjs/go/models'

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const { label: glossaryLabel } = useGlossary()
const ctx = useWaveWorkspaceContext()

const emit = defineEmits<{ imported: [] }>()

const reconciliation = useReconciliationIndex()
watch(ctx.waveId, (id) => void reconciliation.load(id), { immediate: true })

// ── Step: upload ──

type StepKey = 'upload' | 'mapping' | 'preview'
const currentStep = ref<StepKey>('upload')
const wizardSteps = computed<WizardStep[]>(() => [
  { key: 'upload', title: t('waveWorkspace.shipments.import.steps.upload') },
  { key: 'mapping', title: t('waveWorkspace.shipments.import.steps.mapping') },
  { key: 'preview', title: t('waveWorkspace.shipments.import.steps.preview') },
])

const picking = ref(false)
const csvPreview = ref<dto.CSVFilePreviewDTO | null>(null)

async function handlePickFile(): Promise<void> {
  picking.value = true
  try {
    const path = await pickCsvFile()
    if (!path) return
    csvPreview.value = await parseCSVFile(path)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    picking.value = false
  }
}

// ── Step: mapping ──

const reconciliationKey = ref<'byLineId' | 'byBatchAndLineNo'>('byLineId')
const selectedOrderId = ref<number | null>(null)

const orderOptions = computed<SelectOption[]>(() =>
  reconciliation.index.value.orders.map((order) => ({
    label: `${order.batchNo} · ${glossaryLabel('supplierOrderStatus', order.status)}`,
    value: order.id,
  })),
)

watch(
  () => reconciliation.index.value.orders,
  (orders) => {
    if (selectedOrderId.value == null && orders.length > 0) selectedOrderId.value = orders[0].id
  },
  { immediate: true },
)

const mapping = ref<FieldMappingValue>({ columns: {}, defaults: {} })

// Selecting a supplier order pre-fills a default `batchNo` so a single-order
// CSV doesn't need its own batchNo column. A default always wins over a
// column mapping for the same destField (see FieldMappingEditor's own doc
// comment) — the operator can clear this default to fall back to a mapped
// column for a multi-order CSV.
watch(selectedOrderId, (id) => {
  const order = reconciliation.index.value.orders.find((candidate) => candidate.id === id)
  if (order) mapping.value = { ...mapping.value, defaults: { ...mapping.value.defaults, batchNo: order.batchNo } }
})

const keyDestFields = computed<FieldMappingDestField[]>(() =>
  reconciliationKey.value === 'byLineId'
    ? [{ key: 'lineId', label: t('waveWorkspace.shipments.import.mapping.lineId') }]
    : [
        { key: 'batchNo', label: t('waveWorkspace.shipments.import.mapping.batchNo') },
        { key: 'supplierLineNo', label: t('waveWorkspace.shipments.import.mapping.supplierLineNo') },
      ],
)

const destFields = computed<FieldMappingDestField[]>(() => [
  ...keyDestFields.value,
  { key: 'externalShipmentNo', label: t('waveWorkspace.shipments.import.mapping.externalShipmentNo') },
  { key: 'carrierCode', label: t('waveWorkspace.shipments.import.mapping.carrierCode') },
  { key: 'carrierName', label: t('waveWorkspace.shipments.import.mapping.carrierName') },
  { key: 'trackingNo', label: t('waveWorkspace.shipments.import.mapping.trackingNo') },
  { key: 'quantity', label: t('waveWorkspace.shipments.import.mapping.quantity') },
  { key: 'shippedAt', label: t('waveWorkspace.shipments.import.mapping.shippedAt') },
])

function validateShipmentField(destField: string, value: string): string | undefined {
  if (destField === 'quantity') {
    return /^\d+$/.test(value.trim()) && Number(value.trim()) > 0 ? undefined : 'invalid_quantity'
  }
  if (destField === 'supplierLineNo') {
    return value.trim() === '' || /^\d+$/.test(value.trim()) ? undefined : 'invalid_integer'
  }
  return undefined
}

function mappingHasField(field: string): boolean {
  return field in mapping.value.columns || field in mapping.value.defaults
}

const canProceedFromMapping = computed(() => {
  const hasKey = reconciliationKey.value === 'byLineId' ? mappingHasField('lineId') : mappingHasField('batchNo') && mappingHasField('supplierLineNo')
  return hasKey && mappingHasField('quantity')
})

// ── Step: preview + submit ──

const importMode = ref<'skip_invalid' | 'reject_all'>('skip_invalid')

const importModeOptions = computed<SelectOption[]>(() => [
  { label: t('waveWorkspace.shipments.import.importModeOptions.skip_invalid'), value: 'skip_invalid' },
  { label: t('waveWorkspace.shipments.import.importModeOptions.reject_all'), value: 'reject_all' },
])

interface ResolvedEntry {
  supplierOrderLineId: number
  fulfillmentLineId: number
  externalShipmentNo: string
  carrierCode: string
  carrierName: string
  trackingNo: string
  quantity: number
  shippedAt?: string
}

interface ParsedRow {
  originalRowIndex: number
  ok: boolean
  entry?: ResolvedEntry
  sol?: ReconciliationLine
  reason?: string
}

function unresolvedRowReason(): string {
  return t('waveWorkspace.shipments.import.mapping.unresolvedRowReason')
}

function resolveLine(values: Record<string, string>): ReconciliationLine | undefined {
  if (reconciliationKey.value === 'byLineId') {
    const id = Number((values.lineId ?? '').trim())
    return Number.isFinite(id) ? reconciliation.index.value.byLineId.get(id) : undefined
  }
  const batchNo = (values.batchNo ?? '').trim()
  const lineNo = Number((values.supplierLineNo ?? '').trim())
  if (!batchNo || !Number.isFinite(lineNo)) return undefined
  return reconciliation.index.value.byBatchAndLineNo.get(batchLineNoKey(batchNo, lineNo))
}

const parsedRows = computed<ParsedRow[]>(() => {
  if (!csvPreview.value) return []
  const mapped = applyMapping(csvPreview.value.rows, mapping.value.columns, mapping.value.defaults)
  return mapped.map((row, index) => {
    const values = row.values
    const sol = resolveLine(values)
    if (!sol) return { originalRowIndex: index, ok: false, reason: unresolvedRowReason() }
    const quantity = Number((values.quantity ?? '').trim())
    return {
      originalRowIndex: index,
      ok: true,
      sol,
      entry: {
        supplierOrderLineId: sol.lineId,
        fulfillmentLineId: sol.fulfillmentLineId,
        externalShipmentNo: values.externalShipmentNo ?? '',
        carrierCode: values.carrierCode ?? '',
        carrierName: values.carrierName ?? '',
        trackingNo: values.trackingNo ?? '',
        quantity: Number.isFinite(quantity) ? quantity : 0,
        shippedAt: values.shippedAt ? values.shippedAt : undefined,
      },
    }
  })
})

const resolvedRows = computed(() => parsedRows.value.filter((row): row is ParsedRow & { ok: true; entry: ResolvedEntry; sol: ReconciliationLine } => row.ok))
const unresolvedRows = computed(() => parsedRows.value.filter((row) => !row.ok))

interface PreviewRow {
  __rowKey: number
  rowNo: number
  supplierSku: string
  externalShipmentNo: string
  quantity: number
  trackingNo: string
}

const previewGridRows = computed<PreviewRow[]>(() =>
  resolvedRows.value.map((row) => ({
    __rowKey: row.originalRowIndex,
    rowNo: row.originalRowIndex + 1,
    supplierSku: row.sol.supplierSku,
    externalShipmentNo: row.entry.externalShipmentNo,
    quantity: row.entry.quantity,
    trackingNo: row.entry.trackingNo,
  })),
)

const previewColumns = computed(() =>
  createColumns<PreviewRow>([
    { key: 'rowNo', title: '#', type: 'number', width: 60 },
    { key: 'supplierSku', title: t('waveWorkspace.factory.lineColumns.supplierSku'), type: 'text' },
    { key: 'externalShipmentNo', title: t('waveWorkspace.shipments.import.mapping.externalShipmentNo'), type: 'text' },
    { key: 'quantity', title: t('waveWorkspace.shipments.manual.quantity'), type: 'number' },
    { key: 'trackingNo', title: t('waveWorkspace.shipments.import.mapping.trackingNo'), type: 'text' },
  ]),
)

const submitting = ref(false)
const importResult = ref<ImportResultViewData | null>(null)

const canSubmit = computed(() => resolvedRows.value.length > 0 && !submitting.value)

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value || !csvPreview.value) return
  submitting.value = true
  try {
    const entries = resolvedRows.value.map((row) => row.entry)
    const result = await importShipments({
      waveId: ctx.waveId.value,
      // ImportShipmentInput.integrationProfileId is part of the wire
      // contract but never read by shipmentImportUseCase.ImportShipments
      // (verified against internal/app/shipment_import_usecase.go) — no
      // profile selector was built since nothing server-side consumes it.
      integrationProfileId: 0,
      importMode: importMode.value,
      entries,
    })

    const rows = [
      ...unresolvedRows.value.map((row) => ({ rowNo: row.originalRowIndex + 1, reason: row.reason! })),
      ...result.errors.map((err) => ({
        rowNo: resolvedRows.value[err.entryIndex]!.originalRowIndex + 1,
        reason: err.reason,
      })),
    ].sort((a, b) => a.rowNo - b.rowNo)

    importResult.value = {
      total: csvPreview.value.rows.length,
      successCount: result.successCount,
      errorCount: result.errorCount + unresolvedRows.value.length,
      rows,
    }

    if (result.successCount > 0) {
      feedback.success(t('feedback.success'))
      await ctx.refresh()
      emit('imported')
    } else {
      feedback.error(t('feedback.error'))
    }
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}

function resetWizard(): void {
  currentStep.value = 'upload'
  csvPreview.value = null
  mapping.value = { columns: {}, defaults: {} }
  reconciliationKey.value = 'byLineId'
  importMode.value = 'skip_invalid'
  importResult.value = null
}

// ── Nav ──

const canNext = computed(() => {
  if (currentStep.value === 'upload') return !!csvPreview.value
  if (currentStep.value === 'mapping') return canProceedFromMapping.value
  return canSubmit.value
})

function handleNext(): void {
  if (currentStep.value === 'upload') currentStep.value = 'mapping'
  else if (currentStep.value === 'mapping') currentStep.value = 'preview'
}

function handleBack(): void {
  if (currentStep.value === 'preview') currentStep.value = 'mapping'
  else if (currentStep.value === 'mapping') currentStep.value = 'upload'
}
</script>

<template>
  <div class="import-wizard">
    <ImportResultView v-if="importResult" :result="importResult" @new-import="resetWizard" />

    <WizardFrame
      v-else
      :steps="wizardSteps"
      :current="currentStep"
      :can-next="canNext"
      :can-back="!submitting"
      :next-label="t('intakeWizard.nav.next')"
      :back-label="t('intakeWizard.nav.back')"
      :finish-label="t('waveWorkspace.shipments.import.preview.submit')"
      @next="handleNext"
      @back="handleBack"
      @finish="handleSubmit"
    >
      <template v-if="currentStep === 'upload'">
        <div class="import-wizard__upload">
          <p class="import-wizard__hint">{{ t('waveWorkspace.shipments.import.uploadHint') }}</p>
          <NButton :loading="picking" @click="handlePickFile">{{ t('waveWorkspace.shipments.import.pickFile') }}</NButton>
          <span v-if="picking" class="import-wizard__status">{{ t('intakeWizard.sampleUpload.parsing') }}</span>
          <span v-else-if="!csvPreview" class="import-wizard__status">{{ t('intakeWizard.sampleUpload.noFile') }}</span>
          <span v-else class="import-wizard__status">
            {{ t('intakeWizard.sampleUpload.headersDetected', { count: csvPreview.headers.length }) }}
            ·
            {{ t('intakeWizard.sampleUpload.rowsDetected', { count: csvPreview.rows.length }) }}
          </span>
        </div>
      </template>

      <template v-else-if="currentStep === 'mapping'">
        <div class="import-wizard__mapping">
          <div class="import-wizard__mapping-field">
            <span class="import-wizard__mapping-label">{{ t('waveWorkspace.shipments.import.supplierOrder') }}</span>
            <NSelect v-model:value="selectedOrderId" :options="orderOptions" filterable style="max-width: 360px" />
          </div>

          <div class="import-wizard__mapping-field">
            <span class="import-wizard__mapping-label">{{ t('waveWorkspace.shipments.import.mapping.reconciliationKey') }}</span>
            <NRadioGroup v-model:value="reconciliationKey">
              <NRadioButton value="byLineId">{{ t('waveWorkspace.shipments.import.mapping.byLineId') }}</NRadioButton>
              <NRadioButton value="byBatchAndLineNo">{{ t('waveWorkspace.shipments.import.mapping.byBatchAndLineNo') }}</NRadioButton>
            </NRadioGroup>
          </div>

          <CalloutBar tone="info" :message="t('waveWorkspace.shipments.import.mapping.hint')" />

          <FieldMappingEditor
            v-if="csvPreview"
            v-model:model-value="mapping"
            :dest-fields="destFields"
            :source-headers="csvPreview.headers"
            :sample-rows="csvPreview.rows"
            :dest-column-header="t('intakeWizard.mapping.destColumnHeader')"
            :src-column-header="t('intakeWizard.mapping.srcColumnHeader')"
            :preview-title="t('intakeWizard.mapping.previewTitle')"
            :unmapped-label="t('intakeWizard.mapping.unmapped')"
            :validate="validateShipmentField"
          />
        </div>
      </template>

      <template v-else-if="currentStep === 'preview'">
        <div class="import-wizard__preview">
          <div class="import-wizard__mapping-field">
            <span class="import-wizard__mapping-label">{{ t('waveWorkspace.shipments.import.importMode') }}</span>
            <NSelect v-model:value="importMode" :options="importModeOptions" style="max-width: 360px" />
          </div>

          <CalloutBar
            v-if="unresolvedRows.length > 0"
            tone="warning"
            :message="t('waveWorkspace.shipments.import.mapping.unresolved', { count: unresolvedRows.length })"
          />

          <p class="import-wizard__hint">{{ t('waveWorkspace.shipments.import.preview.rowCount', { count: previewGridRows.length }) }}</p>

          <DataGrid :columns="previewColumns" :rows="previewGridRows" row-key="__rowKey" pagination="client" />
        </div>
      </template>
    </WizardFrame>
  </div>
</template>

<style scoped>
.import-wizard {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.import-wizard__upload,
.import-wizard__mapping,
.import-wizard__preview {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.import-wizard__mapping-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.import-wizard__mapping-label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}

.import-wizard__hint {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.import-wizard__status {
  margin-left: var(--space-3);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>
