<script setup lang="ts">
/**
 * ManualEntryModal — single-line manual demand entry.
 * Operator selects source platform + file kind (documentType). Kind is
 * resolved by importDemandDocument / Interpret from that documentType;
 * leftover IntegrationProfile.demandKind is not used to filter platforms
 * or infer kind.
 */
import { computed, ref, watch } from 'vue'
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

const props = defineProps<{
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

function isValidRequestedQuantity(value: number | null): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 1
}

const canSubmit = computed(() => {
  if (profileId.value == null || documentType.value == null) return false
  if (!externalTitle.value.trim()) return false
  if (!isValidRequestedQuantity(requestedQuantity.value)) return false
  if (documentType.value === 'import_entitlement' && !sourceCustomerRef.value.trim()) return false
  return true
})

const sourceCustomerRefPlaceholder = computed(() =>
  t(
    documentType.value === 'import_entitlement'
      ? 'inbox.manualEntry.sourceCustomerRefRequired'
      : 'inbox.manualEntry.sourceCustomerRef',
  ),
)

type ManualEntryLine = {
  lineType: string
  obligationTriggerKind: string
  entitlementAuthority: string
  recipientInputState: string
  routingDisposition: string
  giftLevelSnapshot?: string
  externalTitle: string
  requestedQuantity: number
}

function buildManualEntryLines(
  kind: DemandImportDocumentType,
  title: string,
  quantity: number,
): ManualEntryLine[] {
  if (kind === 'import_entitlement') {
    return [
      {
        lineType: 'entitlement_rule',
        obligationTriggerKind: 'periodic_membership',
        entitlementAuthority: 'manual_grant',
        recipientInputState: 'not_required',
        routingDisposition: 'accepted',
        giftLevelSnapshot: title,
        externalTitle: title,
        requestedQuantity: quantity,
      },
    ]
  }
  return [
    {
      lineType: 'sku_order',
      obligationTriggerKind: 'manual_compensation',
      entitlementAuthority: 'manual_grant',
      recipientInputState: 'ready',
      routingDisposition: 'accepted',
      externalTitle: title,
      requestedQuantity: quantity,
    },
  ]
}

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

// 打开即重置并加载 profiles——NModal 只在关闭路径回发 update:show(false)，
// 打开路径只能由父级改 show prop 驱动，因此在这里监听。
watch(
  () => props.show,
  (show) => {
    if (show) void handleOpen()
  },
)

function handleUpdateShow(value: boolean): void {
  emit('update:show', value)
}

async function handleSubmit(): Promise<void> {
  const title = externalTitle.value.trim()
  const quantity = requestedQuantity.value
  if (profileId.value == null || documentType.value == null || !title) return
  if (!isValidRequestedQuantity(quantity)) return
  if (documentType.value === 'import_entitlement' && !sourceCustomerRef.value.trim()) return
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
      lines: buildManualEntryLines(documentType.value, title, quantity),
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

defineExpose({
  handleOpen,
  handleSubmit,
  profileId,
  documentType,
  externalTitle,
  profileOptions,
  sourceCustomerRef,
  sourceCustomerRefPlaceholder,
  requestedQuantity,
  canSubmit,
})
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
      <NInput v-model:value="sourceCustomerRef" :placeholder="sourceCustomerRefPlaceholder" />
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
