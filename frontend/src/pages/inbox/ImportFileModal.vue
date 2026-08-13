<script setup lang="ts">
/**
 * ImportFileModal — demand CSV import wizard-in-a-modal. WizardFrame shell
 * with three steps: select platform + file kind + pick file → optional
 * this-run mapping override (FieldMappingEditor, seeded from the pair's
 * default template when available) → result report. Actual import remains
 * template-driven via `importDemandCSV` (filePath preferred so backend
 * re-reads with hasHeader / positional rules from MappingRules).
 * documentType is operator-selected, never inferred from profile.demandKind.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NModal, NRadioButton, NRadioGroup, NSelect } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { WizardFrame, type WizardStep } from '@/shared/ui/wizard'
import {
  FieldMappingEditor,
  emptyFieldMapping,
  parseMappingRules,
  serializeMappingRules,
  type FieldMappingDestField,
  type FieldMappingValue,
} from '@/shared/ui/field-mapping'
import { CalloutBar } from '@/shared/ui/guidance'
import { useFeedback } from '@/shared/ui/feedback'
import {
  listProfiles,
  pickTabularFile,
  parseTabularFile,
  importDemandCSV,
  getDefaultTemplateForProfile,
  batchAssignDemandToWave,
  listWavesFiltered,
} from '@/shared/api/bridge'
import type { ImportDemandCSVResult } from '@/entities/demand'
import type { dto } from '@/../wailsjs/go/models'
import {
  INTAKE_V2_DEST_FIELD_ORDER,
  destFieldLabelKey,
  destFieldTooltipKey,
} from '@/pages/integrations/wizard/destFields'
import { canImportDemand } from '@/pages/integrations/profileAvailability'
import ImportEvidenceReference from '@/shared/ui/customer-resolution/ImportEvidenceReference.vue'
import {
  DEMAND_IMPORT_DOCUMENT_TYPES,
  type DemandImportDocumentType,
} from './demandImportDocumentTypes'

const props = defineProps<{
  show: boolean
  /** 波内导入：设置后导入成功即把新单据自动分派进该波次（不再显示「发送到波次」选择器）。 */
  targetWaveId?: number
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  /** Fires once a document was actually persisted (import succeeded, at least partially). */
  (e: 'imported'): void
  /** Fires when the imported document was assigned into a wave (targetWaveId flow or the result-step picker). */
  (e: 'assignedToWave', docIds: number[]): void
}>()

const { t, te } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

type StepKey = 'select' | 'mapping' | 'result'
const currentStep = ref<StepKey>('select')

const wizardSteps = computed<WizardStep[]>(() => [
  { key: 'select', title: t('inbox.importModal.steps.select') },
  { key: 'mapping', title: t('inbox.importModal.steps.mapping') },
  { key: 'result', title: t('inbox.importModal.steps.result') },
])

const profiles = ref<dto.IntegrationProfileDTO[]>([])
const profilesLoading = ref(false)
const profilesError = ref('')
const profileId = ref<number | null>(null)
const documentType = ref<DemandImportDocumentType | null>(null)

const profileOptions = computed<SelectOption[]>(() =>
  profiles.value
    .filter(canImportDemand)
    .map((profile) => ({ label: `${profile.profileKey} (${profile.sourceChannel})`, value: profile.id })),
)

const documentTypeOptions = computed<SelectOption[]>(() =>
  DEMAND_IMPORT_DOCUMENT_TYPES.map((value) => ({
    label: t(`glossary.documentType.${value}.label`),
    value,
  })),
)

const sourceDocumentNo = ref('')
const sourceCustomerRef = ref('')

const filePath = ref('')
const parsing = ref(false)
const headers = ref<string[]>([])
const previewRows = ref<Record<string, string>[]>([])
const pickError = ref('')

const mapping = ref<FieldMappingValue>(emptyFieldMapping())
const templateLoading = ref(false)
const templateNotice = ref('')

const importMode = ref<'reject_all' | 'skip_invalid'>('skip_invalid')
const importing = ref(false)
const importResult = ref<ImportDemandCSVResult | null>(null)

const destFields = computed<FieldMappingDestField[]>(() =>
  INTAKE_V2_DEST_FIELD_ORDER.map((field) => {
    const labelKey = destFieldLabelKey(field)
    const tooltipKey = destFieldTooltipKey(field)
    return {
      key: field,
      label: te(labelKey) ? t(labelKey) : field,
      tooltip: te(tooltipKey) ? t(tooltipKey) : undefined,
    }
  }),
)

const anomalyCount = computed(() => previewRows.value.filter((row) => headers.value.some((header) => !row[header])).length)
const inputFormat = computed(() => filePath.value.split('.').pop()?.toUpperCase() ?? '')
const pairReady = computed(() => profileId.value != null && documentType.value != null)

