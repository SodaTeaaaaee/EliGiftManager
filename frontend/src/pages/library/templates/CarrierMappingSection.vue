<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NModal, NPopconfirm, NSelect, NSpace } from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { DataGrid, createColumns } from '@/shared/ui/data-grid'
import { useFeedback } from '@/shared/ui/feedback'
import {
  createCarrierMapping,
  deleteCarrierMapping,
  importCarrierMappings,
  inspectSampleFile,
  listCarrierMappings,
  pickFile,
  updateCarrierMapping,
} from '@/shared/api/bridge'
import type { CarrierMapping, ImportCarrierMappingsResult, Platform } from '@/entities/models'

/**
 * CarrierMappingSection — 承运商映射 for source platforms: 「承运商名称/描述 →
 * 该平台回填时接受的承运商 ID」. Field semantics on the unchanged struct:
 * InternalName = the name/description as it appears in factory shipment
 * files, ExternalCode = the platform carrier ID, InternalCode = hidden and
 * always sent as ''. Supports inline add / edit / delete and a file import
 * that lets the operator pick the name and ID columns from the first row.
 */
const props = defineProps<{
  platforms: Platform[]
}>()

const { t } = useI18n()
const feedback = useFeedback()

function errMsg(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

const sourcePlatforms = computed(() => props.platforms.filter((p) => p.Kind === 'source'))
const platformOptions = computed(() =>
  sourcePlatforms.value.map((p) => ({ label: `${p.Name} (${p.Key})`, value: p.ID })),
)

const selectedPlatformId = ref<number | null>(null)
const mappings = ref<CarrierMapping[]>([])
const loading = ref(false)
const busy = ref(false)

async function load(): Promise<void> {
  if (!selectedPlatformId.value) {
    mappings.value = []
    return
  }
  loading.value = true
  try {
    mappings.value = await listCarrierMappings(selectedPlatformId.value)
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    loading.value = false
  }
}

watch(sourcePlatforms, (list) => {
  if (selectedPlatformId.value && list.some((p) => p.ID === selectedPlatformId.value)) return
  selectedPlatformId.value = list[0]?.ID ?? null
}, { immediate: true })

watch(selectedPlatformId, () => {
  cancelEdit()
  void load()
})

onMounted(() => {
  void load()
})

// ── Inline add ──

const newName = ref('')
const newCode = ref('')

const canAdd = computed(
  () => Boolean(selectedPlatformId.value) && newName.value.trim() !== '' && newCode.value.trim() !== '',
)

async function handleAdd(): Promise<void> {
  if (!canAdd.value || !selectedPlatformId.value) return
  busy.value = true
  try {
    await createCarrierMapping({
      PlatformID: selectedPlatformId.value,
      InternalName: newName.value.trim(),
      ExternalCode: newCode.value.trim(),
      InternalCode: '',
    })
    newName.value = ''
    newCode.value = ''
    feedback.success(t('library.carrierSuccess'))
    await load()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    busy.value = false
  }
}

// ── Inline edit ──

const editingId = ref<number | null>(null)
const editName = ref('')
const editCode = ref('')

function startEdit(row: CarrierMapping): void {
  editingId.value = row.ID
  editName.value = row.InternalName
  editCode.value = row.ExternalCode
}

function cancelEdit(): void {
  editingId.value = null
  editName.value = ''
  editCode.value = ''
}

async function saveEdit(row: CarrierMapping): Promise<void> {
  if (editName.value.trim() === '' || editCode.value.trim() === '') return
  busy.value = true
  try {
    await updateCarrierMapping({
      ...row,
      InternalName: editName.value.trim(),
      ExternalCode: editCode.value.trim(),
      InternalCode: '',
    })
    cancelEdit()
    feedback.success(t('templates.carrier.updateSuccess'))
    await load()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    busy.value = false
  }
}

async function handleDelete(row: CarrierMapping): Promise<void> {
  busy.value = true
  try {
    await deleteCarrierMapping(row.ID)
    if (editingId.value === row.ID) cancelEdit()
    feedback.success(t('templates.carrier.deleteSuccess'))
    await load()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    busy.value = false
  }
}

const columns = computed(() =>
  createColumns<CarrierMapping>([
    {
      key: 'InternalName',
      title: t('templates.carrier.nameHeader'),
      type: 'text',
      minWidth: 200,
      render: (row) =>
        editingId.value === row.ID
          ? h(NInput, {
              size: 'small',
              value: editName.value,
              'onUpdate:value': (v: string) => (editName.value = v),
            })
          : row.InternalName || '—',
    },
    {
      key: 'ExternalCode',
      title: t('templates.carrier.codeHeader'),
      type: 'text',
      minWidth: 180,
      render: (row) =>
        editingId.value === row.ID
          ? h(NInput, {
              size: 'small',
              value: editCode.value,
              'onUpdate:value': (v: string) => (editCode.value = v),
            })
          : h('code', { class: 'carrier-mapping__code' }, row.ExternalCode || '—'),
    },
    {
      key: 'actions',
      title: t('common.actions'),
      type: 'actions',
      width: 200,
      render: (row) =>
        editingId.value === row.ID
          ? h(NSpace, { size: 'small', justify: 'end', wrap: false }, () => [
              h(
                NButton,
                {
                  size: 'tiny',
                  type: 'primary',
                  loading: busy.value,
                  disabled: editName.value.trim() === '' || editCode.value.trim() === '',
                  onClick: () => void saveEdit(row),
                },
                { default: () => t('common.save') },
              ),
              h(
                NButton,
                { size: 'tiny', quaternary: true, onClick: cancelEdit },
                { default: () => t('common.cancel') },
              ),
            ])
          : h(NSpace, { size: 'small', justify: 'end', wrap: false }, () => [
              h(
                NButton,
                { size: 'tiny', secondary: true, onClick: () => startEdit(row) },
                { default: () => t('common.edit') },
              ),
              h(
                NPopconfirm,
                {
                  onPositiveClick: () => void handleDelete(row),
                  positiveText: t('templates.list.deleteAction'),
                  negativeText: t('common.cancel'),
                },
                {
                  trigger: () =>
                    h(
                      NButton,
                      { size: 'tiny', type: 'error', quaternary: true },
                      { default: () => t('templates.list.deleteAction') },
                    ),
                  default: () => t('templates.carrier.deleteConfirm', { name: row.InternalName }),
                },
              ),
            ]),
    },
  ]),
)

// ── Import from file ──

const importFileFilters = [{ displayName: 'CSV / Excel', pattern: '*.csv;*.xlsx;*.xls' }]
const showImportModal = ref(false)
const importFilePath = ref('')
const importHeaders = ref<string[]>([])
const importNameHeader = ref<string | null>(null)
const importCodeHeader = ref<string | null>(null)
const importInspecting = ref(false)
const importRunning = ref(false)
const importError = ref('')
const importResult = ref<ImportCarrierMappingsResult | null>(null)

function openImport(): void {
  importFilePath.value = ''
  importHeaders.value = []
  importNameHeader.value = null
  importCodeHeader.value = null
  importError.value = ''
  importResult.value = null
  showImportModal.value = true
}

async function handlePickImportFile(): Promise<void> {
  const path = await pickFile(importFileFilters)
  if (!path) return
  importInspecting.value = true
  importError.value = ''
  importResult.value = null
  try {
    const info = await inspectSampleFile(path, '', 5)
    importFilePath.value = path
    importHeaders.value = (info.Records[0] ?? []).map((cell) => cell.trim()).filter((cell) => cell !== '')
    importNameHeader.value = null
    importCodeHeader.value = null
  } catch (err) {
    importError.value = errMsg(err)
  } finally {
    importInspecting.value = false
  }
}

const importHeaderOptions = computed(() =>
  importHeaders.value.map((header) => ({ label: header, value: header })),
)

const canRunImport = computed(
  () =>
    Boolean(selectedPlatformId.value) &&
    importFilePath.value !== '' &&
    Boolean(importNameHeader.value) &&
    Boolean(importCodeHeader.value) &&
    importNameHeader.value !== importCodeHeader.value,
)

async function handleRunImport(): Promise<void> {
  if (!canRunImport.value || !selectedPlatformId.value) return
  importRunning.value = true
  importError.value = ''
  try {
    importResult.value = await importCarrierMappings(
      selectedPlatformId.value,
      importFilePath.value,
      importNameHeader.value ?? '',
      importCodeHeader.value ?? '',
    )
    feedback.success(t('templates.carrier.importSuccess'))
    await load()
  } catch (err) {
    importResult.value = null
    importError.value = errMsg(err)
  } finally {
    importRunning.value = false
  }
}
</script>

<template>
  <SectionCard :title="t('library.carrierMappings')" :description="t('templates.carrier.desc')">
    <template #actions>
      <NSpace align="center">
        <NSelect
          v-model:value="selectedPlatformId"
          :options="platformOptions"
          :placeholder="t('templates.carrier.noSourcePlatform')"
          :disabled="!platformOptions.length"
          size="small"
          class="carrier-mapping__platform-select"
        />
        <NButton size="small" :disabled="!selectedPlatformId" @click="openImport">
          {{ t('templates.carrier.importFromFile') }}
        </NButton>
      </NSpace>
    </template>

    <div v-if="!sourcePlatforms.length" class="carrier-mapping__empty">
      <EmptyState :title="t('templates.carrier.noSourcePlatform')" size="sm" />
    </div>

    <template v-else>
      <div class="carrier-mapping__add-row">
        <NInput
          v-model:value="newName"
          size="small"
          :placeholder="t('templates.carrier.addNamePlaceholder')"
          class="carrier-mapping__add-input"
          @keyup.enter="handleAdd"
        />
        <span class="carrier-mapping__arrow" aria-hidden="true">→</span>
        <NInput
          v-model:value="newCode"
          size="small"
          :placeholder="t('templates.carrier.addCodePlaceholder')"
          class="carrier-mapping__add-input"
          @keyup.enter="handleAdd"
        />
        <NButton size="small" type="primary" :disabled="!canAdd" :loading="busy" @click="handleAdd">
          {{ t('templates.carrier.add') }}
        </NButton>
      </div>

      <div v-if="!loading && !mappings.length" class="carrier-mapping__empty">
        <EmptyState :title="t('templates.carrier.empty')" size="sm" />
      </div>
      <DataGrid
        v-else
        :columns="columns"
        :rows="mappings"
        row-key="ID"
        :loading="loading"
        pagination="none"
      />
    </template>

    <NModal
      v-model:show="showImportModal"
      preset="card"
      :title="t('templates.carrier.importTitle')"
      class="carrier-mapping__import-modal"
    >
      <div class="carrier-mapping__import">
        <p class="carrier-mapping__hint">{{ t('templates.carrier.importPickHint') }}</p>
        <div class="carrier-mapping__file-row">
          <NButton size="small" type="primary" secondary :loading="importInspecting" @click="handlePickImportFile">
            {{ importFilePath ? t('inbox.changeFile') : t('templates.carrier.importPickFile') }}
          </NButton>
          <span v-if="importFilePath" class="carrier-mapping__path" :title="importFilePath">{{ importFilePath }}</span>
          <span v-else class="carrier-mapping__hint">{{ t('inbox.noFileSelected') }}</span>
        </div>

        <template v-if="importHeaders.length">
          <div class="carrier-mapping__pick-row">
            <span class="carrier-mapping__pick-label">{{ t('templates.carrier.importNameColumn') }}</span>
            <NSelect
              v-model:value="importNameHeader"
              size="small"
              :options="importHeaderOptions"
              :placeholder="t('common.pleaseSelect')"
            />
          </div>
          <div class="carrier-mapping__pick-row">
            <span class="carrier-mapping__pick-label">{{ t('templates.carrier.importCodeColumn') }}</span>
            <NSelect
              v-model:value="importCodeHeader"
              size="small"
              :options="importHeaderOptions"
              :placeholder="t('common.pleaseSelect')"
            />
          </div>
          <p
            v-if="importNameHeader && importNameHeader === importCodeHeader"
            class="carrier-mapping__error"
          >
            {{ t('templates.carrier.importSameColumn') }}
          </p>
        </template>
        <p v-else-if="importFilePath" class="carrier-mapping__error">
          {{ t('templates.carrier.importNoHeaders') }}
        </p>

        <p v-if="importError" class="carrier-mapping__error">
          {{ t('templates.carrier.importFailed', { message: importError }) }}
        </p>

        <div v-if="importResult" class="carrier-mapping__result">
          <div class="carrier-mapping__result-title">{{ t('templates.carrier.importResult') }}</div>
          <div class="carrier-mapping__result-stats">
            <span>{{ t('templates.carrier.importCreated', { n: importResult.Created }) }}</span>
            <span>{{ t('templates.carrier.importUpdated', { n: importResult.Updated }) }}</span>
            <span>{{ t('templates.carrier.importSkipped', { n: importResult.Skipped }) }}</span>
          </div>
          <ul v-if="importResult.Issues.length" class="carrier-mapping__issues">
            <li v-for="(issue, index) in importResult.Issues" :key="index" class="carrier-mapping__issue">
              <span class="carrier-mapping__issue-line">#{{ issue.LineNo }}</span>
              <span class="carrier-mapping__issue-key">{{ issue.Key }}</span>
              <span>{{ issue.Message }}</span>
            </li>
          </ul>
        </div>
      </div>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="showImportModal = false">{{ t('common.close') }}</NButton>
          <NButton
            type="primary"
            :loading="importRunning"
            :disabled="!canRunImport"
            @click="handleRunImport"
          >
            {{ t('templates.carrier.importRun') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </SectionCard>
</template>

<style scoped>
.carrier-mapping__platform-select {
  width: 200px;
}

.carrier-mapping__empty {
  padding: var(--space-2) 0;
}

.carrier-mapping__add-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
  flex-wrap: wrap;
}

.carrier-mapping__add-input {
  flex: 1 1 200px;
  max-width: 320px;
}

.carrier-mapping__arrow {
  color: var(--color-text-muted);
}

.carrier-mapping__import-modal {
  width: min(560px, 92vw);
}

.carrier-mapping__import {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.carrier-mapping__hint {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.carrier-mapping__file-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.carrier-mapping__path {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1 1 200px;
  min-width: 0;
}

.carrier-mapping__pick-row {
  display: grid;
  grid-template-columns: 160px 1fr;
  align-items: center;
  gap: var(--space-3);
}

.carrier-mapping__pick-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.carrier-mapping__error {
  margin: 0;
  color: var(--status-error-fg);
  font-size: var(--font-size-sm);
}

.carrier-mapping__result {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  background: var(--color-inset);
}

.carrier-mapping__result-title {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.carrier-mapping__result-stats {
  display: flex;
  gap: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.carrier-mapping__issues {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: var(--font-size-sm);
}

.carrier-mapping__issue {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.carrier-mapping__issue-line {
  font-family: var(--font-mono);
  color: var(--color-text-muted);
  min-width: 40px;
}

.carrier-mapping__issue-key {
  color: var(--color-text-secondary);
}
</style>

<style>
.carrier-mapping__code {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-primary);
}
</style>
