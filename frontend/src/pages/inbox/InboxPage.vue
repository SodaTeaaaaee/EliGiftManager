<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
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
import {
  assignLines,
  attachIdentity,
  ingestDocument,
  listCustomers,
  listInboxRows,
  listWaves,
} from '@/shared/api/bridge'
import type { CustomerProfile, InboxRow, Wave } from '@/entities/models'

const { t } = useI18n()

const loading = ref(false)
const actionLoading = ref(false)
const rows = ref<InboxRow[]>([])
const waves = ref<Wave[]>([])
const customers = ref<CustomerProfile[]>([])
const selectedLineKeys = ref<number[]>([])
const selectedDocumentFilter = ref<string>('all')

const showAssignModal = ref(false)
const selectedWaveId = ref<number | null>(null)

const showAttachModal = ref(false)
const selectedCustomerId = ref<number | null>(null)
const targetIdentityId = ref<number | null>(null)

const showIngestModal = ref(false)
const ingestForm = ref({
  originalName: '',
  kind: 'membership',
  identityValue: '',
  identityType: 'platform_uid',
  membershipLevel: '舰长',
  externalSku: '',
  externalTitle: '',
  quantity: 1,
})

async function loadData() {
  loading.value = true
  try {
    const [inboxRes, waveRes, custRes] = await Promise.all([
      listInboxRows(),
      listWaves(),
      listCustomers(),
    ])
    rows.value = inboxRes
    waves.value = waveRes.filter((w) => w.CloseResult === 'open' || !w.CloseResult)
    customers.value = custRes
  } catch (err) {
    console.error('Failed to load inbox data:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadData()
})

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
  if (selectedDocumentFilter.value === 'all') {
    return rows.value
  }
  return rows.value.filter(
    (r) => r.Document && String(r.Document.ID) === selectedDocumentFilter.value,
  )
})

const waveOptions = computed(() =>
  waves.value.map((w) => ({
    label: `${w.WaveNo} - ${w.Name}`,
    value: w.ID,
  })),
)

const customerOptions = computed(() =>
  customers.value.map((c) => ({
    label: `${c.DisplayName} (ID: ${c.ID})`,
    value: c.ID,
  })),
)

async function handleAssignWave() {
  if (!selectedWaveId.value || selectedLineKeys.value.length === 0) return
  actionLoading.value = true
  try {
    await assignLines(selectedWaveId.value, selectedLineKeys.value)
    showAssignModal.value = false
    selectedLineKeys.value = []
    selectedWaveId.value = null
    await loadData()
  } catch (err) {
    console.error('Failed to assign lines:', err)
  } finally {
    actionLoading.value = false
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
    console.error('Failed to attach identity:', err)
  } finally {
    actionLoading.value = false
  }
}

async function handleIngest() {
  actionLoading.value = true
  try {
    await ingestDocument(
      {
        PlatformID: 1,
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
    showIngestModal.value = false
    await loadData()
  } catch (err) {
    console.error('Failed to ingest document:', err)
  } finally {
    actionLoading.value = false
  }
}

const columns = [
  {
    type: 'selection' as const,
    disabled(row: InboxRow) {
      return row.Assigned
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
    width: 130,
    render(row: InboxRow) {
      return row.Unaligned
        ? h(StatusBadge, {
            dimension: 'blockReason',
            value: 'unaligned_product',
          })
        : h(NTag, { type: 'success', size: 'small' }, { default: () => t('common.yes') })
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
]
</script>

<template>
  <div class="inbox-page">
    <PageHeader :title="t('inbox.title')" :description="t('inbox.subtitle')">
      <template #actions>
        <NSpace>
          <NButton size="small" @click="showIngestModal = true">
            {{ t('inbox.importDocument') }}
          </NButton>
          <NButton
            size="small"
            type="primary"
            :disabled="selectedLineKeys.length === 0"
            @click="showAssignModal = true"
          >
            {{ t('inbox.assignToWave') }} ({{ selectedLineKeys.length }})
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

    <!-- Ingest Document Modal -->
    <NModal
      v-model:show="showIngestModal"
      preset="card"
      :title="t('inbox.importDocument')"
      style="width: 520px"
    >
      <NForm label-placement="left" label-width="100">
        <NFormItem :label="t('library.documentType')">
          <NInput v-model:value="ingestForm.originalName" />
        </NFormItem>
        <NFormItem :label="t('inbox.kind')">
          <NSelect
            v-model:value="ingestForm.kind"
            :options="[
              { label: 'membership', value: 'membership' },
              { label: 'retail_order', value: 'retail_order' },
              { label: 'operator_grant', value: 'operator_grant' },
            ]"
          />
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
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showIngestModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="actionLoading" @click="handleIngest">
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

.inbox-page__empty {
  padding: var(--space-4) 0;
}

.inbox-page__table {
  width: 100%;
}
</style>
