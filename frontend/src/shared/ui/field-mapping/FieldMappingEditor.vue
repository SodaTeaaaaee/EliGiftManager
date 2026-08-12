<script setup lang="ts">
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSelect, NInput, NInputNumber, NRadioGroup, NRadioButton, NSwitch } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import type { FieldMappingDestField, FieldMappingMode, FieldMappingValue } from './types'
import { applyMapping, validateDestFieldValue } from './previewTransform'

/**
 * FieldMappingEditor — visual column/position mapping widget.
 * Supports MappingRules v2: mode header|positional, hasHeader, columnOrder.
 * Left column lists dest fields; right column binds each to a source header
 * (header mode) or 0-based cell index (positional mode), plus a fixed-value
 * input. Live preview of the first 5 sample rows re-renders on change.
 */
const props = withDefaults(
  defineProps<{
    destFields: FieldMappingDestField[]
    /** Real parsed CSV headers (`CSVFilePreviewDTO.headers`). */
    sourceHeaders: string[]
    modelValue: FieldMappingValue
    /** First N parsed CSV rows — only the first 5 are used for the preview. */
    sampleRows: Record<string, string>[]
    /** Already-resolved header for the destField list column. */
    destColumnHeader: string
    /** Already-resolved header for the source-header dropdown column. */
    srcColumnHeader: string
    /** Already-resolved title above the live preview table. */
    previewTitle: string
    /** Already-resolved placeholder for an unbound dropdown / unmapped preview cell. */
    unmappedLabel: string
    /** Already-resolved placeholder for the fixed-value input. */
    fixedValuePlaceholder?: string
    /** Labels for mode / hasHeader / columnOrder controls (already translated). */
    modeLabel?: string
    modeHeaderLabel?: string
    modePositionalLabel?: string
    hasHeaderLabel?: string
    positionPlaceholder?: string
    columnOrderLabel?: string
    columnOrderPlaceholder?: string
    /** Read-only review mode for template-driven imports. */
    readonly?: boolean
    /** Optional detected source metadata. */
    inputFormat?: string
    sheetName?: string
    /** Per-cell validator deciding the red-highlight. Defaults to the demand-intake rule. */
    validate?: (destField: string, value: string) => string | undefined
  }>(),
  {
    fixedValuePlaceholder: undefined,
    modeLabel: undefined,
    modeHeaderLabel: undefined,
    modePositionalLabel: undefined,
    hasHeaderLabel: undefined,
    positionPlaceholder: undefined,
    columnOrderLabel: undefined,
    columnOrderPlaceholder: undefined,
    readonly: false,
    inputFormat: undefined,
    sheetName: undefined,
    // Function prop defaults are used as-is (not as factories) — must be the validator itself.
    validate: (destField: string, value: string) => validateDestFieldValue(destField, value),
  },
)

const { t } = useI18n({ useScope: 'global' })
const resolvedModeHeaderLabel = computed(
  () => props.modeHeaderLabel ?? t('intakeWizard.mapping.modeHeader'),
)
const resolvedModePositionalLabel = computed(
  () => props.modePositionalLabel ?? t('intakeWizard.mapping.modePositional'),
)

const emit = defineEmits<{
  'update:modelValue': [FieldMappingValue]
}>()

const mode = computed<FieldMappingMode>(() =>
  props.modelValue.mode === 'positional' ? 'positional' : 'header',
)

const hasHeader = computed(() => props.modelValue.hasHeader !== false)

const sourceHeaderOptions = computed<SelectOption[]>(() =>
  props.sourceHeaders.map((header) => ({ label: header, value: header })),
)

const transformOptions = computed<SelectOption[]>(() => [
  { label: t('intakeWizard.mapping.transformTrim'), value: 'trim' },
  { label: t('intakeWizard.mapping.transformStripQuotes'), value: 'strip_quotes' },
  { label: t('intakeWizard.mapping.transformStripLeadingQuote'), value: 'strip_leading_quote' },
])

