<script setup lang="ts">
/**
 * StepSampleUploadAndMapping — wizard step 3 (both create and remap modes).
 * Picks a local CSV sample (`pickCsvFile` + `parseCSVFile`, bridge.ts),
 * shows detected header/row counts, then composes `FieldMappingEditor` for
 * the 12 canonical destFields against the real parsed headers/rows.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { FieldMappingEditor, type FieldMappingDestField } from '@/shared/ui/field-mapping'
import { CalloutBar } from '@/shared/ui/guidance'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'
import { INTAKE_DEST_FIELD_ORDER, destFieldLabelKey, destFieldTooltipKey } from './destFields'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t } = useI18n({ useScope: 'global' })

const destFields = computed<FieldMappingDestField[]>(() =>
  INTAKE_DEST_FIELD_ORDER.map((field) => ({
    key: field,
    label: t(destFieldLabelKey(field)),
    tooltip: t(destFieldTooltipKey(field)),
  })),
)

function handlePick(): void {
  void props.state.pickAndParseFile()
}
</script>

<template>
  <div class="step-sample-upload">
    <div class="step-sample-upload__pick-row">
      <NButton :loading="state.parsing.value" @click="handlePick">
        {{ t('intakeWizard.sampleUpload.pickButton') }}
      </NButton>
      <span v-if="state.parsing.value" class="step-sample-upload__status">{{ t('intakeWizard.sampleUpload.parsing') }}</span>
      <span v-else-if="!state.csvHeaders.value.length" class="step-sample-upload__status">{{ t('intakeWizard.sampleUpload.noFile') }}</span>
      <span v-else class="step-sample-upload__status">
        {{ t('intakeWizard.sampleUpload.headersDetected', { count: state.csvHeaders.value.length }) }}
        ·
        {{ t('intakeWizard.sampleUpload.rowsDetected', { count: state.csvRows.value.length }) }}
      </span>
    </div>

    <CalloutBar v-if="state.pickError.value" tone="error" :message="state.pickError.value" />

    <FieldMappingEditor
      v-if="state.csvHeaders.value.length"
      :dest-fields="destFields"
      :source-headers="state.csvHeaders.value"
      :model-value="state.mapping.value"
      :sample-rows="state.csvRows.value"
      :dest-column-header="t('intakeWizard.mapping.destColumnHeader')"
      :src-column-header="t('intakeWizard.mapping.srcColumnHeader')"
      :preview-title="t('intakeWizard.mapping.previewTitle')"
      :unmapped-label="t('intakeWizard.mapping.unmapped')"
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
