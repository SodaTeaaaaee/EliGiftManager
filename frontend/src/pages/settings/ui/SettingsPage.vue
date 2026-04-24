<script setup lang="ts">
import { ColorPaletteOutline, ServerOutline, DesktopOutline, MoonOutline, SunnyOutline } from '@vicons/ionicons5'
import { computed, onMounted, ref, type Component } from 'vue'
import { NButton, NCard, NDescriptions, NDescriptionsItem, NIcon, NRadioButton, NRadioGroup, NTag, useOsTheme } from 'naive-ui'
import { bootstrapApp, getDashboard, isWailsRuntimeAvailable, pingDatabase, WAILS_PREVIEW_MESSAGE, type DashboardPayload } from '@/shared/lib/wails/app'
import { themePreferenceOptions, useThemeStore, type ThemePreference } from '@/shared/model/theme'
import type { BootstrapPayload } from '@/shared/types/app'

const themeStore = useThemeStore()
const osTheme = useOsTheme()
const bootstrap = ref<BootstrapPayload | null>(null)
const dashboard = ref<DashboardPayload | null>(null)
const dbPingResult = ref('')
const isLoading = ref(false)
const isPinging = ref(false)

const themeIcons: Record<ThemePreference, Component> = {
  light: SunnyOutline,
  dark: MoonOutline,
  system: DesktopOutline,
}

const themePreference = computed({
  get: () => themeStore.preference,
  set: (value: ThemePreference) => themeStore.setPreference(value),
})

const resolvedThemeLabel = computed(() => {
  if (themeStore.preference === 'system') {
    return osTheme.value === 'dark' ? '跟随系统：深色' : '跟随系统：浅色'
  }
  return themeStore.preference === 'dark' ? '当前固定为深色主题' : '当前固定为浅色主题'
})

async function loadRuntimeInfo() {
  if (!isWailsRuntimeAvailable()) {
    dbPingResult.value = WAILS_PREVIEW_MESSAGE
    return
  }

  isLoading.value = true
  try {
    const [bootstrapPayload, dashboardPayload] = await Promise.all([bootstrapApp(), getDashboard()])
    bootstrap.value = bootstrapPayload
    dashboard.value = dashboardPayload
  } finally {
    isLoading.value = false
  }
}

async function handlePingDB() {
  if (!isWailsRuntimeAvailable()) {
    dbPingResult.value = WAILS_PREVIEW_MESSAGE
    return
  }

  isPinging.value = true
  try {
    dbPingResult.value = await pingDatabase()
  } finally {
    isPinging.value = false
  }
}

onMounted(loadRuntimeInfo)
</script>

<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Settings</p>
        <h1 class="app-title mt-2">设置</h1>
        <p class="app-copy mt-2">管理主题偏好，并查看当前 Wails 运行环境与 SQLite 数据库状态。</p>
      </div>
      <NButton :loading="isLoading" secondary strong @click="loadRuntimeInfo">刷新运行信息</NButton>
    </header>

    <NCard size="medium">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div class="flex max-w-2xl items-start gap-3">
          <NIcon :size="20"><ColorPaletteOutline /></NIcon>
          <div>
            <h2 class="app-heading-md">主题偏好</h2>
            <p class="app-copy mt-2">提供浅色、深色和跟随系统三种模式，偏好会保存在本地。</p>
          </div>
        </div>

        <div class="flex flex-col items-start gap-3">
          <NRadioGroup v-model:value="themePreference" name="theme-preference">
            <NRadioButton v-for="option in themePreferenceOptions" :key="option.value" :value="option.value">
              <span class="inline-flex items-center gap-2">
                <NIcon :size="16"><component :is="themeIcons[option.value]" /></NIcon>
                {{ option.label }}
              </span>
            </NRadioButton>
          </NRadioGroup>
          <NTag type="info" round>{{ resolvedThemeLabel }}</NTag>
        </div>
      </div>
    </NCard>

    <div class="grid gap-4 xl:grid-cols-2">
      <NCard size="medium">
        <template #header>
          <div class="flex items-center gap-2"><NIcon><ServerOutline /></NIcon><span>SQLite 数据库</span></div>
        </template>
        <template #header-extra>
          <NButton size="small" :loading="isPinging" @click="handlePingDB">读写测试</NButton>
        </template>

        <NDescriptions :column="1" bordered size="small">
          <NDescriptionsItem label="数据库文件">{{ dashboard?.databasePath ?? '等待后端连接' }}</NDescriptionsItem>
          <NDescriptionsItem label="会员 / 商品 / 派发">
            {{ dashboard?.memberCount ?? 0 }} / {{ dashboard?.productCount ?? 0 }} / {{ dashboard?.dispatchCount ?? 0 }}
          </NDescriptionsItem>
          <NDescriptionsItem label="模板配置">{{ dashboard?.templateCount ?? 0 }}</NDescriptionsItem>
        </NDescriptions>

        <NTag v-if="dbPingResult" class="mt-4" :type="dbPingResult.includes('成功') ? 'success' : 'warning'" round>
          {{ dbPingResult }}
        </NTag>
      </NCard>

      <NCard size="medium">
        <template #header>应用运行信息</template>
        <NDescriptions :column="1" bordered size="small">
          <NDescriptionsItem label="应用名">{{ bootstrap?.name ?? 'EliGiftManager' }}</NDescriptionsItem>
          <NDescriptionsItem label="版本">{{ bootstrap?.version ?? '-' }}</NDescriptionsItem>
          <NDescriptionsItem label="Go Runtime">{{ bootstrap?.runtime ?? '-' }}</NDescriptionsItem>
          <NDescriptionsItem label="前端运行时">{{ bootstrap?.frontend ?? '-' }}</NDescriptionsItem>
          <NDescriptionsItem label="模块">{{ bootstrap?.module ?? '-' }}</NDescriptionsItem>
        </NDescriptions>
      </NCard>
    </div>
  </section>
</template>

