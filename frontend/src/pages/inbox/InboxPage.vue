<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  NSpin,
  NTag,
} from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import { useFeedback } from '@/shared/ui/feedback'
import DuplicateDecisionList from './DuplicateDecisionList.vue'
import {
  applyRevision,
  assignLines,
  attachIdentity,
  dismissRevision,
  importFile,
  ingestDocument,
  listCustomers,
  listInboxRows,
  listPlatforms,
  listProducts,
  listTemplates,
  listWaves,
  moveLines,
  pickFile,
  updateAlias,
} from '@/shared/api/bridge'
import { identityTypeValues, inputFactKindValues } from '@/shared/api/generated/enums'
import type {
  CustomerProfile,
  ImportFileResult,
  IngestDocumentResult,
  InboxRow,
  Platform,
  ProductItem,
  TemplateConfig,
  Wave,
} from '@/entities/models'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const feedback = useFeedback()

function errMsg(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

const loading = ref(false)
const actionLoading = ref(false)
const rows = ref<InboxRow[]>([])
const waves = ref<Wave[]>([])
const customers = ref<CustomerProfile[]>([])
const platforms = ref<Platform[]>([])
const templates = ref<TemplateConfig[]>([])
const products = ref<ProductItem[]>([])
const selectedLineKeys = ref<number[]>([])
const selectedDocumentFilter = ref<string>('all')

const showAssignModal = ref(false)
const selectedWaveId = ref<number | null>(null)

const showAttachModal = ref(false)
const selectedCustomerId = ref<number | null>(null)
const targetIdentityId = ref<number | null>(null)

const showIngestModal = ref(false)
const ingestForm = ref({
  platformId: null as number | null,
  originalName: '',
  kind: 'membership',
  identityValue: '',
  identityType: 'platform_uid',
  membershipLevel: '舰长',
  externalSku: '',
  externalTitle: '',
  quantity: 1,
})
const ingestResult = ref<IngestDocumentResult | null>(null)

// ── Move-to-wave (MoveLines) ──

const showMoveModal = ref(false)
const moveTargetWaveId = ref<number | null>(null)
const moveLineIds = ref<number[]>([])
const moveExcludeWaveId = ref<number | null>(null)

// ── File import (ImportFile) ──

const importFileFilters = [{ displayName: 'CSV / Excel', pattern: '*.csv;*.xlsx;*.xls' }]
const showImportFileModal = ref(false)
const importForm = ref({
  platformId: null as number | null,
  templateId: null as number | null,
  filePath: '',
})
const importResult = ref<ImportFileResult | null>(null)
const importError = ref('')

// ── Align-to-product (UpdateAlias) ──

const showAlignModal = ref(false)
const targetAliasId = ref<number | null>(null)
const selectedAlignProductId = ref<number | null>(null)

async function loadData() {
  loading.value = true
  try {
    const [inboxRes, waveRes, custRes, platRes, tmplRes] = await Promise.all([
      listInboxRows(),
      listWaves(),
      listCustomers(),
      listPlatforms(),
      listTemplates(),
    ])
    rows.value = inboxRes
    waves.value = waveRes.filter((w) => w.CloseResult === 'open' || !w.CloseResult)
    customers.value = custRes
    platforms.value = platRes
    templates.value = tmplRes
  } catch (err) {
    console.error('Failed to load inbox data:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadData()
  // Duplicate observations have no inbox row view; the deep link falls back
  // to the four decision-pending row categories and says so once.
  if (homeFilter.value === 'duplicates') {
    feedback.info(t('inbox.duplicatesFilterHint'))
  }
})

// ── Home deep-link filter (?filter=…) ──

const homeFilter = computed(() => {
  const raw = route.query.filter
  return typeof raw === 'string' ? raw : ''
})

/** Home bucket cards deep-link one of these filter values; unknown values
 * keep the safe fallback (no filtering) but hide the tag since there is no
 * honest bucket name to display. */
const homeFilterBucketKeys: Record<string, string> = {
  unassigned: 'home.buckets.unassigned',
  duplicates: 'home.buckets.duplicateAsk',
  alignment: 'home.buckets.alignmentConflict',
  unattached: 'home.buckets.identityUnattached',
  revisions: 'home.buckets.pendingRevisions',
}

const isKnownHomeFilter = computed(() => Boolean(homeFilterBucketKeys[homeFilter.value]))

const deepLinkFilterLabel = computed(() => {
  const key = homeFilterBucketKeys[homeFilter.value]
  return key ? t('common.deepLinkFilter', { filter: t(key) }) : ''
})

function matchesHomeFilter(row: InboxRow): boolean {
  switch (homeFilter.value) {
    case 'unassigned':
      return !row.Assigned && !row.RevisionPending
    case 'duplicates':
      return row.RevisionPending || row.Unaligned || row.Unattached || !row.Assigned
    case 'alignment':
      return row.Unaligned
    case 'unattached':
      return row.Unattached
    case 'revisions':
      return row.RevisionPending
    default:
      return true
  }
}

function clearHomeFilter() {
  void router.replace({ query: { ...route.query, filter: undefined } })
}

const documentOptions = computed(() => {
  const map = new Map<string, string>()
  for (const r of rows.value) {
    if (r.Document) {
      map.set(String(r.Document.ID), r.Document.OriginalName || `Doc #${r.Document.ID}`)
    }
  }
  const opts = [{ label: t('inbox.allDocuments'), value: 'all' }]
  for (const [id, name] of map.entries()) {
    opts.push({ label: name, value: id })
  }
  return opts
})

const filteredRows = computed(() => {
  let list = rows.value
  if (selectedDocumentFilter.value !== 'all') {
    list = list.filter(
      (r) => r.Document && String(r.Document.ID) === selectedDocumentFilter.value,
    )
  }
  if (homeFilter.value) {
    list = list.filter(matchesHomeFilter)
  }
  return list
})

const waveOptions = computed(() =>
  waves.value.map((w) => ({
    label: `${w.WaveNo} - ${w.Name}`,
    value: w.ID,
  })),
)

const moveWaveOptions = computed(() =>
  waves.value
    .filter((w) => w.ID !== moveExcludeWaveId.value)
    .map((w) => ({
      label: `${w.WaveNo} - ${w.Name}`,
      value: w.ID,
    })),
)

/** Rows currently covered by the table's checkbox selection. */
const selectedRows = computed(() => {
  const ids = new Set(selectedLineKeys.value)
  return rows.value.filter((r) => ids.has(r.Line.ID))
})

const selectedUnassignedKeys = computed(() =>
  selectedRows.value.filter((r) => !r.Assigned).map((r) => r.Line.ID),
)

const selectedAssignedKeys = computed(() =>
  selectedRows.value.filter((r) => r.Assigned).map((r) => r.Line.ID),
)

const customerOptions = computed(() =>
  customers.value.map((c) => ({
    label: `${c.DisplayName} (ID: ${c.ID})`,
    value: c.ID,
  })),
)

const platformOptions = computed(() =>
  platforms.value.map((p) => ({
    label: `${p.Name} (${p.Key})`,
    value: p.ID,
  })),
)

/** Input-direction templates for the selected platform. */
const importTemplateOptions = computed(() =>
  templates.value
    .filter(
      (tpl) =>
        tpl.Direction === 'input' &&
        (!importForm.value.platformId || tpl.PlatformID === importForm.value.platformId),
    )
    .map((tpl) => ({
      label: `${tpl.Name} (${tpl.DocumentType})`,
      value: tpl.ID,
    })),
)

const kindOptions = inputFactKindValues.map((kind) => ({
  label: t(`glossary.inputFactKind.${kind}.label`),
  value: kind,
}))

const identityTypeOptions = identityTypeValues.map((type) => ({
  label: t(`glossary.identityType.${type}.label`),
  value: type,
}))

async function handleAssignWave() {
  if (!selectedWaveId.value || selectedUnassignedKeys.value.length === 0) return
  actionLoading.value = true
  try {
    await assignLines(selectedWaveId.value, selectedUnassignedKeys.value)
    showAssignModal.value = false
    selectedLineKeys.value = []
    selectedWaveId.value = null
    await loadData()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

// ── Move-to-wave flow ──

function openMoveModal(lineIds: number[], excludeWaveId: number | null) {
  if (!lineIds.length) return
  moveLineIds.value = lineIds
  moveExcludeWaveId.value = excludeWaveId
  moveTargetWaveId.value = null
  showMoveModal.value = true
}

async function handleMoveLines() {
  if (!moveTargetWaveId.value || moveLineIds.value.length === 0) return
  actionLoading.value = true
  try {
    await moveLines(moveLineIds.value, moveTargetWaveId.value)
    showMoveModal.value = false
    moveLineIds.value = []
    moveExcludeWaveId.value = null
    moveTargetWaveId.value = null
    selectedLineKeys.value = []
    feedback.success(t('inbox.moveSuccess'))
    await loadData()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

// ── Revision decisions (ApplyRevision / DismissRevision) ──

const revisionActingFactId = ref<number | null>(null)

async function handleApplyRevision(row: InboxRow) {
  if (revisionActingFactId.value !== null) return
  revisionActingFactId.value = row.Fact.ID
  try {
    await applyRevision(row.Fact.ID)
    feedback.success(t('inbox.applyRevisionSuccess'))
    await loadData()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    revisionActingFactId.value = null
  }
}

async function handleDismissRevision(row: InboxRow) {
  if (revisionActingFactId.value !== null) return
  revisionActingFactId.value = row.Fact.ID
  try {
    await dismissRevision(row.Fact.ID)
    feedback.success(t('inbox.dismissRevisionSuccess'))
    await loadData()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    revisionActingFactId.value = null
  }
}

function openAttachModal(identityId: number) {
  targetIdentityId.value = identityId
  selectedCustomerId.value = null
  showAttachModal.value = true
}

async function handleAttachIdentity() {
  if (!targetIdentityId.value || !selectedCustomerId.value) return
  actionLoading.value = true
  try {
    await attachIdentity(targetIdentityId.value, selectedCustomerId.value)
    showAttachModal.value = false
    targetIdentityId.value = null
    selectedCustomerId.value = null
    await loadData()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

function openIngestModal() {
  ingestResult.value = null
  showIngestModal.value = true
}

async function handleIngest() {
  if (!ingestForm.value.platformId) return
  actionLoading.value = true
  try {
    const res = await ingestDocument(
      {
        PlatformID: ingestForm.value.platformId,
        DocumentType: 'manual_ingest',
        Direction: 'input',
        OriginalName: ingestForm.value.originalName || 'Manual Ingest',
      },
      [
        {
          Kind: ingestForm.value.kind,
          IdentityType: ingestForm.value.identityType,
          IdentityValue: ingestForm.value.identityValue,
          MembershipLevel: ingestForm.value.membershipLevel,
          Lines: [
            {
              SourceLineNo: 1,
              ExternalSKU: ingestForm.value.externalSku,
              ExternalTitle: ingestForm.value.externalTitle,
              ExternalSpec: '',
              Quantity: ingestForm.value.quantity,
            },
          ],
        },
      ],
    )
    if (res.Duplicates?.length) {
      // Keep the modal open so the operator can decide the fresh asks.
      ingestResult.value = res
    } else {
      showIngestModal.value = false
    }
    await loadData()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

// ── File import flow ──

function openImportFileModal() {
  importForm.value = {
    platformId: platformOptions.value[0]?.value ?? null,
    templateId: null,
    filePath: '',
  }
  importResult.value = null
  importError.value = ''
  showImportFileModal.value = true
}

function handleImportPlatformChange() {
  importForm.value.templateId = null
}

async function handlePickImportFile() {
  const path = await pickFile(importFileFilters)
  if (path) importForm.value.filePath = path
}

async function handleImportFile() {
  if (!importForm.value.platformId || !importForm.value.templateId || !importForm.value.filePath) return
  actionLoading.value = true
  importError.value = ''
  try {
    importResult.value = await importFile(
      importForm.value.platformId,
      importForm.value.templateId,
      importForm.value.filePath,
    )
    await loadData()
  } catch (err) {
    importResult.value = null
    importError.value = err instanceof Error ? err.message : String(err)
  } finally {
    actionLoading.value = false
  }
}

// ── Align-to-product flow ──

/** The row view resolves ExternalSKU to an alias id; null rows cannot align. */
function aliasIdForRow(row: InboxRow): number | null {
  const aliasID = row.AliasID
  return typeof aliasID === 'number' && aliasID > 0 ? aliasID : null
}

async function openAlignModal(row: InboxRow) {
  const aliasID = aliasIdForRow(row)
  if (!aliasID) return
  targetAliasId.value = aliasID
  selectedAlignProductId.value = null
  if (!products.value.length) {
    try {
      products.value = await listProducts()
    } catch (err) {
      feedback.error(t('feedback.error'), errMsg(err))
    }
  }
  showAlignModal.value = true
}

async function handleAlignToProduct() {
  if (!targetAliasId.value || !selectedAlignProductId.value) return
  actionLoading.value = true
  try {
    await updateAlias(targetAliasId.value, selectedAlignProductId.value)
    showAlignModal.value = false
    targetAliasId.value = null
    selectedAlignProductId.value = null
    feedback.success(t('inbox.alignSuccess'))
    await loadData()
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

const columns = [
  {
    type: 'selection' as const,
    // Revision-pending facts must be applied or dismissed before they can
    // enter a wave (the backend refuses the assignment). Already-assigned
    // rows stay selectable for move-to-wave.
    disabled(row: InboxRow) {
      return row.RevisionPending
    },
  },
  {
    title: t('inbox.lineNo'),
    key: 'SourceLineNo',
    width: 80,
    render(row: InboxRow) {
      return `#${row.Line.SourceLineNo}`
    },
  },
  {
    title: t('inbox.kind'),
    key: 'Kind',
    width: 120,
    render(row: InboxRow) {
      return h(StatusBadge, {
        dimension: 'inputFactKind',
        value: row.Fact.Kind || 'membership',
      })
    },
  },
  {
    title: t('inbox.externalInfo'),
    key: 'ExternalInfo',
    render(row: InboxRow) {
      const parts = [
        row.Line.ExternalTitle,
        row.Line.ExternalSKU,
        row.Line.ExternalSpec,
      ].filter(Boolean)
      return parts.length ? parts.join(' / ') : '—'
    },
  },
  {
    title: t('inbox.quantity'),
    key: 'Quantity',
    width: 90,
    render(row: InboxRow) {
      return row.Line.Quantity
    },
  },
  {
    title: t('inbox.assigned'),
    key: 'Assigned',
    width: 110,
    render(row: InboxRow) {
      return row.Assigned
        ? h(NTag, { type: 'success', size: 'small' }, { default: () => t('common.yes') })
        : h(NTag, { type: 'warning', size: 'small' }, { default: () => t('common.no') })
    },
  },
  {
    title: t('inbox.unaligned'),
    key: 'Unaligned',
    width: 200,
    render(row: InboxRow) {
      if (!row.Unaligned) {
        return h(NTag, { type: 'success', size: 'small' }, { default: () => t('common.yes') })
      }
      return h('span', { class: 'inbox-page__unaligned-cell' }, [
        h(StatusBadge, { dimension: 'blockReason', value: 'unaligned_product' }),
        h(
          NButton,
          {
            size: 'tiny',
            type: 'warning',
            secondary: true,
            disabled: !aliasIdForRow(row),
            onClick: () => openAlignModal(row),
          },
          { default: () => t('inbox.alignToProduct') },
        ),
      ])
    },
  },
  {
    title: t('inbox.unattached'),
    key: 'Unattached',
    width: 140,
    render(row: InboxRow) {
      if (row.Unattached && row.Fact.PlatformIdentityID) {
        return h(
          NButton,
          {
            size: 'tiny',
            type: 'warning',
            secondary: true,
            onClick: () => openAttachModal(row.Fact.PlatformIdentityID!),
          },
          { default: () => t('inbox.attachIdentity') },
        )
      }
      return row.Unattached
        ? h(StatusBadge, {
            dimension: 'blockReason',
            value: 'identity_unattached',
          })
        : h(NTag, { type: 'success', size: 'small' }, { default: () => t('common.yes') })
    },
  },
  {
    title: t('inbox.documentNo'),
    key: 'Document',
    render(row: InboxRow) {
      return row.Document?.OriginalName || row.Fact.SourceDocumentNo || '—'
    },
  },
  {
    title: t('inbox.revision'),
    key: 'RevisionPending',
    width: 110,
    render(row: InboxRow) {
      if (!row.RevisionPending) return null
      return h(
        NTag,
        { type: 'warning', size: 'small' },
        { default: () => t('inbox.revisionPending') },
      )
    },
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 240,
    render(row: InboxRow) {
      const buttons = []
      if (row.RevisionPending) {
        buttons.push(
          h(
            NButton,
            {
              size: 'tiny',
              type: 'primary',
              secondary: true,
              loading: revisionActingFactId.value === row.Fact.ID,
              disabled:
                revisionActingFactId.value !== null &&
                revisionActingFactId.value !== row.Fact.ID,
              onClick: () => void handleApplyRevision(row),
            },
            { default: () => t('inbox.applyRevision') },
          ),
          h(
            NButton,
            {
              size: 'tiny',
              type: 'warning',
              secondary: true,
              disabled: revisionActingFactId.value !== null,
              onClick: () => void handleDismissRevision(row),
            },
            { default: () => t('inbox.dismissRevision') },
          ),
        )
      }
      if (row.Assigned) {
        buttons.push(
          h(
            NButton,
            {
              size: 'tiny',
              secondary: true,
              onClick: () =>
                openMoveModal([row.Line.ID], row.Line.WaveID ?? null),
            },
            { default: () => t('inbox.moveToWave') },
          ),
        )
      }
      if (!buttons.length) return null
      return h('span', { class: 'inbox-page__action-cell' }, buttons)
    },
  },
]
</script>

<template>
  <div class="inbox-page">
    <PageHeader :title="t('inbox.title')" :description="t('inbox.subtitle')">
      <template #actions>
        <NSpace>
          <NButton size="small" @click="openIngestModal">
            {{ t('inbox.importDocument') }}
          </NButton>
          <NButton size="small" type="primary" @click="openImportFileModal">
            {{ t('inbox.importFile') }}
          </NButton>
          <NButton
            size="small"
            type="primary"
            :disabled="selectedUnassignedKeys.length === 0"
            @click="showAssignModal = true"
          >
            {{ t('inbox.assignToWave') }} ({{ selectedUnassignedKeys.length }})
          </NButton>
          <NButton
            size="small"
            :disabled="selectedAssignedKeys.length === 0"
            @click="openMoveModal(selectedAssignedKeys, null)"
          >
            {{ t('inbox.moveToWave') }} ({{ selectedAssignedKeys.length }})
          </NButton>
          <NButton size="small" :loading="loading" @click="loadData">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>
    </PageHeader>

    <SectionCard :title="t('inbox.title')">
      <template #actions>
        <div class="inbox-page__filter">
          <NTag
            v-if="isKnownHomeFilter"
            closable
            size="small"
            class="inbox-page__home-filter-tag"
            @close="clearHomeFilter"
          >
            {{ deepLinkFilterLabel }}
          </NTag>
          <NSelect
            v-model:value="selectedDocumentFilter"
            :options="documentOptions"
            size="small"
            style="width: 220px"
          />
        </div>
      </template>

      <NSpin :show="loading">
        <div v-if="!filteredRows.length" class="inbox-page__empty">
          <EmptyState :title="t('inbox.empty')" size="sm" />
        </div>
        <div v-else class="inbox-page__table">
          <NDataTable
            v-model:checked-row-keys="selectedLineKeys"
            :columns="columns"
            :data="filteredRows"
            :row-key="(row: InboxRow) => row.Line.ID"
            size="small"
          />
        </div>
      </NSpin>
    </SectionCard>

    <!-- Assign to Wave Modal -->
    <NModal
      v-model:show="showAssignModal"
      preset="card"
      :title="t('inbox.assignToWave')"
      style="width: 480px"
    >
      <NForm>
        <NFormItem :label="t('inbox.selectWave')">
          <NSelect
            v-model:value="selectedWaveId"
            :options="waveOptions"
            :placeholder="t('inbox.selectWave')"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showAssignModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!selectedWaveId"
            @click="handleAssignWave"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Attach Identity Modal -->
    <NModal
      v-model:show="showAttachModal"
      preset="card"
      :title="t('inbox.attachIdentity')"
      style="width: 480px"
    >
      <NForm>
        <NFormItem :label="t('inbox.selectCustomer')">
          <NSelect
            v-model:value="selectedCustomerId"
            :options="customerOptions"
            :placeholder="t('inbox.selectCustomer')"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showAttachModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!selectedCustomerId"
            @click="handleAttachIdentity"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Import File Modal -->
    <NModal
      v-model:show="showImportFileModal"
      preset="card"
      :title="t('inbox.importFile')"
      style="width: 560px"
    >
      <NForm label-placement="left" label-width="100">
        <NFormItem :label="t('inbox.selectPlatform')">
          <NSelect
            v-model:value="importForm.platformId"
            :options="platformOptions"
            :placeholder="t('common.pleaseSelect')"
            @update:value="handleImportPlatformChange"
          />
        </NFormItem>
        <NFormItem :label="t('inbox.selectTemplate')">
          <NSelect
            v-model:value="importForm.templateId"
            :options="importTemplateOptions"
            :placeholder="importTemplateOptions.length ? t('inbox.selectTemplate') : t('inbox.noTemplateForPlatform')"
          />
        </NFormItem>
        <NFormItem :label="t('inbox.filePath')">
          <div class="inbox-page__file-row">
            <NButton size="small" @click="handlePickImportFile">
              {{ importForm.filePath ? t('inbox.changeFile') : t('inbox.pickFile') }}
            </NButton>
            <span v-if="importForm.filePath" class="inbox-page__file-path">{{ importForm.filePath }}</span>
            <span v-else class="inbox-page__file-hint">{{ t('inbox.noFileSelected') }}</span>
          </div>
        </NFormItem>
      </NForm>

      <p v-if="importError" class="inbox-page__error">{{ importError }}</p>

      <div v-if="importResult" class="inbox-page__receipt">
        <h5 class="inbox-page__receipt-title">{{ t('inbox.importResult') }}</h5>
        <div class="inbox-page__receipt-stats">
          <span>{{ t('inbox.factsCreated') }}: {{ importResult.FactsCreated }}</span>
          <span>{{ t('inbox.linesCreated') }}: {{ importResult.LinesCreated }}</span>
          <span>{{ t('inbox.duplicates') }}: {{ importResult.Duplicates.length }}</span>
          <span>{{ t('inbox.issues') }}: {{ importResult.Issues.length }}</span>
        </div>
        <p class="inbox-page__receipt-hint">{{ t('inbox.duplicatesHint') }}</p>
        <div v-if="importResult.Duplicates.length" class="inbox-page__receipt-block">
          <div class="inbox-page__receipt-subtitle">{{ t('inbox.duplicates') }}</div>
          <DuplicateDecisionList
            :duplicates="importResult.Duplicates"
            @decided="() => void loadData()"
          />
        </div>
        <div v-if="importResult.Issues.length" class="inbox-page__receipt-block">
          <div class="inbox-page__receipt-subtitle">{{ t('inbox.issues') }}</div>
          <ul class="inbox-page__issue-list">
            <li v-for="(issue, index) in importResult.Issues" :key="index" class="inbox-page__issue-row">
              <span class="inbox-page__issue-line">#{{ issue.LineNo }}</span>
              <span class="inbox-page__issue-key">{{ issue.Key }}</span>
              <span>{{ issue.Message }}</span>
            </li>
          </ul>
        </div>
      </div>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="showImportFileModal = false">{{ t('common.close') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!importForm.platformId || !importForm.templateId || !importForm.filePath"
            @click="handleImportFile"
          >
            {{ t('common.import') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Move to Wave Modal -->
    <NModal
      v-model:show="showMoveModal"
      preset="card"
      :title="t('inbox.moveToWave')"
      style="width: 480px"
    >
      <NForm>
        <NFormItem :label="t('inbox.selectWave')">
          <NSelect
            v-model:value="moveTargetWaveId"
            :options="moveWaveOptions"
            :placeholder="t('inbox.selectWave')"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showMoveModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!moveTargetWaveId"
            @click="handleMoveLines"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Align to Product Modal -->
    <NModal
      v-model:show="showAlignModal"
      preset="card"
      :title="t('inbox.alignToProduct')"
      style="width: 480px"
    >
      <NForm>
        <NFormItem :label="t('library.productName')">
          <NSelect
            v-model:value="selectedAlignProductId"
            :options="
              products.map((p) => ({
                label: `${p.Name} (${p.FactorySKU})`,
                value: p.ID,
              }))
            "
            :placeholder="t('library.productName')"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showAlignModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!selectedAlignProductId"
            @click="handleAlignToProduct"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Ingest Document Modal -->
    <NModal
      v-model:show="showIngestModal"
      preset="card"
      :title="t('inbox.importDocument')"
      style="width: 520px"
    >
      <NForm label-placement="left" label-width="100">
        <NFormItem :label="t('inbox.selectPlatform')">
          <NSelect v-model:value="ingestForm.platformId" :options="platformOptions" />
        </NFormItem>
        <NFormItem :label="t('library.documentType')">
          <NInput v-model:value="ingestForm.originalName" />
        </NFormItem>
        <NFormItem :label="t('inbox.kind')">
          <NSelect v-model:value="ingestForm.kind" :options="kindOptions" />
        </NFormItem>
        <NFormItem :label="t('inbox.identityType')">
          <NSelect v-model:value="ingestForm.identityType" :options="identityTypeOptions" />
        </NFormItem>
        <NFormItem :label="t('library.identities')">
          <NInput v-model:value="ingestForm.identityValue" />
        </NFormItem>
        <NFormItem :label="t('library.productName')">
          <NInput v-model:value="ingestForm.externalTitle" />
        </NFormItem>
        <NFormItem :label="t('inbox.quantity')">
          <NInputNumber v-model:value="ingestForm.quantity" :min="1" />
        </NFormItem>
      </NForm>

      <div v-if="ingestResult && ingestResult.Duplicates.length" class="inbox-page__receipt">
        <h5 class="inbox-page__receipt-title">{{ t('inbox.importResult') }}</h5>
        <p class="inbox-page__receipt-hint">{{ t('inbox.duplicatesHint') }}</p>
        <DuplicateDecisionList
          :duplicates="ingestResult.Duplicates"
          @decided="() => void loadData()"
        />
      </div>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="showIngestModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!ingestForm.platformId"
            @click="handleIngest"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.inbox-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.inbox-page__filter {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.inbox-page__home-filter-tag {
  flex-shrink: 0;
}

.inbox-page__empty {
  padding: var(--space-4) 0;
}

.inbox-page__table {
  width: 100%;
}

.inbox-page__unaligned-cell,
.inbox-page__action-cell {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.inbox-page__file-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.inbox-page__file-path {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  word-break: break-all;
}

.inbox-page__file-hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.inbox-page__error {
  margin: 0;
  color: var(--status-error-fg);
  font-size: var(--font-size-sm);
}

.inbox-page__receipt {
  margin-top: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.inbox-page__receipt-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.inbox-page__receipt-stats {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.inbox-page__receipt-hint {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.inbox-page__receipt-block {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.inbox-page__receipt-subtitle {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-secondary);
}

.inbox-page__issue-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: var(--font-size-sm);
}

.inbox-page__issue-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.inbox-page__issue-line {
  font-family: var(--font-mono);
  color: var(--color-text-muted);
  min-width: 40px;
}

.inbox-page__issue-key {
  color: var(--color-text-secondary);
}
</style>
