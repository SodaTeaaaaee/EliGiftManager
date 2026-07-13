<script setup lang="ts">
/**
 * SupplierOrderCard — one supplier order's independently-operable card (P5
 * factory-orders sub-area, plan 3.3.4 first bullet's "多订单并列卡片，各自
 * 独立操作"). All state here is keyed off `order.id` (no shared/global
 * card-index state), so N sibling cards on one wave never interfere.
 *
 * Action gating mirrors the server's hard-checked state machine
 * (`supplier_order_lifecycle_usecase.go`): "mark submitted" only when
 * `status==='draft'`, "record acceptance" only when `status==='submitted'`.
 * "Generate file" has no status gate — regeneration stays possible at any
 * status (`GenerateSupplierOrderFile` is not idempotent/versioned; every
 * click writes a fresh timestamped file, see `bridge.ts`'s doc comment).
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { StatusBadge } from '@/shared/ui/status'
import { DataGrid, createColumns, type DataGridColumnSpec } from '@/shared/ui/data-grid'
import { useFeedback } from '@/shared/ui/feedback'
import { generateSupplierOrderFile, revealInFolder } from '@/shared/api/bridge'
import type { dto } from '@/../wailsjs/go/models'
import MarkSubmittedDialog from './MarkSubmittedDialog.vue'
import RecordAcceptanceDialog from './RecordAcceptanceDialog.vue'

const props = defineProps<{
  order: dto.SupplierOrderDTO
  lines: dto.SupplierOrderLineDTO[]
}>()

const emit = defineEmits<{
  /** Fires after any lifecycle mutation on this card — parent re-`loadAll()`s. */
  changed: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const canMarkSubmitted = computed(() => props.order.status === 'draft')
const canRecordAcceptance = computed(() => props.order.status === 'submitted')

const columns = computed(() =>
  createColumns<dto.SupplierOrderLineDTO>([
    {
      key: 'supplierLineNo',
      title: t('waveWorkspace.factory.lineColumns.supplierLineNo'),
      type: 'number',
      getValue: (line) => line.supplierLineNo ?? null,
    },
    {
      key: 'supplierSku',
      title: t('waveWorkspace.factory.lineColumns.supplierSku'),
      type: 'text',
    },
    {
      key: 'submittedQuantity',
      title: t('waveWorkspace.factory.lineColumns.submittedQuantity'),
      type: 'number',
    },
    {
      key: 'acceptedQuantity',
      title: t('waveWorkspace.factory.lineColumns.acceptedQuantity'),
      type: 'number',
      getValue: (line) => line.acceptedQuantity ?? null,
    },
    {
      key: 'status',
      title: t('waveWorkspace.factory.lineColumns.status'),
      type: 'status',
      dimension: 'supplierOrderStatus',
      size: 'sm',
    },
    {
      key: 'fulfillmentLineId',
      title: t('waveWorkspace.factory.lineColumns.fulfillmentLineId'),
      type: 'number',
    },
  ] satisfies DataGridColumnSpec<dto.SupplierOrderLineDTO>[]),
)

// ── Generate file ──

const generatingFile = ref(false)
const lastFileResult = ref<dto.SupplierOrderFileResultDTO | null>(null)
const openingFolder = ref(false)

async function handleGenerateFile(): Promise<void> {
  generatingFile.value = true
  try {
    const result = await generateSupplierOrderFile(props.order.id)
    lastFileResult.value = result
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    generatingFile.value = false
  }
}

async function handleOpenFolder(): Promise<void> {
  const path = lastFileResult.value?.filePath
  if (!path) return
  openingFolder.value = true
  try {
    await revealInFolder(path)
  } catch (err) {
    feedback.error(t('waveWorkspace.factory.generateFile.openFolderError'), err instanceof Error ? err.message : String(err))
  } finally {
    openingFolder.value = false
  }
}

async function handleCopyPath(): Promise<void> {
  const path = lastFileResult.value?.filePath
  if (!path) return
  try {
    await navigator.clipboard.writeText(path)
    feedback.success(t('waveWorkspace.factory.generateFile.copySuccess'))
  } catch {
    // Clipboard API unavailable/denied (e.g. non-secure context) — the path
    // text is still visible/selectable on the card, so this is non-fatal.
  }
}

// ── Mark submitted / record acceptance dialogs ──

const showMarkSubmittedDialog = ref(false)
const showRecordAcceptanceDialog = ref(false)

function handleMarkSubmittedDone(): void {
  emit('changed')
}

function handleRecordAcceptanceDone(): void {
  emit('changed')
}
</script>

