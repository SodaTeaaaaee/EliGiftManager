<script setup lang="ts">
/**
 * IntegrationDetailDrawer — the per-profile detail panel (plan P4
 * integrations page). Self-fetching: given a `profileId`, loads the full
 * profile + its bindings + the system's document templates on every open.
 * Sections: connector config, capability toggles (editable), document
 * templates (read-only, derived from bindings), template bindings
 * (setDefault / unbind via TemplateController), and an expert-mode fold-out
 * (raw `extraData` JSON + `connectorKey`).
 */
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NSwitch, NSelect, NInput, NSpin } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { DetailDrawer } from '@/shared/ui/drawer'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import { ErrorBanner, useFeedback } from '@/shared/ui/feedback'
import {
  getProfile,
  listBindingsByProfile,
  listDocumentTemplates,
  listConnectorCapabilities,
  updateProfile,
  setDefaultBinding,
  unbindTemplate,
  listCarrierMappings,
  listExternalCarriers,
  bindInternalCarrier,
  deleteCarrierMapping,
  importCarrierMappings,
  pickTabularFile,
  createDocumentTemplate,
  bindTemplateToProfile,
} from '@/shared/api/bridge'
import type { IntegrationProfile } from '@/entities/profile'
import type { dto } from '../../../wailsjs/go/models'
import IntakeWizard from './wizard/IntakeWizard.vue'
import { useCustomerResolutionFeaturePolicy } from '@/shared/composables/useCustomerResolutionFeaturePolicy'
import {
  captureProfileScope,
  isProfileEntityActive,
  isProfileLoadActive,
  isProfileScopeActive,
  type ProfileScope,
} from './profileScopedFlow'

function formatAliases(raw: string | undefined): string {
  if (!raw) return '—'
  try {
    const parsed = JSON.parse(raw) as unknown
    if (Array.isArray(parsed)) {
      const labels = parsed.map((item) => String(item).trim()).filter(Boolean)
      return labels.length ? labels.join(', ') : '—'
    }
  } catch {
    // fall through — treat as plain comma-separated text
  }
  return raw.trim() || '—'
}

const props = defineProps<{
  profileId: number | null
  show: boolean
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  changed: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const featurePolicy = useCustomerResolutionFeaturePolicy()
let profileScopeGeneration = 0
let profileLoadSequence = 0

function currentProfileScope(expectedProfileId?: number): ProfileScope | null {
  if (!props.show) return null
  const scope = captureProfileScope(props.profileId, profileScopeGeneration)
  if (!scope || (expectedProfileId != null && scope.profileId !== expectedProfileId)) return null
  return scope
}

function profileScopeIsCurrent(scope: ProfileScope): boolean {
  return props.show && isProfileScopeActive(scope, props.profileId, profileScopeGeneration)
}

function profileEntityIsCurrent(scope: ProfileScope, entityProfileId: number): boolean {
  return props.show && isProfileEntityActive(
    scope,
    props.profileId,
    profileScopeGeneration,
    entityProfileId,
  )
}

const loading = ref(false)
const loadError = ref('')
const profile = ref<IntegrationProfile | null>(null)
const bindings = ref<dto.ProfileTemplateBindingDTO[]>([])
const allTemplates = ref<dto.DocumentTemplateDTO[]>([])
const carrierMappings = ref<dto.CarrierMappingDTO[]>([])
const externalCarriers = ref<dto.ExternalCarrierDTO[]>([])
const importingCarriers = ref(false)
const deletingCarrierId = ref<number | null>(null)
const lastCarrierImportEvidence = ref<{ importRunId: number; evidenceDisabled: boolean } | null>(null)
const bindingExternalCarrier = ref<dto.ExternalCarrierDTO | null>(null)
const internalCarrierCodeDraft = ref('')
const bindingCarrier = ref(false)
const carrierWritesEnabled = computed(() => featurePolicy.isEnabled('carrierRegistryWritesEnabled'))

function carrierConflictCopy(carrier: dto.ExternalCarrierDTO): string {
  return carrier.conflictReason
}

const boundTemplates = computed(() => {
  const templateIds = new Set(bindings.value.map((b) => b.templateId))
  return allTemplates.value.filter((tmpl) => templateIds.has(tmpl.id))
})

const documentTypeOptions = computed<SelectOption[]>(() => {
  const values = profile.value?.sourceSurface === 'factory'
    ? ['import_product_catalog', 'export_supplier_order', 'import_supplier_shipment']
    : [
        profile.value?.demandKind === 'retail_order' ? 'import_sales_order' : 'import_entitlement',
        'import_carrier_mapping',
        'export_source_tracking_update',
      ]
  return values.map((value) => ({ label: value, value }))
})

const templateFormatOptions = computed<SelectOption[]>(() => {
  const values = templateDraft.documentType === 'import_product_catalog'
    ? ['zip', 'csv', 'xlsx', 'xls']
    : ['csv', 'xlsx', 'xls']
  return values.map((value) => ({ label: value.toUpperCase(), value }))
})

const showTemplateCreator = ref(false)
const creatingTemplate = ref(false)
const templateCreateError = ref('')
const templateDraft = reactive({
  templateKey: '',
  documentType: '',
  format: '',
  mappingRules: '',
  isDefault: true,
})

function openTemplateCreator(): void {
  if (!profile.value || !currentProfileScope(profile.value.id)) return
  templateDraft.templateKey = `${profile.value.profileKey}-template-${Date.now()}`
  templateDraft.documentType = ''
  templateDraft.format = ''
  templateDraft.mappingRules = ''
  templateDraft.isDefault = true
  templateCreateError.value = ''
  showTemplateCreator.value = true
}

const canCreateTemplate = computed(() =>
  profile.value?.id === props.profileId &&
  templateDraft.templateKey.trim().length > 0 &&
  templateDraft.documentType.length > 0 &&
  templateDraft.format.length > 0 &&
  templateDraft.mappingRules.trim().length > 0 &&
  !creatingTemplate.value,
)

async function createAndBindTemplate(): Promise<void> {
  if (!profile.value || !canCreateTemplate.value) return
  const session = currentProfileScope(profile.value.id)
  if (!session) return
  const request = {
    templateKey: templateDraft.templateKey.trim(),
    documentType: templateDraft.documentType,
    format: templateDraft.format,
    mappingRules: templateDraft.mappingRules.trim(),
    isDefault: templateDraft.isDefault,
    hasExistingDefault: bindings.value.some(
      (binding) => binding.documentType === templateDraft.documentType && binding.isDefault,
    ),
  }
  templateCreateError.value = ''
  try {
    const parsed = JSON.parse(request.mappingRules) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error(t('integrations.templates.mappingObjectRequired'))
    }
  } catch (err) {
    templateCreateError.value = err instanceof Error ? err.message : String(err)
    return
  }

  creatingTemplate.value = true
  try {
    const template = await createDocumentTemplate({
      templateKey: request.templateKey,
      documentType: request.documentType,
      format: request.format,
      mappingRules: request.mappingRules,
      extraData: '',
    })
    if (!profileScopeIsCurrent(session)) return
    try {
      const binding = await bindTemplateToProfile({
        integrationProfileId: session.profileId,
        documentType: request.documentType,
        templateId: template.id,
        isDefault: request.isDefault && !request.hasExistingDefault,
      })
      if (!profileScopeIsCurrent(session)) return
      if (request.isDefault && request.hasExistingDefault) {
        await setDefaultBinding(binding.id)
        if (!profileScopeIsCurrent(session)) return
      }
    } catch (err) {
      if (!profileScopeIsCurrent(session)) return
      templateCreateError.value = t('integrations.templates.createdButBindFailed', {
        error: err instanceof Error ? err.message : String(err),
      })
      await loadDetail()
      return
    }
    feedback.success(t('integrations.templates.createdAndBound'))
    showTemplateCreator.value = false
    await loadDetail()
    if (!profileScopeIsCurrent(session)) return
    emit('changed')
  } catch (err) {
    if (!profileScopeIsCurrent(session)) return
    templateCreateError.value = err instanceof Error ? err.message : String(err)
  } finally {
    if (profileScopeIsCurrent(session)) creatingTemplate.value = false
  }
}