const mappedSourceHeaders = computed(() => new Set(Object.values(props.modelValue.columns ?? {})))
const unmappedSourceHeaders = computed(() =>
  mode.value === 'header'
    ? props.sourceHeaders.filter((header) => !mappedSourceHeaders.value.has(header))
    : [],
)

function patch(partial: Partial<FieldMappingValue>): void {
  emit('update:modelValue', {
    version: props.modelValue.version ?? 2,
    mode: props.modelValue.mode ?? 'header',
    hasHeader: props.modelValue.hasHeader ?? true,
    columns: props.modelValue.columns ?? {},
    positions: props.modelValue.positions ?? {},
    defaults: props.modelValue.defaults ?? {},
    transforms: props.modelValue.transforms,
    columnOrder: props.modelValue.columnOrder,
    required: props.modelValue.required,
    sheetName: props.modelValue.sheetName,
    imageLayout: props.modelValue.imageLayout,
    ...partial,
  })
}

function handleModeChange(next: FieldMappingMode): void {
  patch({ mode: next, version: 2 })
}

function handleHasHeaderChange(next: boolean): void {
  patch({ hasHeader: next, version: 2 })
}

function handleSheetNameChange(next: string): void {
  patch({ sheetName: next })
}

function columnValue(destField: string): string | null {
  return props.modelValue.columns[destField] ?? null
}

function positionValue(destField: string): number | null {
  const positions = props.modelValue.positions ?? {}
  return destField in positions ? positions[destField]! : null
}

function defaultValue(destField: string): string {
  return props.modelValue.defaults[destField] ?? ''
}

function isUnmapped(destField: string): boolean {
  if (mode.value === 'positional') {
    return !(destField in (props.modelValue.positions ?? {})) && !(destField in props.modelValue.defaults)
  }
  return !(destField in props.modelValue.columns) && !(destField in props.modelValue.defaults)
}

function handleColumnChange(destField: string, value: string | null): void {
  const columns = { ...props.modelValue.columns }
  if (value) columns[destField] = value
  else delete columns[destField]
  patch({ columns })
}

function handlePositionChange(destField: string, value: number | null): void {
  const positions = { ...(props.modelValue.positions ?? {}) }
  if (value !== null && value !== undefined && Number.isFinite(value) && value >= 0) {
    positions[destField] = Math.floor(value)
  } else {
    delete positions[destField]
  }
  patch({ positions, version: 2 })
}

function handleDefaultChange(destField: string, value: string): void {
  const defaults = { ...props.modelValue.defaults }
  if (value !== '') defaults[destField] = value
  else delete defaults[destField]
  patch({ defaults })
}

function transformValue(destField: string): string[] {
  return props.modelValue.transforms?.[destField] ?? []
}

function handleTransformChange(destField: string, value: string[]): void {
  const transforms = { ...(props.modelValue.transforms ?? {}) }
  if (value.length > 0) transforms[destField] = value
  else delete transforms[destField]
  patch({ transforms })
}

function isRequired(destField: string): boolean {
  return (props.modelValue.required ?? []).includes(destField)
}

function handleRequiredChange(destField: string, value: boolean): void {
  const required = new Set(props.modelValue.required ?? [])
  if (value) required.add(destField)
  else required.delete(destField)
  patch({ required: [...required] })
}

const columnOrderText = computed(() => (props.modelValue.columnOrder ?? []).join(', '))

function handleColumnOrderChange(value: string): void {
  const parts = value
    .split(/[,，\n]/)
    .map((part) => part.trim())
    .filter(Boolean)
  patch({ columnOrder: parts, version: 2 })
}

interface PreviewRow {
  __previewRowIndex: number
  [destFieldKey: string]: string | number
}

const previewRows = computed<PreviewRow[]>(() => {
  const mapped = applyMapping(props.sampleRows.slice(0, 5), props.modelValue, undefined, props.sourceHeaders)
  return mapped.map((row, index) => ({ __previewRowIndex: index, ...row.values }))
})

