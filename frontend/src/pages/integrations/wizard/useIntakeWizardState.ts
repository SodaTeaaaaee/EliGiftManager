/**
 * useIntakeWizardState — the intake wizard's data/state composable. Owns:
 *
 * - Profile-identity fields (profileKey/sourceChannel/sourceSurface/demandKind/
 *   factorySupplierPlatform), seeded from `PLATFORM_PRESETS`.
 * - Demand capability toggles OR factory capability toggles (by surface).
 * - Optional `connectorKey` + `trackingSyncModeOverride`.
 * - The tabular sample-upload + MappingRules v2 column/position mapping state.
 * - `finish()` — createProfile -> createDocumentTemplate -> bindTemplateToProfile
 *   (remap mode skips profile creation). bindConflict / template partial degrades
 *   without aborting after the profile already exists.
 */
import { computed, reactive, ref } from 'vue'
import {
  pickTabularFile,
  parseTabularFile,
  createProfile,
  createDocumentTemplate,
  bindTemplateToProfile,
} from '@/shared/api/bridge'
import {
  PLATFORM_PRESETS,
  mappingFromPreset,
  type IntakeProfileCapabilities,
} from '@/shared/lib/demand-intake/platform-presets'
import {
  emptyFieldMapping,
  serializeMappingRules,
  type FieldMappingValue,
} from '@/shared/ui/field-mapping'
import type { IntegrationProfile } from '@/entities/profile'
import { i18n } from '@/shared/i18n'
import {
  documentTypeForDemandKind,
  documentTypeForFactoryCaps,
  documentTypesForFactoryCaps,
  deriveProfileStrategyDefaults,
  type BusinessSurfaceChoice,
  type DemandKind,
  type FactoryProfileCapabilities,
} from './deriveProfileDefaults'

export type { BusinessSurfaceChoice } from './deriveProfileDefaults'

/**
 * Binds a freshly-created template as the profile's default, gracefully degrading to a
 * non-default binding if one already exists. Known backend gap: no update-in-place for
 * existing default bindings.
 *
 * Returns `'ok' | 'degraded'` — never throws on the default-bind conflict itself.
 * A hard failure on the non-default bind still throws.
 */
async function bindAsDefaultOrDegrade(input: {
  integrationProfileId: number
  documentType: string
  templateId: number
}): Promise<'ok' | 'degraded'> {
  try {
    await bindTemplateToProfile({ ...input, isDefault: true })
    return 'ok'
  } catch {
    await bindTemplateToProfile({ ...input, isDefault: false })
    return 'degraded'
  }
}

/**
 * Legal dest-key prefixes for each document type (mirrors backend
 * `destCatalogByDocType` in mapping_dest_registry.go). Used to keep demand
 * `line.*` / `recipient.*` seeds out of factory product/shipment/export templates.
 */
function allowedDestPrefixesForDocType(docType: string): readonly string[] {
  switch (docType) {
    case 'import_product_catalog':
      return ['product.']
    case 'import_supplier_shipment':
      return ['shipment.']
    case 'export_supplier_order':
      return ['export.']
    case 'import_entitlement':
    case 'import_sales_order':
      return ['line.', 'document.', 'recipient.']
    case 'import_carrier_mapping':
      return ['carrier.']
    case 'export_source_tracking_update':
      return ['tracking.', 'export.']
    default:
      return []
  }
}

function destAllowedForDocType(dest: string, docType: string): boolean {
  const key = dest.trim()
  if (!key) return false
  const prefixes = allowedDestPrefixesForDocType(docType)
  if (prefixes.some((p) => key.startsWith(p))) return true
  // Demand imports still accept bare (unprefixed) v1 line dest keys.
  if (
    (docType === 'import_entitlement' || docType === 'import_sales_order') &&
    !key.includes('.')
  ) {
    return true
  }
  return false
}

