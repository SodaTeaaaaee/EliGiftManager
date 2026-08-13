<script setup lang="ts">
/**
 * ManualEntryModal — single-line manual demand entry.
 * Operator selects source platform + file kind (documentType). Kind is
 * resolved by importDemandDocument / Interpret from that documentType;
 * leftover IntegrationProfile.demandKind is not used to filter platforms
 * or infer kind.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NInputNumber, NModal, NSelect } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { listProfiles, importDemandDocument } from '@/shared/api/bridge'
import type { dto } from '@/../wailsjs/go/models'
import { canImportDemand } from '@/pages/integrations/profileAvailability'
import {
  DEMAND_IMPORT_DOCUMENT_TYPES,
  type DemandImportDocumentType,
} from './demandImportDocumentTypes'

defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'created'): void
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const profiles = ref<dto.IntegrationProfileDTO[]>([])
const profilesLoading = ref(false)
const profileOptions = ref<SelectOption[]>([])

const profileId = ref<number | null>(null)
const documentType = ref<DemandImportDocumentType | null>(null)
const sourceDocumentNo = ref('')
const sourceCustomerRef = ref('')
const customerProfileId = ref<number | null>(null)
const externalTitle = ref('')
const requestedQuantity = ref<number | null>(1)
const submitting = ref(false)

const documentTypeOptions = computed<SelectOption[]>(() =>
  DEMAND_IMPORT_DOCUMENT_TYPES.map((value) => ({
    label: t(`glossary.documentType.${value}.label`),
    value,
  })),
)

const canSubmit = computed(
  () => profileId.value != null && documentType.value != null && !!externalTitle.value.trim(),
)

function resetForm(): void {
  profileId.value = null
  documentType.value = null
  sourceDocumentNo.value = ''
  sourceCustomerRef.value = ''
  customerProfileId.value = null
  externalTitle.value = ''
  requestedQuantity.value = 1
}

async function handleOpen(): Promise<void> {
  resetForm()
  profilesLoading.value = true
  try {
    profiles.value = await listProfiles()
    profileOptions.value = profiles.value
      .filter(canImportDemand)
      .map((profile) => ({
        label: `${profile.profileKey} (${profile.sourceChannel})`,
        value: profile.id,
      }))
  } catch (err) {
    profiles.value = []
    profileOptions.value = []
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    profilesLoading.value = false
  }
}

function handleUpdateShow(value: boolean): void {
  emit('update:show', value)
  if (value) void handleOpen()
}

async function handleSubmit(): Promise<void> {
  if (profileId.value == null || documentType.value == null || !externalTitle.value.trim()) return
  submitting.value = true
  try {
    await importDemandDocument({
      kind: documentType.value,
      documentType: documentType.value,
      captureMode: 'manual_entry',
      sourceChannel: 'manual',
      sourceDocumentNo: sourceDocumentNo.value || `MANUAL-${Date.now()}`,
      sourceCustomerRef: sourceCustomerRef.value,
      customerProfileId: customerProfileId.value ?? undefined,
      integrationProfileId: profileId.value,
      lines: [
        {
          lineType: 'sku_order',
          obligationTriggerKind: 'manual_compensation',
          entitlementAuthority: 'manual_grant',
          recipientInputState: 'ready',
          routingDisposition: 'accepted',
          externalTitle: externalTitle.value.trim(),
          requestedQuantity: requestedQuantity.value ?? 1,
        },
      ],
    })
    feedback.success(t('feedback.success'))
    emit('created')
    handleUpdateShow(false)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal :show="show" preset="card" :title="t('inbox.manualEntry.title')" style="width: 480px" @update:show="handleUpdateShow">
    <div class="manual-entry-modal">
      <p class="manual-entry-modal__label">{{ t('inbox.manualEntry.profileLabel') }}</p>
      <NSelect
        v-model:value="profileId"
        :options="profileOptions"
        :loading="profilesLoading"
        filterable
        :placeholder="t('inbox.manualEntry.profilePlaceholder')"
      />
      <p class="manual-entry-modal__label">{{ t('inbox.manualEntry.documentTypeLabel') }}</p>
      <NSelect
        v-model:value="documentType"
        :options="documentTypeOptions"
        :placeholder="t('inbox.manualEntry.documentTypePlaceholder')"
      />
      <NInput v-model:value="sourceDocumentNo" :placeholder="t('inbox.columns.sourceDoc')" />
      <NInput v-model:value="sourceCustomerRef" :placeholder="t('inbox.manualEntry.sourceCustomerRef')" />
      <NInputNumber v-model:value="customerProfileId" style="width: 100%" :placeholder="t('inbox.manualEntry.customerProfileId')" />
      <NInput v-model:value="externalTitle" :placeholder="t('inbox.manualEntry.externalTitle')" />
      <NInputNumber v-model:value="requestedQuantity" style="width: 100%" :min="1" :placeholder="t('inbox.manualEntry.requestedQuantity')" />

      <div class="manual-entry-modal__actions">
        <NButton @click="handleUpdateShow(false)">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('common.submit') }}
        </NButton>
      </div>
    </div>
  </NModal>
</template>

<style scoped>
.manual-entry-modal {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.manual-entry-modal__label {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--font-size-sm);
}

.manual-entry-modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
