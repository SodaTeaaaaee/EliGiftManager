<template>
  <div class="profile-template-binding-page">
    <h1 class="text-xl font-medium mb-4">{{ t("template.bindings") }}</h1>

    <n-space vertical size="large">
      <!-- Profile selector -->
      <n-form label-placement="left" label-width="100">
        <n-form-item :label="t('template.selectProfile')">
          <n-select
            v-model:value="selectedProfileId"
            :options="profileOptions"
            placeholder="Select profile"
            style="width: 320px"
            @update:value="onProfileChange"
          />
        </n-form-item>
      </n-form>

      <template v-if="selectedProfileId !== null">
        <n-space>
          <n-button type="primary" @click="openBindDrawer">
            {{ t("template.bindToProfile") }}
          </n-button>
        </n-space>

        <n-alert v-if="error" type="error" :title="error" closable @close="error = ''" />
        <n-alert v-if="successMsg" type="success" :title="successMsg" closable @close="successMsg = ''" />

        <n-data-table
          :columns="columns"
          :data="bindings"
          :loading="loading"
          :row-key="(row: dto.ProfileTemplateBindingDTO) => row.id"
          size="small"
        />
      </template>

      <n-empty v-else description="Select a profile first" />
    </n-space>

    <!-- Bind Template Drawer -->
    <n-drawer v-model:show="drawerVisible" :width="480" placement="right">
      <n-drawer-content :title="t('template.bindToProfile')">
        <n-form
          ref="formRef"
          label-placement="left"
          label-width="100"
          :model="formData"
          :rules="formRules"
        >
          <n-form-item :label="t('template.documentType')" path="documentType">
            <n-select
              v-model:value="formData.documentType"
              :options="documentTypeOptions"
              placeholder="Select document type"
            />
          </n-form-item>
          <n-form-item label="Template" path="templateId">
            <n-select
              v-model:value="formData.templateId"
              :options="templateOptions"
              placeholder="Select template"
            />
          </n-form-item>
          <n-form-item :label="t('template.isDefault')">
            <n-switch v-model:value="formData.isDefault" />
          </n-form-item>
        </n-form>

        <template #footer>
          <n-space justify="end">
            <n-button @click="drawerVisible = false">{{ t("common.cancel") }}</n-button>
            <n-button type="primary" @click="submitBind" :loading="submitting">
              {{ t("template.bindToProfile") }}
            </n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, h } from "vue";
import {
  NButton,
  NDataTable,
  NSpace,
  NAlert,
  NEmpty,
  NDrawer,
  NDrawerContent,
  NForm,
  NFormItem,
  NSelect,
  NSwitch,
} from "naive-ui";
import type { DataTableColumns, FormInst, FormRules } from "naive-ui";
import {
  listProfiles,
  listDocumentTemplates,
  listBindingsByProfile,
  bindTemplateToProfile,
} from "@/shared/lib/wails/app";
import { dto } from "@/../wailsjs/go/models";
import { useI18n } from "@/shared/i18n";

const { t } = useI18n();

// ── Options ──

const documentTypeOptions = [
  { label: "Import Entitlement", value: "import_entitlement" },
  { label: "Import Sales Order", value: "import_sales_order" },
  { label: "Import Product Catalog", value: "import_product_catalog" },
  { label: "Export Supplier Order", value: "export_supplier_order" },
  { label: "Import Supplier Shipment", value: "import_supplier_shipment" },
  { label: "Export Source Tracking Update", value: "export_source_tracking_update" },
];

// ── State ──

const profiles = ref<dto.IntegrationProfileDTO[]>([]);
const allTemplates = ref<dto.DocumentTemplateDTO[]>([]);
const bindings = ref<dto.ProfileTemplateBindingDTO[]>([]);
const selectedProfileId = ref<number | null>(null);
const loading = ref(false);
const error = ref("");
const successMsg = ref("");
const drawerVisible = ref(false);
const submitting = ref(false);
const formRef = ref<FormInst | null>(null);

// ── Computed ──

const profileOptions = computed(() =>
  profiles.value.map((p) => ({
    label: `${p.profileKey} (${p.sourceChannel})`,
    value: p.id,
  }))
);

const templateOptions = computed(() =>
  allTemplates.value.map((tmpl) => ({
    label: `[${tmpl.id}] ${tmpl.templateKey} (${tmpl.documentType} / ${tmpl.format})`,
    value: tmpl.id,
  }))
);

// ── Form ──

const makeEmptyForm = () => ({
  documentType: "import_entitlement",
  templateId: null as number | null,
  isDefault: false,
});

const formData = reactive(makeEmptyForm());

function resetForm() {
  Object.assign(formData, makeEmptyForm());
}

const formRules: FormRules = {
  documentType: [
    { required: true, message: "请选择文档类型", trigger: "change" },
  ],
  templateId: [
    { required: true, type: "number", message: "请选择模板", trigger: "change" },
  ],
};

// ── Table columns ──

const columns = computed<DataTableColumns<dto.ProfileTemplateBindingDTO>>(() => [
  { title: "ID", key: "id", width: 60 },
  { title: t("template.documentType"), key: "documentType", width: 220 },
  { title: "Template ID", key: "templateId", width: 90 },
  {
    title: t("template.isDefault"),
    key: "isDefault",
    width: 80,
    render(row) {
      return row.isDefault ? "✓" : "—";
    },
  },
  { title: "Created At", key: "createdAt", width: 180 },
]);

// ── Actions ──

async function loadProfiles() {
  try {
    profiles.value = await listProfiles();
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  }
}

async function loadAllTemplates() {
  try {
    allTemplates.value = await listDocumentTemplates();
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  }
}

async function loadBindings(profileId: number) {
  loading.value = true;
  try {
    bindings.value = await listBindingsByProfile(profileId);
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    loading.value = false;
  }
}

function onProfileChange(val: number | null) {
  if (val !== null) {
    error.value = "";
    successMsg.value = "";
    loadBindings(val);
  }
}

function openBindDrawer() {
  resetForm();
  drawerVisible.value = true;
}

async function submitBind() {
  try {
    await formRef.value?.validate();
  } catch {
    return;
  }
  if (selectedProfileId.value === null || formData.templateId === null) return;
  submitting.value = true;
  error.value = "";
  try {
    await bindTemplateToProfile({
      integrationProfileId: selectedProfileId.value,
      documentType: formData.documentType,
      templateId: formData.templateId,
      isDefault: formData.isDefault,
    });
    successMsg.value = "Template bound";
    drawerVisible.value = false;
    await loadBindings(selectedProfileId.value);
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    submitting.value = false;
  }
}

// ── Lifecycle ──

onMounted(() => {
  loadProfiles();
  loadAllTemplates();
});
</script>
