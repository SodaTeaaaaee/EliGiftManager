<script setup lang="ts">
import { computed, h, onMounted, ref, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NAlert, NButton, NCard, NDataTable, NEmpty, NTag, NSpace, NInput, NSelect, NModal, NForm, NFormItem, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { 
  generateParticipants, 
  listAssignedDemandsByWave, 
  listDemandLines, 
  mapDemandLines, 
  getWaveRoutingStats,
  listAddressesByProfile,
  createAddress,
  updateAddress
} from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { t } = useI18n();
const waveId = computed(() => Number(route.params.waveId) || 0);

const docs = ref<dto.DemandDocumentDTO[]>([]);
const loading = ref(false);
const applying = ref(false);
const participantsGenerated = ref(false);
const lineCache = ref<Record<number, dto.DemandLineDTO[]>>({});
const expandedKeys = ref<number[]>([]);
const blockedSummary = ref<string>("");

// Search, Filter and Stats state
const searchKeyword = ref("");
const channelFilter = ref<string | null>(null);
const routingStats = ref<any | null>(null);

// Inline Address Editor State
const addressModalVisible = ref(false);
const savingAddress = ref(false);
const addressProfileId = ref<number | null>(null);
const addressIdToUpdate = ref<number | null>(null);

const addressForm = reactive({
  recipientName: "",
  phone: "",
  country: "CN",
  province: "",
  city: "",
  district: "",
  addressLine1: "",
  addressLine2: "",
  postalCode: "",
  label: "Default Workspace",
});

const channelOptions = computed(() => {
  const channels = new Set(docs.value.map(d => d.sourceChannel));
  return Array.from(channels).map(c => ({ label: c, value: c }));
});

const filteredDocs = computed(() => {
  return docs.value.filter((doc) => {
    const matchesKeyword = !searchKeyword.value || 
      doc.sourceDocumentNo.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
      (doc.sourceCustomerRef && doc.sourceCustomerRef.toLowerCase().includes(searchKeyword.value.toLowerCase()));
    const matchesChannel = !channelFilter.value || doc.sourceChannel === channelFilter.value;
    return matchesKeyword && matchesChannel;
  });
});

const columns = computed<DataTableColumns<dto.DemandDocumentDTO>>(() => [
  { type: "expand", renderExpand: (row) => renderExpand(row) },
  { title: "ID", key: "id", width: 65 },
  { title: "Kind", key: "kind", width: 140 },
  { title: "Source Document No", key: "sourceDocumentNo", width: 180 },
  { title: "Customer Ref", key: "sourceCustomerRef", width: 160 },
  { 
    title: "Profile", 
    key: "customerProfileId", 
    width: 100,
    render: (row) => row.customerProfileId ? h(NTag, { size: "small", bordered: false }, { default: () => `Profile #${row.customerProfileId}` }) : "—"
  },
  { title: "Channel", key: "sourceChannel", width: 110 },
  { title: "Capture Mode", key: "captureMode", width: 120 },
]);

const lineColumns = computed<DataTableColumns<dto.DemandLineDTO>>(() => [
  { title: t("mapping.columns.line"), key: "sourceLineNo", width: 75 },
  { title: t("mapping.columns.type"), key: "lineType", width: 120 },
  { title: t("mapping.columns.title"), key: "externalTitle" },
  { 
    title: t("mapping.columns.disposition"), 
    key: "routingDisposition", 
    width: 140,
    render(row) {
      const type = row.routingDisposition === "blocked" ? "error" : row.routingDisposition === "deferred" ? "warning" : "success";
      return h(NTag, { type, size: "small", round: true, bordered: false }, { default: () => row.routingDisposition });
    }
  },
  { 
    title: t("mapping.columns.input"), 
    key: "recipientInputState", 
    width: 160,
    render(row) {
      const type = row.recipientInputState === "address_unavailable" || row.recipientInputState === "address_invalid" ? "warning" : "default";
      return h(NTag, { type, size: "small", bordered: false }, { default: () => row.recipientInputState || "none" });
    }
  },
  { title: t("mapping.columns.qty"), key: "requestedQuantity", width: 85 },
  {
    title: "Actions",
    key: "actions",
    width: 120,
    render(row) {
      const needsFix = row.recipientInputState === "address_unavailable" || row.recipientInputState === "address_invalid";
      if (needsFix) {
        return h(
          NButton,
          {
            size: "tiny",
            type: "warning",
            secondary: true,
            onClick: () => {
              const doc = docs.value.find(d => d.id === row.demandDocumentId);
              if (doc && doc.customerProfileId) {
                openAddressFixer(doc.customerProfileId);
              } else {
                message.error("Cannot resolve address: No profile associated with this document.");
              }
            }
          },
          { default: () => "Fix Address" }
        );
      }
      return null;
    }
  }
]);

async function loadDocs() {
  loading.value = true;
  try {
    const [docsResult, statsResult] = await Promise.all([
      listAssignedDemandsByWave(waveId.value),
      getWaveRoutingStats(waveId.value)
    ]);
    docs.value = docsResult;
    routingStats.value = statsResult;
  } finally {
    loading.value = false;
  }
}

async function loadLines(docId: number) {
  if (lineCache.value[docId]) return;
  lineCache.value[docId] = await listDemandLines(docId);
}

function renderExpand(row: dto.DemandDocumentDTO) {
  const lines = lineCache.value[row.id] || [];
  return h(NDataTable, {
    columns: lineColumns.value,
    data: lines,
    size: "small",
    bordered: false,
    pagination: false,
  });
}

async function handleGenerateParticipants() {
  try {
    const count = await generateParticipants(waveId.value);
    participantsGenerated.value = true;
    message.success(`${t("mapping.generateParticipants")}: ${count}`);
    await loadDocs();
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e));
  }
}

