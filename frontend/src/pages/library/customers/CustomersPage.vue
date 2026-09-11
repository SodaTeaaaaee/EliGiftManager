<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
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
} from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { createCustomer, listCustomers } from '@/shared/api/bridge'
import type { CustomerProfile } from '@/entities/models'

const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const actionLoading = ref(false)
const customers = ref<CustomerProfile[]>([])

const showCreateModal = ref(false)
const createForm = ref({
  displayName: '',
  notes: '',
})

async function loadData() {
  loading.value = true
  try {
    customers.value = await listCustomers()
  } catch (err) {
    console.error('Failed to load customers:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadData()
})

function openDetail(customer: CustomerProfile) {
  void router.push(`/library/customers/${customer.ID}`)
}

async function handleCreate() {
  if (!createForm.value.displayName.trim()) return
  actionLoading.value = true
  try {
    const cust = await createCustomer(
      createForm.value.displayName.trim(),
      createForm.value.notes.trim(),
    )
    showCreateModal.value = false
    createForm.value = { displayName: '', notes: '' }
    await loadData()
    if (cust && cust.ID) {
      void router.push(`/library/customers/${cust.ID}`)
    }
  } catch (err) {
    console.error('Failed to create customer:', err)
  } finally {
    actionLoading.value = false
  }
}

const columns = [
  {
    title: '#',
    key: 'ID',
    width: 80,
    render(row: CustomerProfile) {
      return `#${row.ID}`
    },
  },
  {
    title: t('library.displayName'),
    key: 'DisplayName',
  },
  {
    title: t('library.notes'),
    key: 'Notes',
    ellipsis: { tooltip: true },
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 120,
    render(row: CustomerProfile) {
      return h(
        NButton,
        {
          size: 'small',
          type: 'primary',
          quaternary: true,
          onClick: () => openDetail(row),
        },
        { default: () => t('common.details') },
      )
    },
  },
]
</script>

<template>
  <div class="customers-page">
    <SectionCard :title="t('library.customersTab')">
      <template #actions>
        <NSpace>
          <NButton size="small" type="primary" @click="showCreateModal = true">
            {{ t('library.createCustomer') }}
          </NButton>
          <NButton size="small" :loading="loading" @click="loadData">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>

      <NSpin :show="loading">
        <div v-if="!customers.length" class="customers-page__empty">
          <EmptyState :title="t('library.emptyCustomers')" size="sm" />
        </div>
        <div v-else class="customers-page__table">
          <NDataTable
            :columns="columns"
            :data="customers"
            :row-key="(row: CustomerProfile) => row.ID"
            size="small"
          />
        </div>
      </NSpin>
    </SectionCard>

    <!-- Create Customer Modal -->
    <NModal
      v-model:show="showCreateModal"
      preset="card"
      :title="t('library.createCustomer')"
      style="width: 480px"
    >
      <NForm>
        <NFormItem :label="t('library.displayName')">
          <NInput
            v-model:value="createForm.displayName"
            :placeholder="t('library.displayName')"
          />
        </NFormItem>
        <NFormItem :label="t('library.notes')">
          <NInput
            v-model:value="createForm.notes"
            type="textarea"
            :placeholder="t('library.notes')"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!createForm.displayName.trim()"
            @click="handleCreate"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.customers-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.customers-page__empty {
  padding: var(--space-4) 0;
}

.customers-page__table {
  width: 100%;
}
</style>
