<script setup lang="ts">
/**
 * ImportResultView — the CSV import wizard's persistent result screen (P5
 * shipment-backfill, plan 3.3.4 second bullet's "导入结果页逐行渲染错误").
 *
 * Deliberately its OWN component, rendered by `ImportWizard.vue` in place of
 * (not alongside) `WizardFrame` once `importResult` is set, and only ever
 * cleared by the operator's explicit "start a new import" click
 * (`@new-import`). This is the direct fix for the old tree's confirmed bug
 * (`frontend/src/pages/wave-workspace/WaveShipmentStep.vue:234-238`):
 * `resetImportWizard()` wiped `importResult` immediately on any partial
 * success, discarding the very per-row errors it had just collected. Here,
 * nothing about mounting/showing this view ever clears `result` — only the
 * `new-import` emit (wired to the parent's own wizard-reset function) does.
 *
 * `rows` already merges TWO error sources into one line-by-line list
 * (built by `ImportWizard.vue`'s `handleSubmit`): rows this component's
 * caller could never resolve to a supplier order line client-side (bad/
 * missing reconciliation key) AND the backend's own per-row
 * `ImportShipmentResult.errors[]` — so every row's outcome is visible here
 * regardless of which side rejected it, satisfying skip_invalid mode's
 * "must list every skipped row" requirement.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { CalloutBar } from '@/shared/ui/guidance'
import ImportEvidenceReference from '@/shared/ui/customer-resolution/ImportEvidenceReference.vue'

export interface ImportResultViewRow {
  rowNo: number
  reason: string
}

export interface ImportResultViewData {
  importRunId: number
  evidenceDisabled: boolean
  total: number
  successCount: number
  errorCount: number
  rows: ImportResultViewRow[]
  /** Non-blocking, row-level mapping warnings (e.g. unknown mapping-dest
   * vocabulary) — values were still kept and imported. */
  warnings: string[]
}

const props = defineProps<{ result: ImportResultViewData }>()

const emit = defineEmits<{ 'new-import': [] }>()

const { t } = useI18n({ useScope: 'global' })

const hasErrors = computed(() => props.result.rows.length > 0)
const hasWarnings = computed(() => props.result.warnings.length > 0)
const summaryTone = computed(() => (props.result.errorCount > 0 ? 'warning' : 'success'))
</script>

<template>
  <SectionCard :title="t('waveWorkspace.shipments.import.result.title')">
    <template #description>{{ t('waveWorkspace.shipments.import.steps.result') }}</template>

    <div class="import-result-view">
      <CalloutBar
        :tone="summaryTone"
        :message="
          t('waveWorkspace.shipments.import.result.summary', {
            total: result.total,
            success: result.successCount,
            error: result.errorCount,
          })
        "
      />
      <ImportEvidenceReference :import-run-id="result.importRunId" :evidence-disabled="result.evidenceDisabled" />

      <div class="import-result-view__errors">
        <h4 class="import-result-view__errors-title">{{ t('waveWorkspace.shipments.import.result.errorsTitle') }}</h4>
        <p v-if="!hasErrors" class="import-result-view__no-errors">
          {{ t('waveWorkspace.shipments.import.result.noErrors') }}
        </p>
        <ul v-else class="import-result-view__error-list">
          <li v-for="row in result.rows" :key="row.rowNo" class="import-result-view__error-row">
            {{ t('waveWorkspace.shipments.import.result.errorRow', { index: row.rowNo, reason: row.reason }) }}
          </li>
        </ul>
      </div>

      <div v-if="hasWarnings" class="import-result-view__warnings">
        <CalloutBar
          tone="warning"
          :message="t('waveWorkspace.shipments.import.result.warningCount', { count: result.warnings.length })"
        />
        <ul class="import-result-view__warning-list">
          <li v-for="(warning, idx) in result.warnings" :key="idx" class="import-result-view__warning-row">
            {{ warning }}
          </li>
        </ul>
      </div>

      <div class="import-result-view__footer">
        <NButton type="primary" @click="emit('new-import')">
          {{ t('waveWorkspace.shipments.import.result.newImport') }}
        </NButton>
      </div>
    </div>
  </SectionCard>
</template>

<style scoped>
.import-result-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.import-result-view__errors-title {
  margin: 0 0 var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.import-result-view__no-errors {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.import-result-view__error-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  max-height: 320px;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  list-style: none;
}

.import-result-view__error-row {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--status-error-bg);
  color: var(--status-error-fg);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
}

.import-result-view__warnings {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.import-result-view__warning-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  max-height: 320px;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  list-style: none;
}

.import-result-view__warning-row {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--status-warning-bg);
  color: var(--status-warning-fg);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
}

.import-result-view__footer {
  display: flex;
  justify-content: flex-end;
}
</style>
