import { ref, watch } from 'vue'

export type TableMode = 'scroll' | 'paginated'

const tableMode = ref<TableMode>('scroll')

export function useTableMode() {
  return tableMode
}

// Persist across page navigations by syncing to localStorage.
const NEW_KEY = 'eligift_tableMode'
const LEGACY_KEY = 'eligift_scrollMode'

function loadPersistedMode(): TableMode {
  if (typeof localStorage === 'undefined') return 'scroll'
  const newVal = localStorage.getItem(NEW_KEY)
  if (newVal === 'scroll' || newVal === 'paginated') return newVal
  // Backward compat: read legacy boolean key
  const legacy = localStorage.getItem(LEGACY_KEY)
  if (legacy === 'false') return 'paginated'
  if (legacy === 'true') return 'scroll'
  return 'scroll'
}

tableMode.value = loadPersistedMode()

watch(tableMode, (v) => {
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(NEW_KEY, v)
  }
})
