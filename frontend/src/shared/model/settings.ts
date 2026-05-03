import { ref, watch } from 'vue'

const scrollMode = ref(false)

export function useScrollMode() {
  return scrollMode
}

// Persist across page navigations by syncing to localStorage.
const STORAGE_KEY = 'eligift_scrollMode'
const stored = typeof localStorage !== 'undefined' ? localStorage.getItem(STORAGE_KEY) : null
if (stored === 'true') scrollMode.value = true

watch(scrollMode, (v) => {
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(STORAGE_KEY, String(v))
  }
})