async function loadDetail(): Promise<void> {
  const session = currentProfileScope()
  if (!session) return
  const loadSequence = ++profileLoadSequence
  loading.value = true
  loadError.value = ''
  try {
    const [p, b, templates, carriers, observedCarriers] = await Promise.all([
      getProfile(session.profileId),
      listBindingsByProfile(session.profileId),
      listDocumentTemplates(),
      listCarrierMappings(session.profileId),
      listExternalCarriers(session.profileId),
      featurePolicy.load(),
    ])
    if (!props.show || !isProfileLoadActive(
      { ...session, loadSequence },
      props.profileId,
      profileScopeGeneration,
      profileLoadSequence,
      p.id,
    )) return
    profile.value = p
    bindings.value = b
    allTemplates.value = templates
    carrierMappings.value = carriers
    externalCarriers.value = observedCarriers
    factoryPlatformDraft.value = p.factorySupplierPlatform ?? ''
  } catch (err) {
    if (!profileScopeIsCurrent(session) || loadSequence !== profileLoadSequence) return
    profile.value = null
    bindings.value = []
    allTemplates.value = []
    carrierMappings.value = []
    externalCarriers.value = []
    loadError.value = err instanceof Error ? err.message : String(err)
  } finally {
    if (profileScopeIsCurrent(session) && loadSequence === profileLoadSequence) loading.value = false
  }
}

function close(): void {
  emit('update:show', false)
}

// ── Capability editing ──

const CAPABILITY_KEYS = [
  'supportsPartialShipment',
  'supportsApiImport',
  'supportsApiExport',
  'requiresCarrierMapping',
  'requiresExternalOrderNo',
  'allowsManualClosure',
] as const

const FACTORY_CAPABILITY_KEYS = [
  'supportsExportSupplierOrder',
  'supportsImportProductCatalog',
  'supportsImportSupplierShipment',
] as const

type CapabilityKey = (typeof CAPABILITY_KEYS)[number] | (typeof FACTORY_CAPABILITY_KEYS)[number]
const visibleCapabilityKeys = computed<readonly CapabilityKey[]>(() =>
  profile.value?.sourceSurface === 'factory' ? FACTORY_CAPABILITY_KEYS : CAPABILITY_KEYS,
)

