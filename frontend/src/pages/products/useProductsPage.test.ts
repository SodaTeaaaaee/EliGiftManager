// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const mocks = vi.hoisted(() => ({
  listProductMastersPage: vi.fn(),
  registerRefreshTarget: vi.fn(() => () => {}),
}))

vi.mock('@/shared/api/bridge', () => ({
  listProductMastersPage: mocks.listProductMastersPage,
  createProductMaster: vi.fn(),
  updateProductMaster: vi.fn(),
}))

vi.mock('@/shared/lib/view-hotkeys', () => ({
  registerRefreshTarget: mocks.registerRefreshTarget,
}))

vi.mock('@/shared/ui/filter-bar', () => ({
  useUrlFilters: () => ({
    state: { keyword: '', productKind: [] as string[] },
  }),
}))

import { useProductsPage, type UseProductsPageApi } from './useProductsPage'
import type { ProductMaster } from '@/entities/product'

function master(id: number): ProductMaster {
  return {
    id,
    supplierPlatform: 'p',
    factorySku: `sku-${id}`,
    supplierProductRef: '',
    name: `Master ${id}`,
    productKind: 'physical',
    archived: false,
    coverImagePath: '',
    detailImagePaths: '',
    extraData: '',
    createdAt: '',
    updatedAt: '',
  }
}

const Host = defineComponent({
  setup() {
    const page = useProductsPage()
    return { page }
  },
  render: () => null,
})

function mountPage() {
  const wrapper = mount(Host)
  const page = (wrapper.vm as unknown as { page: UseProductsPageApi }).page
  return { wrapper, page }
}

describe('useProductsPage load errors', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('sets error on first-load RPC failure without treating it as an empty catalog success', async () => {
    mocks.listProductMastersPage.mockRejectedValue(new Error('rpc down'))
    const { wrapper, page } = mountPage()
    await flushPromises()

    expect(page.loading.value).toBe(false)
    expect(page.error.value).toBe('rpc down')
    expect(page.masters.value).toEqual([])
    wrapper.unmount()
  })

  it('keeps previous masters when a later load fails', async () => {
    const items = [master(1)]
    mocks.listProductMastersPage.mockResolvedValueOnce({ items, totalCount: 1 })
    const { wrapper, page } = mountPage()
    await flushPromises()

    expect(page.error.value).toBeNull()
    expect(page.masters.value).toEqual(items)

    mocks.listProductMastersPage.mockRejectedValueOnce('boom')
    await page.load()
    await flushPromises()

    expect(page.error.value).toBe('boom')
    expect(page.masters.value).toEqual(items)
    expect(page.loading.value).toBe(false)
    wrapper.unmount()
  })

  it('clears error after a successful retry', async () => {
    mocks.listProductMastersPage.mockRejectedValueOnce(new Error('rpc down'))
    const { wrapper, page } = mountPage()
    await flushPromises()
    expect(page.error.value).toBe('rpc down')

    const items = [master(2)]
    mocks.listProductMastersPage.mockResolvedValueOnce({ items, totalCount: 1 })
    await page.load()
    await flushPromises()

    expect(page.error.value).toBeNull()
    expect(page.masters.value).toEqual(items)
    wrapper.unmount()
  })
})
