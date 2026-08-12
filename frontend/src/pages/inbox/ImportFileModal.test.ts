// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

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

import ImportFileModal from './ImportFileModal.vue'

describe('ImportFileModal targetWaveId 波内导入', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.listProfiles.mockResolvedValue([
      { id: 1, profileKey: 'demo', sourceChannel: 'bilibili', sourceSurface: 'community', demandKind: 'membership_entitlement' },
    ])
    mocks.getDefaultTemplateForProfile.mockResolvedValue(null)
  })

  it('导入成功后把新单据分派到 targetWave 并 emit assignedToWave', async () => {
    mocks.importDemandCSV.mockResolvedValue({
      importRunId: 0,
      evidenceDisabled: false,
      document: { id: 42 },
      errors: [],
      totalProcessed: 1,
      successCount: 1,
      errorCount: 0,
      warnings: [],
    })
    mocks.batchAssignDemandToWave.mockResolvedValue({
      results: [{ demandDocumentId: 42, success: true }],
      successCount: 1,
      failureCount: 0,
    })

    const wrapper = mount(ImportFileModal, {
      props: { show: true, targetWaveId: 7 },
      attachTo: document.body,
    })
    await flushPromises()

    // 走组件实际暴露的流程：打开时加载 profiles → 构造预览状态 → 执行导入。
    const vm = wrapper.vm as unknown as {
      handleOpen(): Promise<void>
      handleImport(): Promise<void>
      profileId: number | null
      filePath: string
      previewRows: Record<string, string>[]
    }
    await vm.handleOpen()
    await flushPromises()
    vm.profileId = 1
    vm.filePath = 'demo.csv'
    vm.previewRows = [{ sku: 'A1' }]
    await flushPromises()

    await vm.handleImport()
    await flushPromises()

    expect(mocks.importDemandCSV).toHaveBeenCalledTimes(1)
    expect(mocks.batchAssignDemandToWave).toHaveBeenCalledWith({ waveId: 7, docIds: [42] })
    expect(wrapper.emitted('assignedToWave')).toBeTruthy()
    expect(wrapper.emitted('assignedToWave')![0][0]).toEqual([42])

    wrapper.unmount()
  })
})