async function handleMap() {
  applying.value = true;
  blockedSummary.value = "";
  try {
    const result = await mapDemandLines(waveId.value);
    if (result.blockedLines?.length) {
      blockedSummary.value = result.blockedLines
        .map((line) => line.demandLineTitle || `#${line.demandLineId}`)
        .join(", ");
    }
    message.success(t("mapping.mapDemandOk"));
    await loadDocs();
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e));
  } finally {
    applying.value = false;
  }
}

// Open the inline address corrector modal
async function openAddressFixer(profileId: number) {
  addressProfileId.value = profileId;
  savingAddress.value = false;
  addressIdToUpdate.value = null;
  
  // Set default blank form values
  addressForm.recipientName = "";
  addressForm.phone = "";
  addressForm.country = "CN";
  addressForm.province = "";
  addressForm.city = "";
  addressForm.district = "";
  addressForm.addressLine1 = "";
  addressForm.addressLine2 = "";
  addressForm.postalCode = "";
  addressForm.label = "Default Workspace";
  
  try {
    const addresses = await listAddressesByProfile(profileId);
    if (addresses && addresses.length > 0) {
      const addr = addresses[0]; // load the first/default address to edit
      addressIdToUpdate.value = addr.id;
      addressForm.recipientName = addr.recipientName || "";
      addressForm.phone = addr.phone || "";
      addressForm.country = addr.country || "CN";
      addressForm.province = addr.province || "";
      addressForm.city = addr.city || "";
      addressForm.district = addr.district || "";
      addressForm.addressLine1 = addr.addressLine1 || "";
      addressForm.addressLine2 = addr.addressLine2 || "";
      addressForm.postalCode = addr.postalCode || "";
      addressForm.label = addr.label || "Default Workspace";
    }
    addressModalVisible.value = true;
  } catch (e: unknown) {
    message.error("Failed to load address profile: " + String(e));
  }
}

async function handleSaveAddress() {
  if (!addressProfileId.value) return;
  savingAddress.value = true;
  try {
    if (addressIdToUpdate.value) {
      await updateAddress({
        id: addressIdToUpdate.value,
        customerProfileId: addressProfileId.value,
        ...addressForm,
        isDefault: true,
        isTest: false,
        validationStatus: "valid",
        validationDetail: "",
        extraData: "",
      });
      message.success("Address updated successfully.");
    } else {
      await createAddress({
        customerProfileId: addressProfileId.value,
        ...addressForm,
        isDefault: true,
        isTest: false,
        validationStatus: "valid",
        validationDetail: "",
        extraData: "",
      });
      message.success("Address created successfully.");
    }
    addressModalVisible.value = false;
    
    // Automatically re-run mapping to resolve the blocker
    await handleMap();
  } catch (e: unknown) {
    message.error("Failed to save address: " + String(e));
  } finally {
    savingAddress.value = false;
  }
}

onMounted(loadDocs);
</script>

