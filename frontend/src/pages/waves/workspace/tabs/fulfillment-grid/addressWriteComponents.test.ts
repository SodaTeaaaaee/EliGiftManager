// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import type { CustomerResolutionFeaturePolicyDTO } from '@/entities/customer-resolution'
import type { FulfillmentGridRow } from './useFulfillmentGrid'

const mocks = vi.hoisted(() => ({
  bindAddressToLine: vi.fn(),
  batchBindAddressToLines: vi.fn(),
  createAddress: vi.fn(),
  listAddressesByProfile: vi.fn(),
  loadPolicy: vi.fn(),
  feedbackError: vi.fn(),
}))

const disabledPolicy = {
  revision: 1,
  customerResolutionWritesEnabled: false,
  candidateScanEnabled: true,
  mergeExecutionEnabled: true,
  splitExecutionEnabled: true,
  importEvidenceEnabled: true,
  carrierRegistryWritesEnabled: true,
  actorRef: '',
  reason: '',
  updatedAt: '2026-07-15T00:00:00Z',
} as CustomerResolutionFeaturePolicyDTO

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/shared/api/bridge', () => ({
  bindAddressToLine: mocks.bindAddressToLine,
  batchBindAddressToLines: mocks.batchBindAddressToLines,
  createAddress: mocks.createAddress,
  listAddressesByProfile: mocks.listAddressesByProfile,
}))

vi.mock('@/shared/composables/useCustomerResolutionFeaturePolicy', () => ({
  useCustomerResolutionFeaturePolicy: () => ({
    policy: { value: disabledPolicy },
    load: mocks.loadPolicy,
  }),
}))

vi.mock('@/shared/ui/feedback', () => ({
  useFeedback: () => ({
    error: mocks.feedbackError,
    success: vi.fn(),
    receipt: vi.fn(),
  }),
}))

vi.mock('@/shared/i18n/glossary', () => ({
  useGlossary: () => ({ label: (_dimension: string, value: string) => value }),
}))

import InlineAddressEditor from './InlineAddressEditor.vue'
import BatchActionBar from './BatchActionBar.vue'

const row = {
  fulfillmentLineId: 11,
  customerProfileId: 7,
} as FulfillmentGridRow

describe('wave address write component gates', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.loadPolicy.mockResolvedValue(disabledPolicy)
    mocks.listAddressesByProfile.mockResolvedValue([{ id: 70, isDefault: true }])
  })

  it('shows the reason and blocks both the button and inline bind handler', async () => {
    const wrapper = shallowMount(InlineAddressEditor, { props: { row } })
    await flushPromises()

    expect(wrapper.text()).toContain('fulfillmentGrid.address.writesDisabledReason')
    const bind = wrapper.get('[data-testid="bind-existing-address"]')
    expect(bind.attributes('disabled')).toBe('true')
    await (wrapper.vm as unknown as { handleBindExisting: () => Promise<void> }).handleBindExisting()
    expect(mocks.bindAddressToLine).not.toHaveBeenCalled()
  })

  it('shows the reason and blocks both the button and batch bind handler', async () => {
    const wrapper = shallowMount(BatchActionBar, {
      props: { selectedRows: [row], waveId: 3 },
    })

    expect(wrapper.text()).toContain('fulfillmentGrid.address.writesDisabledReason')
    const bind = wrapper.get('[data-testid="batch-bind-default-address"]')
    expect(bind.attributes('disabled')).toBe('true')
    await (wrapper.vm as unknown as { handleBindDefaultAddress: () => Promise<void> }).handleBindDefaultAddress()
    expect(mocks.listAddressesByProfile).not.toHaveBeenCalled()
    expect(mocks.batchBindAddressToLines).not.toHaveBeenCalled()
  })
})