const editingCapabilities = ref(false)
const capabilityDraft = reactive<Record<CapabilityKey, boolean>>({
  supportsPartialShipment: false,
  supportsApiImport: false,
  supportsApiExport: false,
  requiresCarrierMapping: false,
  requiresExternalOrderNo: false,
  allowsManualClosure: false,
  supportsExportSupplierOrder: false,
  supportsImportProductCatalog: false,
  supportsImportSupplierShipment: false,
})
const savingCapabilities = ref(false)

function startEditCapabilities(): void {
  if (!profile.value || !currentProfileScope(profile.value.id)) return
  for (const key of [...CAPABILITY_KEYS, ...FACTORY_CAPABILITY_KEYS]) {
    capabilityDraft[key] = profile.value[key]
  }
  editingCapabilities.value = true
}

function cancelEditCapabilities(): void {
  editingCapabilities.value = false
}

function profileUpdateBase(p: IntegrationProfile) {
  return {
    id: p.id,
    profileKey: p.profileKey,
    sourceChannel: p.sourceChannel,
    sourceSurface: p.sourceSurface,
    demandKind: p.demandKind,
    initialAllocationStrategy: p.initialAllocationStrategy,
    identityStrategy: p.identityStrategy,
    entitlementAuthorityMode: p.entitlementAuthorityMode,
    recipientInputMode: p.recipientInputMode,
    referenceStrategy: p.referenceStrategy,
    trackingSyncMode: p.trackingSyncMode,
    closurePolicy: p.closurePolicy,
    supportsPartialShipment: p.supportsPartialShipment,
    supportsApiImport: p.supportsApiImport,
    supportsApiExport: p.supportsApiExport,
    requiresCarrierMapping: p.requiresCarrierMapping,
    requiresExternalOrderNo: p.requiresExternalOrderNo,
    allowsManualClosure: p.allowsManualClosure,
    supportsExportSupplierOrder: p.supportsExportSupplierOrder ?? false,
    supportsImportProductCatalog: p.supportsImportProductCatalog ?? false,
    supportsImportSupplierShipment: p.supportsImportSupplierShipment ?? false,
    connectorKey: p.connectorKey,
    factorySupplierPlatform: p.factorySupplierPlatform ?? '',
    supportedLocales: p.supportedLocales,
    defaultLocale: p.defaultLocale,
    extraData: p.extraData,
  }
}

async function saveCapabilities(): Promise<void> {
  if (!profile.value) return
  const session = currentProfileScope(profile.value.id)
  if (!session) return
  const input = {
    ...profileUpdateBase(profile.value),
    supportsPartialShipment: capabilityDraft.supportsPartialShipment,
    supportsApiImport: capabilityDraft.supportsApiImport,
    supportsApiExport: capabilityDraft.supportsApiExport,
    requiresCarrierMapping: capabilityDraft.requiresCarrierMapping,
    requiresExternalOrderNo: capabilityDraft.requiresExternalOrderNo,
    allowsManualClosure: capabilityDraft.allowsManualClosure,
    supportsExportSupplierOrder: capabilityDraft.supportsExportSupplierOrder,
    supportsImportProductCatalog: capabilityDraft.supportsImportProductCatalog,
    supportsImportSupplierShipment: capabilityDraft.supportsImportSupplierShipment,
  }
  savingCapabilities.value = true
  try {
    await updateProfile(input)
    if (!profileScopeIsCurrent(session)) return
    feedback.success(t('feedback.success'))
    editingCapabilities.value = false
    await loadDetail()
    if (!profileScopeIsCurrent(session)) return
    emit('changed')
  } catch (err) {
    if (!profileScopeIsCurrent(session)) return
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    if (profileScopeIsCurrent(session)) savingCapabilities.value = false
  }
}

// ── Factory supplier platform ──

const factoryPlatformDraft = ref('')
const savingFactoryPlatform = ref(false)

async function saveFactoryPlatform(): Promise<void> {
  if (!profile.value) return
  const session = currentProfileScope(profile.value.id)
  if (!session) return
  const input = {
    ...profileUpdateBase(profile.value),
    factorySupplierPlatform: factoryPlatformDraft.value.trim(),
  }
  savingFactoryPlatform.value = true
  try {
    await updateProfile(input)
    if (!profileScopeIsCurrent(session)) return
    feedback.success(t('feedback.success'))
    await loadDetail()
    if (!profileScopeIsCurrent(session)) return
    emit('changed')
  } catch (err) {
    if (!profileScopeIsCurrent(session)) return
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    if (profileScopeIsCurrent(session)) savingFactoryPlatform.value = false
  }
}

// ── Carrier mappings ──