async function loadTemplatePreview(id: number, docType: DemandImportDocumentType): Promise<void> {
  templateLoading.value = true
  templateNotice.value = ''
  try {
    const tmpl = await getDefaultTemplateForProfile(id, docType)
    if (tmpl?.mappingRules) {
      mapping.value = parseMappingRules(tmpl.mappingRules)
      templateNotice.value = t('inbox.importModal.templateLoaded', { key: tmpl.templateKey })
    } else {
      mapping.value = emptyFieldMapping()
      templateNotice.value = t('inbox.importModal.templateMissing')
    }
  } catch (err) {
    mapping.value = emptyFieldMapping()
    templateNotice.value = err instanceof Error ? err.message : String(err)
  } finally {
    templateLoading.value = false
  }
}

watch([profileId, documentType], ([id, docType]) => {
  if (id != null && docType) {
    void loadTemplatePreview(id, docType)
    return
  }
  mapping.value = emptyFieldMapping()
  templateNotice.value = ''
})

// 打开即重置并加载 profiles——NModal 只在关闭路径回发 update:show(false)，
// 打开路径只能由父级改 show prop 驱动，因此在这里监听。
watch(
  () => props.show,
  (show) => {
    if (show) void handleOpen()
  },
)

async function handleOpen(): Promise<void> {
  currentStep.value = 'select'
  profileId.value = null
  documentType.value = null
  sourceDocumentNo.value = ''
  sourceCustomerRef.value = ''
  filePath.value = ''
  headers.value = []
  previewRows.value = []
  importMode.value = 'skip_invalid'
  importResult.value = null
  mapping.value = emptyFieldMapping()
  pickError.value = ''
  profilesError.value = ''
  templateNotice.value = ''
  showSendPicker.value = false
  targetPickerWaveId.value = null
  sentToWave.value = false

  profilesLoading.value = true
  try {
    profiles.value = await listProfiles()
  } catch (err) {
    profiles.value = []
    profilesError.value = err instanceof Error ? err.message : String(err)
    feedback.error(t('feedback.error'), profilesError.value)
  } finally {
    profilesLoading.value = false
  }
}

async function handlePickFile(): Promise<void> {
  if (!pairReady.value) return
  parsing.value = true
  pickError.value = ''
  try {
    const path = await pickTabularFile()
    if (!path) return
    filePath.value = path
    // Honour profile template hasHeader so headerless membership sheets keep row0 as data.
    const preview = await parseTabularFile(path, mapping.value.hasHeader !== false)
    headers.value = preview.headers
    previewRows.value = preview.rows
  } catch (err) {
    pickError.value = err instanceof Error ? err.message : String(err)
    feedback.error(t('feedback.error'), pickError.value)
  } finally {
    parsing.value = false
  }
}

const canNextFromSelect = computed(
  () => pairReady.value && (!!filePath.value || previewRows.value.length > 0) && !parsing.value,
)

function handleNext(): void {
  if (currentStep.value === 'select' && canNextFromSelect.value) {
    currentStep.value = 'mapping'
  }
}

function handleBack(): void {
  if (currentStep.value === 'mapping') currentStep.value = 'select'
  else if (currentStep.value === 'result') currentStep.value = 'mapping'
}

async function handleImport(): Promise<void> {
  if (currentStep.value === 'result') {
    handleUpdateShow(false)
    return
  }
  if (profileId.value == null || documentType.value == null) return
  importing.value = true
  try {
    const result = await importDemandCSV({
      integrationProfileId: profileId.value,
      documentType: documentType.value,
      sourceDocumentNo: sourceDocumentNo.value,
      sourceCustomerRef: sourceCustomerRef.value,
      importMode: importMode.value,
      // Prefer filePath so backend re-reads with hasHeader/positional rules.
      filePath: filePath.value || undefined,
      rows: previewRows.value,
      mappingRules: serializeMappingRules(mapping.value),
    })
    importResult.value = result
    currentStep.value = 'result'
    if (result.document) {
      emit('imported')
      // 波内导入：导入成功即自动把新单据分派进目标波次。
      if (props.targetWaveId != null) {
        try {
          const assignResult = await batchAssignDemandToWave({
            waveId: props.targetWaveId,
            docIds: [result.document.id],
          })
          if (assignResult.failureCount > 0) {
            feedback.error(t('inbox.import.assignToWaveFailed'))
          } else {
            sentToWave.value = true
            emit('assignedToWave', [result.document.id])
          }
        } catch (err) {
          feedback.error(t('inbox.import.assignToWaveFailed'), err instanceof Error ? err.message : String(err))
        }
      }
    }
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    importing.value = false
  }
}

// ── 结果步「发送到波次」picker（无 targetWaveId 的收件箱用法）──

