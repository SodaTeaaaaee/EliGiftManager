/**
 * useIntakeWizardState — the intake wizard's data/state composable. Owns:
 *
 * - Profile-identity fields (profileKey/sourceChannel/sourceSurface/demandKind),
 *   seeded from `PLATFORM_PRESETS` and re-seeded on every `applyPreset()` call
 *   (create mode only — remap mode never touches these, see `isRemapMode`).
 * - The 6 capability toggles + optional `connectorKey` + its
 *   `trackingSyncModeOverride` (see `deriveProfileDefaults.ts` for why the
 *   sync mode is only ever explicit, never guessed, once a connector is picked).
 * - The CSV sample-upload + column-mapping state (`FieldMappingValue`).
 * - `finish()` — the 2-or-3-call persistence sequence:
 *   - create mode: `createProfile` -> `createDocumentTemplate` -> `bindTemplateToProfile`.
 *   - remap mode (`existingProfile` passed in): `createDocumentTemplate` ->
 *     `bindTemplateToProfile` only, against the existing profile's id.
 *
 * Step navigation is a plain linear index walk over `steps` (`FULL_STEPS` for
 * create mode, `REMAP_STEPS` — sampleUpload/capabilities/confirm only — for
 * remap mode), matching `WizardFrame`'s controlled-`current` contract.
 */
import { computed, reactive, ref } from 'vue'
import {
  pickCsvFile,
  parseCSVFile,
  createProfile,
  createDocumentTemplate,
  bindTemplateToProfile,
} from '@/shared/api/bridge'
import { PLATFORM_PRESETS, type IntakeProfileCapabilities } from '@/shared/lib/demand-intake/platform-presets'
import type { FieldMappingValue } from '@/shared/ui/field-mapping'
import type { IntegrationProfile } from '@/entities/profile'
import { i18n } from '@/shared/i18n'
import { documentTypeForDemandKind, deriveProfileStrategyDefaults, type DemandKind } from './deriveProfileDefaults'

/**
 * Binds a freshly-created template as the profile's default, gracefully degrading to a
 * non-default binding if one already exists. `internal/app/template_usecase.go`'s
 * `BindTemplateToProfile` always INSERTS a new row and REJECTS the call outright when a
 * default binding already exists for the (profile, documentType) pair — there is no
 * update-in-place or delete exposed to the frontend (the repo layer has a `Delete`, but no
 * controller method wires it up). This matters most for the wizard's "remap" mode (re-running
 * the wizard against a profile that already has a default from its original creation) — the
 * re-bind-as-default call will predictably fail there. Rather than leaving the newly-created
 * template fully orphaned, this falls back to a non-default bind and surfaces a translated,
 * specific error so the operator understands the template exists but isn't active yet — a
 * known backend gap, not swallowed.
 */