function filterRecordKeys<T>(
  record: Record<string, T> | undefined,
  keep: (dest: string) => boolean,
): Record<string, T> {
  const out: Record<string, T> = {}
  for (const [k, v] of Object.entries(record ?? {})) {
    if (keep(k)) out[k] = v
  }
  return out
}

/**
 * Project the wizard mapping onto dests legal for `docType`.
 * Returns null when no source columns/positions remain — caller should emit
 * empty mappingRules (`""`) so CreateDocumentTemplate skips dest validation
 * instead of rejecting demand-line seeds on factory doc types.
 */
function filterMappingForDocType(
  mapping: FieldMappingValue,
  docType: string,
): FieldMappingValue | null {
  const keep = (dest: string) => destAllowedForDocType(dest, docType)
  const columns = filterRecordKeys(mapping.columns, keep)
  const positions = filterRecordKeys(mapping.positions, keep)
  const defaults = filterRecordKeys(mapping.defaults, keep)
  const transforms = mapping.transforms ? filterRecordKeys(mapping.transforms, keep) : undefined
  const columnOrder = (mapping.columnOrder ?? []).filter(keep)
  const required = mapping.required?.filter(keep)

  const hasSource =
    mapping.mode === 'positional'
      ? Object.keys(positions).length > 0
      : Object.keys(columns).length > 0
  if (!hasSource) return null

  return {
    version: mapping.version ?? 2,
    mode: mapping.mode,
    hasHeader: mapping.hasHeader,
    columns,
    positions,
    defaults,
    transforms,
    columnOrder,
    required,
  }
}

/**
 * Serialize mapping for a specific document type.
 * Empty string = no rules (backend CreateDocumentTemplate skips Parse/Validate).
 * Never serialize an empty columns object — ParseMappingRules rejects that shape.
 */
function mappingRulesJSONForDocType(mapping: FieldMappingValue, docType: string): string {
  const filtered = filterMappingForDocType(mapping, docType)
  if (!filtered) return ''
  return serializeMappingRules(filtered)
}

export type IntakeWizardStepKey = 'platformPreset' | 'businessSurface' | 'sampleUpload' | 'capabilities' | 'confirm'

const FULL_STEPS: IntakeWizardStepKey[] = ['platformPreset', 'businessSurface', 'sampleUpload', 'capabilities', 'confirm']
const REMAP_STEPS: IntakeWizardStepKey[] = ['sampleUpload', 'capabilities', 'confirm']

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

function surfaceFromProfile(profile: IntegrationProfile | null): BusinessSurfaceChoice {
  if (!profile) return 'membership'
  if (profile.sourceSurface === 'factory') return 'factory'
  if (profile.sourceSurface === 'retail' || profile.demandKind === 'retail_order') return 'retail'
  return 'membership'
}

export interface UseIntakeWizardStateOptions {
  /** When set, the wizard runs in "remap" mode: skips profile creation entirely and only
   *  (re)creates a column-mapping template, binding it as this profile's new default. */
  existingProfile?: IntegrationProfile | null
}

