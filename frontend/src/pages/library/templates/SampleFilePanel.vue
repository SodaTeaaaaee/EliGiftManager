<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NDataTable, NSelect, NSwitch } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { inspectSampleFile, pickFile } from '@/shared/api/bridge'
import type { SampleFileInfo } from '@/entities/models'

/**
 * SampleFilePanel — the 样例文件 section of the template editor. Picks a
 * CSV/XLSX/XLS file, reads its first rows through InspectSampleFile, offers
 * a sheet select for spreadsheets (re-inspecting on change), and hosts the
 * 「第一行是表头」 switch that drives header vs positional mapping. The raw
 * grid shows exactly what the backend read, with 0-based column indexes as
 * titles in positional mode so operators can bind positions by eye.
 */
const props = defineProps<{
  filePath: string
  sample: SampleFileInfo | null
  sheetName: string
  hasHeader: boolean
}>()

const emit = defineEmits<{
  inspected: [filePath: string, sample: SampleFileInfo]
  cleared: []
  'update:sheetName': [string]
  'update:hasHeader': [boolean]
}>()

const { t } = useI18n()

const SAMPLE_LIMIT = 20
const sampleFileFilters = [{ displayName: 'CSV / Excel', pattern: '*.csv;*.xlsx;*.xls' }]

const loading = ref(false)
const error = ref('')

async function inspect(filePath: string, sheetName: string): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const info = await inspectSampleFile(filePath, sheetName, SAMPLE_LIMIT)
    emit('inspected', filePath, info)
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

