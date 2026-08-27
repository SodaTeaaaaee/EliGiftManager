<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
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
  NSpace,
  NSpin,
  NTag,
} from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import {
  closeWave,
  createWave,
  listResultViews,
  listWaves,
  reopenWave,
} from '@/shared/api/bridge'
import type { Wave } from '@/entities/models'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const loading = ref(false)
const actionLoading = ref(false)
const waves = ref<Wave[]>([])

const showCreateModal = ref(false)
const createForm = ref({ name: '', notes: '' })

const showCloseModal = ref(false)
const targetCloseWave = ref<Wave | null>(null)
const closeForm = ref({ result: 'clean', note: '' })

// ── Home deep-link filter (?filter=blocked|writebackFailed|residual) ──

const homeFilter = computed(() => {
  const raw = route.query.filter
  return typeof raw === 'string' ? raw : ''
})

interface WaveFlags {
  blocked: boolean
  writebackFailed: boolean
}

const flagsLoading = ref(false)
const waveFlags = ref<Record<number, WaveFlags>>({})

async function loadWaveFlags() {
  if (homeFilter.value !== 'blocked' && homeFilter.value !== 'writebackFailed') return
  flagsLoading.value = true
  try {
    const flags: Record<number, WaveFlags> = {}
    for (const w of waves.value) {
      const open = w.CloseResult === 'open' || !w.CloseResult
      // The blocked bucket only counts open waves (home semantics); writeback
      // failures can linger on closed waves, so those still get inspected.
      if (homeFilter.value === 'blocked' && !open) {
        flags[w.ID] = { blocked: false, writebackFailed: false }
        continue
      }
      try {
        const views = await listResultViews(w.ID)
        flags[w.ID] = {
          blocked: views.some((v) => v.WorkState === 'blocked'),
          writebackFailed: views.some((v) => v.WritebackFailed),
        }
      } catch (err) {
        console.error('Failed to load result views for wave flags:', err)
        flags[w.ID] = { blocked: false, writebackFailed: false }
      }
    }
    waveFlags.value = flags
  } finally {
    flagsLoading.value = false
  }
}

const displayWaves = computed(() => {
  switch (homeFilter.value) {
    case 'blocked':
      return waves.value.filter((w) => waveFlags.value[w.ID]?.blocked)
    case 'writebackFailed':
      return waves.value.filter((w) => waveFlags.value[w.ID]?.writebackFailed)
    case 'residual':
      return waves.value.filter((w) => w.CloseResult === 'residual')
    default:
      return waves.value
  }
})

function clearHomeFilter() {
  void router.replace({ query: { ...route.query, filter: undefined } })
}

async function refreshAll() {
  await loadData()
  await loadWaveFlags()
}

watch(
  () => route.query.filter,
  () => {
    void loadWaveFlags()
  },
)

onMounted(() => {
  void refreshAll()
})

async function loadData() {
  loading.value = true
  try {
    waves.value = await listWaves()
  } catch (err) {
    console.error('Failed to load waves:', err)
  } finally {
    loading.value = false
  }
}

function openWorkspace(wave: Wave) {
  void router.push(`/waves/${wave.ID}/rules`)
}

async function handleCreate() {
  if (!createForm.value.name.trim()) return
  actionLoading.value = true
  try {
    const wave = await createWave(createForm.value.name.trim(), createForm.value.notes.trim())
    showCreateModal.value = false
    createForm.value = { name: '', notes: '' }
    await loadData()
    if (wave && wave.ID) {
      void router.push(`/waves/${wave.ID}/rules`)
    }
  } catch (err) {
    console.error('Failed to create wave:', err)
  } finally {
    actionLoading.value = false
  }
}

function openCloseModal(wave: Wave) {
  targetCloseWave.value = wave
  closeForm.value = { result: 'clean', note: '' }
  showCloseModal.value = true
}

async function handleClose() {
  if (!targetCloseWave.value) return
  actionLoading.value = true
  try {
    await closeWave(targetCloseWave.value.ID, closeForm.value.result, closeForm.value.note)
    showCloseModal.value = false
    targetCloseWave.value = null
    await loadData()
  } catch (err) {
    console.error('Failed to close wave:', err)
  } finally {
    actionLoading.value = false
  }
}

async function handleReopen(wave: Wave) {
  actionLoading.value = true
  try {
    await reopenWave(wave.ID)
    await loadData()
  } catch (err) {
    console.error('Failed to reopen wave:', err)
  } finally {
    actionLoading.value = false
  }
}