const showSendPicker = ref(false)
const waveOptions = ref<SelectOption[]>([])
const wavesLoading = ref(false)
const targetPickerWaveId = ref<number | null>(null)
const sendingToWave = ref(false)
const sentToWave = ref(false)

async function openSendToWavePicker(): Promise<void> {
  showSendPicker.value = true
  targetPickerWaveId.value = null
  wavesLoading.value = true
  try {
    const page = await listWavesFiltered({ page: 1, pageSize: 200, sortBy: 'updatedAt', sortDesc: true })
    waveOptions.value = page.items.map((wave) => ({ label: `${wave.name} (${wave.waveNo})`, value: wave.id }))
  } catch (err) {
    waveOptions.value = []
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    wavesLoading.value = false
  }
}

const canConfirmSend = computed(() => targetPickerWaveId.value != null && !sendingToWave.value)

async function handleConfirmSendToWave(): Promise<void> {
  const doc = importResult.value?.document
  if (targetPickerWaveId.value == null || doc == null) return
  sendingToWave.value = true
  try {
    const assignResult = await batchAssignDemandToWave({
      waveId: targetPickerWaveId.value,
      docIds: [doc.id],
    })
    if (assignResult.failureCount > 0) {
      feedback.error(t('inbox.import.assignToWaveFailed'))
    } else {
      sentToWave.value = true
      feedback.success(t('feedback.success'))
      emit('assignedToWave', [doc.id])
      showSendPicker.value = false
    }
  } catch (err) {
    feedback.error(t('inbox.import.assignToWaveFailed'), err instanceof Error ? err.message : String(err))
  } finally {
    sendingToWave.value = false
  }
}

function handleUpdateShow(value: boolean): void {
  // 打开路径由 `watch(() => props.show)` 驱动（NModal 只回发关闭）。
  emit('update:show', value)
}

const canNext = computed(() => {
  if (currentStep.value === 'select') return canNextFromSelect.value
  if (currentStep.value === 'mapping') return !importing.value
  // result step: Finish becomes Close
  return true
})
</script>

