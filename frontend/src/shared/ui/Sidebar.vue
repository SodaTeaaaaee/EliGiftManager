<script setup lang="ts">
import {
  AnalyticsOutline,
  GiftOutline,
  LayersOutline,
  PeopleOutline,
  SettingsOutline,
  TicketOutline,
} from '@vicons/ionicons5'
import { computed, h, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NDivider, NIcon, NMenu, NSpace, NText, type MenuOption } from 'naive-ui'

interface SidebarItem {
  name: string
  label: string
  icon: Component
}

const mainItems: SidebarItem[] = [
  { name: 'dashboard', label: '工作台', icon: AnalyticsOutline },
  { name: 'waves', label: '发货任务', icon: TicketOutline },
  { name: 'members', label: '会员与地址库', icon: PeopleOutline },
  { name: 'products', label: '礼物商品库', icon: GiftOutline },
  { name: 'templates', label: '模板', icon: LayersOutline },
]

const settingsItem: SidebarItem = {
  name: 'settings',
  label: '设置',
  icon: SettingsOutline,
}

function renderIcon(icon: Component) {
  return () => h(NIcon, { size: 18 }, { default: () => h(icon) })
}

function createMenuOption(item: SidebarItem): MenuOption {
  return {
    key: item.name,
    label: item.label,
    icon: renderIcon(item.icon),
  }
}

const mainOptions = mainItems.map(createMenuOption)
const settingsOptions = [createMenuOption(settingsItem)]

const route = useRoute()
const router = useRouter()

const selectedKey = computed(() => route.name?.toString() ?? 'dashboard')

function handleNavigate(key: string | number) {
  void router.push({ name: String(key) })
}
</script>

<template>
  <NSpace vertical size="large" class="h-full">
    <div class="px-3 pt-1">
      <NText strong>EliGiftManager</NText>
      <div class="mt-1">
        <NText depth="3">Gift Ops Console</NText>
      </div>
    </div>

    <NMenu :value="selectedKey" :options="mainOptions" @update:value="handleNavigate" />

    <div class="mt-auto">
      <NDivider class="!my-3" />
      <NMenu :value="selectedKey" :options="settingsOptions" @update:value="handleNavigate" />
    </div>
  </NSpace>
</template>

