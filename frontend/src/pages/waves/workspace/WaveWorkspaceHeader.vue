<script setup lang="ts">
/**
 * WaveWorkspaceHeader — the wave-workspace shell's top chrome (plan section
 * 7, unit A). Reads the ONE shared `useWaveWorkspace()` context via
 * `useWaveWorkspaceContext()` — never fetches the snapshot itself; every
 * mutation below re-fetches through the injected `refresh()` so the rest of
 * the tree updates in place (no `:key` bump, no route remount).
 *
 * The drift badge here is DISPLAY-ONLY: `WaveWorkspaceContext` exposes no
 * shared "drift drawer open" state, and the foundations contract's fallback
 * for that case is to keep this badge passive and let unit B's overview tab
 * (`WaveDriftDrawer.vue`) own the openable drawer. Flagged as a followup in
 * case product wants the header badge itself clickable later.
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput } from 'naive-ui'
import {
  ArrowRedoOutline,
  ArrowUndoOutline,
  CheckmarkOutline,
  CloseOutline,
  PencilOutline,
  TimeOutline,
} from '@vicons/ionicons5'
import { PageHeader } from '@/shared/ui/shell'
import { StatusBadge } from '@/shared/ui/status'
import { CalloutBar } from '@/shared/ui/guidance'
import { useFeedback } from '@/shared/ui/feedback'
import { updateWave } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import { useWaveUndoRedo } from './useWaveUndoRedo'
import WaveHistoryDrawer from './WaveHistoryDrawer.vue'

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const ctx = useWaveWorkspaceContext()
const { handleUndo, handleRedo } = useWaveUndoRedo()

// ── inline name edit ──
const editing = ref(false)
const nameDraft = ref('')
const savingName = ref(false)

function startEdit(): void {
  nameDraft.value = ctx.snapshot.value?.wave.name ?? ''
  editing.value = true
}

function cancelEdit(): void {
  editing.value = false
}

async function saveEdit(): Promise<void> {
  const wave = ctx.snapshot.value?.wave
  if (!wave || savingName.value) return
  const trimmed = nameDraft.value.trim()
  if (!trimmed) return
  savingName.value = true
  try {
    await updateWave({ waveId: wave.id, name: trimmed, notes: wave.notes, levelTags: wave.levelTags })
    await ctx.refresh()
    editing.value = false
    feedback.success(t('waveWorkspace.header.name.saveSuccess'))
  } catch (err) {
    feedback.error(t('waveWorkspace.header.name.saveError'), err instanceof Error ? err.message : String(err))
  } finally {
    savingName.value = false
  }
}

// ── history drawer ──
const historyOpen = ref(false)
</script>

<template>
  <div class="wave-workspace-header">
    <PageHeader :kicker="t('waveWorkspace.header.kicker')">
      <template #title>
        <div class="wave-workspace-header__name">
          <template v-if="editing">
            <NInput
              v-model:value="nameDraft"
              size="large"
              autofocus
              :placeholder="t('waveWorkspace.header.name.placeholder')"
              :disabled="savingName"
              class="wave-workspace-header__name-input"
              @keydown.enter.prevent="saveEdit"
              @keydown.esc.prevent="cancelEdit"
            />
            <NButton
              size="small"
              circle
              type="primary"
              :title="t('waveWorkspace.header.name.save')"
              :loading="savingName"
              @click="saveEdit"
            >
              <template #icon><CheckmarkOutline /></template>
            </NButton>
            <NButton size="small" circle :disabled="savingName" :title="t('waveWorkspace.header.name.cancel')" @click="cancelEdit">
              <template #icon><CloseOutline /></template>
            </NButton>
          </template>
          <template v-else>
            <span class="wave-workspace-header__title">{{ ctx.snapshot.value?.wave.name }}</span>
            <button
              type="button"
              class="wave-workspace-header__edit-btn"
              :aria-label="t('waveWorkspace.header.name.edit')"
              :title="t('waveWorkspace.header.name.edit')"
              @click="startEdit"
            >
              <PencilOutline />
            </button>
          </template>
        </div>
      </template>
      <template #description>
        <span class="wave-workspace-header__badges">
          <StatusBadge dimension="lifecycleStage" :value="ctx.snapshot.value?.projectedLifecycleStage ?? ''" show-dot />
          <StatusBadge dimension="driftSummary" :value="ctx.driftSummaryValue.value" show-dot />
        </span>
      </template>
      <template #actions>
        <NButton size="small" :title="t('waveWorkspace.header.undo.aria')" @click="handleUndo">
          <template #icon><ArrowUndoOutline /></template>
          {{ t('waveWorkspace.header.undo.label') }}
        </NButton>
        <NButton size="small" :title="t('waveWorkspace.header.redo.aria')" @click="handleRedo">
          <template #icon><ArrowRedoOutline /></template>
          {{ t('waveWorkspace.header.redo.label') }}
        </NButton>
        <NButton size="small" :title="t('waveWorkspace.header.history.label')" @click="historyOpen = true">
          <template #icon><TimeOutline /></template>
          {{ t('waveWorkspace.header.history.label') }}
        </NButton>
      </template>
    </PageHeader>

    <CalloutBar v-if="ctx.undoBoundaryCrossed.value" tone="warning" :message="t('waveWorkspace.header.undoBoundaryNotice')" />

    <WaveHistoryDrawer v-model:show="historyOpen" />
  </div>
</template>

<style scoped>
.wave-workspace-header {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.wave-workspace-header__name {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.wave-workspace-header__name-input {
  max-width: 28rem;
}

.wave-workspace-header__title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wave-workspace-header__edit-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  border-radius: var(--radius-sm);
  border: none;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}

.wave-workspace-header__edit-btn:hover {
  background: var(--color-inset);
  color: var(--color-text-primary);
}

.wave-workspace-header__edit-btn:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.wave-workspace-header__edit-btn svg {
  width: 16px;
  height: 16px;
}

.wave-workspace-header__badges {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}
</style>
