<template>
  <div class="template-management-page">
    <h1 class="text-xl font-medium mb-4">{{ t("template.title") }}</h1>

    <n-space vertical size="large">
      <n-space>
        <n-button type="primary" @click="openCreateDrawer">
          {{ t("template.create") }}
        </n-button>
      </n-space>

      <n-alert v-if="error" type="error" :title="error" closable @close="error = ''" />
      <n-alert v-if="successMsg" type="success" :title="successMsg" closable @close="successMsg = ''" />

      <n-data-table
        :columns="columns"
        :data="templates"
        :loading="loading"
        :row-key="(row: dto.DocumentTemplateDTO) => row.id"
        size="small"
      />
    </n-space>

    <!-- Create Drawer -->
    <n-drawer v-model:show="drawerVisible" :width="520" placement="right">
      <n-drawer-content :title="t('template.create')">
        <n-form
          ref="formRef"
          label-placement="left"
          label-width="120"
          :model="formData"
          :rules="formRules"
        >
          <n-form-item :label="t('template.templateKey')" path="templateKey">
            <n-input
              v-model:value="formData.templateKey"
              placeholder="e.g. bilibili_entitlement_csv_v1"
            />
          </n-form-item>
          <n-form-item :label="t('template.documentType')" path="documentType">
            <n-select
              v-model:value="formData.documentType"
              :options="documentTypeOptions"
              placeholder="Select document type"
            />
          </n-form-item>
          <n-form-item :label="t('template.format')" path="format">
            <n-select
              v-model:value="formData.format"
              :options="formatOptions"
              placeholder="Select format"
            />
          </n-form-item>
          <n-form-item :label="t('template.mappingRules')" path="mappingRules">
            <n-input
              v-model:value="formData.mappingRules"
              type="textarea"
              placeholder='{"col_a": "field_b", ...}'
              :rows="5"
            />
          </n-form-item>
          <n-form-item label="Extra Data">
            <n-input
              v-model:value="formData.extraData"
              type="textarea"
              placeholder='{"encoding": "utf-8"}'
              :rows="3"
            />
          </n-form-item>
        </n-form>

        <template #footer>
          <n-space justify="end">
            <n-button @click="drawerVisible = false">{{ t("common.cancel") }}</n-button>
            <n-button type="primary" @click="submitForm" :loading="submitting">
              {{ t("template.create") }}
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
  NDrawer,
  NDrawerContent,
  NForm,
  NFormItem,
  NInput,
  NSelect,
} from "naive-ui";
import type { DataTableColumns, FormInst, FormRules } from "naive-ui";
import {
  listDocumentTemplates,
  createDocumentTemplate,
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

const formatOptions = [
  { label: "CSV", value: "csv" },
  { label: "XLSX", value: "xlsx" },
  { label: "JSON", value: "json" },
  { label: "API Payload", value: "api_payload" },
];

// ── State ──

const templates = ref<dto.DocumentTemplateDTO[]>([]);
const loading = ref(false);
const error = ref("");
const successMsg = ref("");
const drawerVisible = ref(false);
const submitting = ref(false);
const formRef = ref<FormInst | null>(null);

// ── Form ──

const makeEmptyForm = () => ({
  templateKey: "",
  documentType: "import_entitlement",
  format: "csv",
  mappingRules: "",
  extraData: "",
});

const formData = reactive(makeEmptyForm());

function resetForm() {
  Object.assign(formData, makeEmptyForm());
}

const formRules: FormRules = {
  templateKey: [
    { required: true, message: "模板标识不能为空", trigger: "blur" },
  ],
  documentType: [
    { required: true, message: "请选择文档类型", trigger: "change" },
  ],
  format: [
    { required: true, message: "请选择格式", trigger: "change" },
  ],
};

// ── Table columns ──

const columns = computed<DataTableColumns<dto.DocumentTemplateDTO>>(() => [
  { title: "ID", key: "id", width: 60 },
  { title: t("template.templateKey"), key: "templateKey", width: 220 },
  { title: t("template.documentType"), key: "documentType", width: 200 },
  { title: t("template.format"), key: "format", width: 100 },
  { title: "Created At", key: "createdAt", width: 180 },
]);

// ── Actions ──

async function loadTemplates() {
  loading.value = true;
  try {
    templates.value = await listDocumentTemplates();
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    loading.value = false;
  }
}

function openCreateDrawer() {
  resetForm();
  drawerVisible.value = true;
}

async function submitForm() {
  try {
    await formRef.value?.validate();
  } catch {
    return;
  }
  submitting.value = true;
  error.value = "";
  try {
    await createDocumentTemplate({ ...formData });
    successMsg.value = "Template created";
    drawerVisible.value = false;
    await loadTemplates();
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    submitting.value = false;
  }
}

// ── Lifecycle ──

onMounted(() => {
  loadTemplates();
});
</script>
