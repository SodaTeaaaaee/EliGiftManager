/**
 * useIntakeWizardState — remap / configure-one-file-kind only.
 * The platform already exists (added via builtin shortcut or custom create).
 * This session binds one documentType: pick kind → sample → confirm.
 */
import { computed, reactive, ref } from 'vue'
import {
  pickTabularFile,
  pickCatalogImportFile,
  parseTabularFile,
  createDocumentTemplate,
  bindTemplateToProfile,
  setDefaultBinding,
  getDefaultTemplateForProfile,
} from '@/shared/api/bridge'
import {
  PLATFORM_PRESETS,
  mappingFromPreset,
  mappingFromPresetForDocumentType,
  type IntakeProfileCapabilities,
} from '@/shared/lib/demand-intake/platform-presets'
import {
  emptyFieldMapping,
  parseMappingRules,
  type FieldMappingValue,
} from '@/shared/ui/field-mapping'
import type { IntegrationProfile } from '@/entities/profile'
import { i18n } from '@/shared/i18n'
import {
  documentTypeForDemandKind,
  deriveProfileStrategyDefaults,
  type BusinessSurfaceChoice,
  type DemandKind,
  type FactoryProfileCapabilities,
} from './deriveProfileDefaults'
import { expectedFileKinds } from '../profileAvailability'
import {
  buildIntakeTemplatePlan,
  canProceedFromSample,
  formatSupportsDocumentType,
  intakeFormatFromPath,
} from './intakeTemplatePlan'

export type { BusinessSurfaceChoice } from './deriveProfileDefaults'

const EXPORT_SAMPLE_EXTENSIONS = ['.csv', '.xlsx']

function samplePickerExtensions(documentType: string): string[] | undefined {
  if (
    documentType === 'export_supplier_order' ||
    documentType === 'export_source_tracking_update'
  ) {
    return [...EXPORT_SAMPLE_EXTENSIONS]
  }
  return undefined
}

function sampleFormatIsAllowed(filePath: string, documentType: string): boolean {
  if (!filePath.trim()) return true
  return formatSupportsDocumentType(intakeFormatFromPath(filePath), documentType)
}

async function bindAsDefaultOrDegrade(input: {
  integrationProfileId: number
  documentType: string
  templateId: number
}): Promise<'ok' | 'conflict'> {
  try {
    await bindTemplateToProfile({ ...input, isDefault: true })
    return 'ok'
  } catch (firstErr) {
    let binding: Awaited<ReturnType<typeof bindTemplateToProfile>>
    try {
      binding = await bindTemplateToProfile({ ...input, isDefault: false })
    } catch {
      throw firstErr
    }
    try {
      await setDefaultBinding(binding.id)
      return 'ok'
    } catch {
      return 'conflict'
    }
  }
}

export type IntakeWizardStepKey = 'documentType' | 'sampleUpload' | 'confirm'

const FILE_SESSION_STEPS: IntakeWizardStepKey[] = ['documentType', 'sampleUpload', 'confirm']

const EMPTY_CAPABILITIES: IntakeProfileCapabilities = {
  supportsPartialShipment: false,
  supportsApiImport: false,
  supportsApiExport: false,
  requiresCarrierMapping: false,
  requiresExternalOrderNo: false,
  allowsManualClosure: false,
}

const EMPTY_FACTORY_CAPABILITIES: FactoryProfileCapabilities = {
  supportsExportSupplierOrder: false,
  supportsImportProductCatalog: false,
  supportsImportSupplierShipment: false,
}

function surfaceFromProfile(profile: IntegrationProfile): BusinessSurfaceChoice {
  return profile.sourceSurface === 'factory' ? 'factory' : 'source'
}

/** Mapping-seed lookup by channel, not a special builtin usage path. */
function mappingPresetKeyFor(profile: IntegrationProfile): string {
  if (profile.sourceChannel === 'bilibili') return 'bilibili'
  return 'custom'
}

