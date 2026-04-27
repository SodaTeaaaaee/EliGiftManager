<script setup lang="ts">
import { ImageOutline, SearchOutline } from '@vicons/ionicons5'
import { computed, onMounted, ref } from 'vue'
import { ChevronBackOutline, ChevronForwardOutline } from '@vicons/ionicons5'
import { NButton, NCard, NDivider, NDrawer, NDrawerContent, NEmpty, NIcon, NInput, NPagination, NSelect, NTag, useMessage } from 'naive-ui'
import { getProductImages, isWailsRuntimeAvailable, listProducts, WAILS_PREVIEW_MESSAGE, type ProductItem } from '@/shared/lib/wails/app'

const message = useMessage()
const products = ref<ProductItem[]>([])
const keyword = ref('')
const platform = ref('')
const page = ref(1)
const pageSize = ref(12)
const total = ref(0)
const isLoading = ref(false)
const errorMessage = ref('')

const platformCatalog = ref<string[]>([])

const detailProduct = ref<ProductItem | null>(null)
const detailImages = ref<{ id: number; path: string; sortOrder: number; sourceDir: string }[]>([])
const showDetail = ref(false)

const mainImages = computed(() => detailImages.value.filter(img => img.sourceDir === '主图'))
const detailOnlyImages = computed(() => detailImages.value.filter(img => img.sourceDir !== '主图'))
const mainIndex = ref(0)
const currentMainImage = computed(() => mainImages.value[mainIndex.value] ?? null)
function prevImage() { if (mainImages.value.length) mainIndex.value = (mainIndex.value - 1 + mainImages.value.length) % mainImages.value.length }
function nextImage() { if (mainImages.value.length) mainIndex.value = (mainIndex.value + 1) % mainImages.value.length }

const platformOptions = computed(() =>
  platformCatalog.value.map((value) => ({ label: value, value })),
)

async function loadProducts() {
  if (!isWailsRuntimeAvailable()) {
    products.value = []
    total.value = 0
    platformCatalog.value = []
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return
  }

  isLoading.value = true
  errorMessage.value = ''

  try {
    const payload = await listProducts(page.value, pageSize.value, keyword.value, platform.value)
    products.value = payload.items
    total.value = payload.total
    platformCatalog.value = payload.platforms
  } catch (error) {
    console.error(error)
    errorMessage.value = '加载礼物库失败。'
  } finally {
    isLoading.value = false
  }
}

function searchProducts() {
  page.value = 1
  void loadProducts()
}

function handlePageChange(nextPage: number) {
  if (nextPage === page.value) return
  page.value = nextPage
  void loadProducts()
}

function handlePageSizeChange(nextPageSize: number) {
  if (nextPageSize === pageSize.value) return
  pageSize.value = nextPageSize
  page.value = 1
  void loadProducts()
}

async function openDetail(product: ProductItem) {
  detailProduct.value = product
  showDetail.value = true
  mainIndex.value = 0
  try {
    detailImages.value = await getProductImages(product.id)
  } catch {
    detailImages.value = []
  }
}

function closeDetail() {
  showDetail.value = false
  detailProduct.value = null
  detailImages.value = []
}

function handleDrawerVisibility(show: boolean) {
  if (!show) closeDetail()
}

onMounted(loadProducts)
</script>