export function useIntakeWizardState(options: UseIntakeWizardStateOptions = {}) {
  const existingProfile = options.existingProfile ?? null
  const isRemapMode = computed(() => !!existingProfile)
  const steps = computed<IntakeWizardStepKey[]>(() => (isRemapMode.value ? REMAP_STEPS : FULL_STEPS))
  const current = ref<IntakeWizardStepKey>(steps.value[0])

  const firstPreset = PLATFORM_PRESETS[0]

  const presetKey = ref(isRemapMode.value ? 'custom' : firstPreset.key)
  const profileKey = ref(existingProfile?.profileKey ?? '')
  const profileKeyTouched = ref(!!existingProfile?.profileKey)
  const sourceChannel = ref(existingProfile?.sourceChannel ?? firstPreset.sourceChannel)
  const sourceSurface = ref(existingProfile?.sourceSurface ?? firstPreset.sourceSurface)
  const demandKind = ref<DemandKind>(
    (existingProfile?.demandKind as DemandKind | undefined) ?? firstPreset.demandKind,
  )
  const factorySupplierPlatform = ref(
    existingProfile?.factorySupplierPlatform ?? firstPreset.factorySupplierPlatform ?? '',
  )

  const businessSurface = ref<BusinessSurfaceChoice>(surfaceFromProfile(existingProfile))

  const capabilities = reactive<IntakeProfileCapabilities>({ ...EMPTY_CAPABILITIES })
  const factoryCapabilities = reactive<FactoryProfileCapabilities>({ ...EMPTY_FACTORY_CAPABILITIES })

  if (existingProfile) {
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
  } else {
    Object.assign(capabilities, firstPreset.defaultCapabilities)
  }

  const isFactorySurface = computed(() => businessSurface.value === 'factory' || sourceSurface.value === 'factory')

  const connectorKey = ref(existingProfile?.connectorKey ?? '')
  /** Only consulted when `connectorKey` is non-empty — see `deriveProfileDefaults.ts`. */
  const trackingSyncModeOverride = ref('document_export')

  const mapping = ref<FieldMappingValue>(
    isRemapMode.value ? emptyFieldMapping() : mappingFromPreset(firstPreset),
  )

  const csvPath = ref('')
  const csvHeaders = ref<string[]>([])
  const csvRows = ref<Record<string, string>[]>([])
  const parsing = ref(false)
  const pickError = ref('')

  /** Apply membership/retail/factory surface choice — updates sourceSurface + demandKind. */
  function setBusinessSurface(surface: BusinessSurfaceChoice): void {
    businessSurface.value = surface
    if (surface === 'factory') {
      sourceSurface.value = 'factory'
      demandKind.value = ''
      // Drop demand-line preset seeds (line.*/recipient.*) so finish() cannot
      // push illegal dests into import_product_catalog / import_supplier_shipment /
      // export_supplier_order. Operator re-maps against product/shipment/export fields.
      mapping.value = emptyFieldMapping()
      // Factory requires at least one factory cap — default product catalog on first pick.
      if (
        !factoryCapabilities.supportsExportSupplierOrder &&
        !factoryCapabilities.supportsImportProductCatalog &&
        !factoryCapabilities.supportsImportSupplierShipment
      ) {
        factoryCapabilities.supportsImportProductCatalog = true
      }
      return
    }
    if (surface === 'retail') {
      sourceSurface.value = 'retail'
      demandKind.value = 'retail_order'
      return
    }
    sourceSurface.value = 'membership'
    demandKind.value = 'membership_entitlement'
  }

  /** Re-seeds channel/surface/demandKind/mapping/capabilities from the selected preset.
   *  No-op in remap mode (profile identity is fixed to the existing profile). */
  function applyPreset(key: string): void {
    presetKey.value = key
    if (isRemapMode.value) return
    const preset = PLATFORM_PRESETS.find((p) => p.key === key)
    if (!preset) return
    sourceChannel.value = preset.sourceChannel
    // Preserve an explicit factory choice if the operator already picked it.
    if (businessSurface.value === 'factory') {
      sourceSurface.value = 'factory'
      demandKind.value = ''
      // Never re-seed demand preset mapping onto a factory surface.
      mapping.value = emptyFieldMapping()
    } else {
      sourceSurface.value = preset.sourceSurface
      demandKind.value = preset.demandKind
      businessSurface.value =
        preset.sourceSurface === 'retail' || preset.demandKind === 'retail_order' ? 'retail' : 'membership'
      mapping.value = mappingFromPreset(preset)
    }
    factorySupplierPlatform.value = preset.factorySupplierPlatform ?? ''
    Object.assign(capabilities, EMPTY_CAPABILITIES, preset.defaultCapabilities)
    if (!profileKeyTouched.value) {
      profileKey.value = preset.sourceChannel ? `${preset.sourceChannel}-1` : ''
    }
  }

  /** The operator edited the profile-key field manually — stop auto-suggesting it on preset change. */
  function setProfileKey(value: string): void {
    profileKeyTouched.value = true
    profileKey.value = value
  }

  async function pickAndParseFile(): Promise<void> {
    pickError.value = ''
    let path: string
    try {
      path = await pickTabularFile()
    } catch (err) {
      pickError.value = err instanceof Error ? err.message : String(err)
      return
    }
    if (!path) return
    csvPath.value = path
    parsing.value = true
    try {
      const preview = await parseTabularFile(path, mapping.value.hasHeader !== false)
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

  function mappingIsConfigured(value: FieldMappingValue): boolean {
    if (value.mode === 'positional') {
      return Object.keys(value.positions ?? {}).length > 0 || Object.keys(value.defaults).length > 0
    }
    return Object.keys(value.columns).length > 0 || Object.keys(value.defaults).length > 0
  }

  /** Gates the Next/Finish button per-step — WizardFrame's `canNext` prop. */
  const canProceedFromCurrentStep = computed<boolean>(() => {
    switch (current.value) {
      case 'platformPreset':
        return profileKey.value.trim().length > 0 && sourceChannel.value.trim().length > 0
      case 'businessSurface':
        if (isFactorySurface.value) {
          return factorySupplierPlatform.value.trim().length > 0
        }
        return sourceSurface.value.trim().length > 0
      case 'sampleUpload': {
        // Factory may skip sample file if operator only needs profile flags.
        if (isFactorySurface.value && !csvPath.value && csvHeaders.value.length === 0) {
          return true
        }
        const hasFile = csvPath.value.length > 0 || csvHeaders.value.length > 0
        if (!hasFile) return false
        if (mapping.value.mode === 'positional') return mappingIsConfigured(mapping.value) || csvHeaders.value.length > 0
        return csvHeaders.value.length > 0
      }
      case 'capabilities':
        if (isFactorySurface.value) {
          return (
            factoryCapabilities.supportsExportSupplierOrder ||
            factoryCapabilities.supportsImportProductCatalog ||
            factoryCapabilities.supportsImportSupplierShipment
          )
        }
        return true
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

  const documentType = computed(() => {
    if (isFactorySurface.value) return documentTypeForFactoryCaps(factoryCapabilities)
    return documentTypeForDemandKind(demandKind.value)
  })

  const persisting = ref(false)
  const persistError = ref('')
  /** Set when template was bound non-default after a default-bind conflict. */
  const bindWarning = ref('')

  async function finish(): Promise<IntegrationProfile> {
    persistError.value = ''
    bindWarning.value = ''
    persisting.value = true
    try {
      const format = csvPath.value.toLowerCase().endsWith('.xlsx') || csvPath.value.toLowerCase().endsWith('.xls')
        ? 'xlsx'
        : 'csv'

      let anyDegraded = false
      let templatesCreated = 0
      let templatesFailed = 0

      if (isRemapMode.value && existingProfile) {
        const docType = documentType.value
        // Remap still scopes mapping to the target docType namespace.
        const rulesJSON = mappingRulesJSONForDocType(mapping.value, docType)
        const template = await createDocumentTemplate({
          templateKey: `${existingProfile.profileKey}-remap-${Date.now()}`,
          documentType: docType,
          format,
          mappingRules: rulesJSON,
          extraData: '',
        })
        const bindResult = await bindAsDefaultOrDegrade({
          integrationProfileId: existingProfile.id,
          documentType: docType,
          templateId: template.id,
        })
        if (bindResult === 'degraded') anyDegraded = true
        if (anyDegraded) {
          bindWarning.value = i18n.global.t('intakeWizard.confirm.bindConflict')
        }
        return existingProfile
      }

      const strategy = derivedStrategy.value
      const profile = await createProfile({
        profileKey: profileKey.value.trim(),
        sourceChannel: sourceChannel.value.trim(),
        sourceSurface: isFactorySurface.value ? 'factory' : sourceSurface.value.trim(),
        demandKind: isFactorySurface.value ? '' : demandKind.value,
        initialAllocationStrategy: strategy.initialAllocationStrategy,
        identityStrategy: strategy.identityStrategy,
        entitlementAuthorityMode: strategy.entitlementAuthorityMode,
        recipientInputMode: strategy.recipientInputMode,
        referenceStrategy: strategy.referenceStrategy,
        trackingSyncMode: strategy.trackingSyncMode,
        closurePolicy: strategy.closurePolicy,
        supportsPartialShipment: isFactorySurface.value ? false : capabilities.supportsPartialShipment,
        supportsApiImport: isFactorySurface.value ? false : capabilities.supportsApiImport,
        supportsApiExport: isFactorySurface.value ? false : capabilities.supportsApiExport,
        requiresCarrierMapping: isFactorySurface.value ? false : capabilities.requiresCarrierMapping,
        requiresExternalOrderNo: isFactorySurface.value ? false : capabilities.requiresExternalOrderNo,
        allowsManualClosure: isFactorySurface.value ? false : capabilities.allowsManualClosure,
        supportsExportSupplierOrder: isFactorySurface.value
          ? factoryCapabilities.supportsExportSupplierOrder
          : false,
        supportsImportProductCatalog: isFactorySurface.value
          ? factoryCapabilities.supportsImportProductCatalog
          : false,
        supportsImportSupplierShipment: isFactorySurface.value
          ? factoryCapabilities.supportsImportSupplierShipment
          : false,
        connectorKey: connectorKey.value.trim(),
        factorySupplierPlatform: factorySupplierPlatform.value.trim(),
        supportedLocales: '',
        defaultLocale: '',
        extraData: '',
      })

      // Profile already exists past this point. Template/bind failures must not
      // throw away the success — surface a bindWarning and still return the profile.
      const docTypes = isFactorySurface.value
        ? documentTypesForFactoryCaps(factoryCapabilities)
        : [documentType.value]

      for (const docType of docTypes) {
        try {
          // Per-docType filter: demand line.* seeds never go into factory caps;
          // multi-cap factory mappings are projected onto each namespace.
          const rulesJSON = mappingRulesJSONForDocType(mapping.value, docType)
          const template = await createDocumentTemplate({
            templateKey: `${profileKey.value.trim()}-${docType}-default`,
            documentType: docType,
            format,
            mappingRules: rulesJSON,
            extraData: '',
          })
          const bindResult = await bindAsDefaultOrDegrade({
            integrationProfileId: profile.id,
            documentType: docType,
            templateId: template.id,
          })
          templatesCreated += 1
          if (bindResult === 'degraded') anyDegraded = true
        } catch {
          templatesFailed += 1
        }
      }

      if (templatesFailed > 0 && templatesCreated === 0) {
        bindWarning.value = i18n.global.t('intakeWizard.confirm.templateAllFailed')
      } else if (templatesFailed > 0) {
        bindWarning.value = i18n.global.t('intakeWizard.confirm.templatePartial')
      } else if (anyDegraded) {
        bindWarning.value = i18n.global.t('intakeWizard.confirm.bindConflict')
      }
      return profile
    } catch (err) {
      persistError.value = err instanceof Error ? err.message : String(err)
      throw err
    } finally {
      persisting.value = false
    }
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
    setProfileKey,
    sourceChannel,
    sourceSurface,
    demandKind,
    businessSurface,
    setBusinessSurface,
    factorySupplierPlatform,
    capabilities,
    factoryCapabilities,
    connectorKey,
    trackingSyncModeOverride,
    mapping,
    csvPath,
    csvHeaders,
    csvRows,
    parsing,
    pickError,
    applyPreset,
    pickAndParseFile,
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
