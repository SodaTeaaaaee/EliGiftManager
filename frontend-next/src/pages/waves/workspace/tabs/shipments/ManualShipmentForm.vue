<script setup lang="ts">
/**
 * ManualShipmentForm — manual shipment entry half of 发货回传 (plan 3.3.4
 * second bullet). Fixes the two confirmed defects in
 * `frontend/src/pages/wave-workspace/WaveShipmentStep.vue`:
 *
 * 1. Lines 332-344 hard-picked `order[0]` from `getSupplierOrderByWave`, so
 *    any second supplier order in a mixed wave was structurally unreachable
 *    from this form. `CreateShipment`'s own `SupplierOrderID` input is
 *    unrestricted server-side (verified against `shipment_usecase.go`) — the
 *    order selector below lists every order in the wave.
 * 2. Line 356 pre-filled the quantity input with the line's full
 *    `submittedQuantity` every time, with no subtraction of prior partial
 *    shipments and no client-side pre-check. This form fetches
 *    `getSupplierOrderLineShippedSummary(orderId)` per selected order and
 *    shows submitted/shipped/remaining per line, warning (not disabling —
 *    the server's `SumShippedQuantityBySOL` check is the actual
 *    enforcement point) when an entered quantity exceeds what remains.
 *
 * One shipment can cover multiple lines in a single submit (mirrors
 * `CreateShipmentInput.lines[]` — a real shipment often bundles several
 * SKUs), so the line list is a set of per-line quantity inputs, not a
 * single-line picker.
 */
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NForm, NFormItem, NSelect, NInput, NInputNumber, NDatePicker, NButton } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { useFeedback } from '@/shared/ui/feedback'
import { getSupplierOrderByWave, getSupplierOrderLineShippedSummary, createShipment } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import type { dto } from '@/../wailsjs/go/models'

const emit = defineEmits<{ created: [] }>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const ctx = useWaveWorkspaceContext()

// ── Supplier order selector (any order in the wave, not order[0]) ──

const orders = ref<dto.SupplierOrderDTO[]>([])
const loadingOrders = ref(false)
const selectedOrderId = ref<number | null>(null)

const orderOptions = computed<SelectOption[]>(() =>
  orders.value.map((order) => ({ label: order.batchNo || `#${order.id}`, value: order.id })),
)

async function loadOrders(): Promise<void> {
  loadingOrders.value = true
  try {
    orders.value = await getSupplierOrderByWave(ctx.waveId.value)
    if (selectedOrderId.value == null && orders.value.length > 0) selectedOrderId.value = orders.value[0].id
  } finally {
    loadingOrders.value = false
  }
}

watch(ctx.waveId, () => void loadOrders(), { immediate: true })

// ── Line list (shipped/remaining) ──

const lineSummaries = ref<dto.SupplierOrderLineShippedDTO[]>([])
const loadingSummary = ref(false)
const quantities = reactive<Record<number, number | null>>({})

function clearQuantities(): void {
  for (const key of Object.keys(quantities)) delete quantities[Number(key)]
}

async function loadSummary(orderId: number): Promise<void> {
  loadingSummary.value = true
  try {
    lineSummaries.value = await getSupplierOrderLineShippedSummary(orderId)
  } catch {
    lineSummaries.value = []
  } finally {
    loadingSummary.value = false
  }
  clearQuantities()
  for (const line of lineSummaries.value) quantities[line.lineId] = null
}

watch(
  selectedOrderId,
  (id) => {
    if (id != null) void loadSummary(id)
    else {
      lineSummaries.value = []
      clearQuantities()
    }
  },
  { immediate: true },
)

function isOverShip(line: dto.SupplierOrderLineShippedDTO): boolean {
  const qty = quantities[line.lineId]
  return qty != null && qty > line.remainingQuantity
}

const hasAnyQuantity = computed(() => Object.values(quantities).some((qty) => (qty ?? 0) > 0))

// ── Header fields + submit ──

const externalShipmentNo = ref('')
const carrierCode = ref('')
const carrierName = ref('')
const trackingNo = ref('')
// Non-nullable (unlike the optional dto field) because the bridge wrapper's
// `createShipment` input types `shippedAt` as a required `string` — default
// to "now" rather than allowing an empty/unparsable value through.
const shippedAtMs = ref<number>(Date.now())
const submitting = ref(false)

const canSubmit = computed(
  () =>
    !submitting.value &&
    selectedOrderId.value != null &&
    hasAnyQuantity.value &&
    externalShipmentNo.value.trim().length > 0,
)