const previewColumns = computed(() =>
  createColumns<PreviewRow>(
    props.destFields.map((field) => ({
      key: field.key,
      title: field.label,
      type: 'text' as const,
      sortable: false,
      render: (row: PreviewRow) => {
        if (isUnmapped(field.key)) {
          return h('span', { class: 'field-mapping-editor__cell field-mapping-editor__cell--unmapped' }, props.unmappedLabel)
        }
        const value = (row[field.key] as string | undefined) ?? ''
        const error = isRequired(field.key) && value.trim() === ''
          ? 'required'
          : props.validate(field.key, value)
        return h(
          'span',
          {
            class: ['field-mapping-editor__cell', { 'field-mapping-editor__cell--invalid': !!error }],
          },
          value,
        )
      },
    })),
  ),
)
</script>

<template>
  <div class="field-mapping-editor">
    <div class="field-mapping-editor__meta">
      <div v-if="inputFormat || sheetName" class="field-mapping-editor__source-meta">
        <span v-if="inputFormat">{{ t('intakeWizard.mapping.inputFormat') }}: {{ inputFormat }}</span>
        <span v-if="sheetName">{{ t('intakeWizard.mapping.sheetName') }}: {{ sheetName }}</span>
      </div>
      <div
        v-if="['XLS', 'XLSX'].includes((inputFormat ?? '').toUpperCase()) || modelValue.sheetName"
        class="field-mapping-editor__meta-row"
      >
        <span class="field-mapping-editor__meta-label">{{ t('intakeWizard.mapping.sheetName') }}</span>
        <NInput
          :value="modelValue.sheetName ?? sheetName ?? ''"
          :disabled="readonly"
          @update:value="handleSheetNameChange"
        />
      </div>
      <div v-if="modeLabel" class="field-mapping-editor__meta-row">
        <span class="field-mapping-editor__meta-label">{{ modeLabel }}</span>
        <NRadioGroup :value="mode" :disabled="readonly" @update:value="(v) => handleModeChange(v as FieldMappingMode)">
          <NRadioButton value="header" :disabled="readonly">{{ resolvedModeHeaderLabel }}</NRadioButton>
          <NRadioButton value="positional" :disabled="readonly">{{ resolvedModePositionalLabel }}</NRadioButton>
        </NRadioGroup>
      </div>
      <div v-if="hasHeaderLabel" class="field-mapping-editor__meta-row">
        <span class="field-mapping-editor__meta-label">{{ hasHeaderLabel }}</span>
        <NSwitch :value="hasHeader" :disabled="readonly" @update:value="handleHasHeaderChange" />
      </div>
      <div v-if="columnOrderLabel" class="field-mapping-editor__meta-row field-mapping-editor__meta-row--wide">
        <span class="field-mapping-editor__meta-label">{{ columnOrderLabel }}</span>
        <NInput
          :value="columnOrderText"
          :placeholder="columnOrderPlaceholder"
          :disabled="readonly"
          @update:value="handleColumnOrderChange"
        />
      </div>
    </div>

    <div class="field-mapping-editor__mapping">
      <div class="field-mapping-editor__mapping-header">
        <span class="field-mapping-editor__mapping-header-cell">{{ destColumnHeader }}</span>
        <span class="field-mapping-editor__mapping-header-cell">{{ srcColumnHeader }}</span>
      </div>
      <div v-for="field in destFields" :key="field.key" class="field-mapping-editor__row">
        <div class="field-mapping-editor__field-label">
          <span>{{ field.label }}</span>
          <span v-if="field.tooltip" class="field-mapping-editor__hint" :title="field.tooltip">?</span>
        </div>
        <div class="field-mapping-editor__field-controls">
          <NSelect
            v-if="mode === 'header'"
            class="field-mapping-editor__column-select"
            :value="columnValue(field.key)"
            :options="sourceHeaderOptions"
            clearable
            filterable
            :tag="!readonly"
            :placeholder="unmappedLabel"
            :disabled="readonly"
            @update:value="(value) => handleColumnChange(field.key, value)"
          />
          <NInputNumber
            v-else
            class="field-mapping-editor__position-input"
            :value="positionValue(field.key)"
            :min="0"
            :precision="0"
            :placeholder="positionPlaceholder ?? '0'"
            clearable
            :disabled="readonly"
            @update:value="(value) => handlePositionChange(field.key, value)"
          />
          <NInput
            class="field-mapping-editor__default-input"
            :value="defaultValue(field.key)"
            :placeholder="fixedValuePlaceholder"
            :disabled="readonly"
            @update:value="(value) => handleDefaultChange(field.key, value)"
          />
          <NSelect
            class="field-mapping-editor__transform-select"
            :value="transformValue(field.key)"
            :options="transformOptions"
            :disabled="readonly"
            multiple
            clearable
            :placeholder="t('intakeWizard.mapping.transformsLabel')"
            @update:value="(value) => handleTransformChange(field.key, value)"
          />
          <label class="field-mapping-editor__required-control">
            <NSwitch
              :value="isRequired(field.key)"
              :disabled="readonly"
              @update:value="(value) => handleRequiredChange(field.key, value)"
            />
            <span>{{ t('intakeWizard.mapping.requiredLabel') }}</span>
          </label>
        </div>
      </div>
    </div>

    <div v-if="unmappedSourceHeaders.length" class="field-mapping-editor__unmapped-sources">
      <strong>{{ t('intakeWizard.mapping.unmappedSourceColumns') }}</strong>
      <span>{{ unmappedSourceHeaders.join(', ') }}</span>
    </div>

    <div class="field-mapping-editor__preview">
      <h4 class="field-mapping-editor__preview-title">{{ previewTitle }}</h4>
      <DataGrid :columns="previewColumns" :rows="previewRows" row-key="__previewRowIndex" pagination="none" />
    </div>
  </div>
