<template>
  <div class="profile-management-page">
    <h1 class="text-xl font-medium mb-4">集成配置管理</h1>

    <n-space vertical size="large">
      <n-space>
        <n-button type="primary" @click="openCreateDrawer">
          创建 Profile
        </n-button>
        <n-button @click="seedDefaults" :loading="seeding">
          初始化默认配置
        </n-button>
      </n-space>

      <n-alert v-if="error" type="error" :title="error" closable @close="error = ''" />
      <n-alert v-if="successMsg" type="success" :title="successMsg" closable @close="successMsg = ''" />

      <n-data-table
        :columns="columns"
        :data="profiles"
        :loading="loading"
        :row-key="(row: dto.IntegrationProfileDTO) => row.id"
        size="small"
      />
    </n-space>

    <!-- Create / Edit Drawer -->
    <n-drawer v-model:show="drawerVisible" :width="520" placement="right">
      <n-drawer-content :title="editingId ? '编辑 Profile' : '创建 Profile'">
        <n-form label-placement="left" label-width="160" :model="formData">
          <n-form-item label="Profile Key" required>
            <n-input v-model:value="formData.profileKey" placeholder="e.g. bilibili_live" />
          </n-form-item>
          <n-form-item label="Source Channel">
            <n-input v-model:value="formData.sourceChannel" placeholder="e.g. taobao" />
          </n-form-item>
          <n-form-item label="Source Surface">
            <n-input v-model:value="formData.sourceSurface" placeholder="e.g. storefront" />
          </n-form-item>
          <n-form-item label="Demand Kind">
            <n-select
              v-model:value="formData.demandKind"
              :options="demandKindOptions"
              placeholder="选择"
            />
          </n-form-item>
          <n-form-item label="Allocation Strategy">
            <n-select
              v-model:value="formData.initialAllocationStrategy"
              :options="allocationStrategyOptions"
              placeholder="选择"
            />
          </n-form-item>
          <n-form-item label="Identity Strategy">
            <n-select
              v-model:value="formData.identityStrategy"
              :options="identityStrategyOptions"
              placeholder="选择"
            />
          </n-form-item>
          <n-form-item label="Entitlement Authority">
            <n-select
              v-model:value="formData.entitlementAuthorityMode"
              :options="entitlementAuthorityModeOptions"
              placeholder="选择"
            />
          </n-form-item>
          <n-form-item label="Recipient Input Mode">
            <n-select
              v-model:value="formData.recipientInputMode"
              :options="recipientInputModeOptions"
              placeholder="选择"
            />
          </n-form-item>
          <n-form-item label="Reference Strategy">
            <n-select
              v-model:value="formData.referenceStrategy"
              :options="referenceStrategyOptions"
              placeholder="选择"
            />
          </n-form-item>
          <n-form-item label="Tracking Sync Mode">
            <n-select
              v-model:value="formData.trackingSyncMode"
              :options="trackingSyncModeOptions"
              placeholder="选择"
            />
          </n-form-item>
          <n-form-item label="Closure Policy">
            <n-select
              v-model:value="formData.closurePolicy"
              :options="closurePolicyOptions"
              placeholder="选择"
            />
          </n-form-item>
          <n-form-item label="Connector Key">
            <n-input v-model:value="formData.connectorKey" placeholder="e.g. taobao_v2" />
          </n-form-item>
          <n-form-item label="Supported Locales">
            <n-input v-model:value="formData.supportedLocales" placeholder="e.g. zh-CN,en" />
          </n-form-item>
          <n-form-item label="Default Locale">
            <n-input v-model:value="formData.defaultLocale" placeholder="e.g. zh-CN" />
          </n-form-item>

          <n-divider />

          <n-form-item label="Partial Shipment">
            <n-switch v-model:value="formData.supportsPartialShipment" />
          </n-form-item>
          <n-form-item label="API Import">
            <n-switch v-model:value="formData.supportsApiImport" />
          </n-form-item>
          <n-form-item label="API Export">
            <n-switch v-model:value="formData.supportsApiExport" />
          </n-form-item>
          <n-form-item label="Carrier Mapping">
            <n-switch v-model:value="formData.requiresCarrierMapping" />
          </n-form-item>
          <n-form-item label="External Order No">
            <n-switch v-model:value="formData.requiresExternalOrderNo" />
          </n-form-item>
          <n-form-item label="Manual Closure">
            <n-switch v-model:value="formData.allowsManualClosure" />
          </n-form-item>

          <n-form-item label="Extra Data">
            <n-input
              v-model:value="formData.extraData"
              type="textarea"
              placeholder="{}"
              :rows="3"
            />
          </n-form-item>
        </n-form>

        <template #footer>
          <n-space justify="end">
            <n-button @click="drawerVisible = false">取消</n-button>
            <n-button type="primary" @click="submitForm" :loading="submitting">
              {{ editingId ? '保存' : '创建' }}
            </n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from "vue";
