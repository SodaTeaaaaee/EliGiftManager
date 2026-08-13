// @vitest-environment happy-dom

import { beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { IntegrationProfile } from '@/entities/profile'

const mocks = vi.hoisted(() => ({
  finish: vi.fn(),
  persistError: { value: '' },
  bindWarning: { value: '' },
  feedbackError: vi.fn(),
  feedbackSuccess: vi.fn(),
  feedbackInfo: vi.fn(),
  beginAnotherFileSession: vi.fn(),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    te: () => false,
  }),
}))

vi.mock('@/shared/ui/feedback', () => ({
  useFeedback: () => ({
    error: mocks.feedbackError,
    success: mocks.feedbackSuccess,
    info: mocks.feedbackInfo,
    receipt: vi.fn(),
  }),
}))

vi.mock('@/shared/ui/wizard', () => ({
  WizardFrame: {
    name: 'WizardFrame',
    props: ['steps', 'current', 'canNext', 'canBack', 'nextLabel', 'backLabel', 'finishLabel', 'cancelLabel'],
    emits: ['next', 'back', 'cancel', 'finish'],
    template: '<button class="wz-finish" @click="$emit(\'finish\')" />',
  },
}))

vi.mock('./useIntakeWizardState', () => ({
  useIntakeWizardState: () => ({
    steps: { value: ['documentType', 'sampleUpload', 'confirm'] },
    current: { value: 'confirm' },
    canProceedFromCurrentStep: { value: true },
    persisting: { value: false },
    persistError: mocks.persistError,
    bindWarning: mocks.bindWarning,
    enabledDocumentTypes: { value: ['import_sales_order'] },
    configuredDocumentTypes: { value: [] },
    finish: mocks.finish,
    beginAnotherFileSession: mocks.beginAnotherFileSession,
  }),
}))

vi.mock('./StepDocumentType.vue', () => ({ default: { template: '<div />' } }))
vi.mock('./StepSampleUploadAndMapping.vue', () => ({ default: { template: '<div />' } }))
vi.mock('./StepConfirm.vue', () => ({ default: { template: '<div />' } }))

import IntakeWizard from './IntakeWizard.vue'

const existingProfile = {
  id: 1,
  profileKey: 'workshop_1',
} as IntegrationProfile

beforeEach(() => {
  mocks.finish.mockReset()
  mocks.persistError.value = ''
  mocks.bindWarning.value = ''
  mocks.feedbackError.mockReset()
  mocks.feedbackSuccess.mockReset()
  mocks.feedbackInfo.mockReset()
  mocks.beginAnotherFileSession.mockReset()
})

describe('IntakeWizard finish catch', () => {
  test('surfaces bindFailed rather than only the raw bind error', async () => {
    mocks.persistError.value = 'intakeWizard.confirm.bindFailed'
    mocks.finish.mockRejectedValue(new Error('bind failed'))

    const wrapper = mount(IntakeWizard, { props: { existingProfile } })
    await wrapper.get('.wz-finish').trigger('click')
    await flushPromises()

    expect(mocks.feedbackError).toHaveBeenCalledWith(
      'intakeWizard.confirm.bindFailed',
      'bind failed',
    )
  })

  test('falls back to bindFailed when persistError was not set', async () => {
    mocks.finish.mockRejectedValue(new Error('bind failed'))

    const wrapper = mount(IntakeWizard, { props: { existingProfile } })
    await wrapper.get('.wz-finish').trigger('click')
    await flushPromises()

    expect(mocks.feedbackError).toHaveBeenCalledWith(
      'intakeWizard.confirm.bindFailed',
      'bind failed',
    )
  })

  test('surfaces bindConflict as info when finish degrades the default bind', async () => {
    mocks.bindWarning.value = 'intakeWizard.confirm.bindConflict'
    mocks.finish.mockResolvedValue(existingProfile)

    const wrapper = mount(IntakeWizard, { props: { existingProfile } })
    await wrapper.get('.wz-finish').trigger('click')
    await flushPromises()

    expect(mocks.feedbackInfo).toHaveBeenCalledWith('intakeWizard.confirm.bindConflict')
    expect(mocks.feedbackError).not.toHaveBeenCalled()
  })
})
