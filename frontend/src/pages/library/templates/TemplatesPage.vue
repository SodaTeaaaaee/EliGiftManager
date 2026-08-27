<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NRadio,
  NRadioGroup,
  NSelect,
  NSpace,
  NSpin,
  NTag,
} from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import {
  FieldMappingEditor,
  emptyFieldMapping,
  parseMappingRules,
  serializeMappingRules,
} from '@/shared/ui/field-mapping'
import type { FieldMappingValue } from '@/shared/ui/field-mapping'
import {
  createCarrierMapping,
  createTemplate,
  getNamedTransformers,
  getSemanticDictionary,
  listCarrierMappings,
  listPlatforms,
  listTemplates,
  pickFile,
  previewTemplate,
} from '@/shared/api/bridge'
import type { CarrierMapping, Platform, TemplateConfig, TemplatePreview } from '@/entities/models'

const { t, te } = useI18n()

const loading = ref(false)
const actionLoading = ref(false)
const templates = ref<TemplateConfig[]>([])
const platforms = ref<Platform[]>([])
const dictionary = ref<string[]>([])
const transformers = ref<string[]>([])

const selectedPlatformForCarriers = ref<number | null>(null)
const carrierMappings = ref<CarrierMapping[]>([])

interface LayoutFormState {
  format: 'csv' | 'xlsx'
  columnOrder: string
  headerNames: { key: string; value: string }[]
}

const showCreateTemplateModal = ref(false)
const templateForm = ref({
  platformId: null as number | null,
  documentType: 'membership_list',
  direction: 'input',
  name: '',
  notes: '',
  mapping: emptyFieldMapping('header') as FieldMappingValue,
  layout: {
    format: 'csv',
    columnOrder: '',
    headerNames: [],
  } as LayoutFormState,
})

const showCreateCarrierModal = ref(false)
const carrierForm = ref({
  externalCode: '',
  internalCode: 'SF',
  internalName: '顺丰速运',
})

// ── Template testing (backend PreviewTemplate) ──

const sampleFileFilters = [{ displayName: 'CSV / Excel', pattern: '*.csv;*.xlsx;*.xls' }]
const showTestModal = ref(false)
const testLoading = ref(false)
const testTemplate = ref<TemplateConfig | null>(null)
const testFilePath = ref('')
const testPreview = ref<TemplatePreview | null>(null)
const testError = ref('')

async function loadData() {
  loading.value = true
  try {
    const [tmplRes, platRes, dictRes, transRes] = await Promise.all([
      listTemplates(),
      listPlatforms(),
      getSemanticDictionary(),
      getNamedTransformers(),
    ])
    templates.value = tmplRes
    platforms.value = platRes
    dictionary.value = dictRes
    transformers.value = transRes
    if (platRes.length > 0 && !selectedPlatformForCarriers.value) {
      selectedPlatformForCarriers.value = platRes[0].ID
    }
  } catch (err) {
    console.error('Failed to load templates data:', err)
  } finally {
    loading.value = false
  }
}

async function loadCarrierMappings() {
  if (!selectedPlatformForCarriers.value) return
  try {
    carrierMappings.value = await listCarrierMappings(selectedPlatformForCarriers.value)
  } catch (err) {
    console.error('Failed to load carrier mappings:', err)
  }
}

watch(selectedPlatformForCarriers, () => {
  void loadCarrierMappings()
})

onMounted(async () => {
  await loadData()
  await loadCarrierMappings()
})

const platformOptions = computed(() =>
  platforms.value.map((p) => ({
    label: `${p.Name} (${p.Key})`,
    value: p.ID,
  })),
)

/** Semantic dictionary keys rendered with their localized display names. */
const destFields = computed(() =>
  dictionary.value.map((key) => {
    const labelKey = `templateEditor.semanticKeys.${key}`
    return {
      key,
      label: te(labelKey) ? t(labelKey) : key,
      tooltip: key,
    }
  }),
)

const documentTypeOptions = [
  'membership_list',
  'order_export',
  'shipment_return',
  'factory_order',
  'writeback',
]

