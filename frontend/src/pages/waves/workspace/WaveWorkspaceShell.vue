<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NRadio,
  NRadioGroup,
  NSpace,
  NSpin,
  NTabs,
  NTab,
} from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { StatusBadge } from '@/shared/ui/status'
import { closeWave, getWave, reopenWave } from '@/shared/api/bridge'
import type { Wave } from '@/entities/models'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.id))
const loading = ref(false)
const actionLoading = ref(false)
const wave = ref<Wave | null>(null)

const showCloseModal = ref(false)
const closeForm = ref({ result: 'clean', note: '' })

const activeTab = computed({
  get() {
    if (route.path.endsWith('/results')) return 'results'
    return 'rules'
  },
  set(val: string) {
    void router.push(`/waves/${waveId.value}/${val}`)
  },
})

async function loadWave() {
  if (!waveId.value || isNaN(waveId.value)) return
  loading.value = true
  try {
    wave.value = await getWave(waveId.value)
  } catch (err) {
    console.error('Failed to load wave:', err)
  } finally {
    loading.value = false
  }
}

watch(waveId, () => {
  void loadWave()
})

onMounted(() => {
  void loadWave()
})

function handleBack() {
  void router.push('/waves')
}

async function handleClose() {
  if (!wave.value) return
  actionLoading.value = true
  try {
    await closeWave(wave.value.ID, closeForm.value.result, closeForm.value.note)
    showCloseModal.value = false
    await loadWave()
  } catch (err) {
    console.error('Failed to close wave:', err)
  } finally {
    actionLoading.value = false
  }
}

async function handleReopen() {
  if (!wave.value) return
  actionLoading.value = true
  try {
    await reopenWave(wave.value.ID)
    await loadWave()
  } catch (err) {
    console.error('Failed to reopen wave:', err)
  } finally {
    actionLoading.value = false
  }
}
</script>

<template>
  <div class="wave-workspace-shell">
    <NSpin :show="loading && !wave">
      <PageHeader
        :title="wave ? `${wave.WaveNo} - ${wave.Name}` : t('waves.title')"
        :description="wave?.Notes || t('waveWorkspace.waveInfo')"
      >
        <template #actions>
          <NSpace align="center">
            <StatusBadge
              v-if="wave"
              dimension="waveCloseResult"
              :value="wave.CloseResult || 'open'"
            />
            <NButton
              v-if="wave && (wave.CloseResult === 'open' || !wave.CloseResult)"
              size="small"
              type="warning"
              @click="showCloseModal = true"
            >
              {{ t('waves.close') }}
            </NButton>
            <NButton
              v-else-if="wave"
              size="small"
              type="info"
              :loading="actionLoading"
              @click="handleReopen"
            >
              {{ t('waves.reopen') }}
            </NButton>
            <NButton size="small" @click="handleBack">
              {{ t('common.back') }}
            </NButton>
          </NSpace>
        </template>
      </PageHeader>

      <div class="wave-workspace-shell__tabs">
        <NTabs v-model:value="activeTab" type="line" animated>
          <NTab name="rules">{{ t('waveWorkspace.rulesTab') }}</NTab>
          <NTab name="results">{{ t('waveWorkspace.resultsTab') }}</NTab>
        </NTabs>
      </div>

      <div class="wave-workspace-shell__content">
        <RouterView :wave-id="waveId" :wave="wave" @refresh="loadWave" />
      </div>
    </NSpin>

    <!-- Close Modal -->
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
.wave-workspace-shell {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.wave-workspace-shell__tabs {
  margin-bottom: var(--space-2);
}

.wave-workspace-shell__content {
  min-height: 400px;
}
</style>