function resetForm(): void {
  externalShipmentNo.value = ''
  carrierCode.value = ''
  carrierName.value = ''
  trackingNo.value = ''
  shippedAtMs.value = Date.now()
  for (const line of lineSummaries.value) quantities[line.lineId] = null
}

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value || selectedOrderId.value == null) return
  const order = orders.value.find((candidate) => candidate.id === selectedOrderId.value)
  if (!order) return

  const lines = lineSummaries.value
    .filter((line) => (quantities[line.lineId] ?? 0) > 0)
    .map((line) => ({
      supplierOrderLineId: line.lineId,
      fulfillmentLineId: line.fulfillmentLineId,
      quantity: quantities[line.lineId] as number,
    }))
  if (lines.length === 0) return

  submitting.value = true
  try {
    await createShipment({
      supplierOrderId: order.id,
      supplierPlatform: order.supplierPlatform,
      // ShipmentNo/Status mirror the CSV import path's own defaults
      // (`shipment_import_usecase.go:276,284` — "IMP-{waveId}-{n}" /
      // "shipped") — neither is a plan-required operator-facing field for
      // this form; ShipmentNo only needs to be unique.
      shipmentNo: `MAN-${ctx.waveId.value}-${Date.now()}`,
      externalShipmentNo: externalShipmentNo.value.trim(),
      carrierCode: carrierCode.value.trim(),
      carrierName: carrierName.value.trim(),
      trackingNo: trackingNo.value.trim(),
      status: 'shipped',
      shippedAt: new Date(shippedAtMs.value).toISOString(),
      basisPayloadSnapshot: '',
      lines,
    })
    feedback.success(t('waveWorkspace.shipments.manual.success'))
    resetForm()
    await Promise.all([loadSummary(order.id), ctx.refresh()])
    emit('created')
  } catch (err) {
    feedback.error(t('waveWorkspace.shipments.manual.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <SectionCard :title="t('waveWorkspace.shipments.manual.title')">
    <EmptyState
      v-if="!loadingOrders && orders.length === 0"
      size="sm"
      :title="t('waveWorkspace.shipments.manual.supplierOrderPlaceholder')"
    />

    <div v-else class="manual-shipment-form">
      <NForm label-placement="top">
        <NFormItem :label="t('waveWorkspace.shipments.manual.supplierOrder')">
          <NSelect
            v-model:value="selectedOrderId"
            :options="orderOptions"
            :loading="loadingOrders"
            :placeholder="t('waveWorkspace.shipments.manual.supplierOrderPlaceholder')"
            filterable
            style="max-width: 360px"
          />
        </NFormItem>
      </NForm>

      <div class="manual-shipment-form__lines">
        <h4 class="manual-shipment-form__lines-title">{{ t('waveWorkspace.shipments.manual.line') }}</h4>
        <p v-if="loadingSummary" class="manual-shipment-form__pending">
          {{ t('waveWorkspace.shipments.manual.remainingPending') }}
        </p>
        <div v-for="line in lineSummaries" :key="line.lineId" class="manual-shipment-form__line-row">
          <div class="manual-shipment-form__line-meta">
            <span class="manual-shipment-form__line-sku">{{ line.supplierSku }}</span>
            <span class="manual-shipment-form__line-stat">
              {{ t('waveWorkspace.shipments.manual.submittedQuantity') }}: {{ line.submittedQuantity }}
              · {{ t('waveWorkspace.shipments.manual.shippedQuantity') }}: {{ line.shippedQuantity }}
              · {{ t('waveWorkspace.shipments.manual.remainingQuantity') }}: {{ line.remainingQuantity }}
            </span>
          </div>
          <NInputNumber
            v-model:value="quantities[line.lineId]"
            :min="0"
            :precision="0"
            :placeholder="t('waveWorkspace.shipments.manual.quantity')"
            :disabled="submitting"
            style="width: 160px"
          />
          <span v-if="isOverShip(line)" class="manual-shipment-form__overship">
            {{ t('waveWorkspace.shipments.manual.overShipWarning') }}
          </span>
        </div>
      </div>

      <NForm label-placement="top" class="manual-shipment-form__header-fields">
        <NFormItem :label="t('waveWorkspace.shipments.manual.externalShipmentNo')">
          <NInput v-model:value="externalShipmentNo" :disabled="submitting" />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.shipments.manual.carrierCode')">
          <NInput v-model:value="carrierCode" :disabled="submitting" />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.shipments.manual.carrierName')">
          <NInput v-model:value="carrierName" :disabled="submitting" />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.shipments.manual.trackingNo')">
          <NInput v-model:value="trackingNo" :disabled="submitting" />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.shipments.manual.shippedAt')">
          <NDatePicker v-model:value="shippedAtMs" type="datetime" style="width: 100%" :disabled="submitting" />
        </NFormItem>
      </NForm>

      <div class="manual-shipment-form__footer">
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('waveWorkspace.shipments.manual.submit') }}
        </NButton>
      </div>
    </div>
  </SectionCard>
</template>

<style scoped>
.manual-shipment-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.manual-shipment-form__lines {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.manual-shipment-form__lines-title {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.manual-shipment-form__pending {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.manual-shipment-form__line-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--card-border-color);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.manual-shipment-form__line-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 200px;
  flex: 1 1 260px;
}

.manual-shipment-form__line-sku {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.manual-shipment-form__line-stat {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.manual-shipment-form__overship {
  flex: 1 1 100%;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--status-warning-fg);
}

.manual-shipment-form__header-fields {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 0 var(--space-3);
}

.manual-shipment-form__footer {
  display: flex;
  justify-content: flex-end;
}
</style>