</template>

<style scoped>
.field-mapping-editor {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.field-mapping-editor__meta {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.field-mapping-editor__source-meta,
.field-mapping-editor__unmapped-sources {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.field-mapping-editor__unmapped-sources {
  padding: var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--status-warning-bg);
  color: var(--status-warning-fg);
}

.field-mapping-editor__meta-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.field-mapping-editor__meta-row--wide {
  flex-direction: column;
  align-items: stretch;
}

.field-mapping-editor__meta-label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
  min-width: 120px;
}

.field-mapping-editor__mapping {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.field-mapping-editor__mapping-header {
  display: grid;
  grid-template-columns: minmax(160px, 1fr) minmax(220px, 1.4fr);
  gap: var(--space-4);
  padding: 0 var(--space-1);
}

.field-mapping-editor__mapping-header-cell {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.field-mapping-editor__row {
  display: grid;
  grid-template-columns: minmax(160px, 1fr) minmax(220px, 1.4fr);
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-2) var(--space-1);
  border-bottom: 1px solid var(--card-border-color);
}

.field-mapping-editor__row:last-child {
  border-bottom: none;
}

.field-mapping-editor__field-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.field-mapping-editor__hint {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 1px solid var(--color-border-strong);
  color: var(--color-text-muted);
  font-size: 10px;
  line-height: 1;
  cursor: help;
  flex-shrink: 0;
}

.field-mapping-editor__field-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}

.field-mapping-editor__column-select,
.field-mapping-editor__position-input {
  flex: 1;
  min-width: 0;
}

.field-mapping-editor__default-input {
  flex: 1;
  min-width: 0;
}

.field-mapping-editor__transform-select {
  flex: 2 1 220px;
}

.field-mapping-editor__required-control {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.field-mapping-editor__preview-title {
  margin: 0 0 var(--space-2);
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.field-mapping-editor__cell {
  display: inline-block;
}

.field-mapping-editor__cell--unmapped {
  color: var(--color-text-muted);
  font-style: italic;
}

.field-mapping-editor__cell--invalid {
  color: var(--status-error-fg);
  background: var(--status-error-bg);
  border-radius: var(--radius-sm);
  padding: 0 var(--space-1);
}
</style>
