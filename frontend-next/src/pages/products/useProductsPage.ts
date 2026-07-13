/**
 * useProductsPage — data/filter/CRUD composable for `ProductsPage.vue`
 * (plan §3.7 first half: 商品主档搜索/筛选/归档视图).
 *
 * Keyword, kind, archive mode, pagination, and sorting are all server-owned.
 */
import { onBeforeMount, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { listProductMastersPage, createProductMaster, updateProductMaster } from '@/shared/api/bridge'
import { useUrlFilters, type FilterSchema } from '@/shared/ui/filter-bar'
import { registerRefreshTarget } from '@/shared/lib/view-hotkeys'
import type { ProductMaster } from '@/entities/product'

const schema = [
  { key: 'keyword', type: 'keyword' },
  { key: 'productKind', type: 'enum-multi', dimension: 'productKind' },
] as const satisfies FilterSchema

const PRODUCTS_DEFAULT_PAGE_SIZE = 50

export interface ProductMasterFormInput {
  supplierPlatform: string
  factorySku: string
  supplierProductRef: string
  name: string
  productKind: string
}

export interface UseProductsPageApi {
  masters: Ref<ProductMaster[]>
  loading: Ref<boolean>
  filters: ReturnType<typeof useUrlFilters<typeof schema>>
  /** Archive view toggle — archive/active has no glossary dimension (boolean, not a domain enum). */
  archivedOnly: Ref<boolean>
  page: Ref<number>
  pageSize: Ref<number>
  totalCount: Ref<number>
  load(): Promise<void>
  onPageChange(page: number, pageSize: number): void
  onSort(sortBy: string | null, sortDir: 'asc' | 'desc' | null): void
  submitCreate(input: ProductMasterFormInput): Promise<ProductMaster>
  submitUpdate(id: number, input: ProductMasterFormInput, archived: boolean): Promise<ProductMaster>
  /** Full-object PUT with `archived` flipped (backend has no dedicated archive endpoint). */
  toggleArchived(master: ProductMaster): Promise<ProductMaster>
}

export function useProductsPage(): UseProductsPageApi {
  const masters = ref<ProductMaster[]>([]) as Ref<ProductMaster[]>
  const loading = ref(true)
  const archivedOnly = ref(false)
  const filters = useUrlFilters(schema)
  const page = ref(1)
  const pageSize = ref(PRODUCTS_DEFAULT_PAGE_SIZE)
  const totalCount = ref(0)
  const sortBy = ref<string | null>(null)
  const sortDir = ref<'asc' | 'desc' | null>(null)

  async function load(): Promise<void> {
    loading.value = true
    try {
      const result = await listProductMastersPage({
        keyword: filters.state.keyword.trim(),
        productKinds: filters.state.productKind,
        archivedOnly: archivedOnly.value,
        sortBy: sortBy.value ?? undefined,
        sortDir: sortDir.value ?? undefined,
        limit: pageSize.value,
        offset: (page.value - 1) * pageSize.value,
      })
      totalCount.value = result.totalCount
      if ((page.value - 1) * pageSize.value >= result.totalCount && page.value > 1) {
        page.value = Math.max(1, Math.ceil(result.totalCount / pageSize.value))
        await load()
        return
      }
      masters.value = result.items
    } finally {
      loading.value = false
    }
  }

  function onPageChange(nextPage: number, nextPageSize: number): void {
    page.value = nextPageSize === pageSize.value ? nextPage : 1
    pageSize.value = nextPageSize
    void load()
  }

  function onSort(nextSortBy: string | null, nextSortDir: 'asc' | 'desc' | null): void {
    sortBy.value = nextSortBy
    sortDir.value = nextSortDir
    page.value = 1
    void load()
  }

  watch(
    [() => filters.state.keyword, () => filters.state.productKind, archivedOnly],
    () => {
      page.value = 1
      void load()
    },
    { deep: true },
  )

  async function submitCreate(input: ProductMasterFormInput): Promise<ProductMaster> {
    const created = await createProductMaster(input)
    await load()
    return created
  }

  async function submitUpdate(id: number, input: ProductMasterFormInput, archived: boolean): Promise<ProductMaster> {
    const updated = await updateProductMaster({ id, ...input, archived })
    await load()
    return updated
  }

  async function toggleArchived(master: ProductMaster): Promise<ProductMaster> {
    const updated = await updateProductMaster({
      id: master.id,
      supplierPlatform: master.supplierPlatform,
      factorySku: master.factorySku,
      supplierProductRef: master.supplierProductRef,
      name: master.name,
      productKind: master.productKind,
      archived: !master.archived,
    })
    await load()
    return updated
  }

  onMounted(load)

  let unregisterRefresh: (() => void) | undefined
  onBeforeMount(() => {
    unregisterRefresh = registerRefreshTarget(load)
  })
  onBeforeUnmount(() => unregisterRefresh?.())

  return {
    masters,
    loading,
    filters,
    archivedOnly,
    page,
    pageSize,
    totalCount,
    load,
    onPageChange,
    onSort,
    submitCreate,
    submitUpdate,
    toggleArchived,
  }
}
