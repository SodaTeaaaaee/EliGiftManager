<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
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
} from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import {
  closeWave,
  createWave,
  listWaves,
  reopenWave,
} from '@/shared/api/bridge'
import type { Wave } from '@/entities/models'

const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const actionLoading = ref(false)
const waves = ref<Wave[]>([])

const showCreateModal = ref(false)
const createForm = ref({ name: '', notes: '' })

const showCloseModal = ref(false)
const targetCloseWave = ref<Wave | null>(null)
const closeForm = ref({ result: 'clean', note: '' })

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

onMounted(() => {
  void loadData()
})

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
          <NButton size="small" :loading="loading" @click="loadData">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>
    </PageHeader>

    <SectionCard :title="t('waves.title')">
      <NSpin :show="loading">
        <div v-if="!waves.length" class="waves-page__empty">
          <EmptyState :title="t('waves.empty')" size="sm" />
        </div>
        <div v-else class="waves-page__table">
          <NDataTable :columns="columns" :data="waves" :row-key="(row: Wave) => row.ID" size="small" />
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

.waves-page__table {
  width: 100%;
}
</style>