<template>
  <div class="demand-mapping-page flex flex-col gap-5">
    <div class="mb-2">
      <div class="app-kicker">{{ t("wave.mapping") }}</div>
      <h2 class="app-title mt-2">{{ t("mapping.title") }}</h2>
      <p class="app-copy mt-2">{{ t("mapping.subtitle") }}</p>
    </div>

    <!-- Mapping Pipeline Funnel Banner -->
    <NCard class="glow-card" v-if="routingStats">
      <div class="flex items-center justify-around gap-6 py-2">
        <div class="flex flex-col items-center">
          <span class="text-xs text-slate-400 font-bold uppercase tracking-wider">Total Lines</span>
          <span class="text-2xl font-bold mt-1">{{ routingStats.totalLines }}</span>
        </div>
        <div class="h-10 w-px bg-slate-700/10 dark:bg-slate-700/30"></div>
        <div class="flex flex-col items-center">
          <span class="text-xs text-slate-400 font-bold uppercase tracking-wider">Accepted & Ready</span>
          <span class="text-2xl font-bold text-emerald-600 dark:text-emerald-400 mt-1">
            {{ routingStats.acceptedReadyCount }}
          </span>
        </div>
        <div class="h-10 w-px bg-slate-700/10 dark:bg-slate-700/30"></div>
        <div class="flex flex-col items-center">
          <span class="text-xs text-slate-400 font-bold uppercase tracking-wider">Waiting Input</span>
          <span class="text-2xl font-bold text-amber-500 mt-1">
            {{ routingStats.acceptedWaitingCount }}
          </span>
        </div>
        <div class="h-10 w-px bg-slate-700/10 dark:bg-slate-700/30"></div>
        <div class="flex flex-col items-center">
          <span class="text-xs text-slate-400 font-bold uppercase tracking-wider">Deferred</span>
          <span class="text-2xl font-bold text-slate-400 mt-1">{{ routingStats.deferredCount }}</span>
        </div>
        <div class="h-10 w-px bg-slate-700/10 dark:bg-slate-700/30"></div>
        <div class="flex flex-col items-center">
          <span class="text-xs text-slate-400 font-bold uppercase tracking-wider">Excluded</span>
          <span class="text-2xl font-bold text-red-500 mt-1">
            {{ routingStats.excludedManualCount + routingStats.excludedDuplicateCount + routingStats.excludedRevokedCount }}
          </span>
        </div>
      </div>
    </NCard>

    <NAlert v-if="blockedSummary" type="warning">
      Some lines are blocked due to incomplete data: {{ blockedSummary }}
    </NAlert>

    <!-- Main Assigned Demands Table -->
    <NCard :title="t('mapping.assigned')" class="glow-card">
      <template #header-extra>
        <NSpace align="center" :size="12">
          <NInput 
            v-model:value="searchKeyword" 
            placeholder="Search document no..." 
            clearable 
            size="small" 
            style="width: 200px" 
          />
          <NSelect
            v-model:value="channelFilter"
            :options="channelOptions"
            placeholder="Filter Channel"
            clearable
            size="small"
            style="width: 150px"
          />
          <NButton size="small" secondary @click="handleGenerateParticipants">
            {{ t("mapping.generateParticipants") }}
          </NButton>
          <NButton
            size="small"
            type="primary"
            :loading="applying"
            @click="handleMap"
          >
            {{ t("mapping.mapDemand") }}
          </NButton>
        </NSpace>
      </template>

      <NEmpty v-if="!loading && docs.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="columns"
        :data="filteredDocs"
        :loading="loading"
        :pagination="{ page: 1, pageSize: 10 }"
        size="small"
        :expanded-row-keys="expandedKeys"
        @update:expanded-row-keys="(keys) => {
          expandedKeys = keys as number[]
          for (const key of keys as number[]) {
            void loadLines(key)
          }
        }"
      />
    </NCard>

    <div class="flex justify-between mt-4">
      <NButton @click="router.push(`/waves/${waveId}/allocation`)">{{ t("wave.prevStep") }}</NButton>
      <NSpace>
        <NButton secondary @click="router.push(`/waves/${waveId}`)">{{ t("wave.backToOverview") }}</NButton>
        <NButton type="primary" @click="router.push(`/waves/${waveId}/adjustment-review`)">{{ t("wave.nextStep") }}</NButton>
      </NSpace>
    </div>

    <!-- Inline Address Corrector Modal -->
    <NModal v-model:show="addressModalVisible" preset="card" title="Inline Address Corrector" style="width: 520px">
      <NForm label-placement="top">
        <NFormItem label="Recipient Name" required>
          <NInput v-model:value="addressForm.recipientName" />
        </NFormItem>
        <NFormItem label="Phone Number" required>
          <NInput v-model:value="addressForm.phone" />
        </NFormItem>
        <NGrid :cols="2" :x-gap="12">
          <NGridItem>
            <NFormItem label="Province / State" required>
              <NInput v-model:value="addressForm.province" placeholder="e.g. Guangdong" />
            </NFormItem>
          </NGridItem>
          <NGridItem>
            <NFormItem label="City" required>
              <NInput v-model:value="addressForm.city" placeholder="e.g. Shenzhen" />
            </NFormItem>
          </NGridItem>
        </NGrid>
        <NGrid :cols="2" :x-gap="12">
          <NGridItem>
            <NFormItem label="District" required>
              <NInput v-model:value="addressForm.district" placeholder="e.g. Nanshan" />
            </NFormItem>
          </NGridItem>
          <NGridItem>
            <NFormItem label="Postal Code">
              <NInput v-model:value="addressForm.postalCode" />
            </NFormItem>
          </NGridItem>
        </NGrid>
        <NFormItem label="Address Line 1" required>
          <NInput v-model:value="addressForm.addressLine1" placeholder="Street, Building, Room" />
        </NFormItem>
        <NFormItem label="Address Line 2">
          <NInput v-model:value="addressForm.addressLine2" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="addressModalVisible = false">{{ t("common.cancel") }}</NButton>
          <NButton type="primary" :loading="savingAddress" @click="handleSaveAddress">
            Save Address & Remap
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.glow-card {
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.02);
}

:root[data-theme='dark'] .glow-card {
  border: 1px solid rgba(255, 255, 255, 0.05);
  background: #111827;
}
</style>
