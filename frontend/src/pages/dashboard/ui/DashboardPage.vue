<script setup lang="ts">
import {
  AlertCircleOutline,
  DocumentTextOutline,
  FlashOutline,
  GiftOutline,
  RocketOutline,
  SearchOutline,
  TicketOutline,
  WarningOutline,
} from '@vicons/ionicons5'
import type { Component } from 'vue'
import { NAlert, NButton, NCard, NIcon, NStatistic, NTag } from 'naive-ui'

interface StatItem {
  label: string
  value: string
  detail: string
  icon: Component
}

interface QuickAction {
  label: string
  icon: Component
}

const stats: StatItem[] = [
  { label: '待派发订单', value: '128', detail: '+18 today', icon: TicketOutline },
  { label: '待补全地址', value: '24', detail: 'needs review', icon: WarningOutline },
  { label: '低库存礼物', value: '7', detail: 'below safety line', icon: GiftOutline },
  { label: '本周已完成', value: '386', detail: '92% fulfilled', icon: RocketOutline },
]

const warnings = [
  { title: '库存风险', detail: '签名明信片库存低于 20 件' },
  { title: '地址缺失', detail: '3 个订单缺少手机号码' },
  { title: '模板授权', detail: '快递模板需要重新授权' },
]

const actions: QuickAction[] = [
  { label: '导入新订单', icon: FlashOutline },
  { label: '批量匹配地址', icon: SearchOutline },
  { label: '生成发货单', icon: RocketOutline },
  { label: '导出今日报表', icon: DocumentTextOutline },
]
</script>

<template>
  <section class="space-y-5">
    <header>
      <p class="app-kicker">Dashboard</p>
      <h1 class="app-title mt-2">工作台</h1>
      <p class="app-copy mt-2">把今日的派发状态、风险提醒和常用操作放在一个视图里。</p>
    </header>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <NCard
        v-for="item in stats"
        :key="item.label"
        size="medium"
      >
        <div class="flex items-start justify-between gap-4">
          <NStatistic :value="item.value">
            <template #label>
              <span class="app-text-muted text-sm">{{ item.label }}</span>
            </template>
          </NStatistic>
          <NIcon :size="20" depth="3">
            <component :is="item.icon" />
          </NIcon>
        </div>
        <p class="app-text-accent mt-3 text-sm font-medium">{{ item.detail }}</p>
      </NCard>
    </div>

    <div class="grid gap-4 xl:grid-cols-[1.35fr_1fr]">
      <NCard size="medium">
        <template #header>
          <div class="flex items-center gap-2">
            <NIcon :size="18">
              <AlertCircleOutline />
            </NIcon>
            <span class="app-heading-md">异常警告</span>
          </div>
        </template>
        <template #header-extra>
          <NTag type="error" size="small" round>{{ warnings.length }} alerts</NTag>
        </template>

        <div class="space-y-3">
          <NAlert
            v-for="warning in warnings"
            :key="warning.title"
            :title="warning.title"
            type="error"
            :show-icon="false"
          >
            {{ warning.detail }}
          </NAlert>
        </div>
      </NCard>

      <NCard size="medium">
        <template #header>
          <div class="flex items-center gap-2">
            <NIcon :size="18">
              <FlashOutline />
            </NIcon>
            <span class="app-heading-md">快捷操作</span>
          </div>
        </template>

        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
          <NButton
            v-for="action in actions"
            :key="action.label"
            block
            secondary
            strong
            class="!justify-start"
          >
            <template #icon>
              <NIcon :size="18">
                <component :is="action.icon" />
              </NIcon>
            </template>
            {{ action.label }}
          </NButton>
        </div>
      </NCard>
    </div>
  </section>
</template>
