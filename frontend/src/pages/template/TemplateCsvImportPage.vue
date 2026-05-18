<template>
  <div class="page">
    <div class="page-header">
      <h1>{{ t("csvImport.title") }}</h1>
      <p>{{ t("csvImport.subtitle") }}</p>
    </div>

    <n-space vertical size="large">
      <n-form label-placement="left" label-width="140">
        <n-form-item :label="t('csvImport.selectProfile')">
          <n-select
            v-model:value="profileId"
            :options="profileOptions"
            style="width: 360px"
          />
        </n-form-item>
        <n-form-item :label="t('csvImport.selectDocType')">
          <n-select
            v-model:value="documentType"
            :options="docTypeOptions"
            style="width: 360px"
          />
        </n-form-item>
        <n-form-item :label="t('csvImport.sourceDocumentNo')">
          <n-input v-model:value="sourceDocumentNo" style="width: 360px" />
        </n-form-item>
        <n-form-item :label="t('csvImport.sourceCustomerRef')">
          <n-input v-model:value="sourceCustomerRef" style="width: 360px" />
        </n-form-item>
        <n-form-item :label="t('csvImport.csvData')">
          <n-input
            v-model:value="csvText"
            type="textarea"
            :rows="12"
            style="width: 600px; font-family: monospace; font-size:0.85rem"
            :placeholder='`[{"Name":"Product A","Qty":"2"},{"Name":"Product B","Qty":"1"}]`'
          />
        </n-form-item>
        <n-text depth="3" style="font-size:0.8rem">{{ t("csvImport.csvDataHint") }}</n-text>
      </n-form>

      <n-space>
        <n-button type="primary" :disabled="!canImport" @click="doImport">
          {{ t("csvImport.import") }}
        </n-button>
        <n-text v-if="rowCount > 0">{{ t("csvImport.rowCount", { n: String(rowCount) }) }}</n-text>
      </n-space>

      <n-card v-if="lastResult" title="Import Result" style="max-width:600px">
        <n-text>ID: {{ lastResult.id }} | {{ lastResult.kind }} | {{ lastResult.sourceDocumentNo }}</n-text>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import {
  NButton, NSelect, NInput, NForm, NFormItem, NSpace, NCard, NText, useMessage,
} from "naive-ui";
import { useI18n } from "@/shared/i18n";
import { listProfiles, importDemandFromCSV } from "@/shared/lib/wails/app";

const { t } = useI18n();
const message = useMessage();

const profileId = ref<number | null>(null);
const documentType = ref("import_entitlement");
const sourceDocumentNo = ref("");
const sourceCustomerRef = ref("");
const csvText = ref("");
const lastResult = ref<any>(null);

const profileOptions = ref<{ label: string; value: number }[]>([]);
const docTypeOptions = [
  { label: "import_entitlement", value: "import_entitlement" },
  { label: "import_sales_order", value: "import_sales_order" },
];

const rowCount = computed(() => {
  try { return JSON.parse(csvText.value).length; } catch { return 0; }
});

const canImport = computed(() =>
  profileId.value != null && csvText.value.trim().length > 0,
);

async function doImport() {
  if (!profileId.value) return;
  let rows: Record<string, string>[];
  try {
    rows = JSON.parse(csvText.value);
  } catch {
    message.error("Invalid JSON");
    return;
  }
  try {
    const result = await importDemandFromCSV({
      integrationProfileId: profileId.value,
      documentType: documentType.value,
      sourceDocumentNo: sourceDocumentNo.value,
      sourceCustomerRef: sourceCustomerRef.value,
      rows,
    });
    lastResult.value = result;
    message.success(t("csvImport.importSuccess"));
  } catch (e: any) {
    message.error(e?.toString() ?? "Import failed");
  }
}

async function loadProfiles() {
  const profiles = await listProfiles();
  profileOptions.value = profiles.map((p: any) => ({
    label: `${p.profileKey} (ID:${p.id})`,
    value: p.id,
  }));
}
loadProfiles();
</script>

<style scoped>
.page { padding: 24px; max-width: 900px; }
.page-header { margin-bottom: 20px; }
.page-header h1 { font-size: 1.5rem; font-weight: 700; margin: 0; }
.page-header p { color: var(--text-muted); margin: 4px 0 0; }
</style>
