<script setup lang="ts">
import { ImageOutline, SearchOutline } from '@vicons/ionicons5'
import { computed, onMounted, ref } from 'vue'
import { NButton, NCard, NEmpty, NIcon, NInput, NModal, NPagination, NSelect, NTag, useMessage } from 'naive-ui'
import { isWailsRuntimeAvailable, listProducts, updateProduct, WAILS_PREVIEW_MESSAGE, type ProductItem, type ProductUpdateInput } from '@/shared/lib/wails/app'

const message = useMessage()
const products = ref<ProductItem[]>([])
const keyword = ref('')
const platform = ref('')
const page = ref(1)
const pageSize = ref(12)
const total = ref(0)
const editing = ref<ProductItem | null>(null)
const editCover = ref('')
const editExtra = ref('{}')
const isLoading = ref(false)
const errorMessage = ref('')

const platformCatalog = ref<string[]>([])

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

function openEdit(product: ProductItem) {
  editing.value = product
  editCover.value = product.coverImage
  editExtra.value = product.extraData || '{}'
}

function closeEditor() {
  editing.value = null
}

function handleEditorVisibility(show: boolean) {
  if (!show) {
    closeEditor()
  }
}

async function saveEdit() {
  if (!editing.value) return

  try {
    const product: ProductUpdateInput = {
      id: editing.value.id,
      platform: editing.value.platform,
      factory: editing.value.factory,
      factorySku: editing.value.factorySku,
      name: editing.value.name,
      coverImage: editCover.value,
      extraData: editExtra.value,
    }
    await updateProduct(product)
    message.success('礼物已更新')
    closeEditor()
    await loadProducts()
  } catch (error) {
    message.error(String(error))
  }
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
        <NCard v-for="product in products" :key="product.id" size="small" hoverable>
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
          <NButton class="mt-3" block secondary @click="openEdit(product)">编辑头图 / Extra</NButton>
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

    <NModal :show="!!editing" preset="card" title="编辑礼物" style="max-width: 560px" @update:show="handleEditorVisibility">
      <div class="space-y-3">
        <NInput v-model:value="editCover" placeholder="头图 URL" />
        <NInput v-model:value="editExtra" type="textarea" :autosize="{ minRows: 5 }" placeholder="ExtraData JSON" />
        <NButton type="primary" block @click="saveEdit">保存</NButton>
      </div>
    </NModal>
  </section>
</template>
