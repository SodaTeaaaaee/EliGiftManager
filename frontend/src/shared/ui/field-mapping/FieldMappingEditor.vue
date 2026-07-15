<script setup lang="ts">
import { computed, h } from 'vue'
import { NSelect, NInput, NTooltip } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import type { FieldMappingDestField, FieldMappingValue } from './types'
import { applyMapping, validateDestFieldValue } from './previewTransform'

/**
 * FieldMappingEditor — the visual CSV column-mapping widget (P4
 * demand-intake wizard's mapping step; reusable later for e.g. a P5
 * shipment-CSV mapping UI). Left column lists the destination fields
 * (label + tooltip); right column is a dropdown binding each destField to
 * a parsed CSV source header, plus a fixed-value input for destFields with
 * no natural per-row column (e.g. a constant `line_type`). A live preview
 * of the first 5 sample rows re-renders on every mapping change, with a
 * red highlight on cells that fail `validate` — 100% client-side, no
 * backend round-trip.
 *
 * `validate` defaults to `validateDestFieldValue` (the demand-intake rule:
 * only `requested_quantity` must parse as an integer); pass a different
 * function to reuse this editor for a domain with different validation
 * rules.
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
    /** Per-cell validator deciding the red-highlight. Defaults to the demand-intake rule. */
    validate?: (destField: string, value: string) => string | undefined
  }>(),
  {
    fixedValuePlaceholder: undefined,
    validate: () => validateDestFieldValue,
  },
)

const emit = defineEmits<{
  'update:modelValue': [FieldMappingValue]
}>()

const sourceHeaderOptions = computed<SelectOption[]>(() =>
  props.sourceHeaders.map((header) => ({ label: header, value: header })),
)

function columnValue(destField: string): string | null {
  return props.modelValue.columns[destField] ?? null
}

function defaultValue(destField: string): string {
  return props.modelValue.defaults[destField] ?? ''
}

function isUnmapped(destField: string): boolean {
  return !(destField in props.modelValue.columns) && !(destField in props.modelValue.defaults)
}

function handleColumnChange(destField: string, value: string | null) {
  const columns = { ...props.modelValue.columns }
  if (value) columns[destField] = value
  else delete columns[destField]
  emit('update:modelValue', { columns, defaults: props.modelValue.defaults })
}

function handleDefaultChange(destField: string, value: string) {
  const defaults = { ...props.modelValue.defaults }
  if (value !== '') defaults[destField] = value
  else delete defaults[destField]
  emit('update:modelValue', { columns: props.modelValue.columns, defaults })
}

interface PreviewRow {
  __previewRowIndex: number
  [destFieldKey: string]: string | number
}

const previewRows = computed<PreviewRow[]>(() => {
  const mapped = applyMapping(props.sampleRows.slice(0, 5), props.modelValue.columns, props.modelValue.defaults)
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
        const error = props.validate(field.key, value)
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
    <div class="field-mapping-editor__mapping">
      <div class="field-mapping-editor__mapping-header">
        <span class="field-mapping-editor__mapping-header-cell">{{ destColumnHeader }}</span>
        <span class="field-mapping-editor__mapping-header-cell">{{ srcColumnHeader }}</span>
      </div>
      <div v-for="field in destFields" :key="field.key" class="field-mapping-editor__row">
        <div class="field-mapping-editor__field-label">
          <span>{{ field.label }}</span>
          <NTooltip v-if="field.tooltip" trigger="hover">
            <template #trigger>
              <span class="field-mapping-editor__hint-icon" aria-hidden="true">?</span>
            </template>
            {{ field.tooltip }}
          </NTooltip>
        </div>
        <div class="field-mapping-editor__field-controls">
          <NSelect
            class="field-mapping-editor__column-select"
            :value="columnValue(field.key)"
            :options="sourceHeaderOptions"
            clearable
            filterable
            :placeholder="unmappedLabel"
            @update:value="(value) => handleColumnChange(field.key, value)"
          />
          <NInput
            class="field-mapping-editor__default-input"
            :value="defaultValue(field.key)"
            :placeholder="fixedValuePlaceholder"
            @update:value="(value) => handleDefaultChange(field.key, value)"
          />
        </div>
      </div>
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

.field-mapping-editor__hint-icon {
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
  align-items: center;
  gap: var(--space-2);
}

.field-mapping-editor__column-select {
  flex: 1;
  min-width: 0;
}

.field-mapping-editor__default-input {
  flex: 1;
  min-width: 0;
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
