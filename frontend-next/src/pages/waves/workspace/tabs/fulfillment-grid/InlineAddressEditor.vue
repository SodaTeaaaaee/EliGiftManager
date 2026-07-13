<script setup lang="ts">
/**
 * InlineAddressEditor — the address-resolution panel embedded in
 * `RowDetailDrawer` for address-abnormal rows (`addressState` `'missing'` or
 * `'invalid'` — the caller decides when to mount this, per the row detail
 * drawer's contract). Two modes:
 * - "pick existing": lists the row's customer's saved addresses
 *   (`listAddressesByProfile`) and binds the selected one
 *   (`bindAddressToLine`).
 * - "author new": a minimal create form (`createAddress`) immediately bound
 *   to this line on success.
 *
 * When the row has no linked `customerProfileId` at all (an anonymous /
 * unlinked participant), there is nothing to look up or bind — this shows
 * an explanatory message instead of a non-functional form. Per the data
 * contract's key set there is no dedicated copy for this exact case, so it
 * reuses `fulfillmentGrid.address.noAddresses` ("this customer has no
 * addresses on file") as the closest honest fit — flagged in the handoff
 * `deviations` rather than inventing a new i18n key.
 *
 * Emits `'bound'` after any successful bind — `RowDetailDrawer` turns this
 * into its own `'changed'` emit.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NForm, NFormItem, NInput, NRadioGroup, NRadio } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { bindAddressToLine, createAddress, listAddressesByProfile } from '@/shared/api/bridge'
import type { dto } from '@/../wailsjs/go/models'
import type { FulfillmentGridRow } from './useFulfillmentGrid'

const props = defineProps<{
  row: FulfillmentGridRow
}>()

const emit = defineEmits<{
  (e: 'bound'): void
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

type Mode = 'pick' | 'create'
const mode = ref<Mode>('pick')

const hasCustomerProfile = computed(() => props.row.customerProfileId != null)

// ── Pick existing ──

const addresses = ref<dto.CustomerAddressDTO[]>([])
const addressesLoading = ref(false)
const addressesLoaded = ref(false)
const selectedAddressId = ref<number | null>(null)
const binding = ref(false)

async function loadAddresses(): Promise<void> {
  const profileId = props.row.customerProfileId
  if (!profileId) {
    addresses.value = []
    addressesLoaded.value = true
    return
  }
  addressesLoading.value = true
  try {
    const list = await listAddressesByProfile(profileId)
    addresses.value = list
    selectedAddressId.value = list.find((address) => address.isDefault)?.id ?? list[0]?.id ?? null
    addressesLoaded.value = true
  } catch (err) {
    // `listAddressesByProfile` is soft-fail only for the "no Wails runtime"
    // case (returns `[]`) — a real backend RPC error still rejects, so this
    // must be caught explicitly rather than left as an unhandled rejection.
    addresses.value = []
    addressesLoaded.value = true
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    addressesLoading.value = false
  }
}

// Re-load whenever the drawer swaps in a different row (Assembly may pass a
// new `row` without unmounting this component between address-abnormal rows).
watch(
  () => props.row.customerProfileId,
  () => {
    mode.value = 'pick'
    selectedAddressId.value = null
    addressesLoaded.value = false
    void loadAddresses()
  },
  { immediate: true },
)

const canBindExisting = computed(() => !binding.value && selectedAddressId.value != null)

async function handleBindExisting(): Promise<void> {
  if (!canBindExisting.value || selectedAddressId.value == null) return
  binding.value = true
  try {
    await bindAddressToLine({
      fulfillmentLineId: props.row.fulfillmentLineId,
      customerAddressId: selectedAddressId.value,
    })
    feedback.success(t('fulfillmentGrid.address.bound'))
    emit('bound')
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    binding.value = false
  }
}

// ── Author new ──

const recipientName = ref('')
const phone = ref('')
const region = ref('')
const detail = ref('')
const creating = ref(false)

function resetCreateForm(): void {
  recipientName.value = ''
  phone.value = ''
  region.value = ''
  detail.value = ''
}

const canCreate = computed(
  () =>
    !creating.value &&
    recipientName.value.trim().length > 0 &&
    phone.value.trim().length > 0 &&
    detail.value.trim().length > 0,
)

async function handleCreateAndBind(): Promise<void> {
  const profileId = props.row.customerProfileId
  if (!canCreate.value || !profileId) return
  creating.value = true
  try {
    const address = await createAddress({
      customerProfileId: profileId,
      label: '',
      recipientName: recipientName.value.trim(),
      phone: phone.value.trim(),
      country: '',
      province: region.value.trim(),
      city: '',
      district: '',
      addressLine1: detail.value.trim(),
      addressLine2: '',
      postalCode: '',
      isDefault: false,
      isTest: false,
      validationStatus: 'unvalidated',
      validationDetail: '',
      extraData: '',
    })
    await bindAddressToLine({
      fulfillmentLineId: props.row.fulfillmentLineId,
      customerAddressId: address.id,
    })
    feedback.success(t('fulfillmentGrid.address.bound'))
    resetCreateForm()
    emit('bound')
    mode.value = 'pick'
    await loadAddresses()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    creating.value = false
  }
}

function addressSummary(address: dto.CustomerAddressDTO): string {
  return [address.province, address.city, address.district, address.addressLine1].filter(Boolean).join(' ')
}
</script>

<template>
  <div class="inline-address-editor">
    <p v-if="!hasCustomerProfile" class="inline-address-editor__hint">{{ t('fulfillmentGrid.address.noAddresses') }}</p>
    <template v-else>
      <div class="inline-address-editor__tabs">
        <NButton size="small" :type="mode === 'pick' ? 'primary' : 'default'" @click="mode = 'pick'">
          {{ t('fulfillmentGrid.address.pickExisting') }}
        </NButton>
        <NButton size="small" :type="mode === 'create' ? 'primary' : 'default'" @click="mode = 'create'">
          {{ t('fulfillmentGrid.address.createNew') }}
        </NButton>
      </div>

      <div v-if="mode === 'pick'" class="inline-address-editor__pick">
        <p v-if="addressesLoaded && addresses.length === 0" class="inline-address-editor__hint">
          {{ t('fulfillmentGrid.address.noAddresses') }}
        </p>
        <NRadioGroup v-else v-model:value="selectedAddressId" class="inline-address-editor__list">
          <NRadio
            v-for="address in addresses"
            :key="address.id"
            :value="address.id"
            :disabled="binding"
            class="inline-address-editor__option"
          >
            <span class="inline-address-editor__option-text">
              <strong>{{ address.recipientName }}</strong>
              <span>{{ address.phone }}</span>
              <span>{{ addressSummary(address) }}</span>
            </span>
          </NRadio>
        </NRadioGroup>
        <NButton type="primary" :loading="binding" :disabled="!canBindExisting" @click="handleBindExisting">
          {{ t('fulfillmentGrid.address.bind') }}
        </NButton>
      </div>

      <NForm v-else label-placement="top" class="inline-address-editor__create">
        <NFormItem :label="t('fulfillmentGrid.address.recipient')">
          <NInput v-model:value="recipientName" :disabled="creating" />
        </NFormItem>
        <NFormItem :label="t('fulfillmentGrid.address.phone')">
          <NInput v-model:value="phone" :disabled="creating" />
        </NFormItem>
        <NFormItem :label="t('fulfillmentGrid.address.region')">
          <NInput v-model:value="region" :disabled="creating" />
        </NFormItem>
        <NFormItem :label="t('fulfillmentGrid.address.detail')">
          <NInput v-model:value="detail" :disabled="creating" />
        </NFormItem>
        <NButton type="primary" :loading="creating" :disabled="!canCreate" @click="handleCreateAndBind">
          {{ t('fulfillmentGrid.address.save') }}
        </NButton>
      </NForm>
    </template>
  </div>
</template>

<style scoped>
.inline-address-editor {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.inline-address-editor__hint {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.inline-address-editor__tabs {
  display: flex;
  gap: var(--space-2);
}

.inline-address-editor__pick,
.inline-address-editor__create {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  align-items: flex-start;
}

.inline-address-editor__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  width: 100%;
}

.inline-address-editor__option {
  display: flex;
  align-items: flex-start;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--card-bg);
}

.inline-address-editor__option-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-left: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.inline-address-editor__option-text span {
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}
</style>