async function handleImportCarrierMappings(): Promise<void> {
  if (!profile.value || !carrierWritesEnabled.value) return
  const session = currentProfileScope(profile.value.id)
  if (!session) return
  importingCarriers.value = true
  try {
    const path = await pickTabularFile()
    if (!path) return
    if (!profileScopeIsCurrent(session)) return
    const result = await importCarrierMappings({
      integrationProfileId: session.profileId,
      importMode: 'skip_invalid',
      filePath: path,
    })
    if (!profileScopeIsCurrent(session)) return
    lastCarrierImportEvidence.value = {
      importRunId: result.importRunId,
      evidenceDisabled: result.evidenceDisabled,
    }
    if (result.errorCount > 0) {
      feedback.error(
        t('feedback.error'),
        t('integrations.carrierMappings.importPartial', {
          success: result.successCount,
          errors: result.errorCount,
        }),
      )
    } else {
      feedback.success(t('feedback.success'))
    }
    if (result.warnings && result.warnings.length > 0) {
      feedback.info(
        t('integrations.carrierMappings.importWarnings', {
          count: result.warnings.length,
          items: t('integrations.carrierMappings.warningDetailsWithheld'),
        }),
      )
    }
    await loadDetail()
    if (!profileScopeIsCurrent(session)) return
    emit('changed')
  } catch (err) {
    if (!profileScopeIsCurrent(session)) return
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    if (profileScopeIsCurrent(session)) importingCarriers.value = false
  }
}

async function handleDeleteCarrierMapping(mapping: dto.CarrierMappingDTO): Promise<void> {
  if (!carrierWritesEnabled.value) return
  const session = currentProfileScope()
  if (!session || !profileEntityIsCurrent(session, mapping.integrationProfileId)) return
  deletingCarrierId.value = mapping.id
  try {
    await deleteCarrierMapping(mapping.id)
    if (!profileScopeIsCurrent(session)) return
    feedback.success(t('feedback.success'))
    await loadDetail()
    if (!profileScopeIsCurrent(session)) return
    emit('changed')
  } catch (err) {
    if (!profileScopeIsCurrent(session)) return
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    if (profileScopeIsCurrent(session)) deletingCarrierId.value = null
  }
}

function openCarrierBinding(carrier: dto.ExternalCarrierDTO): void {
  if (!carrierWritesEnabled.value) return
  const session = currentProfileScope()
  if (!session || !profileEntityIsCurrent(session, carrier.integrationProfileId)) return
  bindingExternalCarrier.value = carrier
  internalCarrierCodeDraft.value = carrier.internalCarrierCode ?? ''
}

function closeCarrierBinding(): void {
  if (bindingCarrier.value) return
  bindingExternalCarrier.value = null
  internalCarrierCodeDraft.value = ''
}

async function saveCarrierBinding(): Promise<void> {
  const carrier = bindingExternalCarrier.value
  const session = currentProfileScope()
  const internalCarrierCode = internalCarrierCodeDraft.value.trim()
  if (
    !carrier
    || !session
    || !internalCarrierCode
    || bindingCarrier.value
    || !carrierWritesEnabled.value
    || !profileEntityIsCurrent(session, carrier.integrationProfileId)
  ) return
  bindingCarrier.value = true
  try {
    await bindInternalCarrier({
      externalCarrierId: carrier.id,
      internalCarrierCode,
    })
    if (!profileScopeIsCurrent(session)) return
    bindingExternalCarrier.value = null
    internalCarrierCodeDraft.value = ''
    await loadDetail()
    if (!profileScopeIsCurrent(session)) return
  } catch (err) {
    if (!profileScopeIsCurrent(session)) return
    feedback.error(t('integrations.carrierRegistry.bindFailed'), err instanceof Error ? err.message : String(err))
  } finally {
    if (profileScopeIsCurrent(session)) bindingCarrier.value = false
  }
}

// ── Bindings management ──

const bindingsExpanded = ref(false)
const settingDefaultId = ref<number | null>(null)
const unbindingId = ref<number | null>(null)

function templateKeyFor(templateId: number): string {
  return allTemplates.value.find((tmpl) => tmpl.id === templateId)?.templateKey ?? String(templateId)
}

async function handleSetDefault(binding: dto.ProfileTemplateBindingDTO): Promise<void> {
  const session = currentProfileScope()
  if (!session || !profileEntityIsCurrent(session, binding.integrationProfileId)) return
  settingDefaultId.value = binding.id
  try {
    await setDefaultBinding(binding.id)
    if (!profileScopeIsCurrent(session)) return
    feedback.success(t('feedback.success'))
    await loadDetail()
    if (!profileScopeIsCurrent(session)) return
    emit('changed')
  } catch (err) {
    if (!profileScopeIsCurrent(session)) return
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    if (profileScopeIsCurrent(session)) settingDefaultId.value = null
  }
}

async function handleUnbind(binding: dto.ProfileTemplateBindingDTO): Promise<void> {
  const session = currentProfileScope()
  if (!session || !profileEntityIsCurrent(session, binding.integrationProfileId)) return
  unbindingId.value = binding.id
  try {
    await unbindTemplate(binding.id)
    if (!profileScopeIsCurrent(session)) return
    feedback.success(t('feedback.success'))
    await loadDetail()
    if (!profileScopeIsCurrent(session)) return
    emit('changed')
  } catch (err) {
    if (!profileScopeIsCurrent(session)) return
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    if (profileScopeIsCurrent(session)) unbindingId.value = null
  }
}

// ── Re-run wizard (remap mode) ──

const showRerunWizard = ref(false)
const rerunWizardSession = ref<ProfileScope | null>(null)

function openRerunWizard(): void {
  if (!profile.value) return
  const session = currentProfileScope(profile.value.id)
  if (!session) return
  rerunWizardSession.value = session
  showRerunWizard.value = true
}

function handleRerunDone(): void {
  const session = rerunWizardSession.value
  if (!session || !profileScopeIsCurrent(session)) return
  showRerunWizard.value = false
  rerunWizardSession.value = null
  void loadDetail()
  emit('changed')
}

