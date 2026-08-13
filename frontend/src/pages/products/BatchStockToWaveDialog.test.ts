// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, shallowMount } from '@vue/test-utils'
import type { SelectOption } from 'naive-ui'

const mocks = vi.hoisted(() => ({
  listWaves: vi.fn(),
  listProductMasters: vi.fn(),
  listProductsByWave: vi.fn(),
  snapshotProductsForWaveDetailed: vi.fn(),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    te: () => false,
  }),
}))

vi.mock('@/shared/i18n/glossary', () => ({
  useGlossary: () => ({ label: (_dimension: string, value: string) => value }),
}))

vi.mock('@/shared/i18n', () => ({
  i18n: {
    global: {
      locale: { value: 'zh-CN' },
      t: (key: string) => key,
    },
  },
}))

vi.mock('@/shared/api/bridge', () => ({
  listWaves: mocks.listWaves,
  listProductMasters: mocks.listProductMasters,
  listProductsByWave: mocks.listProductsByWave,
  snapshotProductsForWaveDetailed: mocks.snapshotProductsForWaveDetailed,
}))

vi.mock('@/shared/ui/feedback', () => ({
  useFeedback: () => ({
    error: vi.fn(),
    success: vi.fn(),
    info: vi.fn(),
    receipt: vi.fn(),
  }),
}))

import BatchStockToWaveDialog from './BatchStockToWaveDialog.vue'
import type { ProductMaster } from '@/entities/product'

const selectedMaster = {
  id: 10,
  supplierPlatform: 'p',
  factorySku: 'sku-10',
  supplierProductRef: '',
  name: 'Widget',
  productKind: 'physical',
  archived: false,
  coverImagePath: '',
  detailImagePaths: '',
  extraData: '',
  createdAt: '',
  updatedAt: '',
} as ProductMaster

describe('BatchStockToWaveDialog wave picker', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.listWaves.mockResolvedValue([
      { id: 1, waveNo: 'W-1', name: 'Intake', lifecycleStage: 'intake' },
      { id: 2, waveNo: 'W-2', name: 'Closed', lifecycleStage: 'closed' },
      { id: 3, waveNo: 'W-3', name: 'Exec', lifecycleStage: 'execution' },
    ])
    mocks.listProductsByWave.mockResolvedValue([])
  })

  it('omits closed waves from the picker options', async () => {
    const wrapper = mount(BatchStockToWaveDialog, {
      props: {
        show: false,
        selectedMasters: [selectedMaster],
      },
      attachTo: document.body,
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    const options = (wrapper.vm as unknown as { waveOptions: SelectOption[] }).waveOptions
    expect(options.map((opt) => opt.value)).toEqual([1, 3])
    expect(options.some((opt) => String(opt.label).includes('Closed'))).toBe(false)
    wrapper.unmount()
  })
})

describe('BatchStockToWaveDialog master picker selection', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.listProductMasters.mockResolvedValue([selectedMaster])
    mocks.listWaves.mockResolvedValue([{ id: 1, waveNo: 'W-1', name: 'Intake', lifecycleStage: 'intake' }])
    mocks.listProductsByWave.mockResolvedValue([])
  })

  async function mountPicker() {
    const wrapper = shallowMount(BatchStockToWaveDialog, {
      props: { show: false },
    })
    await wrapper.setProps({ show: true })
    await flushPromises()
    return wrapper
  }

  it('keeps selected masters when checked keys are strings', async () => {
    const wrapper = await mountPicker()
    const vm = wrapper.vm as unknown as {
      checkedMasterKeys: Array<string | number>
      effectiveMasters: ProductMaster[]
    }

    vm.checkedMasterKeys = [String(selectedMaster.id)]
    await flushPromises()

    expect(vm.effectiveMasters.map((master) => master.id)).toEqual([selectedMaster.id])
    wrapper.unmount()
  })

  it('keeps selected masters when checked keys are numbers', async () => {
    const wrapper = await mountPicker()
    const vm = wrapper.vm as unknown as {
      checkedMasterKeys: Array<string | number>
      effectiveMasters: ProductMaster[]
    }

    vm.checkedMasterKeys = [selectedMaster.id]
    await flushPromises()

    expect(vm.effectiveMasters.map((master) => master.id)).toEqual([selectedMaster.id])
    wrapper.unmount()
  })
})
