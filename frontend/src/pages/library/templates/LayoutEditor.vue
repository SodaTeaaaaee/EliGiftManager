<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NRadio, NRadioGroup } from 'naive-ui'
import { EmptyState } from '@/shared/ui/empty-state'
import type { FieldMappingDestField, FieldMappingGroup } from '@/shared/ui/field-mapping'
import {
  moveLayoutColumn,
  placeholderSampleFor,
  type LayoutColumn,
  type LayoutConfigValue,
  type LayoutFormat,
} from './layoutConfig'

/**
 * LayoutEditor — the 输出布局 section for output templates (factory_order,
 * writeback). Left: the semantic dictionary grouped by the five dictionary
 * groups, click to append. Right: the ordered output columns with move
 * up/down, an inline display-header input (falls back to the localized
 * label when blank), and remove. Below: the format radio and a live header
 * preview (header row + one placeholder data row). Serializes to
 * LayoutConfig v1 through `./layoutConfig`.
 */
const props = defineProps<{
  modelValue: LayoutConfigValue
  destFields: FieldMappingDestField[]
  groups: FieldMappingGroup[]
}>()

const emit = defineEmits<{
  'update:modelValue': [LayoutConfigValue]
}>()

const { t } = useI18n()

function patch(partial: Partial<LayoutConfigValue>): void {
  emit('update:modelValue', { ...props.modelValue, ...partial })
}

const fieldByKey = computed(() => new Map(props.destFields.map((field) => [field.key, field])))

function labelFor(key: string): string {
  return fieldByKey.value.get(key)?.label ?? key
}

const selectedKeys = computed(() => new Set(props.modelValue.columns.map((column) => column.key)))

interface AvailableGroup {
  key: string
  label: string
  fields: FieldMappingDestField[]
}

/** Unselected dictionary keys bucketed by group; empty groups disappear. */
const availableGroups = computed<AvailableGroup[]>(() => {
  const remaining = props.destFields.filter((field) => !selectedKeys.value.has(field.key))
  const out: AvailableGroup[] = []
  for (const group of props.groups) {
    const fields = remaining.filter((field) => field.group === group.key)
    if (fields.length) out.push({ key: group.key, label: group.label, fields })
  }
  const ungrouped = remaining.filter((field) => !props.groups.some((group) => group.key === field.group))
  if (ungrouped.length) out.push({ key: '', label: t('templates.layoutEditor.otherGroup'), fields: ungrouped })
  return out
})

function handleAdd(key: string): void {
  if (selectedKeys.value.has(key)) return
  patch({ columns: [...props.modelValue.columns, { key, header: '' }] })
}

function handleRemove(index: number): void {
  patch({ columns: props.modelValue.columns.filter((_, i) => i !== index) })
}

function handleMove(index: number, delta: -1 | 1): void {
  patch({ columns: moveLayoutColumn(props.modelValue.columns, index, delta) })
}

function handleHeaderChange(index: number, header: string): void {
  const columns: LayoutColumn[] = props.modelValue.columns.map((column, i) =>
    i === index ? { ...column, header } : column,
  )
  patch({ columns })
}

function handleFormatChange(format: string | number | boolean): void {
  patch({ format: format === 'xlsx' ? 'xlsx' : ('csv' as LayoutFormat) })
}

function displayHeader(column: LayoutColumn): string {
  const header = column.header.trim()
  return header !== '' ? header : labelFor(column.key)
}

const previewHeaderRow = computed(() => props.modelValue.columns.map(displayHeader))
const previewSampleRow = computed(() =>
  props.modelValue.columns.map((column) => placeholderSampleFor(column.key, labelFor(column.key))),
)
</script>

