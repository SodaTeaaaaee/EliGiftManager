<script setup lang="ts">
/**
 * App root — NConfigProvider (theme/tokens bridge) wraps FeedbackProvider
 * wraps AppShell (nav + content zone).
 */
import { computed } from 'vue'
import { RouterView } from 'vue-router'
import { NConfigProvider, enUS, zhCN } from 'naive-ui'
import {
  GridOutline,
  FileTrayFullOutline,
  LayersOutline,
  LibraryOutline,
  SettingsOutline,
  FlaskOutline,
} from '@vicons/ionicons5'
import type { NavGroupSpec, NavItemSpec } from '@/shared/ui/shell'
import { AppShell } from '@/shared/ui/shell'
import { FeedbackProvider, DisconnectedBanner, TopProgressBar } from '@/shared/ui/feedback'
import { useNaiveTheme } from '@/shared/theme/naive-bridge'
import { useGlobalViewHotkeys } from '@/shared/lib/view-hotkeys'
import { useAppLocale } from '@/shared/i18n'

const { theme, themeOverrides } = useNaiveTheme()
useGlobalViewHotkeys()

// Naive-rendered internals (NSelect placeholder, modal buttons, ...) follow
// naive-ui's own English defaults unless the provider locale is bound to the
// app i18n locale.
const { locale } = useAppLocale()
const naiveLocale = computed(() => (locale.value === 'zh-CN' ? zhCN : enUS))

const navGroups = computed<NavGroupSpec[]>(() => {
  const groups: NavGroupSpec[] = [
    {
      key: 'primary',
      items: [
        { key: 'home', labelKey: 'nav.home', icon: GridOutline, to: { name: 'home' } },
        { key: 'inbox', labelKey: 'nav.inbox', icon: FileTrayFullOutline, to: { name: 'inbox' } },
        { key: 'waves', labelKey: 'nav.waves', icon: LayersOutline, to: { name: 'waves' } },
        { key: 'library', labelKey: 'nav.library', icon: LibraryOutline, to: { name: 'library-customers' } },
      ],
    },
  ]
  // The dev-tools group (design-lab) only exists in dev builds.
  if (import.meta.env.DEV) {
    groups.push({
      key: 'dev',
      labelKey: 'nav.devSectionLabel',
      items: [{ key: 'design-lab', labelKey: 'designLab.title', icon: FlaskOutline, to: { name: 'design-lab' } }],
    })
  }
  return groups
})

const settingsItem: NavItemSpec = {
  key: 'settings',
  labelKey: 'nav.settings',
  icon: SettingsOutline,
  to: { name: 'settings' },
}
</script>

<template>
  <NConfigProvider
    :locale="naiveLocale"
    :theme="theme"
    :theme-overrides="themeOverrides"
    abstract
  >
    <FeedbackProvider>
      <AppShell :groups="navGroups" :settings-item="settingsItem">
        <DisconnectedBanner />
        <TopProgressBar />
        <RouterView />
      </AppShell>
    </FeedbackProvider>
  </NConfigProvider>
</template>
