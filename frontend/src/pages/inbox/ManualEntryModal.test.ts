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

type ManualEntryLine = {
  lineType: string
  obligationTriggerKind: string
  recipientInputState?: string
  routingDisposition?: string
  giftLevelSnapshot?: string
  externalTitle?: string
  requestedQuantity?: number
}

type ManualEntryPayload = {
  sourceCustomerRef?: string
  lines: ManualEntryLine[]
}

type ManualEntryModalVm = {
  handleOpen(): Promise<void>
  handleSubmit(): Promise<void>
  profileId: number | null
  documentType: string | null
  externalTitle: string
  profileOptions: Array<{ label: string; value: number }>
  sourceCustomerRef: string
  sourceCustomerRefPlaceholder: string
  requestedQuantity: number | null
  canSubmit: boolean
}

const leftoverProfiles = [
  { id: 1, profileKey: 'member-leftover', sourceChannel: 'bilibili', sourceSurface: 'membership', demandKind: 'membership_entitlement' },
  { id: 2, profileKey: 'empty-hint', sourceChannel: 'bilibili', sourceSurface: 'retail', demandKind: '' },
  { id: 3, profileKey: 'retail-leftover', sourceChannel: 'bilibili', sourceSurface: 'retail', demandKind: 'retail_order' },
  { id: 4, profileKey: 'factory', sourceChannel: 'factory-a', sourceSurface: 'factory', demandKind: '' },
]

function submittedPayload(): ManualEntryPayload {
  expect(mocks.importDemandDocument).toHaveBeenCalled()
  return mocks.importDemandDocument.mock.calls[0][0] as ManualEntryPayload
}

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

  it('show 从 false 变为 true 时加载平台，不靠 handleOpen', async () => {
    const wrapper = mount(ManualEntryModal, { props: { show: false }, attachTo: document.body })
    await flushPromises()
    expect(mocks.listProfiles).not.toHaveBeenCalled()

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(mocks.listProfiles).toHaveBeenCalled()
    const vm = wrapper.vm as unknown as ManualEntryModalVm
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
    vm.sourceCustomerRef = 'uid-1001'
    await flushPromises()

    expect(vm.canSubmit).toBe(true)
    await vm.handleSubmit()
    await flushPromises()

    expect(mocks.importDemandDocument).toHaveBeenCalledTimes(1)
    expect(mocks.importDemandDocument).toHaveBeenCalledWith(
      expect.objectContaining({
        documentType: 'import_entitlement',
        kind: 'import_entitlement',
        integrationProfileId: 1,
        captureMode: 'manual_entry',
        sourceCustomerRef: 'uid-1001',
      }),
    )
    const line = submittedPayload().lines[0]
    expect(line).toEqual(expect.objectContaining({
      lineType: 'entitlement_rule',
      obligationTriggerKind: 'periodic_membership',
      recipientInputState: 'not_required',
      routingDisposition: 'accepted',
      giftLevelSnapshot: '手工权益',
    }))
    expect(line.lineType).not.toBe('sku_order')
    expect(line.obligationTriggerKind).not.toBe('manual_compensation')
    expect(wrapper.emitted('created')).toBeTruthy()

    wrapper.unmount()
  })

  it('权益时 sourceCustomerRef 占位不标可选，零售仍标可选', async () => {
    const wrapper = mount(ManualEntryModal, {
      props: { show: true },
      attachTo: document.body,
    })
    await flushPromises()

    const vm = wrapper.vm as unknown as ManualEntryModalVm
    await vm.handleOpen()
    await flushPromises()

    expect(vm.sourceCustomerRefPlaceholder).toBe('inbox.manualEntry.sourceCustomerRef')

    vm.documentType = 'import_entitlement'
    await flushPromises()
    expect(vm.sourceCustomerRefPlaceholder).toBe('inbox.manualEntry.sourceCustomerRefRequired')

    vm.documentType = 'import_sales_order'
    await flushPromises()
    expect(vm.sourceCustomerRefPlaceholder).toBe('inbox.manualEntry.sourceCustomerRef')

    wrapper.unmount()
  })

  it('权益缺 sourceCustomerRef 时不可提交', async () => {
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
    vm.sourceCustomerRef = ''
    await flushPromises()

    expect(vm.canSubmit).toBe(false)
    await vm.handleSubmit()
    await flushPromises()

    expect(mocks.importDemandDocument).not.toHaveBeenCalled()

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

    expect(vm.canSubmit).toBe(true)
    await vm.handleSubmit()
    await flushPromises()

    expect(mocks.importDemandDocument).toHaveBeenCalledWith(
      expect.objectContaining({
        documentType: 'import_sales_order',
        kind: 'import_sales_order',
        integrationProfileId: 2,
        lines: [
          expect.objectContaining({
            lineType: 'sku_order',
            obligationTriggerKind: 'manual_compensation',
            recipientInputState: 'ready',
          }),
        ],
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

    expect(vm.canSubmit).toBe(false)
    await vm.handleSubmit()
    await flushPromises()

    expect(mocks.importDemandDocument).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('requestedQuantity 为空或 0 时不可提交且不默认成 1', async () => {
    const wrapper = mount(ManualEntryModal, {
      props: { show: true },
      attachTo: document.body,
    })
    await flushPromises()

    const vm = wrapper.vm as unknown as ManualEntryModalVm
    await vm.handleOpen()
    await flushPromises()
    vm.profileId = 3
    vm.documentType = 'import_sales_order'
    vm.externalTitle = '手工零售'

    vm.requestedQuantity = null
    await flushPromises()
    expect(vm.canSubmit).toBe(false)
    await vm.handleSubmit()
    await flushPromises()
    expect(mocks.importDemandDocument).not.toHaveBeenCalled()

    vm.requestedQuantity = 0
    await flushPromises()
    expect(vm.canSubmit).toBe(false)
    await vm.handleSubmit()
    await flushPromises()
    expect(mocks.importDemandDocument).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('requestedQuantity 有效值原样提交，不默认成 1', async () => {
    const wrapper = mount(ManualEntryModal, {
      props: { show: true },
      attachTo: document.body,
    })
    await flushPromises()

    const vm = wrapper.vm as unknown as ManualEntryModalVm
    await vm.handleOpen()
    await flushPromises()
    vm.profileId = 3
    vm.documentType = 'import_sales_order'
    vm.externalTitle = '手工零售'
    vm.requestedQuantity = 3
    await flushPromises()

    expect(vm.canSubmit).toBe(true)
    await vm.handleSubmit()
    await flushPromises()

    expect(submittedPayload().lines[0].requestedQuantity).toBe(3)

    wrapper.unmount()
  })
})