function documentTypeLabel(type: string): string {
  const key = `templates.documentTypeOptions.${type}`
  return te(key) ? t(key) : type
}

function openCreateTemplate() {
  templateForm.value = {
    platformId: platformOptions.value[0]?.value ?? null,
    documentType: 'membership_list',
    direction: 'input',
    name: '',
    notes: '',
    mapping: emptyFieldMapping('header'),
    layout: {
      format: 'csv',
      columnOrder: '',
      headerNames: [],
    },
  }
  showCreateTemplateModal.value = true
}

function handleAddHeaderName() {
  templateForm.value.layout.headerNames.push({ key: '', value: '' })
}

function handleRemoveHeaderName(index: number) {
  templateForm.value.layout.headerNames.splice(index, 1)
}

async function handleSaveTemplate() {
  if (!templateForm.value.platformId || !templateForm.value.name.trim()) return
  actionLoading.value = true
  try {
    const headerNames: Record<string, string> = {}
    for (const entry of templateForm.value.layout.headerNames) {
      if (entry.key.trim() !== '' && entry.value.trim() !== '') {
        headerNames[entry.key.trim()] = entry.value.trim()
      }
    }
    const layoutJSON = JSON.stringify({
      version: 1,
      format: templateForm.value.layout.format,
      columnOrder: templateForm.value.layout.columnOrder
        .split(/[,，\n]/)
        .map((part) => part.trim())
        .filter(Boolean),
      headerNames,
    })
    await createTemplate({
      PlatformID: templateForm.value.platformId,
      DocumentType: templateForm.value.documentType,
      Direction: templateForm.value.direction,
      Name: templateForm.value.name.trim(),
      MappingJSON: serializeMappingRules(templateForm.value.mapping),
      LayoutJSON: layoutJSON,
      Notes: templateForm.value.notes.trim(),
    })
    showCreateTemplateModal.value = false
    await loadData()
  } catch (err) {
    console.error('Failed to save template:', err)
  } finally {
    actionLoading.value = false
  }
}

// ── Template test flow ──

function openTestTemplate(row: TemplateConfig) {
  testTemplate.value = row
  testFilePath.value = ''
  testPreview.value = null
  testError.value = ''
  showTestModal.value = true
}

async function handlePickSampleFile() {
  const path = await pickFile(sampleFileFilters)
  if (!path) return
  testFilePath.value = path
  await runTemplateTest()
}

async function runTemplateTest() {
  if (!testTemplate.value || !testFilePath.value) return
  testLoading.value = true
  testError.value = ''
  try {
    testPreview.value = await previewTemplate(testTemplate.value.ID, testFilePath.value, 10)
  } catch (err) {
    testPreview.value = null
    testError.value = err instanceof Error ? err.message : String(err)
  } finally {
    testLoading.value = false
  }
}

const testRowKeys = computed(() => {
  const keys: string[] = []
  for (const row of testPreview.value?.Rows ?? []) {
    for (const key of Object.keys(row)) {
      if (!keys.includes(key)) keys.push(key)
    }
  }
  return keys
})

function testColumnTitle(key: string): string {
  const labelKey = `templateEditor.semanticKeys.${key}`
  return te(labelKey) ? t(labelKey) : key
}

const testPreviewColumns = computed(() =>
  testRowKeys.value.map((key) => ({
    title: testColumnTitle(key),
    key,
    render(row: Record<string, string>) {
      return row[key] ?? '—'
    },
  })),
)

function openCreateCarrier() {
  carrierForm.value = {
    externalCode: '',
    internalCode: 'SF',
    internalName: '顺丰速运',
  }
  showCreateCarrierModal.value = true
}

async function handleSaveCarrier() {
  if (!selectedPlatformForCarriers.value || !carrierForm.value.externalCode.trim()) return
  actionLoading.value = true
  try {
    await createCarrierMapping({
      PlatformID: selectedPlatformForCarriers.value,
      ExternalCode: carrierForm.value.externalCode.trim(),
      InternalCode: carrierForm.value.internalCode.trim(),
      InternalName: carrierForm.value.internalName.trim(),
    })
    showCreateCarrierModal.value = false
    await loadCarrierMappings()
  } catch (err) {
    console.error('Failed to save carrier mapping:', err)
  } finally {
    actionLoading.value = false
  }
}

