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
    fontFamily: "'Aptos', 'Segoe UI', 'Helvetica Neue', sans-serif",
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
    fontFamily: "'Aptos', 'Segoe UI', 'Helvetica Neue', sans-serif",
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

onMounted(() => {
  document.addEventListener('contextmenu', onGlobalContextMenu)
})

onUnmounted(() => {
  document.removeEventListener('contextmenu', onGlobalContextMenu)
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