function handleRerunCancel(): void {
  showRerunWizard.value = false
  rerunWizardSession.value = null
}

// ── Expert mode ──

const expertModeOpen = ref(false)
const connectorKeys = ref<string[]>([])
const expertConnectorKey = ref('')
const expertExtraDataRaw = ref('')
const expertJsonError = ref('')
const savingExpert = ref(false)

const connectorOptions = computed<SelectOption[]>(() => connectorKeys.value.map((key) => ({ label: key, value: key })))

async function openExpertMode(): Promise<void> {
  if (!profile.value) return
  const session = currentProfileScope(profile.value.id)
  if (!session) return
  expertModeOpen.value = true
  expertConnectorKey.value = profile.value.connectorKey
  expertExtraDataRaw.value = profile.value.extraData || '{}'
  expertJsonError.value = ''
  try {
    const caps = await listConnectorCapabilities()
    if (!profileScopeIsCurrent(session)) return
    connectorKeys.value = Object.keys(caps)
  } catch {
    if (!profileScopeIsCurrent(session)) return
    connectorKeys.value = []
  }
}

function validateExpertJson(): boolean {
  try {
    JSON.parse(expertExtraDataRaw.value || '{}')
    expertJsonError.value = ''
    return true
  } catch {
    expertJsonError.value = t('integrations.expertMode.invalidJson')
    return false
  }
}

async function saveExpertMode(): Promise<void> {
  if (!profile.value || !validateExpertJson()) return
  const session = currentProfileScope(profile.value.id)
  if (!session) return
  const input = {
    ...profileUpdateBase(profile.value),
    connectorKey: expertConnectorKey.value,
    extraData: expertExtraDataRaw.value,
  }
  savingExpert.value = true
  try {
    await updateProfile(input)
    if (!profileScopeIsCurrent(session)) return
    feedback.success(t('feedback.success'))
    await loadDetail()
    if (!profileScopeIsCurrent(session)) return
    emit('changed')
  } catch (err) {
    if (!profileScopeIsCurrent(session)) return
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    if (profileScopeIsCurrent(session)) savingExpert.value = false
  }
}

function invalidateProfileScopedState(): void {
  profile.value = null
  bindings.value = []
  allTemplates.value = []
  carrierMappings.value = []
  externalCarriers.value = []
  loadError.value = ''
  loading.value = false

  lastCarrierImportEvidence.value = null
  bindingExternalCarrier.value = null
  internalCarrierCodeDraft.value = ''
  bindingCarrier.value = false
  importingCarriers.value = false
  deletingCarrierId.value = null

  showTemplateCreator.value = false
  creatingTemplate.value = false
  templateCreateError.value = ''
  templateDraft.templateKey = ''
  templateDraft.documentType = ''
  templateDraft.format = ''
  templateDraft.mappingRules = ''
  templateDraft.isDefault = true

  editingCapabilities.value = false
  savingCapabilities.value = false
  for (const key of [...CAPABILITY_KEYS, ...FACTORY_CAPABILITY_KEYS]) capabilityDraft[key] = false
  factoryPlatformDraft.value = ''
  savingFactoryPlatform.value = false

  bindingsExpanded.value = false
  settingDefaultId.value = null
  unbindingId.value = null

  showRerunWizard.value = false
  rerunWizardSession.value = null

  expertModeOpen.value = false
  connectorKeys.value = []
  expertConnectorKey.value = ''
  expertExtraDataRaw.value = ''
  expertJsonError.value = ''
  savingExpert.value = false
}

watch(
  () => [props.show, props.profileId] as const,
  ([visible, id], previous) => {
    const previousId = previous?.[1] ?? null
    if (id !== previousId || !visible) {
      profileScopeGeneration += 1
      profileLoadSequence += 1
      invalidateProfileScopedState()
    }
    if (visible && id) void loadDetail()
  },
  { immediate: true, flush: 'sync' },
)
</script>

