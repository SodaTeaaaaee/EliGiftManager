<script setup lang="ts">
import { computed, h, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NDivider,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NModal,
  NPopconfirm,
  NSelect,
  NSpace,
  NSpin,
  NSwitch,
  NTag,
  useMessage,
} from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import {
  ArrowBackOutline,
  AddCircleOutline,
  TrashOutline,
} from "@vicons/ionicons5";
import {
  bindTemplateToProfile,
  createDocumentTemplate,
  createProfile,
  getProfile,
  listBindingsByProfile,
  listConnectorCapabilities,
  listDocumentTemplates,
  updateProfile,
} from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

import GlassCard from "@/shared/ui/GlassCard.vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const message = useMessage();

// ── Mode detection ──
const profileId = computed(() => Number(route.params.id) || 0);
const isCreateMode = computed(
  () => profileId.value === 0 || route.query.mode === "create",
);

// ── State ──
const profile = ref<dto.IntegrationProfileDTO | null>(null);
const bindings = ref<dto.ProfileTemplateBindingDTO[]>([]);
const allTemplates = ref<dto.DocumentTemplateDTO[]>([]);
const connectorCapabilities = ref<Record<string, any>>({});
const loading = ref(false);
const saving = ref(false);
const error = ref("");

const form = reactive({
  id: 0,
  profileKey: "",
  sourceChannel: "",
  sourceSurface: "",
  demandKind: "membership_entitlement",
  initialAllocationStrategy: "policy_driven",
  identityStrategy: "platform_uid",
  entitlementAuthorityMode: "upstream_platform",
  recipientInputMode: "platform_claim",
  referenceStrategy: "member_level",
  trackingSyncMode: "manual_confirmation",
  closurePolicy: "close_after_manual_confirmation",
  supportsPartialShipment: false,
  supportsApiImport: false,
  supportsApiExport: false,
  requiresCarrierMapping: false,
  requiresExternalOrderNo: false,
  allowsManualClosure: true,
  connectorKey: "",
  supportedLocales: "zh-CN,en",
  defaultLocale: "zh-CN",
  extraData: "",
});

