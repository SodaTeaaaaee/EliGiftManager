<script setup lang="ts">
/**
 * ImportFileModal — the real-file demand CSV import wizard-in-a-modal (plan
 * P4 §3.5): pick an integration profile -> native file picker
 * (`pickCsvFile`) -> parse (`parseCSVFile`) -> preview row/anomaly counts ->
 * pick an `ImportMode` (`reject_all` | `skip_invalid`) -> import
 * (`importDemandCSV`) -> render the per-row result report. `documentType` is
 * left empty (the backend defaults it to `"import_entitlement"` and
 * resolves the profile's default template internally — see
 * `controller_demand_csv_import.go::ImportDemandCSV` — this modal has no
 * template-picker step, matching the P4 unit brief's stated flow).
 *
 * "Anomalies" in the preview step is a CLIENT-SIDE heuristic only (rows with
 * at least one blank cell across the recognized headers) — a rough sniff
 * before the real, template-driven validation the backend performs on
 * import, whose actual per-row errors are what the result report shows.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NAlert, NButton, NInput, NModal, NRadioButton, NRadioGroup, NSelect } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { listProfiles, pickCsvFile, parseCSVFile, importDemandCSV } from '@/shared/api/bridge'
import type { ImportDemandCSVResult } from '@/entities/demand'
import type { dto } from '@/../wailsjs/go/models'

defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  /** Fires once a document was actually persisted (import succeeded, at least partially). */
  (e: 'imported'): void
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

type WizardStep = 'select' | 'preview' | 'result'
const step = ref<WizardStep>('select')

const profiles = ref<dto.IntegrationProfileDTO[]>([])
const profilesLoading = ref(false)
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

const importMode = ref<'reject_all' | 'skip_invalid'>('skip_invalid')
const importing = ref(false)
const importResult = ref<ImportDemandCSVResult | null>(null)

const anomalyCount = computed(() => previewRows.value.filter((row) => headers.value.some((header) => !row[header])).length)

async function handleOpen(): Promise<void> {
  step.value = 'select'
  profileId.value = null
  sourceDocumentNo.value = ''
  sourceCustomerRef.value = ''
  filePath.value = ''
  headers.value = []
  previewRows.value = []
  importMode.value = 'skip_invalid'
  importResult.value = null

  profilesLoading.value = true
  try {
    profiles.value = await listProfiles()
  } catch (err) {
    profiles.value = []
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    profilesLoading.value = false
  }
}

async function handlePickFile(): Promise<void> {
  if (profileId.value == null) return
  parsing.value = true
  try {
    const path = await pickCsvFile()
    if (!path) return
    filePath.value = path
    const preview = await parseCSVFile(path)
    headers.value = preview.headers
    previewRows.value = preview.rows
    step.value = 'preview'
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    parsing.value = false
  }
}

async function handleImport(): Promise<void> {
  if (profileId.value == null) return
  importing.value = true
  try {
    const result = await importDemandCSV({
      integrationProfileId: profileId.value,
      documentType: '',
      sourceDocumentNo: sourceDocumentNo.value,
      sourceCustomerRef: sourceCustomerRef.value,
      importMode: importMode.value,
      rows: previewRows.value,
    })
    importResult.value = result
    step.value = 'result'
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
</script>

<template>
  <NModal :show="show" preset="card" :title="t('inbox.importModal.title')" style="width: 640px" @update:show="handleUpdateShow">
    <div class="import-file-modal">
      <template v-if="step === 'select'">
        <p class="import-file-modal__step-label">{{ t('inbox.importModal.step1SelectProfile') }}</p>
        <NSelect v-model:value="profileId" :options="profileOptions" :loading="profilesLoading" filterable />
        <NInput v-model:value="sourceDocumentNo" :placeholder="t('inbox.columns.sourceDoc')" />
        <p class="import-file-modal__step-label">{{ t('inbox.importModal.step2PickFile') }}</p>
        <NButton type="primary" :disabled="profileId == null" :loading="parsing" @click="handlePickFile">
          {{ t('inbox.importFileButton') }}
        </NButton>
      </template>

      <template v-else-if="step === 'preview'">
        <p class="import-file-modal__step-label">{{ t('inbox.importModal.step3Preview') }}</p>
        <p>{{ t('inbox.importModal.rowsRecognized', { count: previewRows.length }) }}</p>
        <NAlert v-if="anomalyCount > 0" type="warning" :show-icon="false">
          {{ t('inbox.importModal.anomalies', { count: anomalyCount }) }}
        </NAlert>

        <p class="import-file-modal__step-label">{{ t('inbox.importModal.modeLabel') }}</p>
        <NRadioGroup v-model:value="importMode">
          <NRadioButton value="skip_invalid">{{ t('inbox.importModal.modeSkipInvalid') }}</NRadioButton>
          <NRadioButton value="reject_all">{{ t('inbox.importModal.modeRejectAll') }}</NRadioButton>
        </NRadioGroup>

        <div class="import-file-modal__actions">
          <NButton @click="step = 'select'">{{ t('common.back') }}</NButton>
          <NButton type="primary" :loading="importing" @click="handleImport">
            {{ t('inbox.importModal.import') }}
          </NButton>
        </div>
      </template>

      <template v-else-if="step === 'result' && importResult">
        <p class="import-file-modal__step-label">{{ t('inbox.importModal.resultTitle') }}</p>
        <p>{{ t('inbox.importModal.successCount', { count: importResult.successCount }) }}</p>
        <p v-if="importResult.errorCount > 0">{{ t('inbox.importModal.errorCount', { count: importResult.errorCount }) }}</p>
        <ul v-if="importResult.errors.length > 0" class="import-file-modal__errors">
          <li v-for="err in importResult.errors" :key="err.rowIndex">
            {{ t('inbox.importModal.rowError', { row: err.rowIndex, reason: err.reason }) }}
          </li>
        </ul>
        <div class="import-file-modal__actions">
          <NButton type="primary" @click="handleUpdateShow(false)">{{ t('common.close') }}</NButton>
        </div>
      </template>
    </div>
  </NModal>
</template>

<style scoped>
.import-file-modal {
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
</style>
