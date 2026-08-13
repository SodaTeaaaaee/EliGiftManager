import { describe, expect, test, vi } from 'vitest'

vi.mock('@/shared/api/bridge', () => ({
  pickTabularFile: vi.fn(),
  pickCatalogImportFile: vi.fn(),
  parseTabularFile: vi.fn(),
  createDocumentTemplate: vi.fn(),
  bindTemplateToProfile: vi.fn(),
  setDefaultBinding: vi.fn(),
}))

vi.mock('@/shared/i18n', () => ({
  i18n: { global: { t: (key: string) => key } },
}))

import { useIntakeWizardState } from './useIntakeWizardState'
import type { IntegrationProfile } from '@/entities/profile'

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
