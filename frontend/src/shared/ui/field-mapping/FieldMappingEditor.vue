<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NCheckbox,
  NInput,
  NInputNumber,
  NRadioButton,
  NRadioGroup,
  NSelect,
  NSwitch,
} from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import type { FieldMappingDestField, FieldMappingMode, FieldMappingValue } from './types'
import { applyMapping } from './previewTransform'

/**
 * FieldMappingEditor — visual column/position mapping widget for MappingConfig
 * v3. Left column lists semantic keys (built by the page from the backend
 * semantic dictionary); the right side binds each to a source header name or
 * 0-based cell position, a fixed default, a transform chain, a required flag,
 * and a fingerprint flag. Collapsible advanced sections edit splitSkuQuantity,
 * joinSources (joinAddress), and enumMaps (mapEnum). The live preview only
 * mirrors the cheap local transforms; full parsing goes through the backend
 * PreviewTemplate use case.
 */
const props = withDefaults(
  defineProps<{
    destFields: FieldMappingDestField[]
    /** Real parsed source headers; empty list still allows typed header names. */
    sourceHeaders: string[]
    modelValue: FieldMappingValue
    /** First N parsed source rows — the first 5 feed the local preview. */
    sampleRows: Record<string, string>[]
    /** Read-only review mode. */
    readonly?: boolean
  }>(),
  {
    readonly: false,
  },
)

const { t } = useI18n({ useScope: 'global' })

const emit = defineEmits<{
  'update:modelValue': [FieldMappingValue]
}>()

const showAdvanced = ref(false)

const mode = computed<FieldMappingMode>(() =>
  props.modelValue.mode === 'positional' ? 'positional' : 'header',
)

function patch(partial: Partial<FieldMappingValue>): void {
  emit('update:modelValue', {
    version: props.modelValue.version ?? 3,
    mode: props.modelValue.mode ?? 'header',
    hasHeader: (props.modelValue.mode ?? 'header') === 'header',
    columns: props.modelValue.columns ?? {},
    positions: props.modelValue.positions ?? {},
    defaults: props.modelValue.defaults ?? {},
    transforms: props.modelValue.transforms,
    enumMaps: props.modelValue.enumMaps,
    joinSources: props.modelValue.joinSources,
    splitSkuQuantity: props.modelValue.splitSkuQuantity,
    required: props.modelValue.required,
    fingerprint: props.modelValue.fingerprint,
    sheetName: props.modelValue.sheetName,
    ...partial,
  })
}

function handleModeChange(next: FieldMappingMode): void {
  // Mode drives hasHeader in MappingConfig v3 (backend normalize()).
  patch({ mode: next, hasHeader: next === 'header' })
}

function handleSheetNameChange(next: string): void {
  patch({ sheetName: next.trim() !== '' ? next : undefined })
}

const sourceHeaderOptions = computed<SelectOption[]>(() =>
  props.sourceHeaders.map((header) => ({ label: header, value: header })),
)

/** joinSources refs: header names in header mode, decimal indexes otherwise. */
const sourceRefOptions = computed<SelectOption[]>(() => {
  if (mode.value === 'header') return sourceHeaderOptions.value
  const sampleWidth = props.sampleRows[0] ? Object.keys(props.sampleRows[0]).length : 0
  const width = Math.max(props.sourceHeaders.length, sampleWidth, 16)
  return Array.from({ length: width }, (_, i) => ({ label: String(i), value: String(i) }))
})

const transformOptions = computed<SelectOption[]>(() => [
  { label: t('templateEditor.transform.trim'), value: 'trim' },
  { label: t('templateEditor.transform.strip_quotes'), value: 'strip_quotes' },
  { label: t('templateEditor.transform.parseDate'), value: 'parseDate' },
  { label: t('templateEditor.transform.mapEnum'), value: 'mapEnum' },
  { label: t('templateEditor.transform.normalizePhone'), value: 'normalizePhone' },
  { label: t('templateEditor.transform.joinAddress'), value: 'joinAddress' },
])

const mappedSourceHeaders = computed(() => new Set(Object.values(props.modelValue.columns ?? {})))
const unmappedSourceHeaders = computed(() =>
  mode.value === 'header'
    ? props.sourceHeaders.filter((header) => !mappedSourceHeaders.value.has(header))
    : [],
)

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
  const bound =
    mode.value === 'positional'
      ? destField in (props.modelValue.positions ?? {})
      : destField in props.modelValue.columns
  return (
    !bound &&
    !(destField in (props.modelValue.joinSources ?? {})) &&
    !(destField in props.modelValue.defaults)
  )
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
  patch({ positions })
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

