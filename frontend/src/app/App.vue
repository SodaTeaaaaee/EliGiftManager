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
import { useZoom, ZOOM_MIN, ZOOM_MAX, ZOOM_STEP } from '@/shared/model/settings'
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
const zoom = useZoom()

function onGlobalContextMenu(event: MouseEvent) {
  event.preventDefault()
  handleEvent(event)
}

function onGlobalWheel(event: WheelEvent) {
  if (!event.ctrlKey) return
  event.preventDefault()
  const delta = event.deltaY > 0 ? -ZOOM_STEP : ZOOM_STEP
  zoom.value = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, zoom.value + delta))
}

onMounted(() => {
  document.addEventListener('contextmenu', onGlobalContextMenu)
  document.addEventListener('wheel', onGlobalWheel, { passive: false })
})

onUnmounted(() => {
  document.removeEventListener('contextmenu', onGlobalContextMenu)
  document.removeEventListener('wheel', onGlobalWheel)
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
