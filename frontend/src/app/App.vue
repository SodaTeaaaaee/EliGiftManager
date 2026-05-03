<script setup lang="ts">
import { computed, onMounted, onUnmounted, watchEffect } from 'vue'
import {
  NConfigProvider,
  NDialogProvider,
  NGlobalStyle,
  NMessageProvider,
  darkTheme,
  useOsTheme,
  type GlobalThemeOverrides,
} from 'naive-ui'
import { RouterView } from 'vue-router'
import { useThemeStore } from '@/shared/model/theme'
import { useContextMenu } from '@/shared/composables/useContextMenu'
import { saveZoom } from '@/shared/lib/wails/app'
import ContextMenu from '@/shared/ui/ContextMenu.vue'

const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#2563eb',
    primaryColorHover: '#3b82f6',
    primaryColorPressed: '#1d4ed8',
    primaryColorSuppl: '#3b82f6',
    infoColor: '#2563eb',
    successColor: '#16a34a',
    warningColor: '#d97706',
    errorColor: '#dc2626',
    fontFamily: "'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', 'Hiragino Sans GB', sans-serif",
  },
}

const darkThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#60a5fa',
    primaryColorHover: '#93c5fd',
    primaryColorPressed: '#3b82f6',
    primaryColorSuppl: '#60a5fa',
    infoColor: '#60a5fa',
    successColor: '#4ade80',
    warningColor: '#fbbf24',
    errorColor: '#f87171',
    fontFamily: "'Noto Sans SC', 'PingFang SC', 'Microsoft YaHei', 'Hiragino Sans GB', sans-serif",
  },
}

const themeStore = useThemeStore()
themeStore.hydrate()

const osTheme = useOsTheme()

const resolvedTheme = computed<'light' | 'dark'>(() => {
  if (themeStore.preference === 'system') {
    return osTheme.value === 'dark' ? 'dark' : 'light'
  }

  return themeStore.preference
})

const naiveTheme = computed(() => (resolvedTheme.value === 'dark' ? darkTheme : null))
const themeOverrides = computed(() =>
  resolvedTheme.value === 'dark' ? darkThemeOverrides : lightThemeOverrides,
)

watchEffect(() => {
  if (typeof document === 'undefined') {
    return
  }

  document.documentElement.dataset.theme = resolvedTheme.value
  document.documentElement.style.colorScheme = resolvedTheme.value
})

const { handleEvent } = useContextMenu()

function onGlobalContextMenu(event: MouseEvent) {
  event.preventDefault()
  handleEvent(event)
}

// ── zoom persistence via devicePixelRatio ──
// WebView2 native zoom changes devicePixelRatio proportionally.
// Shutdown: Go OnBeforeClose → WindowExecJS → persistZoom() → saveZoom + localStorage.
let baseDPR = 1

function persistZoom() {
  const current = window.devicePixelRatio
  const zoom = Math.round((current / baseDPR) * 100)
  if (zoom < 25 || zoom > 500) return
  saveZoom(zoom) // Go → zoom.cfg
  try {
    localStorage.setItem('eligift_zoom', String(zoom))
  } catch {
    /* ok */
  }
}

// Exposed for Go OnBeforeClose → WindowExecJS call.
;(window as any).__persistZoom = persistZoom

onMounted(() => {
  document.addEventListener('contextmenu', onGlobalContextMenu)
  baseDPR = window.devicePixelRatio
})

onUnmounted(() => {
  document.removeEventListener('contextmenu', onGlobalContextMenu)
  persistZoom()
})
</script>

<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="themeOverrides">
    <NGlobalStyle />
    <NMessageProvider>
      <NDialogProvider>
        <RouterView class="h-full" />
        <ContextMenu />
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>
