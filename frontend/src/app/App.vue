<script setup lang="ts">
import { computed, onErrorCaptured, onMounted, onUnmounted, ref, watchEffect } from "vue";
import { RouterView } from "vue-router";
import {
  NButton,
  NConfigProvider,
  NGlobalStyle,
  NDialogProvider,
  NMessageProvider,
  NResult,
  darkTheme,
  useOsTheme,
  type GlobalThemeOverrides,
} from "naive-ui";
import { useThemeStore } from "@/shared/model/theme";
import { useLocaleStore } from "@/shared/model/locale";
import { useContextMenu } from "@/shared/composables/useContextMenu";
import ContextMenu from "@/shared/ui/ContextMenu.vue";
import { useI18n } from "@/shared/i18n";

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
const localeStore = useLocaleStore()
localeStore.hydrate()
const { locale } = useI18n()

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
  document.documentElement.lang = locale.value
  document.documentElement.style.colorScheme = resolvedTheme.value
})

const { handleEvent } = useContextMenu()

function onGlobalContextMenu(event: MouseEvent) {
  event.preventDefault()
  handleEvent(event)
}

// ── zoom persistence via devicePixelRatio ──
// WebView2 native zoom changes devicePixelRatio proportionally.
// Shutdown: Go OnBeforeClose → WindowExecJS → persistZoom().
// Backend saveZoom binding will be restored when Wails bridge is rebuilt.
let baseDPR = 1

function persistZoom() {
  const current = window.devicePixelRatio
  const zoom = Math.round((current / baseDPR) * 100)
  if (zoom < 25 || zoom > 500) return
  // TODO(V2): restore saveZoom backend call when Wails bridge rebuilt
  try {
    localStorage.setItem('eligift_zoom', String(zoom))
  } catch {
    /* ok */
  }
}

// Exposed for Go OnBeforeClose → WindowExecJS call.
;(window as any).__persistZoom = persistZoom

// ── Error boundary ──
const renderError = ref<Error | null>(null)
const renderErrorInfo = ref('')
const chunkError = ref(false)

onErrorCaptured((err: Error, _instance, info: string) => {
  // Wails bridge not ready during dev — non-fatal, let it propagate normally
  const msg = err?.message ?? ''
  if (msg.includes('wails') || msg.includes('runtime.')) {
    console.warn('[App] Wails bridge error (non-fatal):', err)
    return false
  }
  console.error('[App] Render error captured:', err, '\nComponent info:', info)
  renderError.value = err
  renderErrorInfo.value = info
  return false
})

function reloadPage() {
  window.location.reload()
}

onMounted(() => {
  document.addEventListener('contextmenu', onGlobalContextMenu)
  baseDPR = window.devicePixelRatio
  window.addEventListener('router-chunk-error', () => {
    chunkError.value = true
  })
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
        <!-- Chunk load failure banner — shown above router content, nav stays visible -->
        <div v-if="chunkError" style="padding: 16px">
          <NResult
            status="error"
            title="页面加载失败"
            description="路由模块加载出错，可能是网络问题或版本更新导致。"
          >
            <template #footer>
              <NButton type="primary" @click="reloadPage">重新加载</NButton>
            </template>
          </NResult>
        </div>
        <!-- Render error boundary — replaces router-view area only, nav stays visible -->
        <div v-else-if="renderError" style="padding: 16px">
          <NResult
            status="error"
            title="页面渲染出错"
            :description="renderError.message"
          >
            <template #footer>
              <NButton type="primary" @click="reloadPage">重新加载</NButton>
            </template>
          </NResult>
          <pre
            v-if="renderErrorInfo"
            style="margin-top: 12px; font-size: 12px; opacity: 0.6; white-space: pre-wrap; word-break: break-all"
          >{{ renderErrorInfo }}</pre>
        </div>
        <RouterView v-else />
      </NDialogProvider>
    </NMessageProvider>
    <ContextMenu />
  </NConfigProvider>
</template>