<template>
  <div class="layout-editor">
    <div class="layout-editor__format">
      <span class="layout-editor__label">{{ t('templates.layoutFormat') }}</span>
      <NRadioGroup :value="modelValue.format" @update:value="handleFormatChange">
        <NRadio value="csv">{{ t('templates.formatCsv') }}</NRadio>
        <NRadio value="xlsx">{{ t('templates.formatXlsx') }}</NRadio>
      </NRadioGroup>
    </div>

    <div class="layout-editor__columns">
      <section class="layout-editor__pane">
        <h5 class="layout-editor__pane-title">{{ t('templates.layoutEditor.available') }}</h5>
        <p class="layout-editor__pane-hint">{{ t('templates.layoutEditor.availableHint') }}</p>
        <div v-if="!availableGroups.length" class="layout-editor__pane-empty">
          {{ t('templates.layoutEditor.allSelected') }}
        </div>
        <div v-for="group in availableGroups" :key="group.key || '__other'" class="layout-editor__group">
          <div class="layout-editor__group-title">{{ group.label }}</div>
          <div class="layout-editor__chips">
            <button
              v-for="field in group.fields"
              :key="field.key"
              type="button"
              class="layout-editor__chip"
              :title="field.key"
              @click="handleAdd(field.key)"
            >
              <span class="layout-editor__chip-plus" aria-hidden="true">+</span>
              <span>{{ field.label }}</span>
            </button>
          </div>
        </div>
      </section>

      <section class="layout-editor__pane">
        <h5 class="layout-editor__pane-title">
          {{ t('templates.layoutEditor.selected', { n: modelValue.columns.length }) }}
        </h5>
        <p class="layout-editor__pane-hint">{{ t('templates.layoutEditor.selectedHint') }}</p>
        <div v-if="!modelValue.columns.length" class="layout-editor__pane-empty">
          <EmptyState :title="t('templates.layoutEditor.noColumns')" size="sm" />
        </div>
        <ol v-else class="layout-editor__selected">
          <li
            v-for="(column, index) in modelValue.columns"
            :key="column.key"
            class="layout-editor__selected-row"
          >
            <span class="layout-editor__order">{{ index + 1 }}</span>
            <div class="layout-editor__selected-field">
              <span class="layout-editor__selected-label">{{ labelFor(column.key) }}</span>
              <code class="layout-editor__selected-key">{{ column.key }}</code>
            </div>
            <NInput
              size="small"
              class="layout-editor__header-input"
              :value="column.header"
              :placeholder="t('templates.layoutEditor.headerPlaceholder', { label: labelFor(column.key) })"
              @update:value="(v) => handleHeaderChange(index, v)"
            />
            <div class="layout-editor__row-ops">
              <NButton
                size="tiny"
                quaternary
                :disabled="index === 0"
                :aria-label="t('templates.layoutEditor.moveUp')"
                :title="t('templates.layoutEditor.moveUp')"
                @click="handleMove(index, -1)"
              >
                ↑
              </NButton>
              <NButton
                size="tiny"
                quaternary
                :disabled="index === modelValue.columns.length - 1"
                :aria-label="t('templates.layoutEditor.moveDown')"
                :title="t('templates.layoutEditor.moveDown')"
                @click="handleMove(index, 1)"
              >
                ↓
              </NButton>
              <NButton
                size="tiny"
                quaternary
                type="error"
                :aria-label="t('templateEditor.remove')"
                :title="t('templateEditor.remove')"
                @click="handleRemove(index)"
              >
                ×
              </NButton>
            </div>
          </li>
        </ol>
      </section>
    </div>

    <section class="layout-editor__preview">
      <h5 class="layout-editor__pane-title">{{ t('templates.layoutEditor.headerPreview') }}</h5>
      <p class="layout-editor__pane-hint">{{ t('templates.layoutEditor.headerPreviewHint') }}</p>
      <div v-if="!modelValue.columns.length" class="layout-editor__pane-empty">
        {{ t('templates.layoutEditor.noColumns') }}
      </div>
      <div v-else class="layout-editor__preview-scroll">
        <table class="layout-editor__preview-table">
          <thead>
            <tr>
              <th v-for="(header, index) in previewHeaderRow" :key="`h-${index}`">{{ header }}</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td v-for="(cell, index) in previewSampleRow" :key="`s-${index}`">{{ cell }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.layout-editor {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.layout-editor__format {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.layout-editor__label {
  min-width: 120px;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}

.layout-editor__columns {
  display: grid;
  grid-template-columns: minmax(240px, 2fr) minmax(320px, 3fr);
  gap: var(--space-4);
  align-items: start;
}

.layout-editor__pane {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--card-border-color);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.layout-editor__pane-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.layout-editor__pane-hint {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.layout-editor__pane-empty {
  padding: var(--space-3) 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.layout-editor__group {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin-top: var(--space-2);
}

.layout-editor__group-title {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-secondary);
}

.layout-editor__chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}

.layout-editor__chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px var(--space-2);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-primary);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}

.layout-editor__chip:hover {
  border-color: var(--color-accent);
  background: var(--color-accent-subtle);
}

.layout-editor__chip:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.layout-editor__chip-plus {
  color: var(--color-accent);
  font-weight: var(--font-weight-semibold);
}

.layout-editor__selected {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.layout-editor__selected-row {
  display: grid;
  grid-template-columns: 28px minmax(120px, 1fr) minmax(140px, 1fr) auto;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) 0;
  border-bottom: 1px solid var(--card-border-color);
}

.layout-editor__selected-row:last-child {
  border-bottom: none;
}

.layout-editor__order {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  text-align: right;
}

.layout-editor__selected-field {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.layout-editor__selected-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.layout-editor__selected-key {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.layout-editor__header-input {
  min-width: 0;
}

.layout-editor__row-ops {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.layout-editor__preview {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.layout-editor__preview-scroll {
  overflow-x: auto;
  border: 1px solid var(--card-border-color);
  border-radius: var(--radius-md);
}

.layout-editor__preview-table {
  border-collapse: collapse;
  min-width: 100%;
  font-size: var(--font-size-sm);
}

.layout-editor__preview-table th,
.layout-editor__preview-table td {
  padding: var(--space-2) var(--space-3);
  border-right: 1px solid var(--card-border-color);
  white-space: nowrap;
  text-align: left;
}

.layout-editor__preview-table th:last-child,
.layout-editor__preview-table td:last-child {
  border-right: none;
}

.layout-editor__preview-table th {
  background: var(--color-inset);
  color: var(--color-text-secondary);
  font-weight: var(--font-weight-semibold);
  border-bottom: 1px solid var(--card-border-color);
}

.layout-editor__preview-table td {
  color: var(--color-text-primary);
  font-variant-numeric: tabular-nums;
}
</style>
