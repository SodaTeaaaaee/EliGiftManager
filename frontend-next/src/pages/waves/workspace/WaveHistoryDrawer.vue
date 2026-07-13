<script setup lang="ts">
/**
 * WaveHistoryDrawer — the wave-workspace undo/redo history timeline (plan
 * section 7, unit A). Compact mode renders the injected context's
 * `snapshot.recentHistory` (`HistoryNodeDTO[]`, no branch info — `isPinned`
 * isn't on that shape, so the compact view derives `isCurrentHead` locally
 * by comparing each node's id against `snapshot.historyHeadNodeId`). The
 * "full tree" toggle switches to `getHistoryGraph(waveId)`
 * (`HistoryGraphNodeDTO[]`), which carries server-computed
 * `isCurrentHead`/`isPinned`/`childCount` for the branch-aware view.
 *
 * Visual states reimplement OLD
 * `frontend/src/pages/wave-workspace/WaveHistoryPanel.vue:72-77`'s
 * isCurrentHead/isPinned/checkpointHint tag logic — NOT its NDrawer/
 * NTimeline markup, which is replaced by this app's token-styled timeline
 * inside the shared `DetailDrawer`. `commandSummary` is raw server audit
 * text; rendering it as-is is the foundations contract's documented
 * zero-hardcode exception (decision 4).
 *
 * The GC confirm step deliberately uses a local `NModal preset="dialog"`
 * rather than naive-ui's `useDialog()`: `useDialog()` requires an ancestor
 * `<NDialogProvider>`, and none is mounted anywhere in this app (`App.vue`
 * only wraps `NConfigProvider` — see its own header comment). Adding one
 * would mean editing `App.vue`, which is outside unit A's file set. A
 * self-contained `NModal` is also the codebase's existing confirm-dialog
 * convention (see `CloseWaveDialog.vue` / `RenameWaveDialog.vue`), so this
 * matches local precedent as closely as the "use Naive UI ... for the
 * confirm" instruction intended.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NSwitch } from 'naive-ui'
import { DetailDrawer } from '@/shared/ui/drawer'
import { EmptyState } from '@/shared/ui/empty-state'
import { useFeedback } from '@/shared/ui/feedback'
import { getHistoryGraph, runHistoryGC } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import type { dto } from '../../../../wailsjs/go/models'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ 'update:show': [boolean] }>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const ctx = useWaveWorkspaceContext()

interface DisplayNode {
  id: number
  commandSummary: string
  createdAt: string
  checkpointHint: boolean
  isCurrentHead: boolean
  isPinned: boolean
  childCount: number
}

function byCreatedAtDesc(a: { createdAt: string }, b: { createdAt: string }): number {
  if (a.createdAt < b.createdAt) return 1
  if (a.createdAt > b.createdAt) return -1
  return 0
}

// ── compact mode: the snapshot's own recentHistory, no extra fetch ──
const compactNodes = computed<DisplayNode[]>(() => {
  const snapshot = ctx.snapshot.value
  if (!snapshot) return []
  return [...snapshot.recentHistory].sort(byCreatedAtDesc).map((node) => ({
    id: node.id,
    commandSummary: node.commandSummary,
    createdAt: node.createdAt,
    checkpointHint: node.checkpointHint,
    isCurrentHead: node.id === snapshot.historyHeadNodeId,
    isPinned: false,
    childCount: 0,
  }))
})

// ── full-tree mode: getHistoryGraph(waveId), fetched on demand ──
const fullTree = ref(false)
const graphLoading = ref(false)
const graphNodes = ref<dto.HistoryGraphNodeDTO[]>([])

const treeNodes = computed<DisplayNode[]>(() =>
  [...graphNodes.value].sort(byCreatedAtDesc).map((node) => ({
    id: node.id,
    commandSummary: node.commandSummary,
    createdAt: node.createdAt,
    checkpointHint: node.checkpointHint,
    isCurrentHead: node.isCurrentHead,
    isPinned: node.isPinned,
    childCount: node.childCount,
  })),
)

const displayNodes = computed<DisplayNode[]>(() => (fullTree.value ? treeNodes.value : compactNodes.value))

async function loadGraph(): Promise<void> {
  graphLoading.value = true
  try {
    const graph = await getHistoryGraph(ctx.waveId.value)
    graphNodes.value = graph.nodes ?? []
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    graphLoading.value = false
  }
}

watch(fullTree, (value) => {
  if (value) void loadGraph()
})

// Re-fetch the full tree if the drawer is reopened while already toggled on
// (e.g. after a GC run changed the node set while the drawer was closed).
watch(
  () => props.show,
  (value) => {
    if (value && fullTree.value) void loadGraph()
  },
)

// ── GC (with confirm) ──
const gcConfirmVisible = ref(false)
const gcLoading = ref(false)

async function runGC(): Promise<void> {
  gcLoading.value = true
  try {
    const deleted = await runHistoryGC(ctx.waveId.value)
    feedback.success(t('waveWorkspace.history.gcSuccess', { count: deleted }))
    gcConfirmVisible.value = false
    await ctx.refresh()
    if (fullTree.value) await loadGraph()
  } catch (err) {
    feedback.error(t('waveWorkspace.history.gcError'), err instanceof Error ? err.message : String(err))
  } finally {
    gcLoading.value = false
  }
}

function handleUpdateShow(value: boolean): void {
  emit('update:show', value)
}
</script>

<template>
  <DetailDrawer :show="show" size="md" :title="t('waveWorkspace.history.title')" @update:show="handleUpdateShow">
    <div class="wave-history-drawer__toolbar">
      <label class="wave-history-drawer__toggle">
        <NSwitch v-model:value="fullTree" size="small" :loading="graphLoading" />
        <span>{{ t('waveWorkspace.history.graphToggle') }}</span>
      </label>
      <NButton size="small" @click="gcConfirmVisible = true">
        {{ t('waveWorkspace.history.gcButton') }}
      </NButton>
    </div>

    <EmptyState v-if="!graphLoading && displayNodes.length === 0" size="sm" :title="t('waveWorkspace.history.empty')" />

    <ol v-else class="wave-history-drawer__timeline">
      <li v-for="node in displayNodes" :key="node.id" class="wave-history-drawer__node">
        <div class="wave-history-drawer__node-head">
          <span class="wave-history-drawer__summary">{{ node.commandSummary }}</span>
          <div class="wave-history-drawer__tags">
            <span v-if="node.isCurrentHead" class="wave-history-drawer__tag wave-history-drawer__tag--current">
              {{ t('waveWorkspace.history.currentTag') }}
            </span>
            <span v-if="node.isPinned" class="wave-history-drawer__tag wave-history-drawer__tag--pinned">
              {{ t('waveWorkspace.history.pinnedTag') }}
            </span>
            <span v-if="node.checkpointHint" class="wave-history-drawer__tag wave-history-drawer__tag--checkpoint">
              {{ t('waveWorkspace.history.checkpointTag') }}
            </span>
          </div>
        </div>
        <time class="wave-history-drawer__time tabular-nums">{{ node.createdAt }}</time>
      </li>
    </ol>
  </DetailDrawer>

  <NModal
    :show="gcConfirmVisible"
    preset="dialog"
    type="warning"
    :title="t('waveWorkspace.history.gcButton')"
    :content="t('waveWorkspace.history.gcConfirm')"
    :positive-text="t('common.confirm')"
    :negative-text="t('common.cancel')"
    :loading="gcLoading"
    @positive-click="runGC"
    @negative-click="gcConfirmVisible = false"
    @update:show="(value: boolean) => (gcConfirmVisible = value)"
  />
</template>

<style scoped>
.wave-history-drawer__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--color-border);
}

.wave-history-drawer__toggle {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  cursor: pointer;
}

.wave-history-drawer__timeline {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}

.wave-history-drawer__node {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
}

.wave-history-drawer__node-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.wave-history-drawer__summary {
  flex: 1 1 auto;
  min-width: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  word-break: break-word;
}

.wave-history-drawer__tags {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  flex-shrink: 0;
}

.wave-history-drawer__tag {
  display: inline-flex;
  align-items: center;
  padding: 1px var(--space-2);
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  white-space: nowrap;
  border: 1px solid transparent;
}

.wave-history-drawer__tag--current {
  color: var(--status-success-fg);
  background: var(--status-success-bg);
  border-color: var(--status-success-border);
}

.wave-history-drawer__tag--pinned {
  color: var(--status-warning-fg);
  background: var(--status-warning-bg);
  border-color: var(--status-warning-border);
}

.wave-history-drawer__tag--checkpoint {
  color: var(--status-info-fg);
  background: var(--status-info-bg);
  border-color: var(--status-info-border);
}

.wave-history-drawer__time {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}
</style>
