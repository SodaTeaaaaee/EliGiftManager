import { describe, expect, test, vi, beforeEach } from 'vitest'

vi.mock('@/shared/api/bridge', () => ({
  pickTabularFile: vi.fn(),
  pickCatalogImportFile: vi.fn(),
  parseTabularFile: vi.fn(),
  createDocumentTemplate: vi.fn(),
  bindTemplateToProfile: vi.fn(),
  setDefaultBinding: vi.fn(),
  getDefaultTemplateForProfile: vi.fn().mockResolvedValue(null),
}))

vi.mock('@/shared/i18n', () => ({
  i18n: { global: { t: (key: string) => key } },
}))

import {
  bindTemplateToProfile,
  createDocumentTemplate,
  getDefaultTemplateForProfile,
  parseTabularFile,
  pickTabularFile,
  setDefaultBinding,
} from '@/shared/api/bridge'
import { useIntakeWizardState } from './useIntakeWizardState'
import type { IntegrationProfile } from '@/entities/profile'
import type { FieldMappingValue } from '@/shared/ui/field-mapping'

function profile(overrides: Partial<IntegrationProfile> = {}): IntegrationProfile {
  return {
    id: 1,
    profileKey: 'workshop_1',
    sourceChannel: 'workshop',
    sourceSurface: 'membership',
    demandKind: '',
    initialAllocationStrategy: '',
    identityStrategy: '',
    entitlementAuthorityMode: '',
    recipientInputMode: '',
    referenceStrategy: '',
    trackingSyncMode: 'unsupported',
    closurePolicy: '',
    supportsPartialShipment: false,
    supportsApiImport: false,
    supportsApiExport: false,
    requiresCarrierMapping: false,
    requiresExternalOrderNo: false,
    allowsManualClosure: false,
    supportsExportSupplierOrder: false,
    supportsImportProductCatalog: false,
    supportsImportSupplierShipment: false,
    connectorKey: '',
    factorySupplierPlatform: '',
    supportedLocales: '',
    defaultLocale: '',
    extraData: '',
    createdAt: '',
    updatedAt: '',
    ...overrides,
  } as IntegrationProfile
}

const mappedSalesOrder: FieldMappingValue = {
  version: 2,
  mode: 'header',
  hasHeader: true,
  columns: { 'line.external_title': 'Title' },
  defaults: {},
}

async function readyToFinish() {
  const state = useIntakeWizardState({ existingProfile: profile() })
  await state.setSessionDocumentType('import_sales_order')
  state.mapping.value = mappedSalesOrder
  state.csvPath.value = 'sample.csv'
  return state
}

beforeEach(() => {
  vi.mocked(getDefaultTemplateForProfile).mockReset()
  vi.mocked(getDefaultTemplateForProfile).mockResolvedValue(null)
  vi.mocked(createDocumentTemplate).mockReset()
  vi.mocked(bindTemplateToProfile).mockReset()
  vi.mocked(setDefaultBinding).mockReset()
  vi.mocked(parseTabularFile).mockReset()
  vi.mocked(pickTabularFile).mockReset()
})

describe('intake wizard remap-only steps', () => {
  test('never includes the platform picker, for custom or seeded profiles', () => {
    const custom = useIntakeWizardState({ existingProfile: profile() })
    const seeded = useIntakeWizardState({
      existingProfile: profile({
        profileKey: 'bilibili_membership_demo',
        sourceChannel: 'bilibili',
      }),
    })
    expect(custom.steps.value).toEqual(['documentType', 'sampleUpload', 'confirm'])
    expect(seeded.steps.value).toEqual(custom.steps.value)
    expect(custom.steps.value).not.toContain('platformPreset')
  })

  test('pre-selects a file kind from the shared detail row', () => {
    const state = useIntakeWizardState({
      existingProfile: profile(),
      initialDocumentType: 'import_sales_order',
    })
    expect(state.sessionDocumentType.value).toBe('import_sales_order')
    expect(state.current.value).toBe('sampleUpload')
  })
})

describe('finish()', () => {
  test('marks the file kind configured after create and bind succeed', async () => {
    vi.mocked(createDocumentTemplate).mockResolvedValue({
      id: 10,
      templateKey: 't',
      documentType: 'import_sales_order',
      format: 'csv',
      mappingRules: '{}',
      extraData: '',
      createdAt: '',
      updatedAt: '',
    })
    vi.mocked(bindTemplateToProfile).mockResolvedValue({ id: 1 } as never)

    const state = await readyToFinish()
    const result = await state.finish()

    expect(result.id).toBe(1)
    expect(state.configuredDocumentTypes.value).toContain('import_sales_order')
    expect(createDocumentTemplate).toHaveBeenCalled()
    expect(bindTemplateToProfile).toHaveBeenCalled()
    expect(state.persistError.value).toBe('')
  })

  test('does not mark configured or succeed when template create fails', async () => {
    vi.mocked(createDocumentTemplate).mockRejectedValue(new Error('template create failed'))

    const state = await readyToFinish()
    await expect(state.finish()).rejects.toThrow('template create failed')
    expect(state.configuredDocumentTypes.value).not.toContain('import_sales_order')
    expect(state.persistError.value).toBe('template create failed')
  })

  test('does not mark configured when bind fails after create', async () => {
    vi.mocked(createDocumentTemplate).mockResolvedValue({
      id: 10,
      templateKey: 't',
      documentType: 'import_sales_order',
      format: 'csv',
      mappingRules: '{}',
      extraData: '',
      createdAt: '',
      updatedAt: '',
    })
    vi.mocked(bindTemplateToProfile).mockRejectedValue(new Error('bind failed'))
    vi.mocked(setDefaultBinding).mockRejectedValue(new Error('set default failed'))

    const state = await readyToFinish()
    await expect(state.finish()).rejects.toThrow('bind failed')
    expect(state.configuredDocumentTypes.value).not.toContain('import_sales_order')
    expect(state.persistError.value).toBe('intakeWizard.confirm.bindFailed')
  })

  test('warns with bindConflict when attached but default cannot be set', async () => {
    vi.mocked(createDocumentTemplate).mockResolvedValue({
      id: 10,
      templateKey: 't',
      documentType: 'import_sales_order',
      format: 'csv',
      mappingRules: '{}',
      extraData: '',
      createdAt: '',
      updatedAt: '',
    })
    vi.mocked(bindTemplateToProfile)
      .mockRejectedValueOnce(new Error('default binding already exists for profile 1 / type "import_sales_order" (binding ID 9)'))
      .mockResolvedValueOnce({ id: 2 } as never)
    vi.mocked(setDefaultBinding).mockRejectedValue(new Error('set default failed'))

    const state = await readyToFinish()
    const result = await state.finish()

    expect(result.id).toBe(1)
    expect(state.configuredDocumentTypes.value).toContain('import_sales_order')
    expect(state.persistError.value).toBe('')
    expect(state.bindWarning.value).toBe('intakeWizard.confirm.bindConflict')
  })
})