async function bindAsDefaultOrDegrade(input: {
  integrationProfileId: number
  documentType: string
  templateId: number
}): Promise<void> {
  try {
    await bindTemplateToProfile({ ...input, isDefault: true })
  } catch {
    await bindTemplateToProfile({ ...input, isDefault: false })
    throw new Error(i18n.global.t('intakeWizard.confirm.bindConflict'))
  }
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
  const demandKind = ref<DemandKind>((existingProfile?.demandKind as DemandKind | undefined) ?? firstPreset.demandKind)

  const capabilities = reactive<IntakeProfileCapabilities>({ ...EMPTY_CAPABILITIES })
  if (existingProfile) {
    Object.assign(capabilities, {
      supportsPartialShipment: existingProfile.supportsPartialShipment,
      supportsApiImport: existingProfile.supportsApiImport,
      supportsApiExport: existingProfile.supportsApiExport,
      requiresCarrierMapping: existingProfile.requiresCarrierMapping,
      requiresExternalOrderNo: existingProfile.requiresExternalOrderNo,
      allowsManualClosure: existingProfile.allowsManualClosure,
    })
  } else {
    Object.assign(capabilities, firstPreset.defaultCapabilities)
  }

  const connectorKey = ref(existingProfile?.connectorKey ?? '')
  /** Only consulted when `connectorKey` is non-empty — see `deriveProfileDefaults.ts`. */
  const trackingSyncModeOverride = ref('document_export')

  const mapping = ref<FieldMappingValue>({
    columns: isRemapMode.value ? {} : { ...(firstPreset.defaultColumns as Record<string, string>) },
    defaults: {},
  })

  const csvPath = ref('')
  const csvHeaders = ref<string[]>([])
  const csvRows = ref<Record<string, string>[]>([])
  const parsing = ref(false)
  const pickError = ref('')

  /** Re-seeds channel/surface/demandKind/mapping/capabilities from the selected preset.
   *  No-op in remap mode (profile identity is fixed to the existing profile). */
  function applyPreset(key: string): void {
    presetKey.value = key
    if (isRemapMode.value) return
    const preset = PLATFORM_PRESETS.find((p) => p.key === key)
    if (!preset) return
    sourceChannel.value = preset.sourceChannel
    sourceSurface.value = preset.sourceSurface
    demandKind.value = preset.demandKind
    mapping.value = { columns: { ...(preset.defaultColumns as Record<string, string>) }, defaults: {} }
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
      path = await pickCsvFile()
    } catch (err) {
      pickError.value = err instanceof Error ? err.message : String(err)
      return
    }
    if (!path) return
    csvPath.value = path
    parsing.value = true
    try {
      const preview = await parseCSVFile(path)
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

  /** Gates the Next/Finish button per-step — WizardFrame's `canNext` prop. */
  const canProceedFromCurrentStep = computed<boolean>(() => {
    switch (current.value) {
      case 'platformPreset':
        return profileKey.value.trim().length > 0 && sourceChannel.value.trim().length > 0 && sourceSurface.value.trim().length > 0
      case 'sampleUpload':
        return csvHeaders.value.length > 0
      default:
        return true
    }
  })

  const derivedStrategy = computed(() =>
    deriveProfileStrategyDefaults(demandKind.value, capabilities, {
      hasConnectorKey: connectorKey.value.trim().length > 0,
      trackingSyncModeOverride: trackingSyncModeOverride.value,
    }),
  )
  const documentType = computed(() => documentTypeForDemandKind(demandKind.value))

  const persisting = ref(false)
  const persistError = ref('')

  async function finish(): Promise<IntegrationProfile> {
    persistError.value = ''
    persisting.value = true
    try {
      const docType = documentType.value
      const mappingRulesJSON = JSON.stringify({ columns: mapping.value.columns, defaults: mapping.value.defaults })

      if (isRemapMode.value && existingProfile) {
        const template = await createDocumentTemplate({
          templateKey: `${existingProfile.profileKey}-remap-${Date.now()}`,
          documentType: docType,
          format: 'csv',
          mappingRules: mappingRulesJSON,
          extraData: '',
        })
        await bindAsDefaultOrDegrade({
          integrationProfileId: existingProfile.id,
          documentType: docType,
          templateId: template.id,
        })
        return existingProfile
      }

      const strategy = derivedStrategy.value
      const profile = await createProfile({
        profileKey: profileKey.value.trim(),
        sourceChannel: sourceChannel.value.trim(),
        sourceSurface: sourceSurface.value.trim(),
        demandKind: demandKind.value,
        initialAllocationStrategy: strategy.initialAllocationStrategy,
        identityStrategy: strategy.identityStrategy,
        entitlementAuthorityMode: strategy.entitlementAuthorityMode,
        recipientInputMode: strategy.recipientInputMode,
        referenceStrategy: strategy.referenceStrategy,
        trackingSyncMode: strategy.trackingSyncMode,
        closurePolicy: strategy.closurePolicy,
        supportsPartialShipment: capabilities.supportsPartialShipment,
        supportsApiImport: capabilities.supportsApiImport,
        supportsApiExport: capabilities.supportsApiExport,
        requiresCarrierMapping: capabilities.requiresCarrierMapping,
        requiresExternalOrderNo: capabilities.requiresExternalOrderNo,
        allowsManualClosure: capabilities.allowsManualClosure,
        connectorKey: connectorKey.value.trim(),
        supportedLocales: '',
        defaultLocale: '',
        extraData: '',
      })

      const template = await createDocumentTemplate({
        templateKey: `${profileKey.value.trim()}-default`,
        documentType: docType,
        format: 'csv',
        mappingRules: mappingRulesJSON,
        extraData: '',
      })
      await bindAsDefaultOrDegrade({
        integrationProfileId: profile.id,
        documentType: docType,
        templateId: template.id,
      })
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
    capabilities,
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
    finish,
  }
}

export type UseIntakeWizardStateApi = ReturnType<typeof useIntakeWizardState>
