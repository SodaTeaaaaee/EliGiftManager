<template>
  <div class="page">
    <div class="page-header">
      <h1>{{ t("address.title") }}</h1>
      <p>{{ t("address.subtitle") }}</p>
    </div>

    <div class="toolbar">
      <n-select
        v-model:value="selectedProfileId"
        :options="profileOptions"
        :placeholder="t('address.selectProfile')"
        style="width: 280px"
        @update:value="loadAddresses"
      />
      <n-button type="primary" :disabled="!selectedProfileId" @click="showCreate = true">
        {{ t("address.create") }}
      </n-button>
    </div>

    <n-data-table
      v-if="selectedProfileId"
      :columns="columns"
      :data="addresses"
      :loading="loading"
      size="small"
    />
    <n-empty v-else :description="t('address.selectProfile')" />

    <!-- Create/Edit Modal -->
    <n-modal v-model:show="showCreate" :title="editingId ? t('address.edit') : t('address.create')">
      <n-card style="width: 640px" :bordered="false" role="dialog">
        <n-form :model="form" label-placement="left" label-width="100">
          <n-form-item :label="t('address.label')">
            <n-input v-model:value="form.label" />
          </n-form-item>
          <n-form-item :label="t('address.recipientName')">
            <n-input v-model:value="form.recipientName" />
          </n-form-item>
          <n-form-item :label="t('address.phone')">
            <n-input v-model:value="form.phone" />
          </n-form-item>
          <n-form-item :label="t('address.country')">
            <n-input v-model:value="form.country" />
          </n-form-item>
          <n-form-item :label="t('address.province')">
            <n-input v-model:value="form.province" />
          </n-form-item>
          <n-form-item :label="t('address.city')">
            <n-input v-model:value="form.city" />
          </n-form-item>
          <n-form-item :label="t('address.district')">
            <n-input v-model:value="form.district" />
          </n-form-item>
          <n-form-item :label="t('address.addressLine1')">
            <n-input v-model:value="form.addressLine1" />
          </n-form-item>
          <n-form-item :label="t('address.addressLine2')">
            <n-input v-model:value="form.addressLine2" />
          </n-form-item>
          <n-form-item :label="t('address.postalCode')">
            <n-input v-model:value="form.postalCode" />
          </n-form-item>
          <n-form-item :label="t('address.isDefault')">
            <n-switch v-model:value="form.isDefault" />
          </n-form-item>
          <n-form-item :label="t('address.isTest')">
            <n-switch v-model:value="form.isTest" />
          </n-form-item>
          <n-form-item :label="t('address.validationStatus')">
            <n-select
              v-model:value="form.validationStatus"
              :options="statusOptions"
            />
          </n-form-item>
        </n-form>
        <template #footer>
          <n-space justify="end">
            <n-button @click="showCreate = false; editingId = null">{{ t("common.cancel") }}</n-button>
            <n-button type="primary" @click="saveAddress">{{ t("common.save") }}</n-button>
          </n-space>
        </template>
      </n-card>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, h } from "vue";
import {
  NButton, NDataTable, NForm, NFormItem, NInput, NModal, NCard,
  NSpace, NSelect, NSwitch, NEmpty, useMessage,
} from "naive-ui";
import { useI18n } from "@/shared/i18n";
import {
  listProfiles, createAddress, updateAddress, deleteAddress,
  listAddressesByProfile,
} from "@/shared/lib/wails/app";
import type { CustomerAddressDTO } from "@/entities/address";
import type { DataTableColumn } from "naive-ui";

const { t } = useI18n();
const message = useMessage();

const selectedProfileId = ref<number | null>(null);
const profileOptions = ref<{ label: string; value: number }[]>([]);
const addresses = ref<CustomerAddressDTO[]>([]);
const loading = ref(false);
const showCreate = ref(false);
const editingId = ref<number | null>(null);

const statusOptions = [
  { label: "unvalidated", value: "unvalidated" },
  { label: "valid", value: "valid" },
  { label: "invalid", value: "invalid" },
];

const emptyForm = () => ({
  customerProfileId: selectedProfileId.value ?? 0,
  label: "",
  recipientName: "",
  phone: "",
  country: "",
  province: "",
  city: "",
  district: "",
  addressLine1: "",
  addressLine2: "",
  postalCode: "",
  isDefault: false,
  isTest: false,
  validationStatus: "unvalidated",
  validationDetail: "",
  extraData: "",
});

const form = ref(emptyForm());

const columns: DataTableColumn<CustomerAddressDTO>[] = [
  { title: "ID", key: "id", width: 60 },
  { title: t("address.label"), key: "label", ellipsis: { tooltip: true } },
  { title: t("address.recipientName"), key: "recipientName", ellipsis: { tooltip: true } },
  { title: t("address.city"), key: "city" },
  { title: t("address.isDefault"), key: "isDefault", width: 80, render: (r) => r.isDefault ? "✓" : "" },
  {
    title: t("address.validationStatus"), key: "validationStatus", width: 100,
    render: (r) => {
      const map: Record<string, string> = { unvalidated: t("address.status.unvalidated"), valid: t("address.status.valid"), invalid: t("address.status.invalid") };
      return map[r.validationStatus] ?? r.validationStatus;
    },
  },
  {
    title: "", key: "actions", width: 160,
    render: (r) =>
      h(NSpace, null, {
        default: () => [
          h(NButton, { size: "tiny", onClick: () => editAddress(r) }, { default: () => t("address.edit") }),
          h(NButton, { size: "tiny", type: "error", onClick: () => removeAddress(r.id) }, { default: () => t("address.delete") }),
        ],
      }),
  },
];

async function loadProfiles() {
  const profiles = await listProfiles();
  profileOptions.value = profiles.map((p: any) => ({ label: `${p.profileKey} (ID:${p.id})`, value: p.id }));
}

async function loadAddresses() {
  if (!selectedProfileId.value) return;
  loading.value = true;
  addresses.value = await listAddressesByProfile(selectedProfileId.value);
  loading.value = false;
}

function editAddress(a: CustomerAddressDTO) {
  form.value = {
    customerProfileId: a.customerProfileId,
    label: a.label,
    recipientName: a.recipientName,
    phone: a.phone,
    country: a.country,
    province: a.province,
    city: a.city,
    district: a.district,
    addressLine1: a.addressLine1,
    addressLine2: a.addressLine2,
    postalCode: a.postalCode,
    isDefault: a.isDefault,
    isTest: a.isTest,
    validationStatus: a.validationStatus,
    validationDetail: a.validationDetail,
    extraData: a.extraData,
  };
  editingId.value = a.id;
  showCreate.value = true;
}

async function saveAddress() {
  if (editingId.value) {
    await updateAddress({ id: editingId.value, ...form.value });
  } else {
    await createAddress(form.value);
  }
  showCreate.value = false;
  editingId.value = null;
  form.value = emptyForm();
  await loadAddresses();
  message.success(t("common.save"));
}

async function removeAddress(id: number) {
  await deleteAddress(id);
  await loadAddresses();
}

loadProfiles();
</script>

<style scoped>
.page { padding: 24px; max-width: 1200px; }
.page-header { margin-bottom: 20px; }
.page-header h1 { font-size: 1.5rem; font-weight: 700; margin: 0; }
.page-header p { color: var(--text-muted); margin: 4px 0 0; }
.toolbar { display: flex; gap: 12px; margin-bottom: 16px; align-items: center; }
</style>
