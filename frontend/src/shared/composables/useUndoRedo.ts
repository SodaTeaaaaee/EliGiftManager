import { onMounted, onUnmounted } from 'vue'

export interface UseUndoRedoOptions {
  scopeType: 'wave' | 'global'
  scopeKey: () => number | null
  onNotReady?: () => void
}

export function useUndoRedo(options: UseUndoRedoOptions) {
  function handleKeydown(e: KeyboardEvent) {
    // Don't intercept when focus is in a text input
    const target = e.target as HTMLElement
    if (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.isContentEditable
    ) {
      return
    }

    const isCtrl = e.ctrlKey || e.metaKey

    // Ctrl+Shift+Z or Ctrl+Y → redo
    if (isCtrl && e.shiftKey && e.key === 'Z') {
      e.preventDefault()
      options.onNotReady?.()
      return
    }
    if (isCtrl && e.key === 'y') {
      e.preventDefault()
      options.onNotReady?.()
      return
    }

    // Ctrl+Z → undo
    if (isCtrl && e.key === 'z' && !e.shiftKey) {
      e.preventDefault()
      options.onNotReady?.()
    }
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
  })
}
