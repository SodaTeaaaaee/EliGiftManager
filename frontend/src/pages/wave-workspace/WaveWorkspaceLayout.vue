<script setup lang="ts">
import { computed, ref, provide, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  useMessage,
  NAlert,
  NButton,
  NTag,
  NDrawer,
  NDrawerContent,
  NLayout,
  NLayoutSider,
  NLayoutContent,
  NIcon,
} from 'naive-ui'
import { ArrowBackOutline, GitNetworkOutline } from '@vicons/ionicons5'
import WaveWorkspaceSidebar from '@/shared/ui/WaveWorkspaceSidebar.vue'
import UndoRedoTrayPanel from '@/shared/ui/UndoRedoTrayPanel.vue'
import { useUndoRedo } from '@/shared/composables/useUndoRedo'
import { getWaveWorkspaceSnapshot } from '@/shared/lib/wails/app'
import { dto } from '@/../wailsjs/go/models'
import { useI18n } from '@/shared/i18n'
import { useSidebarStore } from '@/shared/model/sidebar'
import WaveHistoryPanel from './WaveHistoryPanel.vue'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const { t } = useI18n()
const sidebar = useSidebarStore()

const waveId = computed(() => {
  const id = Number(route.params.waveId)
  return Number.isFinite(id) ? id : null
})

const refreshKey = ref(0)
provide('waveRefreshKey', refreshKey)
const workspaceSnapshot = ref<dto.WaveWorkspaceSnapshotDTO | null>(null)
provide('waveWorkspaceSnapshot', workspaceSnapshot)
const loading = ref(false)
const error = ref('')

const historyDrawerOpen = ref(false)
const undoTrayRef = ref<InstanceType<typeof UndoRedoTrayPanel> | null>(null)

async function loadWorkspaceSnapshot() {
  if (!waveId.value) return
  loading.value = true
  error.value = ''
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
    const label = action === 'undo' ? t('undoTray.undoLabel') : t('undoTray.redoLabel')
    message.success(`${label}：${summary}`)
    undoTrayRef.value?.pushEntry(action, summary)
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

watch(
  waveId,
  () => {
    void loadWorkspaceSnapshot()
  },
  { immediate: true },
)

onMounted(() => {
  void loadWorkspaceSnapshot()
})

const stageTagType = computed(() => {
  switch (workspaceSnapshot.value?.projectedLifecycleStage) {
    case 'awaiting_manual_closure':
      return 'error' as const
    case 'syncing_back':
      return 'warning' as const
    case 'closed':
      return 'success' as const
    case 'execution':
      return 'info' as const
    default:
      return 'default' as const
  }
})

function openHistoryTreeSubpage() {
  if (waveId.value) router.push(`/waves/${waveId.value}/history`)
}
</script>

<template>
  <n-layout has-sider class="wave-workspace-layout">
    <n-layout-sider
      bordered
      collapse-mode="width"
      :width="280"
      :collapsed-width="64"
      :collapsed="sidebar.waveCollapsed"
      show-trigger="arrow-circle"
      content-style="background: transparent;"
      style="background: transparent;"
      @update:collapsed="sidebar.setWaveCollapsed"
    >
      <div class="sidebar-wrapper">
        <div class="sidebar-top-actions" :class="{ 'is-collapsed': sidebar.waveCollapsed }">
          <NButton text class="back-btn" @click="router.push('/waves')">
            <template #icon>
              <NIcon><ArrowBackOutline /></NIcon>
            </template>
            <span v-if="!sidebar.waveCollapsed">{{ t('wave.returnToQueue') }}</span>
          </NButton>
        </div>
        <WaveWorkspaceSidebar :snapshot="workspaceSnapshot" :collapsed="sidebar.waveCollapsed" />
      </div>
    </n-layout-sider>

    <n-layout-content content-style="background: transparent;">
      <div class="wave-content-wrapper">
        <!-- Top status bar — stage / drift / actions -->
        <div class="wave-content-header">
          <div class="status-tags">
            <NTag
              v-if="workspaceSnapshot?.projectedLifecycleStage"
              :type="stageTagType"
              size="small"
              round
              :bordered="false"
            >
              {{ workspaceSnapshot?.projectedLifecycleStage }}
            </NTag>
            <NTag
              v-if="workspaceSnapshot?.basisSummary?.hasRequiredReview"
              type="error"
              size="small"
              round
              :bordered="false"
            >
              {{ t('wave.reviewRequired') }}
            </NTag>
            <NTag
              v-else-if="workspaceSnapshot?.basisSummary?.hasDriftedBasis"
              type="warning"
              size="small"
              round
              :bordered="false"
            >
              {{ t('wave.drifted') }}
            </NTag>
          </div>
          <div class="status-actions">
            <UndoRedoTrayPanel ref="undoTrayRef" :wave-id="waveId" />
            <NButton secondary size="small" @click="openHistoryTreeSubpage">
              <template #icon>
                <NIcon><GitNetworkOutline /></NIcon>
              </template>
              {{ t('waveSidebar.historyTree') }}
            </NButton>
            <NButton secondary size="small" @click="historyDrawerOpen = true">
              {{ t('waveSidebar.historyDrawer') }}
            </NButton>
          </div>
        </div>

        <NAlert v-if="error" type="error" class="mb-4" :title="error" />

        <div class="wave-shell-content">
          <router-view :key="refreshKey" />
        </div>
      </div>
    </n-layout-content>

    <NDrawer v-model:show="historyDrawerOpen" :width="380" placement="right">
      <NDrawerContent
        :title="t('wave.historyMeta.historyPanel')"
        :native-scrollbar="false"
        closable
      >
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
  flex-shrink: 0;
  padding: 12px 20px;
  background: var(--surface-strong);
  border-right: 1px solid rgba(148, 163, 184, 0.12);
}

.sidebar-top-actions.is-collapsed {
  padding: 12px 0;
  display: flex;
  justify-content: center;
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
  padding: 20px 28px;
  overflow-y: auto;
}

.wave-content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 18px;
  gap: 12px;
  flex-wrap: wrap;
}

.status-tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.status-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.wave-shell-content {
  flex: 1;
  min-height: 0;
}

.mb-4 {
  margin-bottom: 16px;
}
</style>
