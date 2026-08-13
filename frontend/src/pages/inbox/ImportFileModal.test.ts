// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { FieldMappingValue } from '@/shared/ui/field-mapping'

const mocks = vi.hoisted(() => ({
  listProfiles: vi.fn(),
  pickTabularFile: vi.fn(),
  parseTabularFile: vi.fn(),
  importDemandCSV: vi.fn(),
  getDefaultTemplateForProfile: vi.fn(),
  batchAssignDemandToWave: vi.fn(),
  listWavesFiltered: vi.fn(),
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
  pickTabularFile: mocks.pickTabularFile,
  parseTabularFile: mocks.parseTabularFile,
  importDemandCSV: mocks.importDemandCSV,
  getDefaultTemplateForProfile: mocks.getDefaultTemplateForProfile,
  batchAssignDemandToWave: mocks.batchAssignDemandToWave,
  listWavesFiltered: mocks.listWavesFiltered,
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

vi.mock('@/shared/ui/field-mapping', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/ui/field-mapping')>()
  return {
    ...actual,
    FieldMappingEditor: {
      name: 'FieldMappingEditor',
      props: ['modelValue', 'destFields', 'sourceHeaders', 'sampleRows'],
      template: '<div class="field-mapping-editor-stub" />',
    },
  }
})

import ImportFileModal from './ImportFileModal.vue'

type ImportFileModalVm = {
  handleOpen(): Promise<void>
  handleImport(): Promise<void>
  handlePickFile(): Promise<void>
  handleNext(): void
  openSendToWavePicker(): Promise<void>
  handleConfirmSendToWave(): Promise<void>
  profileId: number | null
  documentType: string | null
  filePath: string
  previewRows: Record<string, string>[]
  headers: string[]
  mapping: FieldMappingValue
  currentStep: string
  importResult: unknown
  canNext: boolean
  canNextFromSelect: boolean
  pickError: string
  templateLoading: boolean
  showSendPicker: boolean
  waveOptions: Array<{ label: string; value: number }>
  waveListTruncated: boolean
  sentToWave: boolean
  targetPickerWaveId: number | null
}

const configuredTemplate = {
  templateKey: 'demo',
  mappingRules: JSON.stringify({
    version: 2,
    mode: 'header',
    hasHeader: true,
    columns: { 'line.external_title': 'sku' },
    defaults: {},
  }),
}

const singleDocResult = {
  importRunId: 0,
  evidenceDisabled: false,
  document: { id: 42 },
  errors: [],
  totalProcessed: 1,
  successCount: 1,
  errorCount: 0,
  warnings: [],
}

const multiDocResult = {
  importRunId: 0,
  evidenceDisabled: false,
  document: { id: 10 },
  documents: [{ id: 10 }, { id: 11 }],
  errors: [],
  totalProcessed: 2,
  successCount: 2,
  errorCount: 0,
  warnings: [],
}

const nModalStub = {
  name: 'NModal',
  inheritAttrs: false,
  props: {
    show: { type: Boolean, default: true },
  },
  template: '<div class="n-modal-stub"><slot /><slot name="footer" /></div>',
}

function findButton(wrapper: VueWrapper, text: string) {
  const fromWrapper = wrapper.findAll('button').find((button) => button.text().includes(text))
  if (fromWrapper) return fromWrapper
  const el = Array.from(document.body.querySelectorAll('button')).find((button) =>
    (button.textContent ?? '').includes(text),
  )
  return el ? new DOMWrapper(el) : undefined
}

function isDisabled(button: { attributes: (name: string) => string | undefined; element: Element }): boolean {
  return button.attributes('disabled') !== undefined || (button.element as HTMLButtonElement).disabled
}

async function mountOpen(props: { show?: boolean; targetWaveId?: number } = {}) {
  const wrapper = mount(ImportFileModal, {
    props: { show: true, ...props },
    attachTo: document.body,
    global: { stubs: { NModal: nModalStub, Modal: nModalStub } },
  })
  await flushPromises()
  const vm = wrapper.vm as unknown as ImportFileModalVm
  await vm.handleOpen()
  await flushPromises()
  return { wrapper, vm }
}

async function fillSelectStep(vm: ImportFileModalVm): Promise<void> {
  vm.profileId = 1
  vm.documentType = 'import_entitlement'
  vm.filePath = 'demo.csv'
  vm.previewRows = [{ sku: 'A1' }]
  await flushPromises()
}

describe('ImportFileModal targetWaveId 波内导入', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    document.body.innerHTML = ''
    mocks.listProfiles.mockResolvedValue([
      // leftover demandKind must not drive documentType (retail would infer import_sales_order).
      { id: 1, profileKey: 'demo', sourceChannel: 'bilibili', sourceSurface: 'membership', demandKind: 'retail_order' },
    ])
    mocks.getDefaultTemplateForProfile.mockResolvedValue(configuredTemplate)
    mocks.batchAssignDemandToWave.mockResolvedValue({
      results: [{ demandDocumentId: 42, success: true }],
      successCount: 1,
      failureCount: 0,
    })
  })

  it('导入成功后把新单据分派到 targetWave 并 emit assignedToWave', async () => {
    mocks.importDemandCSV.mockResolvedValue(singleDocResult)

    const { wrapper, vm } = await mountOpen({ targetWaveId: 7 })
    await fillSelectStep(vm)

    await vm.handleImport()
    await flushPromises()

    expect(mocks.importDemandCSV).toHaveBeenCalledTimes(1)
    expect(mocks.importDemandCSV).toHaveBeenCalledWith(
      expect.objectContaining({
        integrationProfileId: 1,
        documentType: 'import_entitlement',
      }),
    )
    expect(mocks.batchAssignDemandToWave).toHaveBeenCalledWith({ waveId: 7, docIds: [42] })
    expect(wrapper.emitted('assignedToWave')).toBeTruthy()
    expect(wrapper.emitted('assignedToWave')![0][0]).toEqual([42])

    wrapper.unmount()
  })

  it('未选择 documentType 时不发起导入', async () => {
    const { wrapper, vm } = await mountOpen({ targetWaveId: 7 })
    vm.profileId = 1
    vm.filePath = 'demo.csv'
    vm.previewRows = [{ sku: 'A1' }]
    await flushPromises()

    await vm.handleImport()
    await flushPromises()

    expect(mocks.importDemandCSV).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('wizard Next then Finish 导入，mapping 是最后一步', async () => {
    mocks.importDemandCSV.mockResolvedValue(singleDocResult)

    const { wrapper, vm } = await mountOpen({ targetWaveId: 7 })
    await fillSelectStep(vm)

    const next = findButton(wrapper, 'intakeWizard.nav.next')
    expect(next).toBeTruthy()
    expect(isDisabled(next!)).toBe(false)
    await next!.trigger('click')
    await flushPromises()

    expect(vm.currentStep).toBe('mapping')
    expect(findButton(wrapper, 'intakeWizard.nav.next')).toBeUndefined()

    const finish = findButton(wrapper, 'inbox.importModal.import')
    expect(finish).toBeTruthy()
    expect(isDisabled(finish!)).toBe(false)
    await finish!.trigger('click')
    await flushPromises()

    expect(mocks.importDemandCSV).toHaveBeenCalledTimes(1)
    expect(wrapper.text() + (document.body.textContent ?? '')).toContain('inbox.importModal.successCount')
    expect(findButton(wrapper, 'intakeWizard.nav.next')).toBeUndefined()
    expect(findButton(wrapper, 'inbox.importModal.import')).toBeUndefined()

    wrapper.unmount()
  })

  it('documents 数组全部分派到 targetWave', async () => {
    mocks.importDemandCSV.mockResolvedValue(multiDocResult)
    mocks.batchAssignDemandToWave.mockResolvedValue({
      results: [
        { demandDocumentId: 10, success: true },
        { demandDocumentId: 11, success: true },
      ],
      successCount: 2,
      failureCount: 0,
    })

    const { wrapper, vm } = await mountOpen({ targetWaveId: 7 })
    await fillSelectStep(vm)

    await findButton(wrapper, 'intakeWizard.nav.next')!.trigger('click')
    await flushPromises()
    await findButton(wrapper, 'inbox.importModal.import')!.trigger('click')
    await flushPromises()

    expect(mocks.batchAssignDemandToWave).toHaveBeenCalledWith({ waveId: 7, docIds: [10, 11] })
    expect(wrapper.emitted('assignedToWave')![0][0]).toEqual([10, 11])

    wrapper.unmount()
  })

  it('快速切换 profile 时忽略过期的 mapping 预览', async () => {
    let resolveFirst: (value: unknown) => void = () => undefined
    let resolveSecond: (value: unknown) => void = () => undefined
    mocks.getDefaultTemplateForProfile
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve }))
      .mockImplementationOnce(() => new Promise((resolve) => { resolveSecond = resolve }))

    const { wrapper, vm } = await mountOpen()
    vm.profileId = 1
    vm.documentType = 'import_entitlement'
    await flushPromises()

    vm.profileId = 2
    await flushPromises()

    resolveFirst({
      templateKey: 'stale',
      mappingRules: JSON.stringify({
        version: 2,
        mode: 'header',
        columns: { 'line.external_title': 'STALE' },
        defaults: {},
      }),
    })
    await flushPromises()
    expect(vm.mapping.columns['line.external_title']).not.toBe('STALE')

    resolveSecond({
      templateKey: 'fresh',
      mappingRules: JSON.stringify({
        version: 2,
        mode: 'header',
        columns: { 'line.external_title': 'FRESH' },
        defaults: {},
      }),
    })
    await flushPromises()
    expect(vm.mapping.columns['line.external_title']).toBe('FRESH')

    wrapper.unmount()
  })

  it('mapping 未配置时 Finish 不可用', async () => {
    mocks.getDefaultTemplateForProfile.mockResolvedValue(null)

    const { wrapper, vm } = await mountOpen()
    await fillSelectStep(vm)

    await findButton(wrapper, 'intakeWizard.nav.next')!.trigger('click')
    await flushPromises()

    expect(vm.currentStep).toBe('mapping')
    expect(vm.canNext).toBe(false)
    const finish = findButton(wrapper, 'inbox.importModal.import')
    expect(finish).toBeTruthy()
    expect(isDisabled(finish!)).toBe(true)

    vm.mapping = {
      ...vm.mapping,
      columns: { 'line.external_title': 'sku' },
    }
    await flushPromises()
    expect(vm.canNext).toBe(true)
    expect(isDisabled(findButton(wrapper, 'inbox.importModal.import')!)).toBe(false)

    wrapper.unmount()
  })

  it('parse 失败时清空 filePath 和 previewRows', async () => {
    mocks.pickTabularFile.mockResolvedValue('broken.csv')
    mocks.parseTabularFile.mockRejectedValue(new Error('bad csv'))

    const { wrapper, vm } = await mountOpen()
    vm.profileId = 1
    vm.documentType = 'import_entitlement'
    vm.filePath = 'old.csv'
    vm.previewRows = [{ sku: 'keep' }]
    await flushPromises()

    await vm.handlePickFile()
    await flushPromises()

    expect(vm.filePath).toBe('')
    expect(vm.previewRows).toEqual([])
    expect(vm.canNextFromSelect).toBe(false)

    wrapper.unmount()
  })

  it('发送到波次过滤已关闭波次并提示截断，且分派全部单据', async () => {
    mocks.importDemandCSV.mockResolvedValue(multiDocResult)
    mocks.listWavesFiltered.mockResolvedValue({
      items: [
        { id: 3, name: 'Open', waveNo: 'W3', lifecycleStage: 'planning' },
        { id: 4, name: 'Done', waveNo: 'W4', lifecycleStage: 'closed' },
      ],
      pagination: { totalCount: 201 },
    })
    mocks.batchAssignDemandToWave.mockResolvedValue({
      results: [
        { demandDocumentId: 10, success: true },
        { demandDocumentId: 11, success: true },
      ],
      successCount: 2,
      failureCount: 0,
    })

    const { wrapper, vm } = await mountOpen()
    await fillSelectStep(vm)
    await vm.handleImport()
    await flushPromises()

    const pageText = wrapper.text() + (document.body.textContent ?? '')
    expect(pageText).toContain('inbox.import.sendToWave')
    await vm.openSendToWavePicker()
    await flushPromises()

    expect(vm.waveOptions.map((option) => option.value)).toEqual([3])
    expect(vm.waveListTruncated).toBe(true)
    expect(wrapper.text() + (document.body.textContent ?? '')).toContain('inbox.batch.waveListTruncated')

    vm.targetPickerWaveId = 3
    await vm.handleConfirmSendToWave()
    await flushPromises()

    expect(mocks.batchAssignDemandToWave).toHaveBeenCalledWith({ waveId: 3, docIds: [10, 11] })
    expect(wrapper.emitted('assignedToWave')![0][0]).toEqual([10, 11])

    wrapper.unmount()
  })
})
