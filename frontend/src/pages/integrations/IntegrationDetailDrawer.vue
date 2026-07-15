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
  deleteCarrierMapping,
  importCarrierMappings,
  pickTabularFile,
} from '@/shared/api/bridge'
import type { IntegrationProfile } from '@/entities/profile'
import type { dto } from '../../../wailsjs/go/models'
import IntakeWizard from './wizard/IntakeWizard.vue'

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

const loading = ref(false)
const loadError = ref('')
const profile = ref<IntegrationProfile | null>(null)
const bindings = ref<dto.ProfileTemplateBindingDTO[]>([])
const allTemplates = ref<dto.DocumentTemplateDTO[]>([])
const carrierMappings = ref<dto.CarrierMappingDTO[]>([])
const importingCarriers = ref(false)
const deletingCarrierId = ref<number | null>(null)

const boundTemplates = computed(() => {
  const templateIds = new Set(bindings.value.map((b) => b.templateId))
  return allTemplates.value.filter((tmpl) => templateIds.has(tmpl.id))
})

async function loadDetail(): Promise<void> {
  if (!props.profileId) return
  loading.value = true
  loadError.value = ''
  try {
    const [p, b, templates, carriers] = await Promise.all([
      getProfile(props.profileId),
      listBindingsByProfile(props.profileId),
      listDocumentTemplates(),
      listCarrierMappings(props.profileId),
    ])
    profile.value = p
    bindings.value = b
    allTemplates.value = templates
    carrierMappings.value = carriers
    factoryPlatformDraft.value = p.factorySupplierPlatform ?? ''
  } catch (err) {
    profile.value = null
    bindings.value = []
    allTemplates.value = []
    carrierMappings.value = []
    loadError.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.show, props.profileId] as const,
  ([visible, id]) => {
    if (visible && id) void loadDetail()
  },
  { immediate: true },
)

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

const editingCapabilities = ref(false)
const capabilityDraft = reactive<Record<(typeof CAPABILITY_KEYS)[number], boolean>>({
  supportsPartialShipment: false,
  supportsApiImport: false,
  supportsApiExport: false,
  requiresCarrierMapping: false,
  requiresExternalOrderNo: false,
  allowsManualClosure: false,
})
const savingCapabilities = ref(false)

function startEditCapabilities(): void {
  if (!profile.value) return
  for (const key of CAPABILITY_KEYS) capabilityDraft[key] = profile.value[key]
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
  savingCapabilities.value = true
  try {
    await updateProfile({
      ...profileUpdateBase(profile.value),
      supportsPartialShipment: capabilityDraft.supportsPartialShipment,
      supportsApiImport: capabilityDraft.supportsApiImport,
      supportsApiExport: capabilityDraft.supportsApiExport,
      requiresCarrierMapping: capabilityDraft.requiresCarrierMapping,
      requiresExternalOrderNo: capabilityDraft.requiresExternalOrderNo,
      allowsManualClosure: capabilityDraft.allowsManualClosure,
    })
    feedback.success(t('feedback.success'))
    editingCapabilities.value = false
    await loadDetail()
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    savingCapabilities.value = false
  }
}

// ── Factory supplier platform ──

const factoryPlatformDraft = ref('')
const savingFactoryPlatform = ref(false)

async function saveFactoryPlatform(): Promise<void> {
  if (!profile.value) return
  savingFactoryPlatform.value = true
  try {
    await updateProfile({
      ...profileUpdateBase(profile.value),
      factorySupplierPlatform: factoryPlatformDraft.value.trim(),
    })
    feedback.success(t('feedback.success'))
    await loadDetail()
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    savingFactoryPlatform.value = false
  }
}

// ── Carrier mappings ──

async function handleImportCarrierMappings(): Promise<void> {
  if (!profile.value) return
  importingCarriers.value = true
  try {
    const path = await pickTabularFile()
    if (!path) return
    const result = await importCarrierMappings({
      integrationProfileId: profile.value.id,
      importMode: 'skip_invalid',
      filePath: path,
    })
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
          items: result.warnings.join('; '),
        }),
      )
    }
    await loadDetail()
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    importingCarriers.value = false
  }
}

async function handleDeleteCarrierMapping(id: number): Promise<void> {
  deletingCarrierId.value = id
  try {
    await deleteCarrierMapping(id)
    feedback.success(t('feedback.success'))
    await loadDetail()
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    deletingCarrierId.value = null
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
  settingDefaultId.value = binding.id
  try {
    await setDefaultBinding(binding.id)
    feedback.success(t('feedback.success'))
    await loadDetail()
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    settingDefaultId.value = null
  }
}

async function handleUnbind(binding: dto.ProfileTemplateBindingDTO): Promise<void> {
  unbindingId.value = binding.id
  try {
    await unbindTemplate(binding.id)
    feedback.success(t('feedback.success'))
    await loadDetail()
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    unbindingId.value = null
  }
}

// ── Re-run wizard (remap mode) ──

const showRerunWizard = ref(false)

function openRerunWizard(): void {
  showRerunWizard.value = true
}

function handleRerunDone(): void {
  showRerunWizard.value = false
  void loadDetail()
  emit('changed')
}

function handleRerunCancel(): void {
  showRerunWizard.value = false
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
  expertModeOpen.value = true
  expertConnectorKey.value = profile.value.connectorKey
  expertExtraDataRaw.value = profile.value.extraData || '{}'
  expertJsonError.value = ''
  try {
    const caps = await listConnectorCapabilities()
    connectorKeys.value = Object.keys(caps)
  } catch {
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
  savingExpert.value = true
  try {
    await updateProfile({
      ...profileUpdateBase(profile.value),
      connectorKey: expertConnectorKey.value,
      extraData: expertExtraDataRaw.value,
    })
    feedback.success(t('feedback.success'))
    await loadDetail()
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    savingExpert.value = false
  }
}
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
            <dt>{{ t('integrations.detail.fields.demandKind') }}</dt>
            <dd><StatusBadge dimension="demandKind" :value="profile.demandKind" size="sm" /></dd>
            <dt>{{ t('statusKit.dimensionNames.trackingSyncMode') }}</dt>
            <dd><StatusBadge dimension="trackingSyncMode" :value="profile.trackingSyncMode" size="sm" /></dd>
            <dt>{{ t('statusKit.dimensionNames.closurePolicy') }}</dt>
            <dd><StatusBadge dimension="closurePolicy" :value="profile.closurePolicy" size="sm" /></dd>
            <dt>{{ t('integrations.detail.fields.connectorKey') }}</dt>
            <dd>{{ profile.connectorKey || '—' }}</dd>
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
            <template v-for="key in CAPABILITY_KEYS" :key="key">
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
            <NButton size="small" :loading="importingCarriers" @click="handleImportCarrierMappings">
              {{ t('integrations.carrierMappings.import') }}
            </NButton>
          </template>
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
                @click="handleDeleteCarrierMapping(mapping.id)"
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
      :show="showRerunWizard"
      preset="card"
      :title="t('integrations.actions.rerunWizard')"
      :style="{ width: 'min(760px, 94vw)' }"
      :mask-closable="false"
      @update:show="(v: boolean) => (showRerunWizard = v)"
    >
      <IntakeWizard v-if="showRerunWizard && profile" :existing-profile="profile" @done="handleRerunDone" @cancel="handleRerunCancel" />
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
</style>
