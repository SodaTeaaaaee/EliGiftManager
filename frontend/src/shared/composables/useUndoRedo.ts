import { onMounted, onUnmounted } from 'vue'
import { undoWaveAction, redoWaveAction, isWailsRuntimeAvailable } from '@/shared/lib/wails/app'

export interface UseUndoRedoOptions {
  scopeType: 'wave' | 'global'
  scopeKey: () => number | null
  onSuccess?: (summary: string, action: 'undo' | 'redo') => void
  onError?: (error: string) => void
  onNotReady?: () => void
}

export function useUndoRedo(options: UseUndoRedoOptions) {
  async function handleUndo() {
    const key = options.scopeKey()
    if (!key || !isWailsRuntimeAvailable()) {
      options.onNotReady?.()
      return
    }
    try {
      const summary = await undoWaveAction(key)
      options.onSuccess?.(summary, 'undo')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      options.onError?.(msg)
    }
  }

  async function handleRedo() {
    const key = options.scopeKey()
    if (!key || !isWailsRuntimeAvailable()) {
      options.onNotReady?.()
      return
    }
    try {
      const summary = await redoWaveAction(key)
      options.onSuccess?.(summary, 'redo')
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      options.onError?.(msg)
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    const target = e.target as HTMLElement
    if (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.isContentEditable
    ) {
      return
    }

    const isCtrl = e.ctrlKey || e.metaKey

    if (isCtrl && e.shiftKey && e.key === 'Z') {
      e.preventDefault()
      handleRedo()
      return
    }
    if (isCtrl && e.key === 'y') {
      e.preventDefault()
      handleRedo()
      return
    }

    if (isCtrl && e.key === 'z' && !e.shiftKey) {
      e.preventDefault()
      handleUndo()
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