/**
 * Transforms run as an ordered chain on the backend, so the editor manages
 * them as an ordered list: one chip per selected transform with move-up /
 * move-down / remove, plus a single-select "append" picker fed by the
 * not-yet-selected names (each name makes sense at most once per chain).
 */
function transformDisplayName(name: string): string {
  const opt = transformOptions.value.find((o) => o.value === name)
  return opt ? String(opt.label) : name
}

function remainingTransformOptions(destField: string): SelectOption[] {
  const selected = new Set(transformValue(destField))
  return transformOptions.value.filter((o) => !selected.has(String(o.value)))
}

function handleTransformAdd(destField: string, value: string | number | null): void {
  if (value == null) return
  handleTransformChange(destField, [...transformValue(destField), String(value)])
}

function handleTransformMove(destField: string, index: number, delta: -1 | 1): void {
  const chain = [...transformValue(destField)]
  const target = index + delta
  if (target < 0 || target >= chain.length) return
  ;[chain[index], chain[target]] = [chain[target], chain[index]]
  handleTransformChange(destField, chain)
}

function handleTransformRemove(destField: string, index: number): void {
  handleTransformChange(
    destField,
    transformValue(destField).filter((_, i) => i !== index),
  )
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

function isFingerprint(destField: string): boolean {
  return (props.modelValue.fingerprint ?? []).includes(destField)
}

function handleFingerprintChange(destField: string, value: boolean): void {
  const fingerprint = new Set(props.modelValue.fingerprint ?? [])
  if (value) fingerprint.add(destField)
  else fingerprint.delete(destField)
  patch({ fingerprint: [...fingerprint] })
}

// ── Advanced: splitSkuQuantity / joinSources / enumMaps ──

const destKeyOptions = computed<SelectOption[]>(() =>
  props.destFields.map((field) => ({ label: field.label, value: field.key })),
)

function handleSplitSkuQuantityChange(value: string | null): void {
  patch({ splitSkuQuantity: value ?? undefined })
}

const joinSourceEntries = computed(() =>
  Object.keys(props.modelValue.joinSources ?? {}).sort().map((key) => ({
    key,
    field: props.destFields.find((f) => f.key === key),
    refs: props.modelValue.joinSources?.[key] ?? [],
  })),
)

const joinKeyToAdd = ref<string | null>(null)
const joinKeyOptions = computed<SelectOption[]>(() => {
  const taken = new Set(Object.keys(props.modelValue.joinSources ?? {}))
  return destKeyOptions.value.filter((opt) => !taken.has(String(opt.value)))
})

function handleAddJoinKey(key: string | null): void {
  if (!key) return
  const joinSources = { ...(props.modelValue.joinSources ?? {}) }
  if (!(key in joinSources)) joinSources[key] = []
  patch({ joinSources })
  joinKeyToAdd.value = null
}

function handleJoinRefsChange(key: string, refs: string[]): void {
  const joinSources = { ...(props.modelValue.joinSources ?? {}) }
  if (refs.length > 0) joinSources[key] = refs
  else delete joinSources[key]
  patch({ joinSources })
}

const enumMapEntries = computed(() =>
  Object.keys(props.modelValue.enumMaps ?? {}).sort().map((key) => ({
    key,
    field: props.destFields.find((f) => f.key === key),
    pairs: Object.entries(props.modelValue.enumMaps?.[key] ?? {}),
  })),
)

const enumKeyToAdd = ref<string | null>(null)
const enumKeyOptions = computed<SelectOption[]>(() => {
  const taken = new Set(Object.keys(props.modelValue.enumMaps ?? {}))
  return destKeyOptions.value.filter((opt) => !taken.has(String(opt.value)))
})

function handleAddEnumKey(key: string | null): void {
  if (!key) return
  const enumMaps = { ...(props.modelValue.enumMaps ?? {}) }
  if (!(key in enumMaps)) enumMaps[key] = {}
  patch({ enumMaps })
  enumKeyToAdd.value = null
}

function handleRemoveEnumKey(key: string): void {
  const enumMaps = { ...(props.modelValue.enumMaps ?? {}) }
  delete enumMaps[key]
  patch({ enumMaps })
}

function handleEnumPairChange(
  key: string,
  oldExternal: string,
  nextExternal: string,
  nextInternal: string,
): void {
  const enumMaps = { ...(props.modelValue.enumMaps ?? {}) }
  const table = { ...(enumMaps[key] ?? {}) }
  delete table[oldExternal]
  // Keep the row visible while either side still has content; incomplete
  // pairs are dropped again on serialize.
  if (nextExternal !== '' || nextInternal !== '') table[nextExternal] = nextInternal
  enumMaps[key] = table
  patch({ enumMaps })
}

function handleAddEnumPair(key: string): void {
  const enumMaps = { ...(props.modelValue.enumMaps ?? {}) }
  const table = { ...(enumMaps[key] ?? {}) }
  let index = 1
  while (table[`__${index}`] !== undefined) index++
  table[`__${index}`] = ''
  enumMaps[key] = table
  patch({ enumMaps })
}

function handleRemoveEnumPair(key: string, external: string): void {
  const enumMaps = { ...(props.modelValue.enumMaps ?? {}) }
  const table = { ...(enumMaps[key] ?? {}) }
  delete table[external]
  enumMaps[key] = table
  patch({ enumMaps })
}

// ── Local preview ──

interface PreviewRow {
  __previewRowIndex: number
  [destFieldKey: string]: string | number
}

const previewRows = computed<PreviewRow[]>(() => {
  const mapped = applyMapping(props.sampleRows.slice(0, 5), props.modelValue, props.sourceHeaders)
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
          return h(
            'span',
            { class: 'field-mapping-editor__cell field-mapping-editor__cell--unmapped' },
            t('templateEditor.unmapped'),
          )
        }
        const value = (row[field.key] as string | undefined) ?? ''
        const invalid = isRequired(field.key) && value.trim() === ''
        return h(
          'span',
          {
            class: [
              'field-mapping-editor__cell',
              { 'field-mapping-editor__cell--invalid': invalid },
            ],
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
      <div class="field-mapping-editor__meta-row">
        <span class="field-mapping-editor__meta-label">{{ t('templateEditor.mode') }}</span>
        <NRadioGroup
          :value="mode"
          :disabled="readonly"
          @update:value="(v) => handleModeChange(v as FieldMappingMode)"
        >
          <NRadioButton value="header" :disabled="readonly">
            {{ t('templateEditor.modeHeader') }}
          </NRadioButton>
          <NRadioButton value="positional" :disabled="readonly">
            {{ t('templateEditor.modePositional') }}
          </NRadioButton>
        </NRadioGroup>
      </div>
      <div class="field-mapping-editor__meta-row">
        <span class="field-mapping-editor__meta-label">{{ t('templateEditor.sheetName') }}</span>
        <NInput
          class="field-mapping-editor__sheet-input"
          :value="modelValue.sheetName ?? ''"
          :placeholder="t('templateEditor.sheetNamePlaceholder')"
          :disabled="readonly"
          @update:value="handleSheetNameChange"
        />
      </div>
    </div>

    <div class="field-mapping-editor__mapping">
      <div class="field-mapping-editor__mapping-header">
        <span class="field-mapping-editor__mapping-header-cell">{{ t('templateEditor.destColumn') }}</span>
        <span class="field-mapping-editor__mapping-header-cell">{{ t('templateEditor.srcColumn') }}</span>
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
            :placeholder="t('templateEditor.unmapped')"
            :disabled="readonly"
            @update:value="(value) => handleColumnChange(field.key, value)"
          />
          <NInputNumber
            v-else
            class="field-mapping-editor__position-input"
            :value="positionValue(field.key)"
            :min="0"
            :precision="0"
            :placeholder="t('templateEditor.positionPlaceholder')"
            clearable
            :disabled="readonly"
            @update:value="(value) => handlePositionChange(field.key, value)"
          />
          <NInput
            class="field-mapping-editor__default-input"
            :value="defaultValue(field.key)"
            :placeholder="t('templateEditor.fixedValuePlaceholder')"
            :disabled="readonly"
            @update:value="(value) => handleDefaultChange(field.key, value)"
          />
          <div
            class="field-mapping-editor__transform-box"
            :title="t('templateEditor.transformsLabel')"
          >
            <span
              v-for="(name, index) in transformValue(field.key)"
              :key="`${field.key}-${name}`"
              class="field-mapping-editor__transform-chip"
            >
              <span class="field-mapping-editor__transform-name">
                {{ transformDisplayName(name) }}
              </span>
              <span class="field-mapping-editor__transform-ops">
                <button
                  type="button"
                  class="field-mapping-editor__chip-op"
                  :disabled="readonly || index === 0"
                  :aria-label="t('common.actions')"
                  @click="handleTransformMove(field.key, index, -1)"
                >↑</button>
                <button
                  type="button"
                  class="field-mapping-editor__chip-op"
                  :disabled="readonly || index === transformValue(field.key).length - 1"
                  @click="handleTransformMove(field.key, index, 1)"
                >↓</button>
                <button
                  type="button"
                  class="field-mapping-editor__chip-op"
                  :disabled="readonly"
                  @click="handleTransformRemove(field.key, index)"
                >×</button>
              </span>
            </span>
            <NSelect
              class="field-mapping-editor__transform-add"
              :value="null"
              :options="remainingTransformOptions(field.key)"
              size="small"
              :placeholder="t('templateEditor.transformAdd')"
              :disabled="readonly"
              @update:value="(value) => handleTransformAdd(field.key, value)"
            />
          </div>
          <label class="field-mapping-editor__flag-control">
            <NSwitch
              :value="isRequired(field.key)"
              :disabled="readonly"
              @update:value="(value) => handleRequiredChange(field.key, value)"
            />
            <span>{{ t('templateEditor.requiredLabel') }}</span>
          </label>
          <label class="field-mapping-editor__flag-control">
            <NCheckbox
              :checked="isFingerprint(field.key)"
              :disabled="readonly"
              @update:checked="(value) => handleFingerprintChange(field.key, value)"
            />
            <span>{{ t('templateEditor.fingerprintLabel') }}</span>
          </label>
        </div>
      </div>
    </div>

    <NButton
      size="small"
      quaternary
      type="primary"
      :disabled="readonly"
      @click="showAdvanced = !showAdvanced"
    >
      {{ t('templateEditor.advanced') }}
    </NButton>

    <div v-if="showAdvanced" class="field-mapping-editor__advanced">
      <div class="field-mapping-editor__advanced-row">
        <span class="field-mapping-editor__meta-label">{{ t('templateEditor.splitSkuQuantity') }}</span>
        <NSelect
          class="field-mapping-editor__key-select"
          :value="modelValue.splitSkuQuantity ?? null"
          :options="destKeyOptions"
          clearable
          filterable
          :placeholder="t('templateEditor.splitSkuQuantityPlaceholder')"
          :disabled="readonly"
          @update:value="(value) => handleSplitSkuQuantityChange(value)"
        />
      </div>

      <div class="field-mapping-editor__advanced-section">
        <h5 class="field-mapping-editor__advanced-title">{{ t('templateEditor.joinSources') }}</h5>
        <div
          v-for="entry in joinSourceEntries"
          :key="`join-${entry.key}`"
          class="field-mapping-editor__advanced-row"
        >
          <span class="field-mapping-editor__advanced-key">
            {{ entry.field?.label ?? entry.key }}
          </span>
          <NSelect
            class="field-mapping-editor__key-select"
            :value="entry.refs"
            :options="sourceRefOptions"
            multiple
            filterable
            :tag="!readonly"
            :placeholder="t('templateEditor.joinSourcesRefsPlaceholder')"
            :disabled="readonly"
            @update:value="(refs) => handleJoinRefsChange(entry.key, refs)"
          />
        </div>
        <NSelect
          class="field-mapping-editor__key-select"
          :value="joinKeyToAdd"
          :options="joinKeyOptions"
          filterable
          clearable
          :placeholder="t('templateEditor.joinSourcesAddKey')"
          :disabled="readonly || joinKeyOptions.length === 0"
          @update:value="(key) => handleAddJoinKey(key)"
        />
      </div>

      <div class="field-mapping-editor__advanced-section">
        <h5 class="field-mapping-editor__advanced-title">{{ t('templateEditor.enumMaps') }}</h5>
        <div
          v-for="entry in enumMapEntries"
          :key="`enum-${entry.key}`"
          class="field-mapping-editor__enum-block"
        >
          <div class="field-mapping-editor__enum-header">
            <span class="field-mapping-editor__advanced-key">
              {{ entry.field?.label ?? entry.key }}
            </span>
            <NButton
              size="tiny"
              type="error"
              quaternary
              :disabled="readonly"
              @click="handleRemoveEnumKey(entry.key)"
            >
              {{ t('templateEditor.remove') }}
            </NButton>
          </div>
          <div
            v-for="pair in entry.pairs"
            :key="`enum-${entry.key}-${pair[0]}`"
            class="field-mapping-editor__enum-row"
          >
            <NInput
              :value="pair[0]"
              :placeholder="t('templateEditor.enumExternal')"
              :disabled="readonly"
              @update:value="(v) => handleEnumPairChange(entry.key, pair[0], v, pair[1])"
            />
            <span class="field-mapping-editor__enum-arrow">→</span>
            <NInput
              :value="pair[1]"
              :placeholder="t('templateEditor.enumInternal')"
              :disabled="readonly"
              @update:value="(v) => handleEnumPairChange(entry.key, pair[0], pair[0], v)"
            />
            <NButton
              size="tiny"
              quaternary
              :disabled="readonly"
              @click="handleRemoveEnumPair(entry.key, pair[0])"
            >
              {{ t('templateEditor.remove') }}
            </NButton>
          </div>
          <NButton
            size="tiny"
            dashed
            :disabled="readonly"
            @click="handleAddEnumPair(entry.key)"
          >
            {{ t('templateEditor.enumAddRow') }}
          </NButton>
        </div>
        <NSelect
          class="field-mapping-editor__key-select"
          :value="enumKeyToAdd"
          :options="enumKeyOptions"
          filterable
          clearable
          :placeholder="t('templateEditor.enumMapsAddKey')"
          :disabled="readonly || enumKeyOptions.length === 0"
          @update:value="(key) => handleAddEnumKey(key)"
        />
      </div>
    </div>

    <div v-if="unmappedSourceHeaders.length" class="field-mapping-editor__unmapped-sources">
      <strong>{{ t('templateEditor.unmappedSourceColumns') }}</strong>
      <span>{{ unmappedSourceHeaders.join(', ') }}</span>
    </div>

    <div class="field-mapping-editor__preview">
      <h4 class="field-mapping-editor__preview-title">{{ t('templateEditor.previewTitle') }}</h4>
      <p class="field-mapping-editor__preview-note">{{ t('templateEditor.previewNote') }}</p>
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

.field-mapping-editor__meta-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.field-mapping-editor__meta-label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
  min-width: 120px;
}

.field-mapping-editor__sheet-input,
.field-mapping-editor__key-select {
  max-width: 320px;
}

.field-mapping-editor__mapping {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.field-mapping-editor__mapping-header {
  display: grid;
  grid-template-columns: minmax(160px, 1fr) minmax(240px, 2fr);
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
  grid-template-columns: minmax(160px, 1fr) minmax(240px, 2fr);
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-2) var(--space-1);
  border-bottom: 1px solid var(--card-border-color);
}

.field-mapping-editor__row:last-of-type {
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
  min-width: 140px;
}

.field-mapping-editor__default-input {
  flex: 1;
  min-width: 120px;
}

.field-mapping-editor__transform-box {
  flex: 1.6 1 200px;
  display: flex;
  align-items: center;
  gap: var(--space-1);
  flex-wrap: wrap;
}

.field-mapping-editor__transform-chip {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: 0 var(--space-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  background: var(--card-bg, transparent);
}

.field-mapping-editor__transform-name {
  white-space: nowrap;
}

.field-mapping-editor__transform-ops {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.field-mapping-editor__chip-op {
  border: none;
  background: transparent;
  cursor: pointer;
  padding: 0 2px;
  font-size: 11px;
  line-height: 1.2;
  color: var(--color-text-muted);
}

.field-mapping-editor__chip-op:hover:not(:disabled) {
  color: var(--color-text-primary);
}

.field-mapping-editor__chip-op:disabled {
  opacity: 0.4;
  cursor: default;
}

.field-mapping-editor__transform-add {
  flex: 1 1 120px;
  min-width: 110px;
}

.field-mapping-editor__flag-control {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  white-space: nowrap;
}

.field-mapping-editor__advanced {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-4);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-md);
}

.field-mapping-editor__advanced-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.field-mapping-editor__advanced-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.field-mapping-editor__advanced-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.field-mapping-editor__advanced-key {
  min-width: 160px;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.field-mapping-editor__enum-block {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-2) 0;
}

.field-mapping-editor__enum-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.field-mapping-editor__enum-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.field-mapping-editor__enum-arrow {
  color: var(--color-text-muted);
}

.field-mapping-editor__unmapped-sources {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  padding: var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--status-warning-bg);
  color: var(--status-warning-fg);
  font-size: var(--font-size-sm);
}

.field-mapping-editor__preview-title {
  margin: 0 0 var(--space-2);
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.field-mapping-editor__preview-note {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
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
