<script setup lang="ts">
import { computed, ref, provide, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, NAlert, NButton, NTag } from 'naive-ui'
import WaveStepWizard from '@/shared/ui/WaveStepWizard.vue'
import { useUndoRedo } from '@/shared/composables/useUndoRedo'
import { getWaveWorkspaceSnapshot } from '@/shared/lib/wails/app'
import { dto } from '@/../wailsjs/go/models'
import { useI18n } from '@/shared/i18n'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { t } = useI18n()

const waveId = computed(() => {
  const id = Number(route.params.waveId)
  return Number.isFinite(id) ? id : null
})

const refreshKey = ref(0)
provide('waveRefreshKey', refreshKey)
const workspaceSnapshot = ref<dto.WaveWorkspaceSnapshotDTO | null>(null)
provide('waveWorkspaceSnapshot', workspaceSnapshot)
const loading = ref(false)
const error = ref("")

async function loadWorkspaceSnapshot() {
  if (!waveId.value) return
  loading.value = true
  error.value = ""
  try {
    workspaceSnapshot.value = await getWaveWorkspaceSnapshot(waveId.value)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

useUndoRedo({
  scopeType: 'wave',
  scopeKey: () => waveId.value,
  onSuccess: (summary, action) => {
    const label = action === 'undo' ? '撤销' : '重做'
    message.success(`${label}：${summary}`)
    refreshKey.value++
    void loadWorkspaceSnapshot()
  },
  onError: (err) => {
    message.warning(err)
  },
  onNotReady: () => {
    message.info('撤销/重做：后端未连接')
  },
})

watch(waveId, () => {
  void loadWorkspaceSnapshot()
}, { immediate: true })

onMounted(() => {
  void loadWorkspaceSnapshot()
})

const stageTagType = computed(() => {
  switch (workspaceSnapshot.value?.projectedLifecycleStage) {
    case 'awaiting_manual_closure':
      return 'error'
    case 'syncing_back':
      return 'warning'
    case 'closed':
      return 'success'
    case 'execution':
      return 'info'
    default:
      return 'default'
  }
})
</script>

<template>
  <div class="wave-workspace">
    <div class="wave-shell-header">
      <div class="wave-shell-header__main">
        <div class="app-kicker">{{ workspaceSnapshot?.wave?.waveNo || t('wave.overview') }}</div>
        <h1 class="app-title mt-2">{{ workspaceSnapshot?.wave?.name || t('wave.overview') }}</h1>
      </div>
      <div class="wave-shell-header__actions">
        <NTag v-if="workspaceSnapshot?.projectedLifecycleStage" :type="stageTagType as any" size="small" round>
          {{ workspaceSnapshot?.projectedLifecycleStage }}
        </NTag>
        <NTag
          v-if="workspaceSnapshot?.basisSummary?.hasRequiredReview"
          type="error"
          size="small"
          round
        >
          {{ t('wave.reviewRequired') }}
        </NTag>
        <NTag
          v-else-if="workspaceSnapshot?.basisSummary?.hasDriftedBasis"
          type="warning"
          size="small"
          round
        >
          {{ t('wave.drifted') }}
        </NTag>
        <NButton secondary size="small" @click="router.push('/waves')">返回任务列表</NButton>
      </div>
    </div>
    <NAlert v-if="error" type="error" class="mb-4" :title="error" />
    <WaveStepWizard :snapshot="workspaceSnapshot" />
    <div class="wave-shell-content">
      <router-view :key="refreshKey" />
    </div>
  </div>
</template>

<style scoped>
.wave-workspace {
  display: flex;
  flex-direction: column;
  min-height: 100%;
}

.wave-shell-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 18px;
}

.wave-shell-header__actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.wave-shell-content {
  flex: 1;
  min-height: 0;
}
</style>