<template>
  <SectionCard flat>
    <template #title>
      <span class="supplier-order-card__title">
        {{ t('waveWorkspace.factory.card.batchNo') }}: {{ order.batchNo }}
      </span>
    </template>
    <template #actions>
      <StatusBadge dimension="supplierOrderStatus" :value="order.status" show-dot />
    </template>

    <dl class="supplier-order-card__meta">
      <div class="supplier-order-card__meta-item">
        <dt>{{ t('waveWorkspace.factory.card.externalOrderNo') }}</dt>
        <dd>{{ order.externalOrderNo || '—' }}</dd>
      </div>
      <div class="supplier-order-card__meta-item">
        <dt>{{ t('waveWorkspace.factory.card.submissionMode') }}</dt>
        <dd>
          <StatusBadge v-if="order.submissionMode" dimension="submissionMode" :value="order.submissionMode" />
          <span v-else>—</span>
        </dd>
      </div>
      <div class="supplier-order-card__meta-item">
        <dt>{{ t('waveWorkspace.factory.card.submittedAt') }}</dt>
        <dd>{{ order.submittedAt || '—' }}</dd>
      </div>
      <div class="supplier-order-card__meta-item">
        <dt>{{ t('waveWorkspace.factory.card.basisNode') }}</dt>
        <dd>{{ order.basisHistoryNodeId || '—' }}</dd>
      </div>
    </dl>

    <div class="supplier-order-card__toolbar">
      <NButton size="small" :loading="generatingFile" @click="handleGenerateFile">
        {{ t('waveWorkspace.factory.generateFile.action') }}
      </NButton>
      <NButton
        size="small"
        :disabled="!canMarkSubmitted"
        :title="!canMarkSubmitted ? t('waveWorkspace.factory.markSubmitted.disabledHint') : undefined"
        @click="showMarkSubmittedDialog = true"
      >
        {{ t('waveWorkspace.factory.markSubmitted.action') }}
      </NButton>
      <NButton
        size="small"
        :disabled="!canRecordAcceptance"
        :title="!canRecordAcceptance ? t('waveWorkspace.factory.recordAcceptance.disabledHint') : undefined"
        @click="showRecordAcceptanceDialog = true"
      >
        {{ t('waveWorkspace.factory.recordAcceptance.action') }}
      </NButton>
    </div>

    <div v-if="lastFileResult" class="supplier-order-card__file-result">
      <p class="supplier-order-card__file-path">
        {{ t('waveWorkspace.factory.generateFile.result', { path: lastFileResult.filePath }) }}
      </p>
      <p class="supplier-order-card__file-meta">
        {{ t('waveWorkspace.factory.generateFile.lineCount', { count: lastFileResult.lineCount }) }}
        ·
        {{ t('waveWorkspace.factory.generateFile.generatedAt', { time: lastFileResult.generatedAt }) }}
      </p>
      <div class="supplier-order-card__file-actions">
        <NButton size="tiny" :loading="openingFolder" @click="handleOpenFolder">
          {{ t('waveWorkspace.factory.generateFile.openFolder') }}
        </NButton>
        <NButton size="tiny" @click="handleCopyPath">
          {{ t('waveWorkspace.factory.generateFile.copyPath') }}
        </NButton>
      </div>
      <p class="supplier-order-card__reconciliation-hint">
        {{ t('waveWorkspace.factory.generateFile.reconciliationHint') }}
      </p>
    </div>

    <h4 class="supplier-order-card__lines-title">{{ t('waveWorkspace.factory.card.lines') }}</h4>
    <DataGrid :columns="columns" :rows="lines" row-key="id" pagination="none" />

    <MarkSubmittedDialog v-model:show="showMarkSubmittedDialog" :order="order" @done="handleMarkSubmittedDone" />
    <RecordAcceptanceDialog
      v-model:show="showRecordAcceptanceDialog"
      :order="order"
      :lines="lines"
      @done="handleRecordAcceptanceDone"
    />
  </SectionCard>
</template>

<style scoped>
.supplier-order-card__title {
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.supplier-order-card__meta {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: var(--space-3);
  margin: 0 0 var(--space-3);
}

.supplier-order-card__meta-item dt {
  margin: 0 0 2px;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.supplier-order-card__meta-item dd {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.supplier-order-card__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.supplier-order-card__file-result {
  margin-bottom: var(--space-4);
  padding: var(--space-3);
  border: 1px solid var(--card-border-color);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.supplier-order-card__file-path {
  margin: 0 0 var(--space-1);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  overflow-wrap: anywhere;
}

.supplier-order-card__file-meta {
  margin: 0 0 var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.supplier-order-card__file-actions {
  display: flex;
  gap: var(--space-2);
  margin-bottom: var(--space-2);
}

.supplier-order-card__reconciliation-hint {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.supplier-order-card__lines-title {
  margin: 0 0 var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}
</style>
