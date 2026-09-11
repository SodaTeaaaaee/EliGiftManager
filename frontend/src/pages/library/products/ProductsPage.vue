<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  NSpin,
  NDrawer,
  NDrawerContent,
} from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { useFeedback } from '@/shared/ui/feedback'
import {
  createAlias,
  createBundleComponent,
  createProduct,
  listAliases,
  listPlatforms,
  listProducts,
} from '@/shared/api/bridge'
import type { Platform, ProductAlias, ProductItem } from '@/entities/models'

const { t } = useI18n()
const feedback = useFeedback()

function errMsg(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

const loading = ref(false)
const actionLoading = ref(false)
const products = ref<ProductItem[]>([])
const platforms = ref<Platform[]>([])

const showCreateProductModal = ref(false)
const productForm = ref({
  name: '',
  factoryPlatformId: null as number | null,
  factorySku: '',
  notes: '',
})

const showAliasDrawer = ref(false)
const selectedProduct = ref<ProductItem | null>(null)
const productAliases = ref<ProductAlias[]>([])

const showCreateAliasModal = ref(false)
const aliasForm = ref({
  platformId: null as number | null,
  externalProductId: '',
  title: '',
  spec: '',
})

// ── Alias bundle components (CreateBundleComponent) ──

// The backend exposes CreateBundleComponent only — no component-list binding
// yet — so the drawer offers append-only component entry per alias.
const showBundleComponentModal = ref(false)
const bundleTargetAlias = ref<ProductAlias | null>(null)
const bundleComponentForm = ref({
  productItemId: null as number | null,
  quantity: 1,
})

const productOptions = computed(() =>
  products.value.map((p) => ({
    label: `${p.Name} (${p.FactorySKU})`,
    value: p.ID,
  })),
)

function openBundleComponent(alias: ProductAlias) {
  bundleTargetAlias.value = alias
  bundleComponentForm.value = {
    productItemId: products.value[0]?.ID ?? null,
    quantity: 1,
  }
  showBundleComponentModal.value = true
}

async function handleSaveBundleComponent() {
  if (!bundleTargetAlias.value || !bundleComponentForm.value.productItemId) return
  actionLoading.value = true
  try {
    await createBundleComponent({
      AliasID: bundleTargetAlias.value.ID,
      ProductItemID: bundleComponentForm.value.productItemId,
      Quantity: bundleComponentForm.value.quantity,
    })
    showBundleComponentModal.value = false
    feedback.success(t('library.bundleComponentAdded'))
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

async function loadData() {
  loading.value = true
  try {
    const [prodRes, platRes] = await Promise.all([
      listProducts(),
      listPlatforms(),
    ])
    products.value = prodRes
    platforms.value = platRes
  } catch (err) {
    console.error('Failed to load products:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadData()
})

const factoryPlatformOptions = computed(() =>
  platforms.value
    .filter((p) => p.Kind === 'factory')
    .map((p) => ({
      label: `${p.Name} (${p.Key})`,
      value: p.ID,
    })),
)

const sourcePlatformOptions = computed(() =>
  platforms.value
    .filter((p) => p.Kind === 'source')
    .map((p) => ({
      label: `${p.Name} (${p.Key})`,
      value: p.ID,
    })),
)

function openCreateProduct() {
  productForm.value = {
    name: '',
    factoryPlatformId: factoryPlatformOptions.value[0]?.value ?? null,
    factorySku: '',
    notes: '',
  }
  showCreateProductModal.value = true
}

async function handleSaveProduct() {
  if (!productForm.value.name.trim() || !productForm.value.factoryPlatformId || !productForm.value.factorySku.trim()) return
  actionLoading.value = true
  try {
    await createProduct({
      Name: productForm.value.name.trim(),
      FactoryPlatformID: productForm.value.factoryPlatformId,
      FactorySKU: productForm.value.factorySku.trim(),
      Notes: productForm.value.notes.trim(),
    })
    showCreateProductModal.value = false
    await loadData()
  } catch (err) {
    console.error('Failed to save product:', err)
  } finally {
    actionLoading.value = false
  }
}

async function openAliases(product: ProductItem) {
  selectedProduct.value = product
  showAliasDrawer.value = true
  await loadAliases(product.ID)
}

async function loadAliases(productId: number) {
  try {
    productAliases.value = await listAliases(productId)
  } catch (err) {
    console.error('Failed to load aliases:', err)
  }
}

function openCreateAlias() {
  aliasForm.value = {
    platformId: sourcePlatformOptions.value[0]?.value ?? null,
    externalProductId: '',
    title: '',
    spec: '',
  }
  showCreateAliasModal.value = true
}

async function handleSaveAlias() {
  if (!selectedProduct.value || !aliasForm.value.platformId || !aliasForm.value.externalProductId.trim()) return
  actionLoading.value = true
  try {
    await createAlias({
      ProductItemID: selectedProduct.value.ID,
      PlatformID: aliasForm.value.platformId,
      ExternalProductID: aliasForm.value.externalProductId.trim(),
      Title: aliasForm.value.title.trim(),
      Spec: aliasForm.value.spec.trim(),
    })
    showCreateAliasModal.value = false
    await loadAliases(selectedProduct.value.ID)
  } catch (err) {
    console.error('Failed to save alias:', err)
  } finally {
    actionLoading.value = false
  }
}

const columns = [
  {
    title: '#',
    key: 'ID',
    width: 80,
    render(row: ProductItem) {
      return `#${row.ID}`
    },
  },
  {
    title: t('library.productName'),
    key: 'Name',
  },
  {
    title: t('library.factoryPlatform'),
    key: 'FactoryPlatformID',
    render(row: ProductItem) {
      const plat = platforms.value.find((p) => p.ID === row.FactoryPlatformID)
      return plat ? plat.Name : `Factory #${row.FactoryPlatformID}`
    },
  },
  {
    title: t('library.factorySKU'),
    key: 'FactorySKU',
  },
  {
    title: t('library.notes'),
    key: 'Notes',
    ellipsis: { tooltip: true },
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 140,
    render(row: ProductItem) {
      return h(
        NButton,
        {
          size: 'small',
          type: 'primary',
          quaternary: true,
          onClick: () => openAliases(row),
        },
        { default: () => t('library.aliases') },
      )
    },
  },
]

const aliasColumns = [
  {
    title: t('library.factoryPlatform'),
    key: 'PlatformID',
    render(row: ProductAlias) {
      const plat = platforms.value.find((p) => p.ID === row.PlatformID)
      return plat ? plat.Name : `Platform #${row.PlatformID}`
    },
  },
  {
    title: t('library.externalProductID'),
    key: 'ExternalProductID',
  },
  {
    title: t('library.aliasTitle'),
    key: 'Title',
  },
  {
    title: t('library.aliasSpec'),
    key: 'Spec',
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 150,
    render(row: ProductAlias) {
      return h(
        NButton,
        {
          size: 'tiny',
          secondary: true,
          onClick: () => openBundleComponent(row),
        },
        { default: () => t('library.bundleComponents') },
      )
    },
  },
]
</script>

<template>
  <div class="products-page">
    <SectionCard :title="t('library.productsTab')">
      <template #actions>
        <NSpace>
          <NButton size="small" type="primary" @click="openCreateProduct">
            {{ t('library.createProduct') }}
          </NButton>
          <NButton size="small" :loading="loading" @click="loadData">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>

      <NSpin :show="loading">
        <div v-if="!products.length" class="products-page__empty">
          <EmptyState :title="t('library.emptyProducts')" size="sm" />
        </div>
        <div v-else class="products-page__table">
          <NDataTable
            :columns="columns"
            :data="products"
            :row-key="(row: ProductItem) => row.ID"
            size="small"
          />
        </div>
      </NSpin>
    </SectionCard>

    <!-- Create Product Modal -->
    <NModal
      v-model:show="showCreateProductModal"
      preset="card"
      :title="t('library.createProduct')"
      style="width: 500px"
    >
      <NForm label-placement="left" label-width="110">
        <NFormItem :label="t('library.productName')">
          <NInput v-model:value="productForm.name" />
        </NFormItem>
        <NFormItem :label="t('library.factoryPlatform')">
          <NSelect
            v-model:value="productForm.factoryPlatformId"
            :options="factoryPlatformOptions"
          />
        </NFormItem>
        <NFormItem :label="t('library.factorySKU')">
          <NInput v-model:value="productForm.factorySku" />
        </NFormItem>
        <NFormItem :label="t('library.notes')">
          <NInput v-model:value="productForm.notes" type="textarea" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateProductModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!productForm.name.trim() || !productForm.factoryPlatformId || !productForm.factorySku.trim()"
            @click="handleSaveProduct"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Product Aliases Drawer -->
    <NDrawer v-model:show="showAliasDrawer" width="560" placement="right">
      <NDrawerContent
        :title="selectedProduct ? `${selectedProduct.Name} - ${t('library.aliases')}` : t('library.aliases')"
      >
        <div class="products-page__drawer-actions">
          <NButton size="small" type="primary" @click="openCreateAlias">
            {{ t('library.createAlias') }}
          </NButton>
        </div>

        <div v-if="!productAliases.length" class="products-page__empty">
          <EmptyState :title="t('library.aliases')" size="sm" />
        </div>
        <div v-else class="products-page__table">
          <NDataTable
            :columns="aliasColumns"
            :data="productAliases"
            :row-key="(row: ProductAlias) => row.ID"
            size="small"
          />
        </div>
      </NDrawerContent>
    </NDrawer>

    <!-- Create Alias Modal -->
    <NModal
      v-model:show="showCreateAliasModal"
      preset="card"
      :title="t('library.createAlias')"
      style="width: 480px"
    >
      <NForm label-placement="left" label-width="120">
        <NFormItem :label="t('library.factoryPlatform')">
          <NSelect v-model:value="aliasForm.platformId" :options="sourcePlatformOptions" />
        </NFormItem>
        <NFormItem :label="t('library.externalProductID')">
          <NInput v-model:value="aliasForm.externalProductId" />
        </NFormItem>
        <NFormItem :label="t('library.aliasTitle')">
          <NInput v-model:value="aliasForm.title" />
        </NFormItem>
        <NFormItem :label="t('library.aliasSpec')">
          <NInput v-model:value="aliasForm.spec" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateAliasModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!aliasForm.platformId || !aliasForm.externalProductId.trim()"
            @click="handleSaveAlias"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Add Bundle Component Modal -->
    <NModal
      v-model:show="showBundleComponentModal"
      preset="card"
      :title="t('library.bundleComponents')"
      style="width: 480px"
    >
      <p class="products-page__hint">
        {{ t('library.bundleComponentHint') }}
      </p>
      <NForm label-placement="left" label-width="110">
        <NFormItem :label="t('library.alias')">
          <span class="products-page__alias-label">
            {{ bundleTargetAlias ? `${bundleTargetAlias.ExternalProductID} - ${bundleTargetAlias.Title || '—'}` : '' }}
          </span>
        </NFormItem>
        <NFormItem :label="t('library.productName')">
          <NSelect
            v-model:value="bundleComponentForm.productItemId"
            :options="productOptions"
            :placeholder="t('common.pleaseSelect')"
          />
        </NFormItem>
        <NFormItem :label="t('inbox.quantity')">
          <NInputNumber v-model:value="bundleComponentForm.quantity" :min="1" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showBundleComponentModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!bundleComponentForm.productItemId"
            @click="handleSaveBundleComponent"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.products-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.products-page__empty {
  padding: var(--space-4) 0;
}

.products-page__table {
  width: 100%;
}

.products-page__drawer-actions {
  margin-bottom: var(--space-3);
  display: flex;
  justify-content: flex-end;
}

.products-page__hint {
  margin: 0 0 var(--space-3);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.products-page__alias-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}
</style>
