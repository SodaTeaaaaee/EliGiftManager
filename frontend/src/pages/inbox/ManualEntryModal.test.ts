// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const mocks = vi.hoisted(() => ({
  listProfiles: vi.fn(),
  importDemandDocument: vi.fn(),
  feedbackError: vi.fn(),
  feedbackSuccess: vi.fn(),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    te: (_key: string) => false,
  }),
}))

vi.mock('@/shared/api/bridge', () => ({
  listProfiles: mocks.listProfiles,
  importDemandDocument: mocks.importDemandDocument,
}))

vi.mock('@/shared/ui/feedback', () => ({
  useFeedback: () => ({
    error: mocks.feedbackError,
    success: mocks.feedbackSuccess,
    info: vi.fn(),
    receipt: vi.fn(),
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

import ManualEntryModal from './ManualEntryModal.vue'

type ManualEntryModalVm = {
  handleOpen(): Promise<void>
  handleSubmit(): Promise<void>
  profileId: number | null
  documentType: string | null
  externalTitle: string
  profileOptions: Array<{ label: string; value: number }>
}

const leftoverProfiles = [
  { id: 1, profileKey: 'member-leftover', sourceChannel: 'bilibili', sourceSurface: 'membership', demandKind: 'membership_entitlement' },
  { id: 2, profileKey: 'empty-hint', sourceChannel: 'bilibili', sourceSurface: 'community', demandKind: '' },
  { id: 3, profileKey: 'retail-leftover', sourceChannel: 'bilibili', sourceSurface: 'community', demandKind: 'retail_order' },
  { id: 4, profileKey: 'factory', sourceChannel: 'factory-a', sourceSurface: 'factory', demandKind: '' },
]

describe('ManualEntryModal 平台 + 文件种类', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.listProfiles.mockResolvedValue(leftoverProfiles)
    mocks.importDemandDocument.mockResolvedValue({ id: 99 })
  })

  it('非 factory 来源平台都可进，不按 leftover demandKind 过滤', async () => {
    const wrapper = mount(ManualEntryModal, {
      props: { show: true },
      attachTo: document.body,
    })
    await flushPromises()

    const vm = wrapper.vm as unknown as ManualEntryModalVm
    await vm.handleOpen()
    await flushPromises()

    const ids = vm.profileOptions.map((option) => option.value)
    expect(ids).toEqual([1, 2, 3])

    wrapper.unmount()
  })

  it('提交权益时传 documentType，不发 leftover retail_order kind', async () => {
    const wrapper = mount(ManualEntryModal, {
      props: { show: true },
      attachTo: document.body,
    })
    await flushPromises()

    const vm = wrapper.vm as unknown as ManualEntryModalVm
    await vm.handleOpen()
    await flushPromises()
    vm.profileId = 1
    vm.documentType = 'import_entitlement'
    vm.externalTitle = '手工权益'
    await flushPromises()

    await vm.handleSubmit()
    await flushPromises()

    expect(mocks.importDemandDocument).toHaveBeenCalledTimes(1)
    expect(mocks.importDemandDocument).toHaveBeenCalledWith(
      expect.objectContaining({
        documentType: 'import_entitlement',
        kind: 'import_entitlement',
        integrationProfileId: 1,
        captureMode: 'manual_entry',
      }),
    )
    expect(wrapper.emitted('created')).toBeTruthy()

    wrapper.unmount()
  })

  it('提交零售时传 import_sales_order', async () => {
    const wrapper = mount(ManualEntryModal, {
      props: { show: true },
      attachTo: document.body,
    })
    await flushPromises()

    const vm = wrapper.vm as unknown as ManualEntryModalVm
    await vm.handleOpen()
    await flushPromises()
    vm.profileId = 2
    vm.documentType = 'import_sales_order'
    vm.externalTitle = '手工零售'
    await flushPromises()

    await vm.handleSubmit()
    await flushPromises()

    expect(mocks.importDemandDocument).toHaveBeenCalledWith(
      expect.objectContaining({
        documentType: 'import_sales_order',
        kind: 'import_sales_order',
        integrationProfileId: 2,
      }),
    )

    wrapper.unmount()
  })

  it('未选择 documentType 时不发起录入', async () => {
    const wrapper = mount(ManualEntryModal, {
      props: { show: true },
      attachTo: document.body,
    })
    await flushPromises()

    const vm = wrapper.vm as unknown as ManualEntryModalVm
    await vm.handleOpen()
    await flushPromises()
    vm.profileId = 3
    vm.externalTitle = '缺种类'
    await flushPromises()

    await vm.handleSubmit()
    await flushPromises()

    expect(mocks.importDemandDocument).not.toHaveBeenCalled()

    wrapper.unmount()
  })
})