<template>
  <DetailDrawer :show="show" :title="t('integrations.detail.title')" size="lg" @update:show="(v) => emit('update:show', v)">
    <template #title>{{ profile?.profileKey ?? t('integrations.detail.title') }}</template>

    <NSpin :show="loading">
      <ErrorBanner
        v-if="loadError"
        :message="t('integrations.detail.loadError')"
        :detail="loadError"
        @retry="loadDetail"
      />
      <template v-else-if="profile">
        <SectionCard :title="t('integrations.detail.sections.connector')" flat>
          <template #actions>
            <NButton size="small" @click="openRerunWizard">{{ t('integrations.actions.rerunWizard') }}</NButton>
          </template>
          <dl class="integration-detail__kv">
            <dt>{{ t('integrations.detail.fields.profileKey') }}</dt>
            <dd>{{ profile.profileKey }}</dd>
            <dt>{{ t('integrations.detail.fields.sourceChannel') }}</dt>
            <dd>{{ profile.sourceChannel || '—' }}</dd>
            <dt>{{ t('integrations.detail.fields.sourceSurface') }}</dt>
            <dd>{{ profile.sourceSurface || '—' }}</dd>
            <template v-if="profile.sourceSurface !== 'factory'">
              <dt>{{ t('integrations.detail.fields.demandKind') }}</dt>
              <dd><StatusBadge dimension="demandKind" :value="profile.demandKind" size="sm" /></dd>
            </template>
            <template v-if="profile.sourceSurface !== 'factory'">
              <dt>{{ t('statusKit.dimensionNames.trackingSyncMode') }}</dt>
              <dd><StatusBadge dimension="trackingSyncMode" :value="profile.trackingSyncMode" size="sm" /></dd>
              <dt>{{ t('statusKit.dimensionNames.closurePolicy') }}</dt>
              <dd><StatusBadge dimension="closurePolicy" :value="profile.closurePolicy" size="sm" /></dd>
            </template>
            <dt>{{ t('integrations.detail.fields.connectorKey') }}</dt>
            <dd>{{ profile.connectorKey || '—' }}</dd>
            <template v-if="profile.sourceSurface === 'factory'">
              <dt>{{ t('integrations.detail.fields.factorySupplierPlatform') }}</dt>
              <dd class="integration-detail__factory-platform">
                <NInput
                  v-model:value="factoryPlatformDraft"
                  size="small"
                  :placeholder="t('integrations.detail.fields.factorySupplierPlatformPlaceholder')"
                  style="max-width: 220px"
                />
                <NButton size="tiny" type="primary" :loading="savingFactoryPlatform" @click="saveFactoryPlatform">
                  {{ t('common.save') }}
                </NButton>
              </dd>
            </template>
          </dl>
        </SectionCard>

        <SectionCard :title="t('integrations.detail.sections.capabilities')" flat>
          <template #actions>
            <NButton v-if="!editingCapabilities" size="small" @click="startEditCapabilities">
              {{ t('integrations.actions.editCapabilities') }}
            </NButton>
            <template v-else>
              <NButton size="small" :disabled="savingCapabilities" @click="cancelEditCapabilities">{{ t('common.cancel') }}</NButton>
              <NButton size="small" type="primary" :loading="savingCapabilities" @click="saveCapabilities">{{ t('common.save') }}</NButton>
            </template>
          </template>
          <dl class="integration-detail__kv">
            <template v-for="key in visibleCapabilityKeys" :key="key">
              <dt>{{ t(`intakeWizard.capabilities.${key}.label`) }}</dt>
              <dd>
                <NSwitch
                  v-if="editingCapabilities"
                  v-model:value="capabilityDraft[key]"
                  :disabled="savingCapabilities"
                />
                <span v-else>{{ profile[key] ? t('common.yes') : t('common.no') }}</span>
              </dd>
            </template>
          </dl>
        </SectionCard>

        <SectionCard :title="t('integrations.detail.sections.templates')" flat>
          <template #actions>
            <NButton size="small" @click="openTemplateCreator">
              {{ t('integrations.templates.createAndBind') }}
            </NButton>
          </template>
          <EmptyState v-if="!boundTemplates.length" size="sm" :title="t('integrations.detail.noTemplates')" />
          <dl v-else class="integration-detail__kv">
            <template v-for="tmpl in boundTemplates" :key="tmpl.id">
              <dt>{{ t('integrations.detail.fields.templateKey') }}</dt>
              <dd>{{ tmpl.templateKey }} — <StatusBadge dimension="documentType" :value="tmpl.documentType" size="sm" /></dd>
            </template>
          </dl>
        </SectionCard>

        <SectionCard :title="t('integrations.detail.sections.carrierMappings')" flat>
          <template #actions>
            <NButton size="small" :loading="importingCarriers" :disabled="!carrierWritesEnabled" @click="handleImportCarrierMappings">
              {{ t('integrations.carrierMappings.import') }}
            </NButton>
          </template>
          <p v-if="!carrierWritesEnabled" class="integration-detail__carrier-policy-note">
            {{ t('integrations.carrierRegistry.disabledReason') }}
          </p>
          <p v-if="lastCarrierImportEvidence" class="integration-detail__carrier-policy-note">
            {{ lastCarrierImportEvidence.evidenceDisabled
              ? t('integrations.carrierMappings.evidenceDisabled')
              : t('integrations.carrierMappings.importRun', { id: lastCarrierImportEvidence.importRunId }) }}
          </p>
          <h4 class="integration-detail__carrier-subtitle">{{ t('integrations.carrierRegistry.observedTitle') }}</h4>
          <p class="integration-detail__carrier-policy-note">{{ t('integrations.carrierRegistry.observedHint') }}</p>
          <EmptyState v-if="!externalCarriers.length" size="sm" :title="t('integrations.carrierRegistry.empty')" />
          <div v-else class="integration-detail__carriers">
            <div v-for="carrier in externalCarriers" :key="carrier.id" class="integration-detail__carrier-row">
              <div class="integration-detail__carrier-info">
                <span class="integration-detail__carrier-code">{{ carrier.externalCarrierCode || '—' }}</span>
                <span>{{ carrier.externalCarrierName || '—' }}</span>
                <code>{{ carrier.status }}</code>
              </div>
              <div class="integration-detail__carrier-aliases">
                <span v-if="carrier.internalCarrierCode">{{ carrier.internalCarrierCode }}</span>
                <span v-else>{{ t('integrations.carrierRegistry.unbound') }}</span>
                <span v-if="carrier.conflictReason" class="integration-detail__carrier-conflict">{{ carrierConflictCopy(carrier) }}</span>
              </div>
              <NButton size="tiny" :disabled="!carrierWritesEnabled" @click="openCarrierBinding(carrier)">
                {{ t('integrations.carrierRegistry.bindAction') }}
              </NButton>
            </div>
          </div>

          <h4 class="integration-detail__carrier-subtitle">{{ t('integrations.carrierRegistry.mappingsTitle') }}</h4>
          <p class="integration-detail__carrier-policy-note">{{ t('integrations.carrierRegistry.mappingsHint') }}</p>
          <EmptyState v-if="!carrierMappings.length" size="sm" :title="t('integrations.carrierMappings.empty')" />
          <div v-else class="integration-detail__carriers">
            <div v-for="mapping in carrierMappings" :key="mapping.id" class="integration-detail__carrier-row">
              <div class="integration-detail__carrier-info">
                <span class="integration-detail__carrier-code">{{ mapping.internalCarrierCode }}</span>
                <span class="integration-detail__carrier-arrow">→</span>
                <span>{{ mapping.externalCarrierCode }}</span>
                <span v-if="mapping.externalCarrierName" class="integration-detail__carrier-name">
                  ({{ mapping.externalCarrierName }})
                </span>
                <span v-if="mapping.isDefault" class="integration-detail__default-tag">
                  {{ t('integrations.detail.fields.isDefault') }}
                </span>
              </div>
              <div class="integration-detail__carrier-aliases">
                <span class="integration-detail__carrier-aliases-label">
                  {{ t('integrations.carrierMappings.aliases') }}:
                </span>
                {{ formatAliases(mapping.aliases) }}
              </div>
              <NButton
                size="tiny"
                :loading="deletingCarrierId === mapping.id"
                :disabled="!carrierWritesEnabled"
                @click="handleDeleteCarrierMapping(mapping)"
              >
                {{ t('common.delete') }}
              </NButton>
            </div>
          </div>
        </SectionCard>

        <SectionCard :title="t('integrations.detail.sections.bindings')" flat>
          <template #actions>
            <NButton size="small" @click="bindingsExpanded = !bindingsExpanded">{{ t('integrations.actions.manageBindings') }}</NButton>
          </template>
          <EmptyState v-if="!bindings.length" size="sm" :title="t('integrations.detail.noBindings')" />
          <div v-else class="integration-detail__bindings">
            <div v-for="binding in bindings" :key="binding.id" class="integration-detail__binding-row">
              <div class="integration-detail__binding-info">
                <StatusBadge dimension="documentType" :value="binding.documentType" size="sm" />
                <span class="integration-detail__binding-template">{{ templateKeyFor(binding.templateId) }}</span>
                <span v-if="binding.isDefault" class="integration-detail__default-tag">{{ t('integrations.detail.fields.isDefault') }}</span>
              </div>
              <div v-if="bindingsExpanded" class="integration-detail__binding-actions">
                <NButton
                  v-if="!binding.isDefault"
                  size="tiny"
                  :loading="settingDefaultId === binding.id"
                  :disabled="unbindingId === binding.id"
                  @click="handleSetDefault(binding)"
                >
                  {{ t('integrations.actions.setDefault') }}
                </NButton>
                <NButton
                  size="tiny"
                  :loading="unbindingId === binding.id"
                  :disabled="settingDefaultId === binding.id"
                  @click="handleUnbind(binding)"
                >
                  {{ t('integrations.actions.unbind') }}
                </NButton>
              </div>
            </div>
          </div>
        </SectionCard>

        <SectionCard flat>
          <template #title>
            <button type="button" class="integration-detail__expert-toggle" @click="openExpertMode">
              {{ t('integrations.expertMode.title') }}
            </button>
          </template>
          <div v-if="expertModeOpen" class="integration-detail__expert-body">
            <label class="integration-detail__expert-label">{{ t('integrations.expertMode.connectorKeyLabel') }}</label>
            <NSelect v-model:value="expertConnectorKey" :options="connectorOptions" clearable filterable tag />

            <label class="integration-detail__expert-label">{{ t('integrations.expertMode.rawJsonLabel') }}</label>
            <NInput v-model:value="expertExtraDataRaw" type="textarea" :autosize="{ minRows: 4, maxRows: 10 }" />
            <p v-if="expertJsonError" class="integration-detail__expert-error">{{ expertJsonError }}</p>

            <div class="integration-detail__expert-actions">
              <NButton size="small" @click="validateExpertJson">{{ t('integrations.expertMode.validate') }}</NButton>
              <NButton size="small" type="primary" :loading="savingExpert" @click="saveExpertMode">
                {{ t('integrations.expertMode.save') }}
              </NButton>
            </div>
          </div>
        </SectionCard>
      </template>
      <EmptyState v-else-if="!loading" size="sm" :title="t('integrations.detail.title')" />
    </NSpin>

    <NModal
      :show="bindingExternalCarrier != null"
      preset="card"
      :title="t('integrations.carrierRegistry.bindTitle')"
      :style="{ width: 'min(520px, 94vw)' }"
      @update:show="(v: boolean) => { if (!v) closeCarrierBinding() }"
    >
      <div class="integration-detail__template-form">
        <p v-if="bindingExternalCarrier" class="integration-detail__carrier-policy-note">
          {{ bindingExternalCarrier.externalCarrierCode || '—' }} · {{ bindingExternalCarrier.externalCarrierName || '—' }}
        </p>
        <label class="integration-detail__expert-label">{{ t('integrations.carrierRegistry.internalCode') }}</label>
        <NInput v-model:value="internalCarrierCodeDraft" :placeholder="t('integrations.carrierRegistry.internalCodePlaceholder')" />
        <div class="integration-detail__expert-actions">
          <NButton :disabled="bindingCarrier" @click="closeCarrierBinding">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="bindingCarrier" :disabled="!internalCarrierCodeDraft.trim() || !carrierWritesEnabled" @click="saveCarrierBinding">
            {{ t('integrations.carrierRegistry.bindAction') }}
          </NButton>
        </div>
      </div>
    </NModal>

    <NModal
      :show="showRerunWizard"
      preset="card"
      :title="t('integrations.actions.rerunWizard')"
      :style="{ width: 'min(760px, 94vw)' }"
      :mask-closable="false"
      @update:show="(v: boolean) => { if (!v) handleRerunCancel() }"
    >
      <IntakeWizard v-if="showRerunWizard && profile" :existing-profile="profile" @done="handleRerunDone" @cancel="handleRerunCancel" />
    </NModal>

    <NModal
      :show="showTemplateCreator"
      preset="card"
      :title="t('integrations.templates.createAndBind')"
      :style="{ width: 'min(720px, 94vw)' }"
      :mask-closable="false"
      @update:show="(v: boolean) => (showTemplateCreator = v)"
    >
      <div class="integration-detail__template-form">
        <label class="integration-detail__expert-label">{{ t('integrations.detail.fields.templateKey') }}</label>
        <NInput v-model:value="templateDraft.templateKey" />

        <label class="integration-detail__expert-label">{{ t('integrations.detail.fields.documentType') }}</label>
        <NSelect v-model:value="templateDraft.documentType" :options="documentTypeOptions" filterable />

        <label class="integration-detail__expert-label">{{ t('integrations.templates.format') }}</label>
        <NSelect v-model:value="templateDraft.format" :options="templateFormatOptions" />

        <label class="integration-detail__expert-label">{{ t('integrations.templates.mappingRules') }}</label>
        <NInput
          v-model:value="templateDraft.mappingRules"
          type="textarea"
          :autosize="{ minRows: 8, maxRows: 18 }"
          :placeholder="t('integrations.templates.mappingPlaceholder')"
        />

        <label class="integration-detail__template-default">
          <NSwitch v-model:value="templateDraft.isDefault" />
          <span>{{ t('integrations.templates.setAsDefault') }}</span>
        </label>

        <ErrorBanner
          v-if="templateCreateError"
          :message="t('integrations.templates.createFailed')"
          :detail="templateCreateError"
        />

        <div class="integration-detail__expert-actions">
          <NButton :disabled="creatingTemplate" @click="showTemplateCreator = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :disabled="!canCreateTemplate"
            :loading="creatingTemplate"
            @click="createAndBindTemplate"
          >
            {{ t('integrations.templates.createAndBind') }}
          </NButton>
        </div>
      </div>
    </NModal>
  </DetailDrawer>