export interface UseIntakeWizardStateOptions {
  /** Required: wizard only configures a file kind on an already-added platform. */
  existingProfile: IntegrationProfile
  /** When set (from the file-kind row), pre-select that kind and start on sample upload. */
  initialDocumentType?: string
}

export function useIntakeWizardState(options: UseIntakeWizardStateOptions) {
  const existingProfile = options.existingProfile
  const isRemapMode = computed(() => true)

  const persistedProfile = ref<IntegrationProfile | null>(existingProfile)
  const configuredDocumentTypes = ref<string[]>([])

  const steps = computed<IntakeWizardStepKey[]>(() => FILE_SESSION_STEPS)
  const current = ref<IntakeWizardStepKey>('documentType')

  const presetKey = ref(mappingPresetKeyFor(existingProfile))
  const profileKey = ref(existingProfile.profileKey)
  const sourceChannel = ref(existingProfile.sourceChannel)
  const sourceSurface = ref(existingProfile.sourceSurface)
  const demandKind = ref<DemandKind>((existingProfile.demandKind as DemandKind | undefined) ?? '')
  const factorySupplierPlatform = ref(existingProfile.factorySupplierPlatform ?? '')
  const businessSurface = ref<BusinessSurfaceChoice>(surfaceFromProfile(existingProfile))

  const capabilities = reactive<IntakeProfileCapabilities>({ ...EMPTY_CAPABILITIES })
  const factoryCapabilities = reactive<FactoryProfileCapabilities>({ ...EMPTY_FACTORY_CAPABILITIES })
  const enableEntitlementImport = ref(true)
  const enableSalesOrderImport = ref(true)

  Object.assign(capabilities, {
    supportsPartialShipment: existingProfile.supportsPartialShipment,
    supportsApiImport: existingProfile.supportsApiImport,
    supportsApiExport: existingProfile.supportsApiExport,
    requiresCarrierMapping: existingProfile.requiresCarrierMapping,
    requiresExternalOrderNo: existingProfile.requiresExternalOrderNo,
    allowsManualClosure: existingProfile.allowsManualClosure,
  })
  Object.assign(factoryCapabilities, {
    supportsExportSupplierOrder: existingProfile.supportsExportSupplierOrder ?? false,
    supportsImportProductCatalog: existingProfile.supportsImportProductCatalog ?? false,
    supportsImportSupplierShipment: existingProfile.supportsImportSupplierShipment ?? false,
  })

  const isFactorySurface = computed(
    () => businessSurface.value === 'factory' || sourceSurface.value === 'factory',
  )

  const connectorKey = ref(existingProfile.connectorKey ?? '')
  const trackingSyncModeOverride = ref('document_export')
  const mapping = ref(emptyFieldMapping())
  const csvPath = ref('')
  const csvHeaders = ref<string[]>([])
  const csvRows = ref<Record<string, string>[]>([])
  const parsing = ref(false)
  const pickError = ref('')
  const sessionDocumentType = ref('')

  const enabledDocumentTypes = computed<string[]>(() => expectedFileKinds(existingProfile))

  const remainingDocumentTypes = computed(() =>
    enabledDocumentTypes.value.filter((type) => !configuredDocumentTypes.value.includes(type)),
  )

  const documentType = computed(() => sessionDocumentType.value)

  function clearSampleState(): void {
    csvPath.value = ''
    csvHeaders.value = []
    csvRows.value = []
    pickError.value = ''
  }

  function mappingSeedForDocumentType(docType: string) {
    if (isFactorySurface.value) return emptyFieldMapping()
    const preset = PLATFORM_PRESETS.find((item) => item.key === presetKey.value)
    if (!preset) return emptyFieldMapping()
    const byType = mappingFromPresetForDocumentType(preset, docType)
    if (byType) return byType
    if (documentTypeForDemandKind(preset.demandKind ?? '') === docType) return mappingFromPreset(preset)
    return emptyFieldMapping()
  }

  async function hydrateMappingForDocumentType(docType: string): Promise<void> {
    try {
      const tmpl = await getDefaultTemplateForProfile(existingProfile.id, docType)
      if (sessionDocumentType.value !== docType) return
      if (tmpl?.mappingRules) {
        mapping.value = parseMappingRules(tmpl.mappingRules)
        return
      }
    } catch {
      if (sessionDocumentType.value !== docType) return
    }
    mapping.value = mappingSeedForDocumentType(docType)
  }

  async function setSessionDocumentType(docType: string): Promise<void> {
    if (!enabledDocumentTypes.value.includes(docType)) return
    sessionDocumentType.value = docType
    clearSampleState()
    mapping.value = mappingSeedForDocumentType(docType)
    await hydrateMappingForDocumentType(docType)
  }

  function beginAnotherFileSession(): void {
    clearSampleState()
    mapping.value = emptyFieldMapping()
    pickError.value = ''
    persistError.value = ''
    bindWarning.value = ''
    sessionDocumentType.value = ''
    current.value = 'documentType'
  }

  async function pickAndParseFile(): Promise<void> {
    pickError.value = ''
    const docType = sessionDocumentType.value
    let path: string
    try {
      path = docType === 'import_product_catalog'
        ? await pickCatalogImportFile()
        : await pickTabularFile(samplePickerExtensions(docType))
    } catch (err) {
      pickError.value = err instanceof Error ? err.message : String(err)
      return
    }
    if (!path) return
    if (!sampleFormatIsAllowed(path, docType)) {
      pickError.value = i18n.global.t('intakeWizard.confirm.mappingRequired')
      csvPath.value = ''
      csvHeaders.value = []
      csvRows.value = []
      return
    }
    csvPath.value = path
    if (path.toLowerCase().endsWith('.zip')) {
      csvHeaders.value = []
      csvRows.value = []
      mapping.value = {
        ...mapping.value,
        imageLayout: mapping.value.imageLayout ?? {
          enabled: true,
          matchField: 'product.name',
          namePattern: '{match}#{nn}',
          coverPick: 'lowest_nn',
          tabularGlob: '*.csv',
        },
      }
      return
    }
    await parseCurrentFile(path, mapping.value.hasHeader !== false)
  }

  async function parseCurrentFile(path: string, hasHeader: boolean): Promise<void> {
    if (path.toLowerCase().endsWith('.zip')) return
    parsing.value = true
    try {
      const preview = await parseTabularFile(path, hasHeader)
      csvHeaders.value = preview.headers
      csvRows.value = preview.rows
    } catch (err) {
      pickError.value = err instanceof Error ? err.message : String(err)
      csvHeaders.value = []
      csvRows.value = []
    } finally {
      parsing.value = false
    }
  }

  async function applyMappingUpdate(next: FieldMappingValue): Promise<void> {
    const prevHasHeader = mapping.value.hasHeader !== false
    const nextHasHeader = next.hasHeader !== false
    mapping.value = next
    const path = csvPath.value
    if (!path || path.toLowerCase().endsWith('.zip') || prevHasHeader === nextHasHeader) return
    pickError.value = ''
    await parseCurrentFile(path, nextHasHeader)
  }

  const currentIndex = computed(() => steps.value.indexOf(current.value))
  const isFirstStep = computed(() => currentIndex.value <= 0)
  const isLastStep = computed(() => currentIndex.value === steps.value.length - 1)

  function goNext(): void {
    const idx = currentIndex.value
    if (idx < steps.value.length - 1) current.value = steps.value[idx + 1]
  }
  function goBack(): void {
    const idx = currentIndex.value
    if (idx > 0) current.value = steps.value[idx - 1]
  }

  const canProceedFromCurrentStep = computed<boolean>(() => {
    switch (current.value) {
      case 'documentType':
        return enabledDocumentTypes.value.includes(sessionDocumentType.value)
      case 'sampleUpload':
        return sampleFormatIsAllowed(csvPath.value, sessionDocumentType.value) && canProceedFromSample({
          isFactorySurface: isFactorySurface.value,
          filePath: csvPath.value,
          detectedHeaders: csvHeaders.value,
          mapping: mapping.value,
        })
      default:
        return true
    }
  })

  const derivedStrategy = computed(() =>
    deriveProfileStrategyDefaults(demandKind.value, capabilities, {
      hasConnectorKey: connectorKey.value.trim().length > 0,
      trackingSyncModeOverride: trackingSyncModeOverride.value,
      isFactorySurface: isFactorySurface.value,
    }),
  )

  const persisting = ref(false)
  const persistError = ref('')
  const bindWarning = ref('')

  async function finish(): Promise<IntegrationProfile> {
    persistError.value = ''
    bindWarning.value = ''
    persisting.value = true
    try {
      const docType = sessionDocumentType.value
      if (!docType) throw new Error(i18n.global.t('intakeWizard.confirm.mappingRequired'))

      const profile = persistedProfile.value
      if (!profile) throw new Error(i18n.global.t('intakeWizard.confirm.mappingRequired'))

      const [planned] = buildIntakeTemplatePlan(mapping.value, docType, csvPath.value)
      if (!planned) throw new Error(i18n.global.t('intakeWizard.confirm.mappingRequired'))

      const templateKey = `${profileKey.value.trim()}-${docType}-${Date.now()}`

      const template = await createDocumentTemplate({
        templateKey,
        documentType: docType,
        format: planned.format,
        mappingRules: planned.mappingRules,
        extraData: '',
      })
      try {
        const bindOutcome = await bindAsDefaultOrDegrade({
          integrationProfileId: profile.id,
          documentType: docType,
          templateId: template.id,
        })
        if (bindOutcome === 'conflict') {
          bindWarning.value = i18n.global.t('intakeWizard.confirm.bindConflict')
        }
      } catch (bindErr) {
        persistError.value = i18n.global.t('intakeWizard.confirm.bindFailed')
        throw bindErr
      }

      if (!configuredDocumentTypes.value.includes(docType)) {
        configuredDocumentTypes.value = [...configuredDocumentTypes.value, docType]
      }
      return profile
    } catch (err) {
      if (!persistError.value) {
        persistError.value = err instanceof Error ? err.message : String(err)
      }
      throw err
    } finally {
      persisting.value = false
    }
  }

  const initialDocumentType = options.initialDocumentType ?? ''
  if (initialDocumentType && enabledDocumentTypes.value.includes(initialDocumentType)) {
    void setSessionDocumentType(initialDocumentType)
    current.value = 'sampleUpload'
  }

  return {
    isRemapMode,
    isFactorySurface,
    steps,
    current,
    currentIndex,
    isFirstStep,
    isLastStep,
    presetKey,
    profileKey,
    sourceChannel,
    sourceSurface,
    demandKind,
    businessSurface,
    factorySupplierPlatform,
    capabilities,
    factoryCapabilities,
    enableEntitlementImport,
    enableSalesOrderImport,
    enabledDocumentTypes,
    remainingDocumentTypes,
    configuredDocumentTypes,
    sessionDocumentType,
    setSessionDocumentType,
    persistedProfile,
    beginAnotherFileSession,
    connectorKey,
    trackingSyncModeOverride,
    mapping,
    csvPath,
    csvHeaders,
    csvRows,
    parsing,
    pickError,
    pickAndParseFile,
    applyMappingUpdate,
    goNext,
    goBack,
    canProceedFromCurrentStep,
    derivedStrategy,
    documentType,
    persisting,
    persistError,
    bindWarning,
    finish,
  }
}

export type UseIntakeWizardStateApi = ReturnType<typeof useIntakeWizardState>
