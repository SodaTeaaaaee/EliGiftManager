<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  NTimeline,
  NTimelineItem,
  NTag,
  NButton,
  NSpin,
  NEmpty,
  NTooltip,
  useMessage,
} from 'naive-ui'
import { getHistoryGraph, runHistoryGC, type HistoryGraphNodeDTO } from '@/shared/lib/wails/app'
import { useI18n } from '@/shared/i18n'

const props = defineProps<{ waveId: number | null }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const gcLoading = ref(false)
const nodes = ref<HistoryGraphNodeDTO[]>([])
const currentHeadId = ref(0)

// Display newest-first (reverse chronological)
const displayNodes = computed(() =>
  [...nodes.value].sort((a, b) => {
    if (a.createdAt < b.createdAt) return 1
    if (a.createdAt > b.createdAt) return -1
    return 0
  })
)

async function load() {
  if (!props.waveId) return
  loading.value = true
  try {
    const graph = await getHistoryGraph(props.waveId)
    nodes.value = graph.nodes ?? []
    currentHeadId.value = graph.currentHeadId ?? 0
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    loading.value = false
  }
}

async function handleCleanUp() {
  if (!props.waveId) return
  gcLoading.value = true
  try {
    const deleted = await runHistoryGC(props.waveId)
    const msg = t('wave.historyMeta.cleanUpDone').replace('{n}', String(deleted))
    message.success(msg)
    await load()
  } catch (e) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    gcLoading.value = false
  }
}

function commandKindLabel(kind: string): string {
  const key = `wave.commandKinds.${kind}` as any
  const resolved = t(key)
  // If t() returns the path key itself, it means no translation — fall back to raw kind
  return resolved === key ? kind : resolved
}

function timelineType(node: HistoryGraphNodeDTO): 'success' | 'info' | 'warning' | 'default' {
  if (node.isCurrentHead) return 'success'
  if (node.isPinned) return 'warning'
  if (node.checkpointHint) return 'info'
  return 'default'
}

watch(() => props.waveId, () => { void load() }, { immediate: true })
</script>

<template>
  <div class="history-panel">
    <div class="history-panel__header">
      <span class="history-panel__title">{{ t('wave.historyMeta.historyPanel') }}</span>
      <NButton size="small" :loading="gcLoading" @click="handleCleanUp">
        {{ t('wave.historyMeta.cleanUp') }}
      </NButton>
    </div>

    <NSpin v-if="loading" class="history-panel__spin" />

    <NEmpty
      v-else-if="!loading && displayNodes.length === 0"
      :description="t('wave.historyMeta.noHistory')"
      class="history-panel__empty"
    />

    <NTimeline v-else class="history-panel__timeline">
      <NTimelineItem
        v-for="node in displayNodes"
        :key="node.id"
        :type="timelineType(node)"
        :time="node.createdAt"
      >
        <template #header>
          <div class="history-node__header">
            <span class="history-node__summary">{{ node.commandSummary }}</span>
            <div class="history-node__tags">
              <NTag v-if="node.isCurrentHead" type="success" size="tiny" round>
                {{ t('wave.historyMeta.currentHead') }}
              </NTag>
              <NTag v-if="node.isPinned" type="warning" size="tiny" round>
                {{ t('wave.historyMeta.pinned') }}
              </NTag>
              <NTag v-if="node.checkpointHint" type="info" size="tiny" round>
                {{ t('wave.historyMeta.checkpoint') }}
              </NTag>
              <NTooltip v-if="node.childCount > 1" trigger="hover">
                <template #trigger>
                  <NTag type="default" size="tiny" round>{{ t('wave.historyMeta.branch') }}</NTag>
                </template>
                {{ node.childCount }} children
              </NTooltip>
            </div>
          </div>
        </template>
        <NTag size="tiny" :bordered="false" style="opacity: 0.7">
          {{ commandKindLabel(node.commandKind) }}
        </NTag>
      </NTimelineItem>
    </NTimeline>
  </div>
</template>

<style scoped>
.history-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px;
  overflow: hidden;
}

.history-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  flex-shrink: 0;
}

.history-panel__title {
  font-size: 15px;
  font-weight: 600;
}

.history-panel__spin {
  margin: auto;
}

.history-panel__empty {
  margin-top: 40px;
}

.history-panel__timeline {
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  padding-right: 4px;
}

.history-node__header {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  flex-wrap: wrap;
}

.history-node__summary {
  font-size: 13px;
  flex: 1;
  min-width: 0;
}

.history-node__tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  flex-shrink: 0;
}
</style>
