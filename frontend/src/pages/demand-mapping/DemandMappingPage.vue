<script setup lang="ts">
import { computed, h, onMounted, ref, reactive, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NAlert, NButton, NCard, NDataTable, NEmpty, NTag, NSpace, NInput, NSelect, NModal, NForm, NFormItem, useMessage, NIcon } from "naive-ui";
import { ListOutline, CheckmarkCircleOutline, DocumentTextOutline } from "@vicons/ionicons5";
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
import PageHeader from "@/shared/ui/PageHeader.vue";
import GlassCard from "@/shared/ui/GlassCard.vue";
import SplitPane from "@/shared/ui/SplitPane.vue";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { t } = useI18n();
const waveId = computed(() => Number(route.params.waveId) || 0);

const docs = ref<dto.DemandDocumentDTO[]>([]);
const loading = ref(false);
const applying = ref(false);
const participantsGenerated = ref(false);
const blockedSummary = ref<string>("");

// Search, Filter and Stats state
const searchKeyword = ref("");
const channelFilter = ref<string | null>(null);
const routingStats = ref<any | null>(null);

// SplitPane selection state
const selectedDocId = ref<number | null>(null);
const selectedDocLines = ref<dto.DemandLineDTO[]>([]);
const linesLoading = ref(false);

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
    // If route.params.demandKind is set, filter by kind
    const demandKindParam = route.params.demandKind as string;
    if (demandKindParam && doc.kind !== demandKindParam) {
      return false;
    }

    const matchesKeyword = !searchKeyword.value || 
      doc.sourceDocumentNo.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
      (doc.sourceCustomerRef && doc.sourceCustomerRef.toLowerCase().includes(searchKeyword.value.toLowerCase()));
    const matchesChannel = !channelFilter.value || doc.sourceChannel === channelFilter.value;
    return matchesKeyword && matchesChannel;
  });
});

const selectedDoc = computed(() => docs.value.find(d => d.id === selectedDocId.value));

const columns = computed<DataTableColumns<dto.DemandDocumentDTO>>(() => [
  { title: "ID", key: "id", width: 65 },
  { title: "Source No", key: "sourceDocumentNo" },
  { title: "Channel", key: "sourceChannel", width: 100 },
]);