/** Read-only mapping summary for the template list (no editor without an UpdateTemplate port). */
function mappingSummary(row: TemplateConfig): string {
  const mapping = parseMappingRules(row.MappingJSON)
  const count =
    mapping.mode === 'positional'
      ? Object.keys(mapping.positions ?? {}).length
      : Object.keys(mapping.columns).length
  const modeLabel =
    mapping.mode === 'positional'
      ? t('templateEditor.modePositional')
      : t('templateEditor.modeHeader')
  return t('templates.mappingSummary', { mode: modeLabel, count })
}

const templateColumns = [
  {
    title: '#',
    key: 'ID',
    width: 70,
    render(row: TemplateConfig) {
      return `#${row.ID}`
    },
  },
  {
    title: t('library.templateName'),
    key: 'Name',
  },
  {
    title: t('library.factoryPlatform'),
    key: 'PlatformID',
    render(row: TemplateConfig) {
      const plat = platforms.value.find((p) => p.ID === row.PlatformID)
      return plat ? plat.Name : `Platform #${row.PlatformID}`
    },
  },
  {
    title: t('library.documentType'),
    key: 'DocumentType',
    render(row: TemplateConfig) {
      return documentTypeLabel(row.DocumentType)
    },
  },
  {
    title: t('library.direction'),
    key: 'Direction',
    width: 110,
    render(row: TemplateConfig) {
      return h(StatusBadge, {
        dimension: 'templateDirection',
        value: row.Direction || 'input',
      })
    },
  },
  {
    title: t('templates.mapping'),
    key: 'MappingJSON',
    render(row: TemplateConfig) {
      return mappingSummary(row)
    },
  },
  {
    title: t('library.version'),
    key: 'Version',
    width: 70,
    render(row: TemplateConfig) {
      return `v${row.Version}`
    },
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 90,
    render(row: TemplateConfig) {
      return h(
        NButton,
        {
          size: 'tiny',
          quaternary: true,
          type: 'primary',
          onClick: () => openTestTemplate(row),
        },
        { default: () => t('templates.test') },
      )
    },
  },
]

const carrierColumns = [
  {
    title: t('library.externalCode'),
    key: 'ExternalCode',
  },
  {
    title: t('library.internalCode'),
    key: 'InternalCode',
  },
  {
    title: t('library.internalName'),
    key: 'InternalName',
  },
]
</script>

