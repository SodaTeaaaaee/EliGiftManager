<script setup lang="ts">
import {
  CloudDownloadOutline,
  DocumentTextOutline,
  GiftOutline,
  LayersOutline,
  PeopleOutline,
  RocketOutline,
} from '@vicons/ionicons5'
import type { Component } from 'vue'
import { NAlert, NCard, NDescriptions, NDescriptionsItem, NIcon, NList, NListItem, NTag, NThing } from 'naive-ui'

interface TemplateCard {
  title: string
  type: string
  stage: string
  status: string
  summary: string
  fields: string[]
  icon: Component
}

const templateCards: TemplateCard[] = [
  {
    title: '会员导入模板',
    type: 'import_member',
    stage: '导入',
    status: '已启用',
    summary: '将平台会员 CSV 映射到标准会员、昵称与地址模型。',
    fields: ['平台 UID', '昵称', '手机号 / 地址'],
    icon: PeopleOutline,
  },
  {
    title: '商品导入模板',
    type: 'import_product',
    stage: '导入',
    status: '待补字段',
    summary: '用于工厂商品主数据入库，保留 SKU、名称和扩展元数据。',
    fields: ['工厂 SKU', '商品名', '图片路径'],
    icon: GiftOutline,
  },
  {
    title: '发货记录导入模板',
    type: 'import_dispatch_record',
    stage: '导入',
    status: '已校验',
    summary: '承接批次结果和发货明细，为地址预校验和导出做准备。',
    fields: ['批次名', '会员 ID', '商品 ID'],
    icon: DocumentTextOutline,
  },
  {
    title: '订单导出模板',
    type: 'export_order',
    stage: '导出',
    status: '草稿',
    summary: '把派发结果整理成面向仓库或快递渠道的导出结构。',
    fields: ['收件信息', '商品清单', '导出状态'],
    icon: CloudDownloadOutline,
  },
]

const templateRules = [
  '模板类型直接对应后端 TemplateConfig.type，避免前后端枚举再次翻译。',
  '导入模板负责字段映射，设置页只保存导入默认规则，不再混放。',
  '模板页面保留版本化空间，后续接数据库时可以直接映射到 TemplateConfig 列表。',
]

function getStatusClass(status: string) {
  if (status === '已启用' || status === '已校验') {
    return 'success'
  }

  if (status === '待补字段') {
    return 'warning'
  }

  return 'default'
}
</script>

<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Templates</p>
        <h1 class="app-title mt-2">模板</h1>
        <p class="app-copy mt-2 max-w-3xl">
          这里集中管理导入导出模板本身。页面内容已经按后端真实模板类型拆开，后续接数据库列表时可以直接落到同一结构。
        </p>
      </div>
      <NTag type="info" size="medium" round>当前模板类型：{{ templateCards.length }}</NTag>
    </header>

    <div class="grid gap-4 xl:grid-cols-[1.45fr_1fr]">
      <section class="grid gap-4 md:grid-cols-2">
        <NCard
          v-for="card in templateCards"
          :key="card.type"
          size="medium"
        >
          <NThing>
            <template #avatar>
              <NIcon :size="18">
                <component :is="card.icon" />
              </NIcon>
            </template>
            <template #header>{{ card.title }}</template>
            <template #description>{{ card.summary }}</template>
            <template #header-extra>
              <NTag :type="getStatusClass(card.status)" size="small" round>{{ card.status }}</NTag>
            </template>
          </NThing>

          <div class="mt-4">
            <NTag type="info" size="small" round>{{ card.stage }}</NTag>
          </div>

          <NDescriptions class="mt-4" :column="1" bordered size="small">
            <NDescriptionsItem label="Template Type">
              {{ card.type }}
            </NDescriptionsItem>
          </NDescriptions>

          <NList class="mt-4">
            <NListItem
              v-for="field in card.fields"
              :key="field"
            >
              <div class="flex items-center justify-between gap-3">
                <span>{{ field }}</span>
                <NTag size="small" round>映射项</NTag>
              </div>
            </NListItem>
          </NList>
        </NCard>
      </section>

      <NCard size="medium">
        <template #header>
          <div class="flex items-center gap-2">
            <NIcon :size="18">
              <LayersOutline />
            </NIcon>
            <span class="app-heading-md">模板约束</span>
          </div>
        </template>
        <template #header-extra>
          <NIcon :size="18">
            <RocketOutline />
          </NIcon>
        </template>

        <p class="app-copy">
          模板页现在先展示结构约束和后端类型映射，等模板列表接口接进来后，这个布局可以直接替换成真实数据。
        </p>

        <div class="mt-4 space-y-3">
          <NAlert
            v-for="rule in templateRules"
            :key="rule"
            type="info"
            :show-icon="false"
          >
            {{ rule }}
          </NAlert>
        </div>
      </NCard>
    </div>
  </section>
</template>
