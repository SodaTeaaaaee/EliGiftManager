<script setup lang="ts">
/**
 * StepSampleUploadAndMapping — wizard step 3 (both create and remap modes).
 * Picks a local tabular sample (`pickTabularFile` + `parseTabularFile`), shows
 * detected header/row counts, then composes `FieldMappingEditor` for the
 * dest catalog appropriate to the surface (demand line/document/recipient, or
 * factory product/shipment/export namespaces).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { FieldMappingEditor, type FieldMappingDestField } from '@/shared/ui/field-mapping'
import { CalloutBar } from '@/shared/ui/guidance'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'
import {
  EXPORT_DEST_FIELDS,
  INTAKE_V2_DEST_FIELD_ORDER,
  PRODUCT_DEST_FIELDS,
  SHIPMENT_DEST_FIELDS,
  destFieldLabelKey,
  destFieldTooltipKey,
} from './destFields'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t, te } = useI18n({ useScope: 'global' })

function toDestFields(keys: readonly string[]): FieldMappingDestField[] {
  return keys.map((field) => {
    const labelKey = destFieldLabelKey(field)
    const tooltipKey = destFieldTooltipKey(field)
    return {
      key: field,
      label: te(labelKey) ? t(labelKey) : field,
      tooltip: te(tooltipKey) ? t(tooltipKey) : undefined,
    }
  })
}

const destFields = computed<FieldMappingDestField[]>(() => {
  if (!props.state.isFactorySurface.value) {
    return toDestFields(INTAKE_V2_DEST_FIELD_ORDER)
  }
  const keys: string[] = []
  const caps = props.state.factoryCapabilities
  if (caps.supportsImportProductCatalog) keys.push(...PRODUCT_DEST_FIELDS)
  if (caps.supportsImportSupplierShipment) keys.push(...SHIPMENT_DEST_FIELDS)
  if (caps.supportsExportSupplierOrder) keys.push(...EXPORT_DEST_FIELDS)
  // Fallback so the editor is never empty when operator has not toggled caps yet.
  if (keys.length === 0) keys.push(...PRODUCT_DEST_FIELDS)
  // Dedupe while preserving order.
  return toDestFields([...new Set(keys)])
})

function handlePick(): void {
  void props.state.pickAndParseFile()
}
</script>

<template>
  <div class="step-sample-upload">
    <CalloutBar
      v-if="state.isFactorySurface.value"
      tone="info"
      :message="t('intakeWizard.sampleUpload.factoryHint')"
    />

    <div class="step-sample-upload__pick-row">
      <NButton :loading="state.parsing.value" @click="handlePick">
        {{ t('intakeWizard.sampleUpload.pickButton') }}
      </NButton>
      <span v-if="state.parsing.value" class="step-sample-upload__status">{{ t('intakeWizard.sampleUpload.parsing') }}</span>
      <span v-else-if="!state.csvPath.value && !state.csvHeaders.value.length" class="step-sample-upload__status">
        {{
          state.isFactorySurface.value
            ? t('intakeWizard.sampleUpload.factoryOptional')
            : t('intakeWizard.sampleUpload.noFile')
        }}
      </span>
      <span v-else class="step-sample-upload__status">
        <template v-if="state.csvPath.value">{{ state.csvPath.value.split(/[/\\]/).pop() }} · </template>
        {{ t('intakeWizard.sampleUpload.headersDetected', { count: state.csvHeaders.value.length }) }}
        ·
        {{ t('intakeWizard.sampleUpload.rowsDetected', { count: state.csvRows.value.length }) }}
      </span>
    </div>

    <CalloutBar v-if="state.pickError.value" tone="error" :message="state.pickError.value" />

    <FieldMappingEditor
      v-if="state.csvPath.value || state.csvHeaders.value.length"
      :dest-fields="destFields"
      :source-headers="state.csvHeaders.value"
      :model-value="state.mapping.value"
      :sample-rows="state.csvRows.value"
      :dest-column-header="t('intakeWizard.mapping.destColumnHeader')"
      :src-column-header="t('intakeWizard.mapping.srcColumnHeader')"
      :preview-title="t('intakeWizard.mapping.previewTitle')"
      :unmapped-label="t('intakeWizard.mapping.unmapped')"
      :fixed-value-placeholder="t('intakeWizard.mapping.fixedValuePlaceholder')"
      :mode-label="t('intakeWizard.mapping.modeLabel')"
      :mode-header-label="t('intakeWizard.mapping.modeHeader')"
      :mode-positional-label="t('intakeWizard.mapping.modePositional')"
      :has-header-label="t('intakeWizard.mapping.hasHeaderLabel')"
      :position-placeholder="t('intakeWizard.mapping.positionPlaceholder')"
      :column-order-label="t('intakeWizard.mapping.columnOrderLabel')"
      :column-order-placeholder="t('intakeWizard.mapping.columnOrderPlaceholder')"
      @update:model-value="(value) => (state.mapping.value = value)"
    />
  </div>
</template>

<style scoped>
.step-sample-upload {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.step-sample-upload__pick-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.step-sample-upload__status {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>
