<script setup lang="ts">
import { CubeOutline, ImageOutline, SearchOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { NAvatar, NButton, NCard, NDataTable, NEmpty, NIcon, NInput, NTag, type DataTableColumns } from 'naive-ui'
import { WAILS_PREVIEW_MESSAGE, isWailsRuntimeAvailable, listProducts, type ProductItem } from '@/shared/lib/wails/app'

const products = ref<ProductItem[]>([])
const keyword = ref('')
const isLoading = ref(false)
const errorMessage = ref('')

const filteredProducts = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return products.value
  return products.value.filter((product) =>
    [product.name, product.factory, product.factorySku, product.extraData].some((value) => value.toLowerCase().includes(text)),
  )
})

const usedProductCount = computed(() => products.value.filter((product) => product.dispatchCount > 0).length)
const totalQuantity = computed(() => products.value.reduce((sum, product) => sum + product.totalQuantity, 0))

const columns: DataTableColumns<ProductItem> = [
  {
    title: '商品',
    key: 'name',
    minWidth: 220,
    render: (row) => h('div', { class: 'flex items-center gap-3' }, [
      h(NAvatar, { size: 40, color: 'var(--accent-surface)' }, { default: () => h(NIcon, { size: 20, color: 'var(--accent)' }, { default: () => h(row.imagePath ? ImageOutline : CubeOutline) }) }),
      h('div', [h('div', { class: 'font-semibold' }, row.name), h('div', { class: 'app-copy' }, row.factory)]),
    ]),
  },
  { title: '工厂 SKU', key: 'factorySku', minWidth: 160 },
  { title: '派发记录', key: 'dispatchCount', width: 110 },
  { title: '派发件数', key: 'totalQuantity', width: 110 },
  {
    title: '图片',
    key: 'imagePath',
    width: 100,
    render: (row) => h(NTag, { type: row.imagePath ? 'success' : 'default', size: 'small', round: true }, { default: () => (row.imagePath ? '已配置' : '未配置') }),
  },
  { title: '扩展数据', key: 'extraData', minWidth: 220, ellipsis: { tooltip: true } },
]

async function loadProducts() {
  if (!isWailsRuntimeAvailable()) {
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return
  }

  isLoading.value = true
  errorMessage.value = ''
  try {
    products.value = await listProducts()
  } catch (error) {
    console.error('加载商品失败', error)
    errorMessage.value = '加载商品数据库失败，请查看后端日志。'
  } finally {
    isLoading.value = false
  }
}

onMounted(loadProducts)
</script>

<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Products</p>
        <h1 class="app-title mt-2">礼物商品库</h1>
        <p class="app-copy mt-2">读取 products，并统计每个商品在派发记录中的使用情况。</p>
      </div>
      <NButton :loading="isLoading" secondary strong @click="loadProducts">刷新商品</NButton>
    </header>

    <div class="grid gap-4 md:grid-cols-3">
      <NCard><p class="app-copy">商品总数</p><p class="mt-1 text-2xl font-semibold">{{ products.length }}</p></NCard>
      <NCard><p class="app-copy">已派发商品</p><p class="mt-1 text-2xl font-semibold">{{ usedProductCount }}</p></NCard>
      <NCard><p class="app-copy">累计件数</p><p class="mt-1 text-2xl font-semibold">{{ totalQuantity }}</p></NCard>
    </div>

    <NCard size="medium">
      <template #header>商品列表</template>
      <template #header-extra>
        <NInput v-model:value="keyword" clearable placeholder="搜索商品、工厂或 SKU" style="width: 280px">
          <template #prefix><NIcon><SearchOutline /></NIcon></template>
        </NInput>
      </template>
      <NEmpty v-if="errorMessage" :description="errorMessage" />
      <NDataTable v-else :columns="columns" :data="filteredProducts" :loading="isLoading" :bordered="false" :scroll-x="900" :pagination="{ pageSize: 12 }" />
    </NCard>
  </section>
</template>
