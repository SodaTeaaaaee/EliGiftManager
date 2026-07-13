<script setup lang="ts">
/**
 * ManualEntryModal — the secondary, single-line manual demand entry form
 * (plan P4 §3.5). Ports the OLD tree's manual-entry shape
 * (`frontend/src/pages/demand-inbox/DemandInboxPage.vue`'s
 * `submitManualEntry()`) verbatim: `kind: 'retail_order'`,
 * `captureMode: 'manual_entry'`, `sourceChannel: 'manual'`, a single line
 * with `lineType: 'sku_order'`, `obligationTriggerKind:
 * 'manual_compensation'`, `entitlementAuthority: 'manual_grant'`,
 * `recipientInputState: 'ready'`, `routingDisposition: 'accepted'` — with
 * every user-visible string now driven through i18n instead of the old
 * tree's hardcoded English literals.
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NInputNumber, NModal, NSelect } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { listProfiles, importDemandDocument } from '@/shared/api/bridge'
import type { dto } from '@/../wailsjs/go/models'

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
const sourceDocumentNo = ref('')
const sourceCustomerRef = ref('')
const customerProfileId = ref<number | null>(null)
const externalTitle = ref('')
const requestedQuantity = ref<number | null>(1)
const submitting = ref(false)

function resetForm(): void {
  profileId.value = null
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
    profileOptions.value = profiles.value.map((profile) => ({
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
  if (profileId.value == null || !externalTitle.value.trim()) return
  submitting.value = true
  try {
    await importDemandDocument({
      kind: 'retail_order',
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
      <NSelect
        v-model:value="profileId"
        :options="profileOptions"
        :loading="profilesLoading"
        filterable
        :placeholder="t('inbox.columns.profile')"
      />
      <NInput v-model:value="sourceDocumentNo" :placeholder="t('inbox.columns.sourceDoc')" />
      <NInput v-model:value="sourceCustomerRef" :placeholder="t('inbox.manualEntry.sourceCustomerRef')" />
      <NInputNumber v-model:value="customerProfileId" style="width: 100%" :placeholder="t('inbox.manualEntry.customerProfileId')" />
      <NInput v-model:value="externalTitle" :placeholder="t('inbox.manualEntry.externalTitle')" />
      <NInputNumber v-model:value="requestedQuantity" style="width: 100%" :min="1" :placeholder="t('inbox.manualEntry.requestedQuantity')" />

      <div class="manual-entry-modal__actions">
        <NButton @click="handleUpdateShow(false)">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="profileId == null || !externalTitle.trim()" @click="handleSubmit">
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

.manual-entry-modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