// ── Options ──
const demandKindOptions = [
  { label: "Membership Entitlement", value: "membership_entitlement" },
  { label: "Retail Order", value: "retail_order" },
];
const allocationStrategyOptions = [
  { label: "Policy Driven", value: "policy_driven" },
  { label: "Demand Driven", value: "demand_driven" },
];
const identityStrategyOptions = [
  { label: "Platform UID", value: "platform_uid" },
  { label: "Email", value: "email" },
  { label: "External Buyer ID", value: "external_buyer_id" },
];
const entitlementAuthorityOptions = [
  { label: "Local Policy", value: "local_policy" },
  { label: "Upstream Platform", value: "upstream_platform" },
  { label: "Manual Grant Only", value: "manual_grant_only" },
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

const documentTypeOptions = [
  { label: "Import Entitlement", value: "import_entitlement" },
  { label: "Import Sales Order", value: "import_sales_order" },
  { label: "Import Product Catalog", value: "import_product_catalog" },
  { label: "Export Supplier Order", value: "export_supplier_order" },
  { label: "Import Supplier Shipment", value: "import_supplier_shipment" },
  { label: "Export Source Tracking Update", value: "export_source_tracking_update" },
];

// ── Loaders ──
async function loadProfile() {
  if (isCreateMode.value) return;
  loading.value = true;
  error.value = "";
  try {
    profile.value = await getProfile(profileId.value);
    if (profile.value) {
      Object.assign(form, profile.value);
    }
    bindings.value = await listBindingsByProfile(profileId.value);
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

async function loadLookups() {
  try {
    allTemplates.value = await listDocumentTemplates();
    connectorCapabilities.value = await listConnectorCapabilities();
  } catch (e: unknown) {
    /* non-fatal */
  }
}

watch(profileId, () => {
  void loadProfile();
});

onMounted(async () => {
  await loadLookups();
  await loadProfile();
});

// ── Save / create ──
async function save() {
  if (!form.profileKey || !form.sourceChannel) {
    message.warning("Profile Key and Source Channel are required");
    return;
  }
  saving.value = true;
  error.value = "";
  try {
    if (isCreateMode.value) {
      const created = await createProfile({
        profileKey: form.profileKey,
        sourceChannel: form.sourceChannel,
        sourceSurface: form.sourceSurface,
        demandKind: form.demandKind,
        initialAllocationStrategy: form.initialAllocationStrategy,
        identityStrategy: form.identityStrategy,
        entitlementAuthorityMode: form.entitlementAuthorityMode,
        recipientInputMode: form.recipientInputMode,
        referenceStrategy: form.referenceStrategy,
        trackingSyncMode: form.trackingSyncMode,
        closurePolicy: form.closurePolicy,
        supportsPartialShipment: form.supportsPartialShipment,
        supportsApiImport: form.supportsApiImport,
        supportsApiExport: form.supportsApiExport,
        requiresCarrierMapping: form.requiresCarrierMapping,
        requiresExternalOrderNo: form.requiresExternalOrderNo,
        allowsManualClosure: form.allowsManualClosure,
        connectorKey: form.connectorKey,
        supportedLocales: form.supportedLocales,
        defaultLocale: form.defaultLocale,
        extraData: form.extraData,
      });
      message.success("Profile created");
      router.replace(`/profiles/${created.id}`);
    } else {
      await updateProfile({ ...form, id: profileId.value });
      message.success("Profile saved");
      await loadProfile();
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    saving.value = false;
  }
}

function goBack() {
  router.push("/profiles");
}

// ── Bindings ──
const bindModalVisible = ref(false);
const newBindModal = reactive({
  templateId: null as number | null,
  documentType: "import_entitlement",
});

async function submitBindTemplate() {
  if (!newBindModal.templateId) {
    message.warning("Choose a template");
    return;
  }
  try {
    await bindTemplateToProfile({
      integrationProfileId: profileId.value,
      documentType: newBindModal.documentType,
      templateId: newBindModal.templateId,
    });
    message.success("Template bound");
    bindModalVisible.value = false;
    newBindModal.templateId = null;
    bindings.value = await listBindingsByProfile(profileId.value);
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

const newTemplateModalVisible = ref(false);
const newTemplate = reactive({
  templateKey: "",
  documentType: "import_entitlement",
  format: "csv",
  mappingRules: "{}",
  extraData: "",
  bindAfterCreate: true,
});

async function submitCreateTemplate() {
  if (!newTemplate.templateKey) {
    message.warning("Template key required");
    return;
  }
  try {
    const created = await createDocumentTemplate({
      templateKey: newTemplate.templateKey,
      documentType: newTemplate.documentType,
      format: newTemplate.format,
      mappingRules: newTemplate.mappingRules,
      extraData: newTemplate.extraData,
    });
    message.success("Template created");
    if (newTemplate.bindAfterCreate && profileId.value > 0) {
      await bindTemplateToProfile({
        integrationProfileId: profileId.value,
        documentType: newTemplate.documentType,
        templateId: created.id,
      });
      message.success("Template bound to profile");
    }
    newTemplateModalVisible.value = false;
    newTemplate.templateKey = "";
    newTemplate.mappingRules = "{}";
    newTemplate.extraData = "";
    allTemplates.value = await listDocumentTemplates();
    bindings.value = await listBindingsByProfile(profileId.value);
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

// ── Bindings table ──
const bindingsAugmented = computed(() => {
  const tmplMap = new Map<number, dto.DocumentTemplateDTO>();
  for (const t of allTemplates.value) tmplMap.set(t.id, t);
  return bindings.value.map((b) => ({
    ...b,
    templateKey: tmplMap.get(b.templateId)?.templateKey || `#${b.templateId}`,
    templateFormat: tmplMap.get(b.templateId)?.format || "—",
  }));
});

const bindingColumns = computed<DataTableColumns<any>>(() => [
  { title: "ID", key: "id", width: 60 },
  { title: "Document Type", key: "documentType", width: 220 },
  { title: "Template", key: "templateKey" },
  { title: "Format", key: "templateFormat", width: 90 },
  {
    title: "Default",
    key: "isDefault",
    width: 80,
    render: (row) =>
      row.isDefault
        ? h(NTag, { size: "tiny", type: "success", round: true, bordered: false }, { default: () => "yes" })
        : "—",
  },
]);

const templateOptions = computed(() =>
  allTemplates.value.map((t) => ({
    label: `${t.templateKey} (${t.format})`,
    value: t.id,
  })),
);

// ── Capability flags ──
const capabilityFields: Array<{ key: keyof typeof form; label: string }> = [
  { key: "supportsPartialShipment", label: "Partial Shipment" },
  { key: "supportsApiImport", label: "API Import" },
  { key: "supportsApiExport", label: "API Export" },
  { key: "requiresCarrierMapping", label: "Requires Carrier Mapping" },
  { key: "requiresExternalOrderNo", label: "Requires External Order No" },
  { key: "allowsManualClosure", label: "Allows Manual Closure" },
];

// ── Connector display ──
const connectorEntry = computed(() => {
  if (!form.connectorKey) return null;
  return connectorCapabilities.value?.[form.connectorKey] ?? null;
});
</script>

<template>
  <div class="profile-detail-page">
    <!-- Top Bar -->
    <div class="detail-header">
      <NButton text @click="goBack">
        <template #icon>
          <NIcon><ArrowBackOutline /></NIcon>
        </template>
        {{ t("profileDetail.backToList") }}
      </NButton>

      <div class="detail-header-main">
        <div class="app-kicker">{{ t("nav.profiles") }}</div>
        <h1 class="detail-title">
          {{ isCreateMode ? "Create New Profile" : (form.profileKey || "—") }}
        </h1>
        <div v-if="!isCreateMode" class="detail-subtitle">
          {{ form.sourceChannel }} · {{ form.sourceSurface || "—" }}
        </div>
      </div>

      <NSpace>
        <NButton type="primary" :loading="saving" @click="save">
          {{ isCreateMode ? "Create" : "Save" }}
        </NButton>
      </NSpace>
    </div>

    <NAlert
      v-if="error"
      type="error"
      class="mb-4"
      :title="error"
      closable
      @close="error = ''"
    />

    <NSpin :show="loading">
      <div class="detail-sections">
        <!-- Section 1: Business Semantics -->
        <GlassCard>
          <div class="section-title">{{ t("profileDetail.sectionSemantics") }}</div>
          <NForm
            label-placement="left"
            label-width="180"
            :model="form"
            class="detail-form"
          >
            <NFormItem :label="t('profileDetail.profileKey')">
              <NInput v-model:value="form.profileKey" placeholder="bilibili_live_membership" />
            </NFormItem>
            <NFormItem :label="t('profileDetail.sourceChannel')">
              <NInput v-model:value="form.sourceChannel" placeholder="bilibili / patreon / fanbox" />
            </NFormItem>
            <NFormItem :label="t('profileDetail.sourceSurface')">
              <NInput v-model:value="form.sourceSurface" placeholder="live_room / shop_purchase / membership" />
            </NFormItem>
            <NFormItem :label="t('profileDetail.demandKind')">
              <NSelect v-model:value="form.demandKind" :options="demandKindOptions" />
            </NFormItem>
            <NFormItem :label="t('profileDetail.captureMode') || 'Allocation Strategy'">
              <NSelect
                v-model:value="form.initialAllocationStrategy"
                :options="allocationStrategyOptions"
              />
            </NFormItem>
            <NFormItem :label="t('profileDetail.entitlementAuthority')">
              <NSelect
                v-model:value="form.entitlementAuthorityMode"
                :options="entitlementAuthorityOptions"
              />
            </NFormItem>
            <NFormItem label="Identity Strategy">
              <NSelect
                v-model:value="form.identityStrategy"
                :options="identityStrategyOptions"
              />
            </NFormItem>
            <NFormItem :label="t('profileDetail.inputMode')">
              <NSelect
                v-model:value="form.recipientInputMode"
                :options="recipientInputModeOptions"
              />
            </NFormItem>
            <NFormItem label="Reference Strategy">
              <NSelect
                v-model:value="form.referenceStrategy"
                :options="referenceStrategyOptions"
              />
            </NFormItem>
          </NForm>
        </GlassCard>

        <!-- Section 2: Capabilities -->
        <GlassCard>
          <div class="section-title">{{ t("profileDetail.sectionCapabilities") }}</div>
          <div class="capability-grid">
            <div
              v-for="f in capabilityFields"
              :key="f.key"
              class="cap-item"
            >
              <NSwitch v-model:value="form[f.key] as boolean" />
              <span class="cap-label">{{ f.label }}</span>
            </div>
          </div>
        </GlassCard>

        <!-- Section 3: Bound Templates -->
        <GlassCard>
          <div class="section-header">
            <div class="section-title">{{ t("profileDetail.sectionTemplates") }}</div>
            <NSpace v-if="!isCreateMode">
              <NButton size="small" secondary @click="bindModalVisible = true">
                <template #icon>
                  <NIcon><AddCircleOutline /></NIcon>
                </template>
                {{ t("profileDetail.addTemplate") }}
              </NButton>
              <NButton size="small" type="primary" @click="newTemplateModalVisible = true">
                {{ t("profileDetail.newTemplate") }}
              </NButton>
            </NSpace>
          </div>

          <NEmpty
            v-if="isCreateMode || bindings.length === 0"
            :description="isCreateMode ? 'Save the profile first to manage templates.' : t('profileDetail.noTemplates')"
            class="empty-block"
          />
          <NDataTable
            v-else
            :columns="bindingColumns"
            :data="bindingsAugmented"
            :pagination="false"
            size="small"
          />
        </GlassCard>

        <!-- Section 4: Connector -->
        <GlassCard>
          <div class="section-title">{{ t("profileDetail.sectionConnector") }}</div>
          <NForm label-placement="left" label-width="180" :model="form">
            <NFormItem label="Connector Key">
              <NInput v-model:value="form.connectorKey" placeholder="e.g. shopee_sg_v2 / bilibili_api_prod" />
            </NFormItem>
            <NFormItem label="Tracking Sync Mode">
              <NSelect v-model:value="form.trackingSyncMode" :options="trackingSyncModeOptions" />
            </NFormItem>
          </NForm>
          <div v-if="connectorEntry" class="connector-info">
            <NTag size="small" :bordered="false">connector capabilities</NTag>
            <pre class="json-preview">{{ JSON.stringify(connectorEntry, null, 2) }}</pre>
          </div>
          <NEmpty v-else description="No connector bound. Profile is currently a strategy + capability declaration only." class="empty-block-sm" />
        </GlassCard>

        <!-- Section 5: Closure Strategy -->
        <GlassCard>
          <div class="section-title">{{ t("profileDetail.sectionClosure") }}</div>
          <NForm label-placement="left" label-width="180" :model="form">
            <NFormItem :label="t('profileDetail.closureStrategy')">
              <NSelect v-model:value="form.closurePolicy" :options="closurePolicyOptions" />
            </NFormItem>
            <NFormItem label="Allows Manual Closure">
              <NSwitch v-model:value="form.allowsManualClosure" />
            </NFormItem>
            <NFormItem label="Supported Locales">
              <NInput v-model:value="form.supportedLocales" placeholder="zh-CN,en,ja" />
            </NFormItem>
            <NFormItem label="Default Locale">
              <NInput v-model:value="form.defaultLocale" placeholder="zh-CN" />
            </NFormItem>
            <NFormItem label="Extra Data (JSON)">
              <NInput
                v-model:value="form.extraData"
                type="textarea"
                :rows="3"
                placeholder='{"webhook_url": "https://...", "timeout_ms": 5000}'
              />
            </NFormItem>
          </NForm>
        </GlassCard>

        <!-- Footer note -->
        <NAlert type="info">
          {{ t("profileDetail.notesHint") }}
        </NAlert>
      </div>
    </NSpin>

    <!-- Bind Existing Template Modal -->
    <NModal v-model:show="bindModalVisible" preset="card" title="Bind Template" style="width: 480px;">
      <NForm label-placement="left" label-width="140">
        <NFormItem label="Document Type">
          <NSelect v-model:value="newBindModal.documentType" :options="documentTypeOptions" />
        </NFormItem>
        <NFormItem label="Template">
          <NSelect
            v-model:value="newBindModal.templateId"
            :options="templateOptions"
            placeholder="Choose a template"
          />
        </NFormItem>
      </NForm>
      <NSpace justify="end" style="margin-top: 12px;">
        <NButton @click="bindModalVisible = false">Cancel</NButton>
        <NButton type="primary" @click="submitBindTemplate">Bind</NButton>
      </NSpace>
    </NModal>

    <!-- Create New Template Modal -->
    <NModal v-model:show="newTemplateModalVisible" preset="card" title="New Template" style="width: 520px;">
      <NForm label-placement="left" label-width="140">
        <NFormItem label="Template Key">
          <NInput v-model:value="newTemplate.templateKey" placeholder="e.g. bilibili_live_demand_csv" />
        </NFormItem>
        <NFormItem label="Document Type">
          <NSelect v-model:value="newTemplate.documentType" :options="documentTypeOptions" />
        </NFormItem>
        <NFormItem label="Format">
          <NSelect
            v-model:value="newTemplate.format"
            :options="[
              { label: 'CSV', value: 'csv' },
              { label: 'Excel', value: 'xlsx' },
              { label: 'JSON', value: 'json' },
            ]"
          />
        </NFormItem>
        <NFormItem label="Mapping Rules (JSON)">
          <NInput v-model:value="newTemplate.mappingRules" type="textarea" :rows="5" />
        </NFormItem>
        <NFormItem label="Bind After Create">
          <NSwitch v-model:value="newTemplate.bindAfterCreate" />
        </NFormItem>
      </NForm>
      <NSpace justify="end" style="margin-top: 12px;">
        <NButton @click="newTemplateModalVisible = false">Cancel</NButton>
        <NButton type="primary" @click="submitCreateTemplate">Create</NButton>
      </NSpace>
    </NModal>
  </div>
</template>

<style scoped>
.profile-detail-page {
  display: flex;
  flex-direction: column;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 18px;
  padding-bottom: 14px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.12);
}

.detail-header-main {
  flex: 1;
  min-width: 0;
}

.detail-title {
  font-size: 1.5rem;
  font-weight: 800;
  color: var(--text);
  margin: 4px 0 0;
  letter-spacing: -0.02em;
}

.detail-subtitle {
  color: var(--muted);
  font-size: 0.85rem;
  margin-top: 2px;
}

.detail-sections {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-title {
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--muted);
  margin-bottom: 14px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}

.detail-form {
  max-width: 720px;
}

.capability-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.cap-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 8px;
  background: rgba(148, 163, 184, 0.06);
}

.cap-label {
  font-size: 0.85rem;
  color: var(--text);
}

.connector-info {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.json-preview {
  background: rgba(148, 163, 184, 0.08);
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 0.75rem;
  font-family: monospace;
  margin: 0;
  overflow-x: auto;
}

.empty-block {
  padding: 32px 0;
}

.empty-block-sm {
  padding: 16px 0;
}

.mb-4 {
  margin-bottom: 16px;
}
</style>
