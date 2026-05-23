<script setup lang="ts">
import { computed, ref, provide, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, NAlert, NButton, NTag, NDrawer, NDrawerContent, NLayout, NLayoutSider, NLayoutContent, NIcon } from 'naive-ui'
import { ArrowBackOutline } from '@vicons/ionicons5'
import WaveWorkspaceSidebar from '@/shared/ui/WaveWorkspaceSidebar.vue'
import { useUndoRedo } from '@/shared/composables/useUndoRedo'
import { getWaveWorkspaceSnapshot } from '@/shared/lib/wails/app'
import { dto } from '@/../wailsjs/go/models'
import { useI18n } from '@/shared/i18n'
import WaveHistoryPanel from './WaveHistoryPanel.vue'

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

const historyDrawerOpen = ref(false)

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
    message.info(t('wave.undoRedoBackendUnavailable'))
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
  <n-layout has-sider class="wave-workspace-layout">
    <n-layout-sider bordered :width="260" content-style="background: transparent;" style="background: transparent;">
      <div class="sidebar-wrapper">
        <div class="sidebar-top-actions">
          <NButton text class="back-btn" @click="router.push('/waves')">
            <template #icon>
              <NIcon><ArrowBackOutline /></NIcon>
            </template>
            {{ t('wave.returnToQueue') }}
          </NButton>
        </div>
        <WaveWorkspaceSidebar :snapshot="workspaceSnapshot" />
      </div>
    </n-layout-sider>
    
    <n-layout-content content-style="background: transparent;">
      <div class="wave-content-wrapper">
        <div class="wave-content-header mb-4 flex justify-between items-center">
          <div class="flex items-center gap-3">
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
          </div>
          <NButton secondary size="small" @click="historyDrawerOpen = true">
            {{ t('wave.historyMeta.historyPanel') }}
          </NButton>
        </div>
        
        <NAlert v-if="error" type="error" class="mb-4" :title="error" />
        
        <div class="wave-shell-content">
          <router-view :key="refreshKey" />
        </div>
      </div>
    </n-layout-content>

    <NDrawer v-model:show="historyDrawerOpen" :width="360" placement="right">
      <NDrawerContent :title="t('wave.historyMeta.historyPanel')" :native-scrollbar="false" closable>
        <WaveHistoryPanel :wave-id="waveId" @close="historyDrawerOpen = false" />
      </NDrawerContent>
    </NDrawer>
  </n-layout>
</template>

<style scoped>
.wave-workspace-layout {
  height: 100%;
  background: transparent;
}

.sidebar-wrapper {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.sidebar-top-actions {
  padding: 12px 20px;
  background: var(--surface-strong);
  border-right: 1px solid rgba(148, 163, 184, 0.12);
}

:root[data-theme='dark'] .sidebar-top-actions {
  background: rgba(15, 23, 42, 0.6);
  border-right-color: rgba(255, 255, 255, 0.05);
}

.back-btn {
  font-size: 13px;
  color: var(--muted);
  transition: color 0.2s ease;
}

.back-btn:hover {
  color: var(--text);
}

.wave-content-wrapper {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 24px 32px;
  overflow-y: auto;
}

.wave-shell-content {
  flex: 1;
  min-height: 0;
}
</style>