<template>
  <div class="templates-page">
    <!-- Templates List -->
    <SectionCard :title="t('library.templatesTab')">
      <template #actions>
        <NSpace>
          <NButton size="small" type="primary" @click="openCreateTemplate">
            {{ t('library.createTemplate') }}
          </NButton>
          <NButton size="small" :loading="loading" @click="loadData">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>

      <NSpin :show="loading">
        <div v-if="!templates.length" class="templates-page__empty">
          <EmptyState :title="t('library.emptyTemplates')" size="sm" />
        </div>
        <div v-else class="templates-page__table">
          <NDataTable
            :columns="templateColumns"
            :data="templates"
            :row-key="(row: TemplateConfig) => row.ID"
            size="small"
          />
        </div>
      </NSpin>
    </SectionCard>

    <!-- Carrier Mappings Section -->
    <SectionCard :title="t('library.carrierMappings')">
      <template #actions>
        <NSpace align="center">
          <NSelect
            v-model:value="selectedPlatformForCarriers"
            :options="platformOptions"
            size="small"
            style="width: 180px"
          />
          <NButton size="small" type="primary" @click="openCreateCarrier">
            {{ t('library.createCarrierMapping') }}
          </NButton>
        </NSpace>
      </template>

      <div v-if="!carrierMappings.length" class="templates-page__empty">
        <EmptyState :title="t('library.carrierMappings')" size="sm" />
      </div>
      <div v-else class="templates-page__table">
        <NDataTable
          :columns="carrierColumns"
          :data="carrierMappings"
          :row-key="(row: CarrierMapping) => row.ID"
          size="small"
        />
      </div>
    </SectionCard>

    <!-- Semantic Dictionary & Named Transformers -->
    <div class="templates-page__meta-grid">
      <SectionCard :title="t('library.semanticDictionary')">
        <div class="templates-page__tags">
          <NTag v-for="key in dictionary" :key="key" size="small" type="info">
            {{ key }}
          </NTag>
        </div>
      </SectionCard>

      <SectionCard :title="t('library.namedTransformers')">
        <div class="templates-page__tags">
          <NTag v-for="trans in transformers" :key="trans" size="small" type="success">
            {{ trans }}
          </NTag>
        </div>
      </SectionCard>
    </div>

    <!-- Create Template Modal -->
    <NModal
      v-model:show="showCreateTemplateModal"
      preset="card"
      :title="t('library.createTemplate')"
      class="templates-page__create-modal"
    >
      <NForm label-placement="left" label-width="110">
        <NFormItem :label="t('library.factoryPlatform')">
          <NSelect v-model:value="templateForm.platformId" :options="platformOptions" />
        </NFormItem>
        <NFormItem :label="t('library.templateName')">
          <NInput v-model:value="templateForm.name" />
        </NFormItem>
        <NFormItem :label="t('library.documentType')">
          <NSelect
            v-model:value="templateForm.documentType"
            :options="documentTypeOptions.map((type) => ({ label: documentTypeLabel(type), value: type }))"
            tag
            filterable
          />
        </NFormItem>
        <NFormItem :label="t('library.direction')">
          <NRadioGroup v-model:value="templateForm.direction">
            <NSpace>
              <NRadio value="input">{{ t('glossary.templateDirection.input.label') }}</NRadio>
              <NRadio value="output">{{ t('glossary.templateDirection.output.label') }}</NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="t('library.notes')">
          <NInput v-model:value="templateForm.notes" type="textarea" />
        </NFormItem>
      </NForm>

      <h5 class="templates-page__section-title">{{ t('templates.mapping') }}</h5>
      <FieldMappingEditor
        v-model="templateForm.mapping"
        :dest-fields="destFields"
        :source-headers="[]"
        :sample-rows="[]"
      />

      <h5 class="templates-page__section-title">{{ t('templates.layout') }}</h5>
      <NForm label-placement="left" label-width="110">
        <NFormItem :label="t('templates.layoutFormat')">
          <NRadioGroup v-model:value="templateForm.layout.format">
            <NSpace>
              <NRadio value="csv">{{ t('templates.formatCsv') }}</NRadio>
              <NRadio value="xlsx">{{ t('templates.formatXlsx') }}</NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="t('templates.columnOrder')">
          <NInput
            v-model:value="templateForm.layout.columnOrder"
            :placeholder="t('templates.columnOrderPlaceholder')"
          />
        </NFormItem>
      </NForm>
      <div class="templates-page__header-names">
        <div class="templates-page__header-names-title">{{ t('templates.headerNames') }}</div>
        <div
          v-for="(entry, index) in templateForm.layout.headerNames"
          :key="index"
          class="templates-page__header-name-row"
        >
          <NInput
            v-model:value="entry.key"
            :placeholder="t('templates.headerNameKey')"
          />
          <NInput
            v-model:value="entry.value"
            :placeholder="t('templates.headerNameValue')"
          />
          <NButton
            size="tiny"
            quaternary
            type="error"
            @click="handleRemoveHeaderName(index)"
          >
            {{ t('templateEditor.remove') }}
          </NButton>
        </div>
        <NButton size="tiny" dashed @click="handleAddHeaderName">
          {{ t('templates.addHeaderName') }}
        </NButton>
      </div>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateTemplateModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!templateForm.platformId || !templateForm.name.trim()"
            @click="handleSaveTemplate"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Template Test Modal -->
    <NModal
      v-model:show="showTestModal"
      preset="card"
      :title="t('templates.testTitle')"
      class="templates-page__test-modal"
    >
      <NSpin :show="testLoading">
        <div class="templates-page__test-file">
          <NButton type="primary" size="small" @click="handlePickSampleFile">
            {{ testFilePath ? t('templates.changeSample') : t('templates.uploadSample') }}
          </NButton>
          <span v-if="testFilePath" class="templates-page__test-path">{{ testFilePath }}</span>
        </div>

        <p v-if="testError" class="templates-page__test-error">
          {{ t('templates.testFailed', { message: testError }) }}
        </p>

        <template v-if="testPreview">
          <h5 class="templates-page__section-title">
            {{ t('templates.previewRows', { n: testPreview.Rows.length }) }}
          </h5>
          <div v-if="!testPreview.Rows.length" class="templates-page__empty">
            <EmptyState :title="t('templates.noRows')" size="sm" />
          </div>
          <div v-else class="templates-page__table">
            <NDataTable
              :columns="testPreviewColumns"
              :data="testPreview.Rows"
              :row-key="(row: Record<string, string>) => testPreview?.Rows.indexOf(row) ?? 0"
              size="small"
            />
          </div>

          <h5 class="templates-page__section-title">
            {{ t('templates.previewIssues', { n: testPreview.Issues.length }) }}
          </h5>
          <div v-if="!testPreview.Issues.length" class="templates-page__test-hint">
            {{ t('templates.noIssues') }}
          </div>
          <ul v-else class="templates-page__issue-list">
            <li v-for="(issue, index) in testPreview.Issues" :key="index">
              <span class="templates-page__issue-line">#{{ issue.LineNo }}</span>
              <span class="templates-page__issue-key">{{ testColumnTitle(issue.Key) }}</span>
              <span>{{ issue.Message }}</span>
            </li>
          </ul>
        </template>
      </NSpin>
    </NModal>

    <!-- Create Carrier Modal -->
    <NModal
      v-model:show="showCreateCarrierModal"
      preset="card"
      :title="t('library.createCarrierMapping')"
      style="width: 480px"
    >
      <NForm label-placement="left" label-width="110">
        <NFormItem :label="t('library.externalCode')">
          <NInput v-model:value="carrierForm.externalCode" />
        </NFormItem>
        <NFormItem :label="t('library.internalCode')">
          <NInput v-model:value="carrierForm.internalCode" />
        </NFormItem>
        <NFormItem :label="t('library.internalName')">
          <NInput v-model:value="carrierForm.internalName" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateCarrierModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!carrierForm.externalCode.trim()"
            @click="handleSaveCarrier"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.templates-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.templates-page__empty {
  padding: var(--space-4) 0;
}

.templates-page__table {
  width: 100%;
}

.templates-page__meta-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}

.templates-page__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.templates-page__create-modal {
  width: min(960px, 92vw);
}

.templates-page__create-modal :deep(.n-card__content) {
  max-height: 68vh;
  overflow-y: auto;
}

.templates-page__section-title {
  margin: var(--space-5) 0 var(--space-2);
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.templates-page__header-names {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-2);
}

.templates-page__header-names-title {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.templates-page__header-name-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
}

.templates-page__test-modal {
  width: min(860px, 92vw);
}

.templates-page__test-file {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
}

.templates-page__test-path {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  word-break: break-all;
}

.templates-page__test-error {
  margin: 0 0 var(--space-2);
  color: var(--status-error-fg);
  font-size: var(--font-size-sm);
}

.templates-page__test-hint {
  color: var(--color-text-muted);
  font-size: var(--font-size-sm);
}

.templates-page__issue-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: var(--font-size-sm);
}

.templates-page__issue-line {
  font-family: var(--font-mono);
  color: var(--color-text-muted);
  min-width: 48px;
  display: inline-block;
}

.templates-page__issue-key {
  color: var(--color-text-secondary);
  margin-right: var(--space-2);
}
</style>