import {
  NButton,
  NDataTable,
  NSpace,
  NAlert,
  NDrawer,
  NDrawerContent,
  NForm,
  NFormItem,
  NInput,
  NSelect,
  NSwitch,
  NDivider,
  useDialog,
} from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import {
  listProfiles,
  createProfile,
  updateProfile,
  deleteProfile,
  seedDefaultProfiles,
} from "@/shared/lib/wails/app";
import { dto } from "@/../wailsjs/go/models";

// ── Options ──

const demandKindOptions = [
  { label: "Membership Entitlement", value: "membership_entitlement" },
  { label: "Retail Order", value: "retail_order" },
];

const allocationStrategyOptions = [
  { label: "Policy Driven", value: "policy_driven" },
  { label: "Demand Driven", value: "demand_driven" },
];

const trackingSyncModeOptions = [
  { label: "API Push", value: "api_push" },
  { label: "Document Export", value: "document_export" },
  { label: "Manual Confirmation", value: "manual_confirmation" },
  { label: "Unsupported", value: "unsupported" },
];

const closurePolicyOptions = [
  { label: "Close After Sync", value: "close_after_sync" },
  { label: "Close After Manual Confirmation", value: "close_after_manual_confirmation" },
  { label: "Close After Shipment", value: "close_after_shipment" },
];

const identityStrategyOptions = [
  { label: "Platform UID", value: "platform_uid" },
  { label: "Email", value: "email" },
  { label: "External Buyer ID", value: "external_buyer_id" },
];

const recipientInputModeOptions = [
  { label: "None", value: "none" },
  { label: "Platform Claim", value: "platform_claim" },
  { label: "External Form", value: "external_form" },
  { label: "Manual Collection", value: "manual_collection" },
];

const referenceStrategyOptions = [
  { label: "Member Level", value: "member_level" },
  { label: "Order Level", value: "order_level" },
  { label: "Order Line Level", value: "order_line_level" },
];

const entitlementAuthorityModeOptions = [
  { label: "Local Policy", value: "local_policy" },
  { label: "Upstream Platform", value: "upstream_platform" },
  { label: "Manual Grant Only", value: "manual_grant_only" },
];

// ── State ──

const profiles = ref<dto.IntegrationProfileDTO[]>([]);
const loading = ref(false);
const error = ref("");
const successMsg = ref("");
const seeding = ref(false);
const drawerVisible = ref(false);
const editingId = ref<number | null>(null);
const submitting = ref(false);
const dialog = useDialog();

// ── Form ──

const makeEmptyForm = () => ({
  profileKey: "",
  sourceChannel: "",
  sourceSurface: "",
  demandKind: "membership_entitlement",
  initialAllocationStrategy: "policy_driven",
  identityStrategy: "platform_uid",
  entitlementAuthorityMode: "local_policy",
  recipientInputMode: "none",
  referenceStrategy: "member_level",
  trackingSyncMode: "api_push",
  closurePolicy: "close_after_sync",
  supportsPartialShipment: false,
  supportsApiImport: false,
  supportsApiExport: false,
  requiresCarrierMapping: false,
  requiresExternalOrderNo: false,
  allowsManualClosure: false,
  connectorKey: "",
  supportedLocales: "",
  defaultLocale: "zh-CN",
  extraData: "",
});