const lineColumns = computed<DataTableColumns<dto.DemandLineDTO>>(() => [
  { title: t("mapping.columns.line"), key: "sourceLineNo", width: 75 },
  { title: t("mapping.columns.type"), key: "lineType", width: 120 },
  { title: t("mapping.columns.title"), key: "externalTitle" },
  { 
    title: t("mapping.columns.disposition"), 
    key: "routingDisposition", 
    width: 120,
    render(row) {
      const type = row.routingDisposition === "blocked" ? "error" : row.routingDisposition === "deferred" ? "warning" : "success";
      return h(NTag, { type, size: "small", round: true, bordered: false }, { default: () => row.routingDisposition });
    }
  },
  { 
    title: t("mapping.columns.input"), 
    key: "recipientInputState", 
    width: 140,
    render(row) {
      const type = row.recipientInputState === "address_unavailable" || row.recipientInputState === "address_invalid" ? "warning" : "default";
      return h(NTag, { type, size: "small", bordered: false }, { default: () => row.recipientInputState || "none" });
    }
  },
  { title: t("mapping.columns.qty"), key: "requestedQuantity", width: 65 },
  {
    title: "Actions",
    key: "actions",
    width: 110,
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
              if (selectedDoc.value && selectedDoc.value.customerProfileId) {
                openAddressFixer(selectedDoc.value.customerProfileId);
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

async function loadLinesForSelection() {
  if (!selectedDocId.value) {
    selectedDocLines.value = [];
    return;
  }
  linesLoading.value = true;
  try {
    selectedDocLines.value = await listDemandLines(selectedDocId.value);
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e));
  } finally {
    linesLoading.value = false;
  }
}

watch(selectedDocId, loadLinesForSelection);

function selectRow(row: dto.DemandDocumentDTO) {
  selectedDocId.value = row.id;
}

function rowProps(row: dto.DemandDocumentDTO) {
  return {
    style: 'cursor: pointer;',
    class: selectedDocId.value === row.id ? 'bg-blue-50 dark:bg-blue-900/20' : '',
    onClick: () => selectRow(row)
  };
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
    if (selectedDocId.value) await loadLinesForSelection();
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
  <div class="demand-mapping-page flex flex-col h-full">
    <PageHeader 
      :title="t('mapping.title')" 
      :description="t('mapping.subtitle')" 
    >
      <template #actions>
        <NButton secondary @click="handleGenerateParticipants">
          {{ t("mapping.generateParticipants") }}
        </NButton>
        <NButton type="primary" :loading="applying" @click="handleMap" icon-placement="left">
          <template #icon><NIcon><CheckmarkCircleOutline /></NIcon></template>
          {{ t("mapping.mapDemand") }}
        </NButton>
      </template>
    </PageHeader>

    <NAlert v-if="blockedSummary" type="warning" class="mb-4">
      Some lines are blocked due to incomplete data: {{ blockedSummary }}
    </NAlert>

    <div class="flex-1 min-h-0 relative -mx-8 -mb-6 mt-2 border-t border-slate-700/10 dark:border-slate-700/30">
      <SplitPane :initial-split="35" :min-left="25" :min-right="35">
        <template #left>
          <div class="h-full flex flex-col bg-slate-50/50 dark:bg-slate-900/20 border-r border-slate-700/10 dark:border-slate-700/30">
            <div class="p-4 border-b border-slate-700/10 dark:border-slate-700/30 flex flex-col gap-3 shrink-0">
              <div class="font-semibold text-slate-800 dark:text-slate-200">{{ t('mapping.assigned') }} ({{ filteredDocs.length }})</div>
              <NSpace align="center" :size="8" wrap>
                <NInput 
                  v-model:value="searchKeyword" 
                  placeholder="Search No..." 
                  clearable 
                  size="small" 
                  style="width: 140px" 
                />
                <NSelect
                  v-model:value="channelFilter"
                  :options="channelOptions"
                  placeholder="Channel"
                  clearable
                  size="small"
                  style="width: 120px"
                />
              </NSpace>
            </div>
            <div class="flex-1 overflow-auto p-4">
              <NEmpty v-if="!loading && docs.length === 0" :description="t('common.empty')" />
              <NDataTable
                v-else
                :columns="columns"
                :data="filteredDocs"
                :loading="loading"
                :pagination="false"
                :row-props="rowProps"
                size="small"
                :bordered="false"
              />
            </div>
          </div>
        </template>
        
        <template #right>
          <div class="h-full flex flex-col bg-white dark:bg-[#1e293b]">
            <div v-if="!selectedDocId" class="flex-1 flex flex-col items-center justify-center text-slate-400">
              <NIcon size="64" class="mb-4 opacity-50"><DocumentTextOutline /></NIcon>
              <p>Select a demand document from the left to view its lines.</p>
            </div>
            <template v-else>
              <div class="p-5 border-b border-slate-700/10 dark:border-slate-700/30 shrink-0">
                <div class="flex justify-between items-start mb-2">
                  <div>
                    <div class="app-kicker">{{ selectedDoc?.sourceChannel }}</div>
                    <h3 class="text-xl font-bold mt-1 text-slate-800 dark:text-slate-100">{{ selectedDoc?.sourceDocumentNo }}</h3>
                  </div>
                  <NTag v-if="selectedDoc?.customerProfileId" size="small" type="info">Profile #{{ selectedDoc.customerProfileId }}</NTag>
                </div>
                <div class="text-sm text-slate-500 mt-2">
                  Customer Ref: {{ selectedDoc?.sourceCustomerRef || 'N/A' }} | Capture Mode: {{ selectedDoc?.captureMode }}
                </div>
              </div>
              
              <div class="flex-1 overflow-auto p-5">
                <h4 class="font-semibold text-slate-800 dark:text-slate-200 mb-3 flex items-center gap-2">
                  <NIcon><ListOutline /></NIcon> Demand Lines
                </h4>
                <NDataTable
                  :columns="lineColumns"
                  :data="selectedDocLines"
                  :loading="linesLoading"
                  size="small"
                  :bordered="true"
                  :pagination="false"
                />
              </div>
            </template>
          </div>
        </template>
      </SplitPane>
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
        <div class="grid grid-cols-2 gap-3">
          <NFormItem label="Province / State" required>
            <NInput v-model:value="addressForm.province" placeholder="e.g. Guangdong" />
          </NFormItem>
          <NFormItem label="City" required>
            <NInput v-model:value="addressForm.city" placeholder="e.g. Shenzhen" />
          </NFormItem>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <NFormItem label="District" required>
            <NInput v-model:value="addressForm.district" placeholder="e.g. Nanshan" />
          </NFormItem>
          <NFormItem label="Postal Code">
            <NInput v-model:value="addressForm.postalCode" />
          </NFormItem>
        </div>
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

