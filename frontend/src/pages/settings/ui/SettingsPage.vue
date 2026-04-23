<script setup lang="ts">
import {
  ColorPaletteOutline,
  ContrastOutline,
  DesktopOutline,
  DocumentTextOutline,
  DownloadOutline,
  MoonOutline,
  ShieldCheckmarkOutline,
  SunnyOutline,
} from '@vicons/ionicons5'
import { computed, type Component } from 'vue'
import { NCard, NIcon, NList, NListItem, NRadioButton, NRadioGroup, NTag, useOsTheme } from 'naive-ui'
import { themePreferenceOptions, useThemeStore, type ThemePreference } from '@/shared/model/theme'

interface SettingGroup {
  title: string
  tag: string
  description: string
  icon: Component
  items: Array<{
    label: string
    value: string
  }>
}

const settingGroups: SettingGroup[] = [
  {
    title: '导入默认项',
    tag: 'Import',
    icon: DownloadOutline,
    description: '控制成员、商品与派发记录导入时的标准化策略。',
    items: [
      { label: 'CSV 编码探测', value: 'UTF-8 / Shift-JIS 自动识别' },
      { label: '重复会员处理', value: '按平台 UID 合并主档案' },
      { label: '默认平台要求', value: '缺失平台标识时阻止导入' },
    ],
  },
  {
    title: '地址校验',
    tag: 'Validate',
    icon: ShieldCheckmarkOutline,
    description: '决定批量派发前的地址完整度和异常拦截规则。',
    items: [
      { label: '手机号规则', value: '缺失或长度异常时标记待校验' },
      { label: '门牌缺失策略', value: '允许保存，但阻止进入导出批次' },
      { label: '导出前预校验', value: '始终执行 ValidateBatch' },
    ],
  },
  {
    title: '导出偏好',
    tag: 'Export',
    icon: DocumentTextOutline,
    description: '统一发货单、日报和快递输出时的默认格式。',
    items: [
      { label: '发货单命名', value: '批次名 + 日期 + 序号' },
      { label: '日报时间基准', value: 'Asia/Tokyo' },
      { label: '快递模板回退', value: '未命中时使用标准字段顺序' },
    ],
  },
  {
    title: '桌面工作区',
    tag: 'Workspace',
    icon: DesktopOutline,
    description: '保留桌面端特有的数据库和工作区默认行为。',
    items: [
      { label: '数据库位置', value: '用户配置目录 / data / eligiftmanager.db' },
      { label: '启动首页', value: '工作台' },
      { label: '批次结果缓存', value: '会话级保留，重启后刷新' },
    ],
  },
]

const themeIcons: Record<ThemePreference, Component> = {
  system: ContrastOutline,
  light: SunnyOutline,
  dark: MoonOutline,
}

const themeStore = useThemeStore()
themeStore.hydrate()

const osTheme = useOsTheme()

const themePreference = computed<ThemePreference>({
  get: () => themeStore.preference,
  set: (value) => themeStore.setPreference(value),
})

const resolvedThemeLabel = computed(() => {
  if (themePreference.value === 'system') {
    return osTheme.value === 'dark' ? '当前跟随系统，实际为深色' : '当前跟随系统，实际为浅色'
  }

  return themePreference.value === 'dark' ? '当前固定为深色主题' : '当前固定为浅色主题'
})
</script>

<template>
  <section class="space-y-5">
    <header>
      <p class="app-kicker">Settings</p>
      <h1 class="app-title mt-2">设置</h1>
      <p class="app-copy mt-2 max-w-3xl">
        这里放全局运行规则，不处理具体模板内容。模板本身已经拆到单独页面，设置页只负责默认策略和工作区行为。
      </p>
    </header>

    <div class="grid gap-4 xl:grid-cols-2">
      <NCard class="xl:col-span-2" size="medium">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div class="flex max-w-2xl items-start gap-3">
            <NIcon :size="18">
              <ColorPaletteOutline />
            </NIcon>
            <div>
              <h2 class="app-heading-md">主题偏好</h2>
              <p class="app-copy mt-2">
                提供浅色、深色和跟随系统三种模式。默认值为“跟随系统”，会自动响应操作系统的外观设置。
              </p>
            </div>
          </div>

          <div class="flex flex-col items-start gap-3">
            <NRadioGroup v-model:value="themePreference" name="theme-preference">
              <NRadioButton
                v-for="option in themePreferenceOptions"
                :key="option.value"
                :value="option.value"
              >
                <span class="inline-flex items-center gap-2">
                  <NIcon :size="16">
                    <component :is="themeIcons[option.value]" />
                  </NIcon>
                  {{ option.label }}
                </span>
              </NRadioButton>
            </NRadioGroup>
            <NTag type="info" round>{{ resolvedThemeLabel }}</NTag>
          </div>
        </div>
      </NCard>

      <NCard
        v-for="group in settingGroups"
        :key="group.title"
        size="medium"
      >
        <template #header>
          <div class="flex items-center gap-2">
            <NIcon :size="18">
              <component :is="group.icon" />
            </NIcon>
            <span class="app-heading-md">{{ group.title }}</span>
          </div>
        </template>
        <template #header-extra>
          <NTag size="small" round>{{ group.tag }}</NTag>
        </template>

        <p class="app-copy">{{ group.description }}</p>

        <NList class="mt-4">
          <NListItem
            v-for="item in group.items"
            :key="item.label"
          >
            <div class="flex flex-col gap-2 md:flex-row md:items-start md:justify-between md:gap-4">
              <p class="text-sm" style="color: var(--muted)">{{ item.label }}</p>
              <p class="text-sm font-medium leading-6 md:max-w-[58%]" style="color: var(--text)">
                {{ item.value }}
              </p>
            </div>
          </NListItem>
        </NList>
      </NCard>
    </div>
  </section>
</template>