const formData = reactive(makeEmptyForm());

function resetForm() {
  Object.assign(formData, makeEmptyForm());
}

// ── Table columns ──

const columns: DataTableColumns<dto.IntegrationProfileDTO> = [
  { title: "ID", key: "id", width: 60 },
  { title: "Profile Key", key: "profileKey", width: 180 },
  { title: "Source Channel", key: "sourceChannel", width: 130 },
  { title: "Demand Kind", key: "demandKind", width: 160 },
  { title: "Allocation Strategy", key: "initialAllocationStrategy", width: 150 },
  { title: "Tracking Sync", key: "trackingSyncMode", width: 120 },
  {
    title: "操作",
    key: "actions",
    width: 150,
    render(row) {
      return h(NSpace, { size: "small" }, () => [
        h(
          NButton,
          { size: "small", onClick: () => openEditDrawer(row) },
          () => "编辑",
        ),
        h(
          NButton,
          { size: "small", type: "error", onClick: () => confirmDelete(row) },
          () => "删除",
        ),
      ]);
    },
  },
];

// ── Actions ──

async function loadProfiles() {
  loading.value = true;
  try {
    profiles.value = await listProfiles();
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    loading.value = false;
  }
}

function openCreateDrawer() {
  resetForm();
  editingId.value = null;
  drawerVisible.value = true;
}

function openEditDrawer(row: dto.IntegrationProfileDTO) {
  editingId.value = row.id;
  Object.assign(formData, {
    profileKey: row.profileKey,
    sourceChannel: row.sourceChannel,
    sourceSurface: row.sourceSurface,
    demandKind: row.demandKind,
    initialAllocationStrategy: row.initialAllocationStrategy,
    identityStrategy: row.identityStrategy,
    entitlementAuthorityMode: row.entitlementAuthorityMode,
    recipientInputMode: row.recipientInputMode,
    referenceStrategy: row.referenceStrategy,
    trackingSyncMode: row.trackingSyncMode,
    closurePolicy: row.closurePolicy,
    supportsPartialShipment: row.supportsPartialShipment,
    supportsApiImport: row.supportsApiImport,
    supportsApiExport: row.supportsApiExport,
    requiresCarrierMapping: row.requiresCarrierMapping,
    requiresExternalOrderNo: row.requiresExternalOrderNo,
    allowsManualClosure: row.allowsManualClosure,
    connectorKey: row.connectorKey,
    supportedLocales: row.supportedLocales,
    defaultLocale: row.defaultLocale,
    extraData: row.extraData,
  });
  drawerVisible.value = true;
}

async function submitForm() {
  if (!formData.profileKey) {
    error.value = "Profile Key 不能为空";
    return;
  }
  submitting.value = true;
  error.value = "";
  try {
    if (editingId.value) {
      await updateProfile({ id: editingId.value, ...formData });
      successMsg.value = "Profile 已更新";
    } else {
      await createProfile({ ...formData });
      successMsg.value = "Profile 已创建";
    }
    drawerVisible.value = false;
    await loadProfiles();
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    submitting.value = false;
  }
}

function confirmDelete(row: dto.IntegrationProfileDTO) {
  dialog.warning({
    title: "确认删除",
    content: `确定要删除 Profile "${row.profileKey}" 吗？`,
    positiveText: "删除",
    negativeText: "取消",
    onPositiveClick: async () => {
      try {
        await deleteProfile(row.id);
        successMsg.value = `Profile "${row.profileKey}" 已删除`;
        await loadProfiles();
      } catch (e: any) {
        error.value = e?.message ?? String(e);
      }
    },
  });
}

async function seedDefaults() {
  seeding.value = true;
  error.value = "";
  try {
    await seedDefaultProfiles();
    successMsg.value = "默认配置已初始化";
    await loadProfiles();
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    seeding.value = false;
  }
}

// ── Lifecycle ──

onMounted(() => {
  loadProfiles();
});
</script>
