<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NSpace,
  NSpin,
  NSwitch,
  NTag,
} from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { createAddress, getCustomer, listAddresses } from '@/shared/api/bridge'
import type { CustomerProfile, RecipientAddress } from '@/entities/models'

const props = defineProps<{
  id?: string | number
}>()

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const customerId = computed(() => Number(props.id ?? route.params.id))
const loading = ref(false)
const actionLoading = ref(false)
const customer = ref<CustomerProfile | null>(null)
const addresses = ref<RecipientAddress[]>([])

const showAddressModal = ref(false)
const addressForm = ref({
  recipientName: '',
  phone: '',
  country: '中国',
  province: '',
  city: '',
  district: '',
  addressLine1: '',
  postalCode: '',
  isDefault: true,
})

async function loadData() {
  if (!customerId.value || isNaN(customerId.value)) return
  loading.value = true
  try {
    const [cust, addrs] = await Promise.all([
      getCustomer(customerId.value),
      listAddresses(customerId.value),
    ])
    customer.value = cust
    addresses.value = addrs
  } catch (err) {
    console.error('Failed to load customer detail:', err)
  } finally {
    loading.value = false
  }
}

watch(customerId, () => {
  void loadData()
})

onMounted(() => {
  void loadData()
})

function handleBack() {
  void router.push('/library/customers')
}

function openCreateAddress() {
  addressForm.value = {
    recipientName: customer.value?.DisplayName || '',
    phone: '',
    country: '中国',
    province: '',
    city: '',
    district: '',
    addressLine1: '',
    postalCode: '',
    isDefault: addresses.value.length === 0,
  }
  showAddressModal.value = true
}

async function handleSaveAddress() {
  if (!addressForm.value.recipientName.trim() || !addressForm.value.addressLine1.trim()) return
  actionLoading.value = true
  try {
    await createAddress({
      CustomerProfileID: customerId.value,
      RecipientName: addressForm.value.recipientName.trim(),
      Phone: addressForm.value.phone.trim(),
      Country: addressForm.value.country.trim(),
      Province: addressForm.value.province.trim(),
      City: addressForm.value.city.trim(),
      District: addressForm.value.district.trim(),
      AddressLine1: addressForm.value.addressLine1.trim(),
      PostalCode: addressForm.value.postalCode.trim(),
      IsDefault: addressForm.value.isDefault,
    })
    showAddressModal.value = false
    await loadData()
  } catch (err) {
    console.error('Failed to save address:', err)
  } finally {
    actionLoading.value = false
  }
}

const addressColumns = [
  {
    title: t('library.recipientName'),
    key: 'RecipientName',
    width: 140,
  },
  {
    title: t('library.phone'),
    key: 'Phone',
    width: 140,
  },
  {
    title: t('library.fullAddress'),
    key: 'AddressLine1',
    render(row: RecipientAddress) {
      const parts = [
        row.Province,
        row.City,
        row.District,
        row.AddressLine1,
        row.AddressLine2,
      ].filter(Boolean)
      return parts.length ? parts.join(' ') : row.AddressLine1 || '—'
    },
  },
  {
    title: t('library.isDefault'),
    key: 'IsDefault',
    width: 100,
    render(row: RecipientAddress) {
      return row.IsDefault
        ? h(NTag, { type: 'success', size: 'small' }, { default: () => t('common.yes') })
        : h(NTag, { type: 'default', size: 'small' }, { default: () => t('common.no') })
    },
  },
]
</script>

<template>
  <div class="customer-detail-page">
    <SectionCard :title="customer ? `${customer.DisplayName} (#${customer.ID})` : t('common.details')">
      <template #actions>
        <NButton size="small" @click="handleBack">
          {{ t('common.back') }}
        </NButton>
      </template>

      <NSpin :show="loading">
        <div v-if="customer" class="customer-detail-page__info">
          <p v-if="customer.Notes" class="customer-detail-page__notes">
            {{ customer.Notes }}
          </p>
        </div>
      </NSpin>
    </SectionCard>

    <SectionCard :title="t('library.addresses')">
      <template #actions>
        <NSpace>
          <NButton size="small" type="primary" @click="openCreateAddress">
            {{ t('library.createAddress') }}
          </NButton>
          <NButton size="small" :loading="loading" @click="loadData">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>

      <NSpin :show="loading">
        <div v-if="!addresses.length" class="customer-detail-page__empty">
          <EmptyState :title="t('library.addresses')" size="sm" />
        </div>
        <div v-else class="customer-detail-page__table">
          <NDataTable
            :columns="addressColumns"
            :data="addresses"
            :row-key="(row: RecipientAddress) => row.ID"
            size="small"
          />
        </div>
      </NSpin>
    </SectionCard>

    <!-- Create Address Modal -->
    <NModal
      v-model:show="showAddressModal"
      preset="card"
      :title="t('library.createAddress')"
      style="width: 520px"
    >
      <NForm label-placement="left" label-width="100">
        <NFormItem :label="t('library.recipientName')">
          <NInput v-model:value="addressForm.recipientName" />
        </NFormItem>
        <NFormItem :label="t('library.phone')">
          <NInput v-model:value="addressForm.phone" />
        </NFormItem>
        <NFormItem :label="t('library.fullAddress')">
          <NInput v-model:value="addressForm.addressLine1" />
        </NFormItem>
        <NFormItem :label="t('library.isDefault')">
          <NSwitch v-model:value="addressForm.isDefault" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showAddressModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!addressForm.recipientName.trim() || !addressForm.addressLine1.trim()"
            @click="handleSaveAddress"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.customer-detail-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.customer-detail-page__info {
  margin-bottom: var(--space-2);
}

.customer-detail-page__notes {
  color: var(--color-text-secondary);
  font-size: var(--font-size-sm);
  margin: 0;
}

.customer-detail-page__empty {
  padding: var(--space-4) 0;
}

.customer-detail-page__table {
  width: 100%;
}
</style>