</template>

<style scoped>
.integration-detail__kv {
  display: grid;
  grid-template-columns: minmax(140px, 1fr) minmax(160px, 1.6fr);
  gap: var(--space-2) var(--space-4);
  margin: 0;
}

.integration-detail__kv dt {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.integration-detail__kv dd {
  margin: 0;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.integration-detail__bindings {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.integration-detail__binding-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--card-border-color);
}

.integration-detail__binding-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.integration-detail__binding-template {
  color: var(--color-text-secondary);
}

.integration-detail__default-tag {
  font-size: var(--font-size-xs);
  color: var(--status-success-fg);
  background: var(--status-success-bg);
  border-radius: var(--radius-sm);
  padding: 0 var(--space-1);
}

.integration-detail__binding-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.integration-detail__expert-toggle {
  border: none;
  background: none;
  padding: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  cursor: pointer;
}

.integration-detail__expert-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.integration-detail__expert-label {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.integration-detail__expert-error {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--status-error-fg);
}

.integration-detail__expert-actions {
  display: flex;
  gap: var(--space-2);
}

.integration-detail__template-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.integration-detail__template-default {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.integration-detail__factory-platform {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.integration-detail__carriers {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.integration-detail__carrier-row {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--card-border-color);
}

.integration-detail__carrier-info {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-1);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.integration-detail__carrier-code {
  font-weight: var(--font-weight-semibold);
}

.integration-detail__carrier-arrow {
  color: var(--color-text-muted);
}

.integration-detail__carrier-name {
  color: var(--color-text-secondary);
}

.integration-detail__carrier-aliases {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.integration-detail__carrier-aliases-label {
  color: var(--color-text-muted);
}

.integration-detail__carrier-subtitle {
  margin: var(--space-4) 0 var(--space-1);
  color: var(--color-text-primary);
  font-size: var(--font-size-sm);
}

.integration-detail__carrier-policy-note {
  margin: var(--space-1) 0;
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}

.integration-detail__carrier-conflict {
  display: block;
  color: var(--status-warning-fg);
}
</style>
