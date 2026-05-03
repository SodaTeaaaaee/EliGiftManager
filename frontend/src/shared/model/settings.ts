import { ref, watch } from 'vue'

const scrollMode = ref(false)

export function useScrollMode() {
  return scrollMode
}

// Persist across page navigations by syncing to sessionStorage.
const STORAGE_KEY = 'eligift_scrollMode'
const stored = typeof sessionStorage !== 'undefined' ? sessionStorage.getItem(STORAGE_KEY) : null
if (stored === 'true') scrollMode.value = true

watch(scrollMode, (v) => {
  if (typeof sessionStorage !== 'undefined') {
    sessionStorage.setItem(STORAGE_KEY, String(v))
  }
})
