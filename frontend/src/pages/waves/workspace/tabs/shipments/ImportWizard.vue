<script setup lang="ts">
/**
 * ImportWizard — factory-return import on the 发货回传 tab.
 *
 * The operator picks a factory platform (required). File kind is implied
 * `import_supplier_shipment`; import uses that platform's default binding.
 * There is no client FieldMappingEditor / reconciliation-key path and this
 * entry does not send mappingRules. Submit is always
 * `mapAndReconcileShipments({ waveId, integrationProfileId, importMode, filePath })`.
 *
 * Missing default binding blocks Next/Finish — the operator is not offered a
 * DIY column-mapping table. The sticky `ImportResultView` is only cleared
 * when they start a new import.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NSelect } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { WizardFrame, type WizardStep } from '@/shared/ui/wizard'
import { CalloutBar } from '@/shared/ui/guidance'
import { StatusBadge } from '@/shared/ui/status'
import { useFeedback } from '@/shared/ui/feedback'
import { pickTabularFile, parseTabularFile, mapAndReconcileShipments, listProfiles } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import ImportResultView from './ImportResultView.vue'
import type { ImportResultViewData } from './ImportResultView.vue'
import { canImportSupplierShipment } from '@/pages/integrations/profileAvailability'
import {
  SUPPLIER_SHIPMENT_IMPORT_DOCUMENT_TYPE,
  loadShipmentImportDefaultBinding,
  type ShipmentImportBindingState,
} from './shipmentImportDefaults'
import type { dto } from '@/../wailsjs/go/models'

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const ctx = useWaveWorkspaceContext()

const emit = defineEmits<{ imported: [] }>()

type StepKey = 'upload' | 'preview'
const currentStep = ref<StepKey>('upload')
const wizardSteps = computed<WizardStep[]>(() => [
  { key: 'upload', title: t('waveWorkspace.shipments.import.steps.upload') },
  { key: 'preview', title: t('waveWorkspace.shipments.import.steps.preview') },
])

const picking = ref(false)
const csvPreview = ref<dto.CSVFilePreviewDTO | null>(null)
const tabularFilePath = ref('')
const templateProfileId = ref<number | null>(null)
const profileOptions = ref<SelectOption[]>([])
const profilesError = ref('')
const defaultBinding = ref<ShipmentImportBindingState>({ status: 'idle' })
let bindingRequestSeq = 0

const importMode = ref<'skip_invalid' | 'reject_all'>('skip_invalid')
const submitting = ref(false)
const importResult = ref<ImportResultViewData | null>(null)

const hasLoadedBinding = computed(() => defaultBinding.value.status === 'loaded')

const importModeOptions = computed<SelectOption[]>(() => [
  { label: t('waveWorkspace.shipments.import.importModeOptions.skip_invalid'), value: 'skip_invalid' },
  { label: t('waveWorkspace.shipments.import.importModeOptions.reject_all'), value: 'reject_all' },
])

async function loadProfiles(): Promise<void> {
  profilesError.value = ''
  try {
    const profiles = await listProfiles()
    profileOptions.value = profiles
      .filter(canImportSupplierShipment)
      .map((p) => ({
        label: `${p.profileKey} (${p.sourceChannel})`,
        value: p.id,
      }))
  } catch (err) {
    profileOptions.value = []
    profilesError.value = err instanceof Error ? err.message : String(err)
  }
}
void loadProfiles()

watch(templateProfileId, (id) => {
  bindingRequestSeq += 1
  const seq = bindingRequestSeq
  if (id == null) {
    defaultBinding.value = { status: 'idle' }
    return
  }
  defaultBinding.value = { status: 'loading' }
  void loadShipmentImportDefaultBinding(id).then((result) => {
    if (seq !== bindingRequestSeq) return
    defaultBinding.value = result
  })
})

async function handlePickFile(): Promise<void> {
  if (!hasLoadedBinding.value) return
  picking.value = true
  try {
    const path = await pickTabularFile()
    if (!path) return
    tabularFilePath.value = path
    csvPreview.value = await parseTabularFile(path)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    picking.value = false
  }
}

const canProceedFromUpload = computed(() =>
  templateProfileId.value != null
  && hasLoadedBinding.value
  && !!tabularFilePath.value,
)

const canSubmit = computed(() => canProceedFromUpload.value && !submitting.value)

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value || templateProfileId.value == null || !tabularFilePath.value) return
  submitting.value = true
  try {
    const result = await mapAndReconcileShipments({
      waveId: ctx.waveId.value,
      integrationProfileId: templateProfileId.value,
      importMode: importMode.value,
      filePath: tabularFilePath.value,
    })
    importResult.value = {
      importRunId: result.importRunId,
      evidenceDisabled: result.evidenceDisabled,
      total: result.totalProcessed ?? result.successCount + result.errorCount,
      successCount: result.successCount,
      errorCount: result.errorCount,
      rows: result.errors.map((err) => ({
        rowNo: err.entryIndex + 1,
        reason: err.reason,
      })),
      warnings: result.warnings ?? [],
    }
    if (result.successCount > 0) {
      feedback.success(t('feedback.success'))
      await ctx.refresh()
      emit('imported')
    } else {
      feedback.error(t('feedback.error'))
    }
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}

function resetWizard(): void {
  bindingRequestSeq += 1
  currentStep.value = 'upload'
  csvPreview.value = null
  tabularFilePath.value = ''
  templateProfileId.value = null
  defaultBinding.value = { status: 'idle' }
  importMode.value = 'skip_invalid'
  importResult.value = null
}

const canNext = computed(() => {
  if (currentStep.value === 'upload') return canProceedFromUpload.value
  return canSubmit.value
})

function handleNext(): void {
  if (currentStep.value === 'upload') currentStep.value = 'preview'
}

function handleBack(): void {
  if (currentStep.value === 'preview') currentStep.value = 'upload'
}
</script>

<template>
  <div class="import-wizard">
    <ImportResultView v-if="importResult" :result="importResult" @new-import="resetWizard" />

    <WizardFrame
      v-else
      :steps="wizardSteps"
      :current="currentStep"
      :can-next="canNext"
      :can-back="!submitting"
      :next-label="t('intakeWizard.nav.next')"
      :back-label="t('intakeWizard.nav.back')"
      :finish-label="t('waveWorkspace.shipments.import.preview.submit')"
      @next="handleNext"
      @back="handleBack"
      @finish="handleSubmit"
    >
      <template v-if="currentStep === 'upload'">
        <div class="import-wizard__upload">
          <p class="import-wizard__hint">{{ t('waveWorkspace.shipments.import.uploadHint') }}</p>
          <CalloutBar v-if="profilesError" tone="error" :message="profilesError" />

          <div class="import-wizard__mapping-field">
            <span class="import-wizard__mapping-label">{{ t('waveWorkspace.shipments.import.documentTypeNote') }}</span>
            <StatusBadge
              dimension="documentType"
              :value="SUPPLIER_SHIPMENT_IMPORT_DOCUMENT_TYPE"
              size="sm"
            />
          </div>

          <div class="import-wizard__mapping-field">
            <span class="import-wizard__mapping-label">{{ t('waveWorkspace.shipments.import.templateProfile') }}</span>
            <NSelect
              v-model:value="templateProfileId"
              :options="profileOptions"
              filterable
              :placeholder="t('waveWorkspace.shipments.import.templateProfilePlaceholder')"
              style="max-width: 360px"
            />
          </div>

          <CalloutBar
            v-if="defaultBinding.status === 'loaded'"
            tone="success"
            :message="t('waveWorkspace.shipments.import.defaultBindingLoaded', { key: defaultBinding.templateKey })"
          />
          <CalloutBar
            v-else-if="defaultBinding.status === 'missing'"
            tone="error"
            :message="t('waveWorkspace.shipments.import.defaultBindingMissing')"
          />
          <CalloutBar
            v-else-if="defaultBinding.status === 'error'"
            tone="error"
            :message="defaultBinding.message"
          />

          <CalloutBar
            v-if="hasLoadedBinding"
            tone="info"
            :message="t('waveWorkspace.shipments.import.templateDrivenMappingHint')"
          />

          <NButton :loading="picking" :disabled="!hasLoadedBinding" @click="handlePickFile">
            {{ t('waveWorkspace.shipments.import.pickFile') }}
          </NButton>
          <span v-if="picking" class="import-wizard__status">{{ t('intakeWizard.sampleUpload.parsing') }}</span>
          <span v-else-if="!csvPreview && !tabularFilePath" class="import-wizard__status">{{ t('intakeWizard.sampleUpload.noFile') }}</span>
          <span v-else class="import-wizard__status">
            <template v-if="tabularFilePath">{{ tabularFilePath.split(/[/\\]/).pop() }} · </template>
            {{ t('intakeWizard.sampleUpload.headersDetected', { count: csvPreview?.headers.length ?? 0 }) }}
            ·
            {{ t('intakeWizard.sampleUpload.rowsDetected', { count: csvPreview?.rows.length ?? 0 }) }}
          </span>
        </div>
      </template>

      <template v-else-if="currentStep === 'preview'">
        <div class="import-wizard__preview">
          <div class="import-wizard__mapping-field">
            <span class="import-wizard__mapping-label">{{ t('waveWorkspace.shipments.import.importMode') }}</span>
            <NSelect v-model:value="importMode" :options="importModeOptions" style="max-width: 360px" />
          </div>

          <CalloutBar tone="info" :message="t('waveWorkspace.shipments.import.templateDrivenPreviewHint')" />
        </div>
      </template>
    </WizardFrame>
  </div>
</template>

<style scoped>
.import-wizard {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.import-wizard__upload,
.import-wizard__preview {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.import-wizard__mapping-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.import-wizard__mapping-label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}

.import-wizard__hint {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.import-wizard__status {
  margin-left: var(--space-3);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>
