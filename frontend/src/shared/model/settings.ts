import { ref, watch } from 'vue'

function read<K extends string>(key: K, fallback: string): string {
  if (typeof localStorage === 'undefined') return fallback
  return localStorage.getItem(key) ?? fallback
}

function persist<K extends string>(key: K, value: string) {
  if (typeof localStorage !== 'undefined') localStorage.setItem(key, value)
}

// ── scroll mode ──

const scrollMode = ref(read('eligift_scrollMode', 'true') === 'true')

export function useScrollMode() {
  return scrollMode
}

watch(scrollMode, (v) => persist('eligift_scrollMode', String(v)))

// ── zoom ──

export const ZOOM_MIN = 50
export const ZOOM_MAX = 200
export const ZOOM_STEP = 5

const zoom = ref(100)

const storedZoom = parseInt(read('eligift_zoom', '100'), 10)
if (!isNaN(storedZoom) && storedZoom >= ZOOM_MIN && storedZoom <= ZOOM_MAX) {
  zoom.value = storedZoom
}

export function useZoom() {
  return zoom
}

watch(zoom, (v) => persist('eligift_zoom', String(v)))

// Apply zoom to root element.
watch(
  zoom,
  (v) => {
    if (typeof document !== 'undefined') {
      document.documentElement.style.zoom = `${v}%`
    }
  },
  { immediate: true },
)