<template>
  <NModal :show="show" preset="card" :title="t('inbox.importModal.title')" style="width: min(720px, 94vw)" @update:show="handleUpdateShow">
    <div class="import-file-modal">
      <WizardFrame
        :steps="wizardSteps"
        :current="currentStep"
        :can-next="canNext"
        :can-back="!importing && currentStep !== 'result'"
        :next-label="t('intakeWizard.nav.next')"
        :back-label="t('intakeWizard.nav.back')"
        :finish-label="currentStep === 'result' ? t('common.close') : t('inbox.importModal.import')"
        :cancel-label="currentStep === 'result' ? undefined : t('common.close')"
        @next="handleNext"
        @back="handleBack"
        @finish="handleImport"
        @cancel="handleUpdateShow(false)"
      >
        <template v-if="currentStep === 'select'">
          <div class="import-file-modal__section">
            <p class="import-file-modal__step-label">{{ t('inbox.importModal.step1SelectProfile') }}</p>
            <CalloutBar v-if="profilesError" tone="error" :message="profilesError" />
            <NSelect
              v-model:value="profileId"
              :options="profileOptions"
              :loading="profilesLoading"
              filterable
              :placeholder="t('inbox.importModal.step1SelectProfile')"
            />
            <p class="import-file-modal__step-label">{{ t('inbox.importModal.documentTypeLabel') }}</p>
            <NSelect
              v-model:value="documentType"
              :options="documentTypeOptions"
              :placeholder="t('inbox.importModal.documentTypePlaceholder')"
            />
            <NInput v-model:value="sourceDocumentNo" :placeholder="t('inbox.columns.sourceDoc')" />
            <NInput v-model:value="sourceCustomerRef" :placeholder="t('inbox.importModal.sourceCustomerRef')" />

            <p class="import-file-modal__step-label">{{ t('inbox.importModal.step2PickFile') }}</p>
            <div class="import-file-modal__pick-row">
              <NButton type="primary" :disabled="!pairReady" :loading="parsing" @click="handlePickFile">
                {{ t('inbox.importFileButton') }}
              </NButton>
              <span v-if="filePath" class="import-file-modal__status">
                {{ filePath.split(/[/\\]/).pop() }}
                ·
                {{ t('inbox.importModal.rowsRecognized', { count: previewRows.length }) }}
              </span>
            </div>
            <CalloutBar v-if="pickError" tone="error" :message="pickError" />
          </div>
        </template>

        <template v-else-if="currentStep === 'mapping'">
          <div class="import-file-modal__section">
            <CalloutBar tone="info" :message="t('inbox.importModal.overrideThisRunOnly')" />
            <CalloutBar v-if="templateNotice" :tone="templateLoading ? 'info' : 'neutral'" :message="templateNotice" />

            <p class="import-file-modal__meta">
              {{ t('inbox.importModal.rowsRecognized', { count: previewRows.length }) }}
              <template v-if="anomalyCount > 0">
                · {{ t('inbox.importModal.anomalies', { count: anomalyCount }) }}
              </template>
            </p>

            <p class="import-file-modal__step-label">{{ t('inbox.importModal.modeLabel') }}</p>
            <NRadioGroup v-model:value="importMode">
              <NRadioButton value="skip_invalid">{{ t('inbox.importModal.modeSkipInvalid') }}</NRadioButton>
              <NRadioButton value="reject_all">{{ t('inbox.importModal.modeRejectAll') }}</NRadioButton>
            </NRadioGroup>

            <FieldMappingEditor
              v-if="headers.length || previewRows.length"
              v-model:model-value="mapping"
              :dest-fields="destFields"
              :source-headers="headers"
              :sample-rows="previewRows"
              :dest-column-header="t('intakeWizard.mapping.destColumnHeader')"
              :src-column-header="t('intakeWizard.mapping.srcColumnHeader')"
              :preview-title="t('intakeWizard.mapping.previewTitle')"
              :unmapped-label="t('intakeWizard.mapping.unmapped')"
              :fixed-value-placeholder="t('intakeWizard.mapping.fixedValuePlaceholder')"
              :mode-label="t('intakeWizard.mapping.modeLabel')"
              :mode-header-label="t('intakeWizard.mapping.modeHeader')"
              :mode-positional-label="t('intakeWizard.mapping.modePositional')"
              :has-header-label="t('intakeWizard.mapping.hasHeaderLabel')"
              :position-placeholder="t('intakeWizard.mapping.positionPlaceholder')"
              :column-order-label="t('intakeWizard.mapping.columnOrderLabel')"
              :column-order-placeholder="t('intakeWizard.mapping.columnOrderPlaceholder')"
              :input-format="inputFormat"
            />
          </div>
        </template>

        <template v-else-if="currentStep === 'result' && importResult">
          <div class="import-file-modal__section">
            <p class="import-file-modal__step-label">{{ t('inbox.importModal.resultTitle') }}</p>
            <p>{{ t('inbox.importModal.successCount', { count: importResult.successCount }) }}</p>
            <p v-if="importResult.errorCount > 0">{{ t('inbox.importModal.errorCount', { count: importResult.errorCount }) }}</p>
            <ImportEvidenceReference :import-run-id="importResult.importRunId" :evidence-disabled="importResult.evidenceDisabled" />
            <ul v-if="importResult.errors.length > 0" class="import-file-modal__errors">
              <li v-for="err in importResult.errors" :key="err.rowIndex">
                {{ t('inbox.importModal.rowError', { row: err.rowIndex, reason: err.reason }) }}
              </li>
            </ul>
            <CalloutBar
              v-if="importResult.warnings && importResult.warnings.length > 0"
              tone="warning"
              :message="t('inbox.importModal.warningCount', { count: importResult.warnings.length })"
            />
            <ul v-if="importResult.warnings && importResult.warnings.length > 0" class="import-file-modal__warnings">
              <li v-for="(warning, idx) in importResult.warnings" :key="idx">{{ warning }}</li>
            </ul>
            <div class="import-file-modal__actions">
              <NButton
                v-if="targetWaveId == null && importResult.document && !sentToWave"
                secondary
                @click="openSendToWavePicker"
              >
                {{ t('inbox.import.sendToWave') }}
              </NButton>
              <NButton type="primary" @click="handleUpdateShow(false)">{{ t('common.close') }}</NButton>
            </div>
          </div>
        </template>
      </WizardFrame>
    </div>
  </NModal>

  <NModal v-model:show="showSendPicker" preset="card" :title="t('inbox.batch.chooseWave')" style="width: 420px">
    <NSelect
      v-model:value="targetPickerWaveId"
      :options="waveOptions"
      :loading="wavesLoading"
      filterable
      :placeholder="t('inbox.batch.chooseWave')"
    />
    <template #footer>
      <div class="import-file-modal__actions">
        <NButton @click="showSendPicker = false">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="sendingToWave" :disabled="!canConfirmSend" @click="handleConfirmSendToWave">
          {{ t('inbox.batch.confirm') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.import-file-modal {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.import-file-modal__section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.import-file-modal__step-label {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}

.import-file-modal__pick-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.import-file-modal__status,
.import-file-modal__meta {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.import-file-modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.import-file-modal__errors {
  max-height: 200px;
  overflow-y: auto;
  margin: 0;
  padding-left: var(--space-4);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--status-error-fg);
}

.import-file-modal__warnings {
  max-height: 200px;
  overflow-y: auto;
  margin: 0;
  padding-left: var(--space-4);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--status-warning-fg);
}
</style>