<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Product Library</p>
        <h1 class="app-title mt-2">礼物管理</h1>
        <p class="app-copy mt-2">礼物按平台隔离，每次导入生成独立内部 ID。</p>
      </div>
      <NButton :loading="isLoading" secondary strong @click="loadProducts">刷新礼物</NButton>
    </header>

    <div class="grid gap-4 md:grid-cols-2">
      <NCard>
        <p class="app-copy">礼物总数</p>
        <p class="mt-1 text-2xl font-semibold">{{ total }}</p>
      </NCard>
      <NCard>
        <p class="app-copy">当前页条数</p>
        <p class="mt-1 text-2xl font-semibold">{{ products.length }}</p>
      </NCard>
    </div>

    <NCard size="medium">
      <div class="flex flex-col gap-3 md:flex-row">
        <NInput v-model:value="keyword" clearable placeholder="搜索名称 / SKU / 工厂" @keyup.enter="searchProducts">
          <template #prefix>
            <NIcon><SearchOutline /></NIcon>
          </template>
        </NInput>
        <NSelect v-model:value="platform" clearable :options="platformOptions" placeholder="平台筛选" style="max-width: 180px" @update:value="searchProducts" />
        <NButton type="primary" @click="searchProducts">搜索</NButton>
      </div>
    </NCard>

    <NEmpty v-if="errorMessage" :description="errorMessage" />
    <template v-else>
      <NEmpty v-if="!products.length" description="暂无礼物" />
      <div v-else class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <NCard v-for="product in products" :key="product.id" size="small" hoverable class="cursor-pointer" @click="openDetail(product)">
          <div class="aspect-[4/3] overflow-hidden rounded-xl bg-slate-100 dark:bg-slate-800">
            <img v-if="product.coverImage" :src="'/local-images/' + product.coverImage" class="h-full w-full object-cover" />
            <div v-else class="flex h-full items-center justify-center">
              <NIcon size="42" depth="3"><ImageOutline /></NIcon>
            </div>
          </div>
          <div class="mt-3 flex items-start justify-between gap-3">
            <div>
              <strong>{{ product.name }}</strong>
              <p class="app-copy mt-1">{{ product.factory }} / {{ product.factorySku }}</p>
            </div>
            <NTag size="small" round>{{ product.platform }}</NTag>
          </div>
          <p class="app-copy mt-2 line-clamp-2">{{ product.extraData }}</p>
        </NCard>
      </div>

      <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <p class="app-copy">共 {{ total }} 条记录</p>
        <NPagination
          :page="page"
          :page-size="pageSize"
          :item-count="total"
          :page-sizes="[12, 24, 48]"
          show-size-picker
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </template>

    <NDrawer :show="showDetail" :width="680" @update:show="handleDrawerVisibility">
      <NDrawerContent title="商品详情" closable>
        <template v-if="detailProduct">
          <!-- 主图轮播 -->
          <!-- 主图区域 — 内容驱动高度，自适应 -->
          <div v-if="mainImages.length" class="relative bg-black/5 rounded overflow-hidden">
            <img v-if="currentMainImage" :src="'/local-images/' + currentMainImage.path" class="w-full block" style="max-height: 60vh; object-fit: contain;" />
            <NButton v-if="mainImages.length > 1" size="small" circle quaternary class="absolute left-2 top-1/2 -translate-y-1/2 !bg-white/80" @click="prevImage"><template #icon><NIcon><ChevronBackOutline /></NIcon></template></NButton>
            <NButton v-if="mainImages.length > 1" size="small" circle quaternary class="absolute right-2 top-1/2 -translate-y-1/2 !bg-white/80" @click="nextImage"><template #icon><NIcon><ChevronForwardOutline /></NIcon></template></NButton>
          </div>
          <div v-if="mainImages.length > 1" class="flex justify-center gap-1.5 mt-1">
            <span v-for="(img, i) in mainImages" :key="img.id" class="w-1.5 h-1.5 rounded-full cursor-pointer transition-colors" :class="i === mainIndex ? 'bg-gray-700 dark:bg-gray-300' : 'bg-gray-300 dark:bg-gray-600'" @click="mainIndex = i" />
          </div>
          <NEmpty v-if="!mainImages.length && !detailOnlyImages.length" description="暂无商品图片" class="py-6" />

          <!-- 商品信息 -->
          <div class="mt-1 space-y-1">
            <h2 class="text-xl font-semibold">{{ detailProduct.name }}</h2>
            <div class="flex flex-wrap items-center gap-2">
              <span class="app-copy">{{ detailProduct.factory }} / {{ detailProduct.factorySku }}</span>
              <NTag size="small" round>{{ detailProduct.platform }}</NTag>
            </div>
            <p v-if="detailProduct.extraData && detailProduct.extraData !== '{}'" class="app-copy text-sm">{{ detailProduct.extraData }}</p>
          </div>

          <!-- 详情图片 -->
          <template v-if="detailOnlyImages.length">
            <NDivider>详情图片</NDivider>
            <div class="space-y-3">
              <img v-for="img in detailOnlyImages" :key="img.id" :src="'/local-images/' + img.path" class="w-full rounded-lg object-contain" />
            </div>
          </template>
        </template>
      </NDrawerContent>
    </NDrawer>
  </section>
</template>
