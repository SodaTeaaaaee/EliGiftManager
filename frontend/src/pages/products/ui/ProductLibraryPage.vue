<script setup lang="ts">
import {
  AddOutline,
  AlbumsOutline,
  DiamondOutline,
  GiftOutline,
  PricetagsOutline,
  RibbonOutline,
} from '@vicons/ionicons5'
import type { Component } from 'vue'
import { NButton, NCard, NDescriptions, NDescriptionsItem, NIcon, NTag } from 'naive-ui'

interface ProductCard {
  name: string
  stock: number
  tag: string
  icon: Component
  note: string
}

const products: ProductCard[] = [
  { name: '限定徽章套装', stock: 186, tag: '热门', icon: RibbonOutline, note: '活动主推款，适合优先加入派发批次。' },
  { name: '签名明信片', stock: 42, tag: '低库存', icon: AlbumsOutline, note: '库存水位接近预警，建议尽快补货。' },
  { name: '应援手幅', stock: 96, tag: '常规', icon: GiftOutline, note: '基础款式，适合大批量自动匹配。' },
  { name: '周年纪念盒', stock: 28, tag: '限量', icon: DiamondOutline, note: '高价值礼盒，推荐单独建立模板。' },
  { name: '透明卡套', stock: 210, tag: '新品', icon: PricetagsOutline, note: '新上架周边，可优先接入展示位。' },
  { name: '活动贴纸包', stock: 164, tag: '组合', icon: GiftOutline, note: '适合和其他礼物组合出库。' },
]

function getTagClass(tag: string) {
  if (tag === '低库存' || tag === '限量') {
    return 'warning'
  }

  if (tag === '新品') {
    return 'info'
  }

  if (tag === '热门') {
    return 'success'
  }

  return 'default'
}
</script>

<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
      <div>
        <p class="app-kicker">Product Library</p>
        <h1 class="app-title mt-2">礼物商品库</h1>
        <p class="app-copy mt-2">用卡片网格预览实物周边、库存和状态。</p>
      </div>
      <NButton type="primary">
        <template #icon>
          <NIcon :size="18">
            <AddOutline />
          </NIcon>
        </template>
        新增商品
      </NButton>
    </header>

    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <NCard
        v-for="product in products"
        :key="product.name"
        size="medium"
      >
        <template #cover>
          <div
            class="flex h-32 items-center justify-center"
            style="background: var(--surface-muted)"
          >
            <NIcon :size="40" depth="3">
              <component :is="product.icon" />
            </NIcon>
          </div>
        </template>

        <div class="flex items-start justify-between gap-3">
          <div>
            <h2 class="app-heading-sm">{{ product.name }}</h2>
            <p class="app-copy mt-2">{{ product.note }}</p>
          </div>
          <NTag :type="getTagClass(product.tag)" size="small" round>{{ product.tag }}</NTag>
        </div>

        <NDescriptions class="mt-4" :column="2" bordered size="small">
          <NDescriptionsItem label="当前库存">
            {{ product.stock }}
          </NDescriptionsItem>
          <NDescriptionsItem label="库存标签">
            <NTag :type="getTagClass(product.tag)" size="small" round>{{ product.tag }}</NTag>
          </NDescriptionsItem>
        </NDescriptions>
      </NCard>
    </div>
  </section>
</template>
