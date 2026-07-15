<script setup lang="ts">
/**
 * ImportFileModal — demand CSV import wizard-in-a-modal. WizardFrame shell
 * with three steps: select profile + pick file → mapping preview
 * (FieldMappingEditor, seeded from the profile's default template when
 * available) → result report. Actual import remains template-driven via
 * `importDemandCSV` (filePath preferred so backend re-reads with hasHeader /
 * positional rules from MappingRules).
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
} from '@/shared/api/bridge'
import type { ImportDemandCSVResult } from '@/entities/demand'
import type { dto } from '@/../wailsjs/go/models'
import {
  INTAKE_V2_DEST_FIELD_ORDER,
  destFieldLabelKey,
  destFieldTooltipKey,
} from '@/pages/integrations/wizard/destFields'
import { documentTypeForDemandKind, type DemandKind } from '@/pages/integrations/wizard/deriveProfileDefaults'

defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  /** Fires once a document was actually persisted (import succeeded, at least partially). */
  (e: 'imported'): void
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

const profileOptions = computed<SelectOption[]>(() =>
  profiles.value.map((profile) => ({ label: `${profile.profileKey} (${profile.sourceChannel})`, value: profile.id })),
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

async function loadTemplatePreview(id: number): Promise<void> {
  templateLoading.value = true
  templateNotice.value = ''
  try {
    const profile = profiles.value.find((p) => p.id === id)
    const demandKind = (profile?.demandKind as DemandKind | undefined) ?? 'membership_entitlement'
    const docType = documentTypeForDemandKind(demandKind)
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

watch(profileId, (id) => {
  if (id != null) void loadTemplatePreview(id)
})

async function handleOpen(): Promise<void> {
  currentStep.value = 'select'
  profileId.value = null
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
  if (profileId.value == null) return
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
  () => profileId.value != null && (!!filePath.value || previewRows.value.length > 0) && !parsing.value,
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
  if (profileId.value == null) return
  importing.value = true
  try {
    const result = await importDemandCSV({
      integrationProfileId: profileId.value,
      documentType: '',
      sourceDocumentNo: sourceDocumentNo.value,
      sourceCustomerRef: sourceCustomerRef.value,
      importMode: importMode.value,
      // Prefer filePath so backend re-reads with hasHeader/positional rules.
      filePath: filePath.value || undefined,
      rows: previewRows.value,
    })
    importResult.value = result
    currentStep.value = 'result'
    if (result.document) {
      emit('imported')
    }
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    importing.value = false
  }
}

function handleUpdateShow(value: boolean): void {
  emit('update:show', value)
  if (value) void handleOpen()
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
            <NInput v-model:value="sourceDocumentNo" :placeholder="t('inbox.columns.sourceDoc')" />
            <NInput v-model:value="sourceCustomerRef" :placeholder="t('inbox.importModal.sourceCustomerRef')" />

            <p class="import-file-modal__step-label">{{ t('inbox.importModal.step2PickFile') }}</p>
            <div class="import-file-modal__pick-row">
              <NButton type="primary" :disabled="profileId == null" :loading="parsing" @click="handlePickFile">
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
            <CalloutBar tone="info" :message="t('inbox.importModal.templateDrivenHint')" />
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
            />
          </div>
        </template>

        <template v-else-if="currentStep === 'result' && importResult">
          <div class="import-file-modal__section">
            <p class="import-file-modal__step-label">{{ t('inbox.importModal.resultTitle') }}</p>
            <p>{{ t('inbox.importModal.successCount', { count: importResult.successCount }) }}</p>
            <p v-if="importResult.errorCount > 0">{{ t('inbox.importModal.errorCount', { count: importResult.errorCount }) }}</p>
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
              <NButton type="primary" @click="handleUpdateShow(false)">{{ t('common.close') }}</NButton>
            </div>
          </div>
        </template>
      </WizardFrame>
    </div>
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