const columns = [
  {
    title: t('waves.waveNo'),
    key: 'WaveNo',
    width: 140,
  },
  {
    title: t('waves.name'),
    key: 'Name',
  },
  {
    title: t('waves.status'),
    key: 'CloseResult',
    width: 130,
    render(row: Wave) {
      return h(StatusBadge, {
        dimension: 'waveCloseResult',
        value: row.CloseResult || 'open',
      })
    },
  },
  {
    title: t('waves.notes'),
    key: 'Notes',
    ellipsis: { tooltip: true },
  },
  {
    title: t('waves.createdAt'),
    key: 'CreatedAt',
    width: 170,
    render(row: Wave) {
      if (!row.CreatedAt) return '—'
      const date = new Date(row.CreatedAt)
      return isNaN(date.getTime()) ? row.CreatedAt : date.toLocaleDateString()
    },
  },
  {
    title: t('waves.actions'),
    key: 'actions',
    width: 220,
    render(row: Wave) {
      const isOpen = row.CloseResult === 'open' || !row.CloseResult
      return h(NSpace, { size: 'small' }, () => [
        h(
          NButton,
          {
            size: 'small',
            type: 'primary',
            quaternary: true,
            onClick: () => openWorkspace(row),
          },
          { default: () => t('waves.open') },
        ),
        isOpen
          ? h(
              NButton,
              {
                size: 'small',
                type: 'warning',
                quaternary: true,
                onClick: () => openCloseModal(row),
              },
              { default: () => t('waves.close') },
            )
          : h(
              NButton,
              {
                size: 'small',
                type: 'info',
                quaternary: true,
                onClick: () => handleReopen(row),
              },
              { default: () => t('waves.reopen') },
            ),
      ])
    },
  },
]
</script>

<template>
  <div class="waves-page">
    <PageHeader :title="t('waves.title')" :description="t('waves.subtitle')">
      <template #actions>
        <NSpace>
          <NButton type="primary" size="small" @click="showCreateModal = true">
            {{ t('waves.createWave') }}
          </NButton>
          <NButton size="small" :loading="loading" @click="refreshAll">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>
    </PageHeader>

    <SectionCard :title="t('waves.title')">
      <template #actions>
        <div class="waves-page__filter">
          <NTag
            v-if="homeFilter"
            closable
            size="small"
            @close="clearHomeFilter"
          >
            {{ t('common.deepLinkFilter') }}
          </NTag>
        </div>
      </template>
      <NSpin :show="loading || flagsLoading">
        <div v-if="!displayWaves.length" class="waves-page__empty">
          <EmptyState :title="t('waves.empty')" size="sm" />
        </div>
        <div v-else class="waves-page__table">
          <NDataTable
            :columns="columns"
            :data="displayWaves"
            :row-key="(row: Wave) => row.ID"
            size="small"
          />
        </div>
      </NSpin>
    </SectionCard>

    <!-- Create Wave Modal -->
    <NModal
      v-model:show="showCreateModal"
      preset="card"
      :title="t('waves.createWave')"
      style="width: 480px"
    >
      <NForm>
        <NFormItem :label="t('waves.name')">
          <NInput v-model:value="createForm.name" :placeholder="t('waves.name')" />
        </NFormItem>
        <NFormItem :label="t('waves.notes')">
          <NInput
            v-model:value="createForm.notes"
            type="textarea"
            :placeholder="t('waves.notes')"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!createForm.name.trim()"
            @click="handleCreate"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Close Wave Modal -->
    <NModal
      v-model:show="showCloseModal"
      preset="card"
      :title="t('waves.close')"
      style="width: 480px"
    >
      <NForm>
        <NFormItem :label="t('waves.status')">
          <NRadioGroup v-model:value="closeForm.result">
            <NSpace>
              <NRadio value="clean">{{ t('waves.cleanClose') }}</NRadio>
              <NRadio value="residual">{{ t('waves.residualClose') }}</NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="t('waves.closeNote')">
          <NInput
            v-model:value="closeForm.note"
            type="textarea"
            :placeholder="t('waves.closeNote')"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCloseModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="actionLoading" @click="handleClose">
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.waves-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.waves-page__empty {
  padding: var(--space-4) 0;
}

.waves-page__filter {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.waves-page__table {
  width: 100%;
}
</style>
