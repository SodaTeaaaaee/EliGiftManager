import { onMounted, onUnmounted } from 'vue'

export function useUndoRedo(options: {
  onUndo: () => void
  onRedo: () => void
}) {
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

    if ((e.ctrlKey || e.metaKey) && e.key === 'z') {
      e.preventDefault()
      if (e.shiftKey) {
        options.onRedo()
      } else {
        options.onUndo()
      }
    }
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
  })
}