async function handlePickFile(): Promise<void> {
  const path = await pickFile(sampleFileFilters)
  if (!path) return
  loading.value = true
  error.value = ''
  try {
    // Keep a sheet name the mapping already carries (e.g. copied from a
    // built-in); only when the new file lacks that sheet fall back to the
    // first sheet so the operator can re-pick.
    if (props.sheetName) {
      try {
        emit('inspected', path, await inspectSampleFile(path, props.sheetName, SAMPLE_LIMIT))
        return
      } catch {
        emit('update:sheetName', '')
      }
    }
    emit('inspected', path, await inspectSampleFile(path, '', SAMPLE_LIMIT))
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

async function handleSheetChange(next: string | null): Promise<void> {
  const sheet = next ?? ''
  emit('update:sheetName', sheet)
  if (props.filePath) await inspect(props.filePath, sheet)
}

function handleClear(): void {
  error.value = ''
  emit('cleared')
}

const sheetOptions = computed(() =>
  (props.sample?.Sheets ?? []).map((name) => ({ label: name, value: name })),
)

const showSheetSelect = computed(() => sheetOptions.value.length > 0)

/** The sheet the backend actually read: explicit choice, else the first one. */
const effectiveSheet = computed(() => props.sheetName || props.sample?.Sheets[0] || '')

const records = computed(() => props.sample?.Records ?? [])
const width = computed(() => records.value.reduce((max, row) => Math.max(max, row.length), 0))

interface RawGridRow {
  __index: number
  cells: string[]
}

const rawRows = computed<RawGridRow[]>(() => {
  const data = props.hasHeader ? records.value.slice(1) : records.value
  return data.map((cells, index) => ({ __index: index, cells }))
})

const rawColumns = computed<DataTableColumns<RawGridRow>>(() => {
  const header = props.hasHeader ? records.value[0] ?? [] : []
  const indexColumn = {
    key: '__row',
    title: '#',
    width: 56,
    render: (row: RawGridRow) =>
      h(
        'span',
        { class: 'sample-file-panel__row-no' },
        // Source row numbers are 1-based and count the header row when present.
        String(row.__index + (props.hasHeader ? 2 : 1)),
      ),
  }
  const cellColumns = Array.from({ length: width.value }, (_, i) => ({
    key: `c${i}`,
    minWidth: 110,
    ellipsis: { tooltip: true },
    title: () =>
      h('span', { class: 'sample-file-panel__col-title' }, [
        h('span', { class: 'sample-file-panel__col-index' }, String(i)),
        props.hasHeader ? h('span', { class: 'sample-file-panel__col-header' }, header[i] ?? '') : null,
      ]),
    render: (row: RawGridRow) => row.cells[i] ?? '',
  }))
  return [indexColumn, ...cellColumns]
})

const rowSummary = computed(() => {
  if (!props.sample) return ''
  const shown = rawRows.value.length
  return t('templates.sample.rowSummary', { shown, total: props.sample.Total })
})
</script>

<template>
  <div class="sample-file-panel">
    <div class="sample-file-panel__toolbar">
      <NButton size="small" type="primary" secondary :loading="loading" @click="handlePickFile">
        {{ filePath ? t('templates.changeSample') : t('templates.sample.pick') }}
      </NButton>
      <span v-if="filePath" class="sample-file-panel__path" :title="filePath">{{ filePath }}</span>
      <span v-else class="sample-file-panel__hint">{{ t('templates.sample.none') }}</span>
      <NButton v-if="filePath" size="tiny" quaternary @click="handleClear">
        {{ t('templates.sample.clear') }}
      </NButton>
    </div>

    <div class="sample-file-panel__controls">
      <label class="sample-file-panel__switch">
        <NSwitch :value="hasHeader" @update:value="(v) => emit('update:hasHeader', v)" />
        <span>{{ t('templates.sample.firstRowIsHeader') }}</span>
      </label>
      <span class="sample-file-panel__mode-hint">
        {{ hasHeader ? t('templates.sample.headerModeHint') : t('templates.sample.positionalModeHint') }}
      </span>
      <div v-if="showSheetSelect" class="sample-file-panel__sheet">
        <span class="sample-file-panel__sheet-label">{{ t('templateEditor.sheetName') }}</span>
        <NSelect
          size="small"
          class="sample-file-panel__sheet-select"
          :value="effectiveSheet || null"
          :options="sheetOptions"
          :loading="loading"
          @update:value="(v) => handleSheetChange(v == null ? null : String(v))"
        />
      </div>
    </div>

    <p v-if="error" class="sample-file-panel__error">
      {{ t('templates.sample.inspectFailed', { message: error }) }}
    </p>

    <template v-if="sample">
      <div class="sample-file-panel__summary">
        <span>{{ t('templates.sample.format', { format: sample.Format.toUpperCase() }) }}</span>
        <span>{{ rowSummary }}</span>
      </div>
      <div v-if="!records.length" class="sample-file-panel__hint">
        {{ t('templates.sample.emptyFile') }}
      </div>
      <NDataTable
        v-else
        class="sample-file-panel__grid"
        :columns="rawColumns"
        :data="rawRows"
        :row-key="(row: RawGridRow) => row.__index"
        :scroll-x="Math.max(width * 120 + 56, 480)"
        :max-height="260"
        size="small"
        striped
      />
    </template>
  </div>
</template>

<style scoped>
.sample-file-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.sample-file-panel__toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.sample-file-panel__path {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
  flex: 1 1 200px;
  min-width: 0;
}

.sample-file-panel__hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.sample-file-panel__controls {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.sample-file-panel__switch {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  cursor: pointer;
}

.sample-file-panel__mode-hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.sample-file-panel__sheet {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
}

.sample-file-panel__sheet-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.sample-file-panel__sheet-select {
  width: 200px;
}

.sample-file-panel__error {
  margin: 0;
  color: var(--status-error-fg);
  font-size: var(--font-size-sm);
}

.sample-file-panel__summary {
  display: flex;
  gap: var(--space-4);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.sample-file-panel__grid {
  width: 100%;
}
</style>

<style>
/* Column titles/cells render through NDataTable's `render`, outside scope. */
.sample-file-panel__row-no {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.sample-file-panel__col-title {
  display: inline-flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.2;
}

.sample-file-panel__col-index {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--color-text-muted);
}

.sample-file-panel__col-header {
  font-size: var(--font-size-xs);
  color: var(--color-text-primary);
}
</style>
