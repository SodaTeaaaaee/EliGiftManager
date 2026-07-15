/**
 * useWaveUndoRedo — wave-workspace undo/redo action handlers + the global
 * Ctrl+Z / Ctrl+Shift+Z / Ctrl+Y keyboard shortcuts (plan section 7, unit
 * A). The keydown handling + text-input focus guard below is ported
 * near-verbatim from the OLD frontend's
 * `frontend-legacy/src/shared/composables/useUndoRedo.ts` (lines 43-70).
 *
 * The one deliberate behavior change from the OLD composable: on success
 * this calls the injected `WaveWorkspaceContext.refresh()` to re-fetch the
 * snapshot IN PLACE. It never bumps a `refreshKey` / re-keys a
 * `<router-view>` — that was OLD `WaveWorkspaceLayout.vue`'s
 * (lines 61-77, 184) pattern, and is exactly the "撤销丢失 UI 状态" bug this
 * port fixes. Buttons/shortcuts stay always-enabled; there is no server
 * precheck for whether undo/redo is currently possible, so every failure
 * (including "nothing to undo") surfaces via `useFeedback().error()`.
 */
import { onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useFeedback } from '@/shared/ui/feedback'
import { undoWaveAction, redoWaveAction } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'

export interface UseWaveUndoRedoResult {
  handleUndo(): Promise<void>
  handleRedo(): Promise<void>
}

export function useWaveUndoRedo(): UseWaveUndoRedoResult {
  const ctx = useWaveWorkspaceContext()
  const { t } = useI18n({ useScope: 'global' })
  const feedback = useFeedback()

  async function handleUndo(): Promise<void> {
    try {
      const summary = await undoWaveAction(ctx.waveId.value)
      await ctx.refresh()
      feedback.success(t('waveWorkspace.header.undo.success', { summary }))
      feedback.receipt({ kind: 'undo', summary })
    } catch (err) {
      feedback.error(t('waveWorkspace.header.undo.error'), err instanceof Error ? err.message : String(err))
    }
  }

  async function handleRedo(): Promise<void> {
    try {
      const summary = await redoWaveAction(ctx.waveId.value)
      await ctx.refresh()
      feedback.success(t('waveWorkspace.header.redo.success', { summary }))
      feedback.receipt({ kind: 'redo', summary })
    } catch (err) {
      feedback.error(t('waveWorkspace.header.redo.error'), err instanceof Error ? err.message : String(err))
    }
  }

  // Ported near-verbatim from OLD useUndoRedo.ts:43-70 — same focus guard,
  // same key combinations (Ctrl/Cmd+Z, Ctrl/Cmd+Shift+Z, Ctrl/Cmd+Y).
  function handleKeydown(e: KeyboardEvent): void {
    const target = e.target as HTMLElement
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
      return
    }

    const isCtrl = e.ctrlKey || e.metaKey

    if (isCtrl && e.shiftKey && e.key === 'Z') {
      e.preventDefault()
      void handleRedo()
      return
    }
    if (isCtrl && e.key === 'y') {
      e.preventDefault()
      void handleRedo()
      return
    }

    if (isCtrl && e.key === 'z' && !e.shiftKey) {
      e.preventDefault()
      void handleUndo()
    }
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
  })

  return { handleUndo, handleRedo }
}