describe('reconfigure mapping hydrate', () => {
  test('loads existing default template mappingRules instead of an empty seed', async () => {
    vi.mocked(getDefaultTemplateForProfile).mockResolvedValue({
      id: 9,
      templateKey: 'existing',
      documentType: 'import_sales_order',
      format: 'csv',
      mappingRules: JSON.stringify({
        version: 2,
        mode: 'header',
        hasHeader: true,
        columns: { 'line.external_title': 'Reward' },
        defaults: {},
      }),
      extraData: '',
      createdAt: '',
      updatedAt: '',
    })

    const state = useIntakeWizardState({
      existingProfile: profile(),
      initialDocumentType: 'import_sales_order',
    })

    await vi.waitFor(() => {
      expect(state.mapping.value.columns['line.external_title']).toBe('Reward')
    })
    expect(getDefaultTemplateForProfile).toHaveBeenCalledWith(1, 'import_sales_order')
  })
})

describe('applyMappingUpdate hasHeader re-parse', () => {
  test('re-parses the current file when hasHeader is toggled', async () => {
    vi.mocked(parseTabularFile).mockResolvedValue({
      headers: ['A'],
      rows: [{ A: '1' }],
    } as never)

    const state = useIntakeWizardState({ existingProfile: profile() })
    await state.setSessionDocumentType('import_sales_order')
    state.csvPath.value = 'sample.csv'
    state.mapping.value = {
      version: 2,
      mode: 'header',
      hasHeader: true,
      columns: {},
      defaults: {},
    }

    await state.applyMappingUpdate({
      ...state.mapping.value,
      hasHeader: false,
    })

    expect(parseTabularFile).toHaveBeenCalledWith('sample.csv', false)
    expect(state.csvHeaders.value).toEqual(['A'])
    expect(state.mapping.value.hasHeader).toBe(false)
  })
})

describe('export sample picker rejects xls', () => {
  const exportMapping: FieldMappingValue = {
    version: 2,
    mode: 'header',
    hasHeader: true,
    columns: { 'export.factory_sku': 'SKU' },
    defaults: {},
  }

  test.each([
    {
      documentType: 'export_supplier_order',
      existing: profile({
        sourceSurface: 'factory',
        supportsExportSupplierOrder: true,
        factorySupplierPlatform: 'workshop',
      }),
    },
    {
      documentType: 'export_source_tracking_update',
      existing: profile({
        trackingSyncMode: 'document_export',
      }),
    },
  ])('$documentType does not offer or accept .xls', async ({ documentType, existing }) => {
    vi.mocked(pickTabularFile).mockResolvedValue('sample.xls')

    const state = useIntakeWizardState({ existingProfile: existing })
    await state.setSessionDocumentType(documentType)
    state.current.value = 'sampleUpload'

    await state.pickAndParseFile()

    expect(pickTabularFile).toHaveBeenCalledWith(['.csv', '.xlsx'])
    expect(state.csvPath.value).toBe('')
    expect(state.pickError.value).toBe('intakeWizard.confirm.mappingRequired')
    expect(parseTabularFile).not.toHaveBeenCalled()

    state.mapping.value = exportMapping
    state.csvPath.value = 'sample.xls'
    expect(state.canProceedFromCurrentStep.value).toBe(false)

    state.csvPath.value = 'sample.xlsx'
    expect(state.canProceedFromCurrentStep.value).toBe(true)

    state.csvPath.value = 'sample.csv'
    expect(state.canProceedFromCurrentStep.value).toBe(true)
  })

  test('import_sales_order still allows xls and does not filter picker extensions', async () => {
    vi.mocked(pickTabularFile).mockResolvedValue('orders.xls')
    vi.mocked(parseTabularFile).mockResolvedValue({
      headers: ['Title'],
      rows: [{ Title: 'A' }],
    } as never)

    const state = useIntakeWizardState({ existingProfile: profile() })
    await state.setSessionDocumentType('import_sales_order')
    await state.pickAndParseFile()

    expect(pickTabularFile).toHaveBeenCalledWith(undefined)
    expect(state.csvPath.value).toBe('orders.xls')
    expect(state.pickError.value).toBe('')
    expect(parseTabularFile).toHaveBeenCalledWith('orders.xls', true)
  })
})
