<script setup lang="ts">
/**
 * App root — NConfigProvider (theme/tokens bridge) wraps FeedbackProvider
 * wraps AppShell (nav + content zone).
 */
import { computed } from 'vue'
import { RouterView } from 'vue-router'
import { NConfigProvider } from 'naive-ui'
import {
  GridOutline,
  FileTrayFullOutline,
  LayersOutline,
  LibraryOutline,
  SettingsOutline,
  FlaskOutline,
} from '@vicons/ionicons5'
import { AppShell, type NavGroupSpec, type NavItemSpec } from '@/shared/ui/shell'
import { FeedbackProvider, DisconnectedBanner, TopProgressBar } from '@/shared/ui/feedback'
import { useNaiveTheme } from '@/shared/theme/naive-bridge'
import { useGlobalViewHotkeys } from '@/shared/lib/view-hotkeys'

const { theme, themeOverrides } = useNaiveTheme()
useGlobalViewHotkeys()

const navGroups = computed<NavGroupSpec[]>(() => [
  {
    key: 'primary',
    items: [
      { key: 'home', labelKey: 'nav.home', icon: GridOutline, to: { name: 'home' } },
      { key: 'inbox', labelKey: 'nav.inbox', icon: FileTrayFullOutline, to: { name: 'inbox' } },
      { key: 'waves', labelKey: 'nav.waves', icon: LayersOutline, to: { name: 'waves' } },
      { key: 'library', labelKey: 'nav.library', icon: LibraryOutline, to: { name: 'library-customers' } },
    ],
  },
  {
    key: 'dev',
    labelKey: 'nav.devSectionLabel',
    items: [{ key: 'design-lab', labelKey: 'designLab.title', icon: FlaskOutline, to: { name: 'design-lab' } }],
  },
])

const settingsItem: NavItemSpec = {
  key: 'settings',
  labelKey: 'nav.settings',
  icon: SettingsOutline,
  to: { name: 'settings' },
}
</script>

<template>
  <NConfigProvider :theme="theme" :theme-overrides="themeOverrides" abstract>
    <FeedbackProvider>
      <AppShell :groups="navGroups" :settings-item="settingsItem">
        <DisconnectedBanner />
        <TopProgressBar />
        <RouterView />
      </AppShell>
    </FeedbackProvider>
  </NConfigProvider>
</template>
