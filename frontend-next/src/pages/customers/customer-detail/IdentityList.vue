<script setup lang="ts">
/**
 * IdentityList — the customer detail page's "identities" section (plan
 * §3.6): list of bound identities (platform + value + type + primary flag)
 * with add/delete. `identityType` is a closed domain enum (rendered via
 * `StatusBadge`/glossary, never a raw string) — `identityPlatform` is real
 * dynamic data (not an enum), so it stays a free-text field on the add form.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSwitch } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { StatusBadge } from '@/shared/ui/status'
import { EmptyState } from '@/shared/ui/empty-state'
import { useFeedback } from '@/shared/ui/feedback'
import { addCustomerIdentity, deleteCustomerIdentity } from '@/shared/api/bridge'
import type { CustomerIdentityDTO } from '@/entities/customer'

const props = defineProps<{
  customerProfileId: number
  identities: CustomerIdentityDTO[]
}>()

const emit = defineEmits<{
  changed: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

// Closed set per `internal/domain/enums.go`'s IdentityType constants (see
// contract `backendContract` (a)) — the only enum source, no bridge lookup.
const identityTypeOptions = computed<SelectOption[]>(() => [
  { label: t('glossary.identityType.platform_uid.label'), value: 'platform_uid' },
  { label: t('glossary.identityType.email.label'), value: 'email' },
  { label: t('glossary.identityType.username.label'), value: 'username' },
  { label: t('glossary.identityType.external_buyer_id.label'), value: 'external_buyer_id' },
])

const showAdd = ref(false)
const platform = ref('')
const value = ref('')
const identityType = ref('platform_uid')
const isPrimary = ref(false)
const submitting = ref(false)

function resetForm(): void {
  platform.value = ''
  value.value = ''
  identityType.value = 'platform_uid'
  isPrimary.value = false
}

watch(showAdd, (visible) => {
  if (visible) resetForm()
})

const canSubmit = computed(() => !submitting.value && platform.value.trim().length > 0 && value.value.trim().length > 0)

async function handleAdd(): Promise<void> {
  if (!canSubmit.value) return
  submitting.value = true
  try {
    await addCustomerIdentity({
      customerProfileId: props.customerProfileId,
      identityPlatform: platform.value.trim(),
      identityValue: value.value.trim(),
      identityType: identityType.value,
      isPrimary: isPrimary.value,
      extraData: '',
    })
    showAdd.value = false
    feedback.success(t('customerDetail.feedback.identityAdded'))
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}

const pendingDeleteId = ref<number | null>(null)
const deleting = ref(false)

function requestDelete(id: number): void {
  pendingDeleteId.value = id
}

function cancelDelete(): void {
  pendingDeleteId.value = null
}

async function confirmDelete(): Promise<void> {
  if (pendingDeleteId.value == null) return
  deleting.value = true
  try {
    await deleteCustomerIdentity(pendingDeleteId.value)
    pendingDeleteId.value = null
    feedback.success(t('customerDetail.feedback.identityDeleted'))
    emit('changed')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="identity-list">
    <div class="identity-list__toolbar">
      <NButton size="small" @click="showAdd = true">{{ t('customerDetail.identities.addAction') }}</NButton>
    </div>

    <EmptyState v-if="identities.length === 0" :title="t('customerDetail.identities.empty')" size="sm" />

    <ul v-else class="identity-list__items">
      <li v-for="identity in identities" :key="identity.id" class="identity-list__item">
        <div class="identity-list__main">
          <span class="identity-list__platform">{{ identity.identityPlatform }}</span>
          <span class="identity-list__value">{{ identity.identityValue }}</span>
        </div>
        <div class="identity-list__meta">
          <StatusBadge dimension="identityType" :value="identity.identityType" size="sm" />
          <span v-if="identity.isPrimary" class="identity-list__primary-tag">
            {{ t('customerDetail.identities.isPrimaryLabel') }}
          </span>
          <NButton size="tiny" quaternary @click="requestDelete(identity.id)">
            {{ t('customerDetail.identities.deleteAction') }}
          </NButton>
        </div>
      </li>
    </ul>

    <NModal
      :show="showAdd"
      preset="card"
      :title="t('customerDetail.identities.createAction')"
      :style="{ width: 'min(420px, 92vw)' }"
      :mask-closable="!submitting"
      :close-on-esc="!submitting"
      @update:show="(v: boolean) => (showAdd = v)"
    >
      <NForm label-placement="top">
        <NFormItem :label="t('customerDetail.identities.platformLabel')">
          <NInput v-model:value="platform" :disabled="submitting" @keydown.enter.prevent="handleAdd" />
        </NFormItem>
        <NFormItem :label="t('customerDetail.identities.valueLabel')">
          <NInput v-model:value="value" :disabled="submitting" @keydown.enter.prevent="handleAdd" />
        </NFormItem>
        <NFormItem :label="t('customerDetail.identities.typeLabel')">
          <NSelect v-model:value="identityType" :options="identityTypeOptions" :disabled="submitting" />
        </NFormItem>
        <NFormItem :label="t('customerDetail.identities.isPrimaryLabel')">
          <NSwitch v-model:value="isPrimary" :disabled="submitting" />
        </NFormItem>
      </NForm>
      <template #footer>
        <div class="identity-list__form-footer">
          <NButton :disabled="submitting" @click="showAdd = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleAdd">
            {{ t('common.confirm') }}
          </NButton>
        </div>
      </template>
    </NModal>

    <NModal
      :show="pendingDeleteId != null"
      preset="dialog"
      type="warning"
      :title="t('customerDetail.identities.deleteAction')"
      :content="t('customerDetail.identities.deleteConfirm')"
      :positive-text="t('common.confirm')"
      :negative-text="t('common.cancel')"
      :loading="deleting"
      @positive-click="confirmDelete"
      @negative-click="cancelDelete"
      @update:show="(v: boolean) => { if (!v) cancelDelete() }"
    />
  </div>
</template>

<style scoped>
.identity-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.identity-list__toolbar {
  display: flex;
  justify-content: flex-end;
}

.identity-list__items {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.identity-list__item {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
}

.identity-list__main {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  min-width: 0;
}

.identity-list__platform {
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.identity-list__value {
  color: var(--color-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.identity-list__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
}

.identity-list__primary-tag {
  display: inline-flex;
  align-items: center;
  padding: 1px var(--space-2);
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--status-info-fg);
  background: var(--status-info-bg);
  border: 1px solid var(--status-info-border);
}

.identity-list__form-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
