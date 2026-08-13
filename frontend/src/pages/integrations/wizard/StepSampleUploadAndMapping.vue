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
import { NButton, NInput, NSwitch } from 'naive-ui'
import {
  FieldMappingEditor,
  type CatalogImageLayoutValue,
  type FieldMappingDestField,
} from '@/shared/ui/field-mapping'
import { CalloutBar } from '@/shared/ui/guidance'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'
import {
  destFieldLabelKey,
  destFieldTooltipKey,
  destKeysForDocumentType,
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

const sessionDocType = computed(
  () => props.state.sessionDocumentType.value || props.state.documentType.value,
)

const destFields = computed<FieldMappingDestField[]>(() =>
  toDestFields(destKeysForDocumentType(sessionDocType.value)),
)

const inputFormat = computed(() => props.state.csvPath.value.split('.').pop()?.toUpperCase() ?? '')
const isCatalogZip = computed(() =>
  sessionDocType.value === 'import_product_catalog' &&
  props.state.csvPath.value.toLowerCase().endsWith('.zip'),
)
const imageLayout = computed<CatalogImageLayoutValue>(() =>
  props.state.mapping.value.imageLayout ?? { enabled: true },
)

function patchImageLayout(partial: Partial<CatalogImageLayoutValue>): void {
  props.state.mapping.value = {
    ...props.state.mapping.value,
    imageLayout: { ...imageLayout.value, ...partial },
  }
}

function patchImageExtensions(value: string): void {
  patchImageLayout({
    imageExts: value.split(/[,，\s]+/).map((part) => part.trim()).filter(Boolean),
  })
}

function handlePick(): void {
  void props.state.pickAndParseFile()
}
</script>

<template>
  <div class="step-sample-upload">
    <CalloutBar
      tone="info"
      :message="t('intakeWizard.sampleUpload.singleTypeHint')"
    />

    <div class="step-sample-upload__pick-row">
      <NButton :loading="state.parsing.value" @click="handlePick">
        {{ t('intakeWizard.sampleUpload.pickButton') }}
      </NButton>
      <span v-if="state.parsing.value" class="step-sample-upload__status">{{ t('intakeWizard.sampleUpload.parsing') }}</span>
      <span v-else-if="!state.csvPath.value && !state.csvHeaders.value.length" class="step-sample-upload__status">
        {{ t('intakeWizard.sampleUpload.noFile') }}
      </span>
      <span v-else class="step-sample-upload__status">
        <template v-if="state.csvPath.value">{{ state.csvPath.value.split(/[/\\]/).pop() }} · </template>
        {{ t('intakeWizard.sampleUpload.headersDetected', { count: state.csvHeaders.value.length }) }}
        ·
        {{ t('intakeWizard.sampleUpload.rowsDetected', { count: state.csvRows.value.length }) }}
      </span>
    </div>

    <CalloutBar v-if="state.pickError.value" tone="error" :message="state.pickError.value" />

    <div v-if="isCatalogZip" class="step-sample-upload__image-layout">
      <label class="step-sample-upload__layout-toggle">
        <NSwitch
          :value="imageLayout.enabled"
          @update:value="(value) => patchImageLayout({ enabled: value })"
        />
        <span>{{ t('intakeWizard.sampleUpload.imageLayoutEnabled') }}</span>
      </label>
      <NInput
        :value="imageLayout.tabularGlob"
        :placeholder="t('intakeWizard.sampleUpload.tabularGlob')"
        @update:value="(value) => patchImageLayout({ tabularGlob: value })"
      />
      <NInput
        :value="imageLayout.matchField"
        :placeholder="t('intakeWizard.sampleUpload.imageMatchField')"
        @update:value="(value) => patchImageLayout({ matchField: value })"
      />
      <NInput
        :value="imageLayout.coverDir"
        :placeholder="t('intakeWizard.sampleUpload.coverDir')"
        @update:value="(value) => patchImageLayout({ coverDir: value })"
      />
      <NInput
        :value="imageLayout.detailDir"
        :placeholder="t('intakeWizard.sampleUpload.detailDir')"
        @update:value="(value) => patchImageLayout({ detailDir: value })"
      />
      <NInput
        :value="imageLayout.namePattern"
        :placeholder="t('intakeWizard.sampleUpload.imageNamePattern')"
        @update:value="(value) => patchImageLayout({ namePattern: value })"
      />
      <NInput
        :value="(imageLayout.imageExts ?? []).join(', ')"
        :placeholder="t('intakeWizard.sampleUpload.imageExtensions')"
        @update:value="patchImageExtensions"
      />
      <CalloutBar tone="info" :message="t('intakeWizard.sampleUpload.zipManualHeaders')" />
    </div>

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
      :input-format="inputFormat"
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

.step-sample-upload__image-layout {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-2);
}

.step-sample-upload__layout-toggle {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.step-sample-upload__image-layout :deep(.callout-bar) {
  grid-column: 1 / -1;
}
</style>
