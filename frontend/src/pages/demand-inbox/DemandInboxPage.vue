<script setup lang="ts">
import { computed, h, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NAlert,
  NButton,
  NDataTable,
  NEmpty,
  NGrid,
  NGridItem,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  NSpin,
  NTag,
  useMessage,
} from "naive-ui";
import type { DataTableColumns, DataTableRowKey } from "naive-ui";
import {
  AddCircleOutline,
  CloudUploadOutline,
  ArrowForwardOutline,
} from "@vicons/ionicons5";
import {
  batchUpdateDemandLineRouting,
  importDemandDocument,
  listDemandInboxRows,
  listDemandLines,
  listProfiles,
  updateDemandLineRouting,
} from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

import PageHeader from "@/shared/ui/PageHeader.vue";
import GlassCard from "@/shared/ui/GlassCard.vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const message = useMessage();

// ── State ──
const loading = ref(false);
const error = ref("");
const inbox = ref<dto.DemandInboxRowDTO[]>([]);
const profiles = ref<dto.IntegrationProfileDTO[]>([]);

const selectedDocId = ref<number | null>(null);
const demandLines = ref<dto.DemandLineDTO[]>([]);
const linesLoading = ref(false);
const updatingLineId = ref<number | null>(null);
const dirtyLineIds = ref<Set<number>>(new Set());
const editedLines = reactive<Record<number, { routingDisposition: string; recipientInputState: string }>>({});

const filters = reactive({
  assignment: "all",
  demandKind: "",
  profileId: null as number | null,
});

const importModalVisible = ref(false);
const manualModalVisible = ref(false);
const submitting = ref(false);

// ── Bulk routing state (for selected demand-line rows) ──
const checkedLineIds = ref<DataTableRowKey[]>([]);
const bulkRoutingValue = ref<string | null>(null);

// ── Lookup options ──
const assignmentOptions = computed(() => [
  { label: t("demandIntake.all"), value: "all" },
  { label: t("demandIntake.assigned"), value: "assigned" },
  { label: t("demandIntake.unassigned"), value: "unassigned" },
]);

const demandKindOptions = computed(() => [
  { label: t("demandIntake.all"), value: "" },
  { label: "Membership", value: "membership_entitlement" },
  { label: "Retail", value: "retail_order" },
]);

const profileFilterOptions = computed(() => [
  { label: t("demandIntake.all"), value: null },
  ...profiles.value.map((p) => ({
    label: `${p.profileKey} (${p.sourceChannel})`,
    value: p.id,
  })),
]);

const profileOptions = computed(() =>
  profiles.value.map((p) => ({
    label: `${p.profileKey} (${p.sourceChannel})`,
    value: p.id,
  })),
);

const ROUTING_OPTIONS = computed(() => [
  { label: t("demandIntake.routing.accepted"), value: "accepted" },
  { label: t("demandIntake.routing.deferred"), value: "deferred" },
  { label: t("demandIntake.routing.excluded"), value: "excluded_manual" },
]);

const INPUT_STATE_OPTIONS = [
  { label: "Not Required", value: "not_required" },
  { label: "Waiting", value: "waiting_for_input" },
  { label: "Partially Collected", value: "partially_collected" },
  { label: "Ready", value: "ready" },
  { label: "Waived", value: "waived" },
  { label: "Expired", value: "expired" },
];

// ── Filtered rows (client-side profile filter) ──
const filteredInbox = computed(() => {
  if (filters.profileId == null) return inbox.value;
  return inbox.value.filter(
    (row) => row.integrationProfileId === filters.profileId,
  );
});

// ── Selected document ──
const selectedDoc = computed(() => {
  if (selectedDocId.value == null) return null;
  return (
    inbox.value.find((r) => r.demandDocumentId === selectedDocId.value) || null
  );
});

// ── Routing summary tag rendering ──
function dispositionTagType(d: string): "success" | "warning" | "error" | "default" {
  switch (d) {
    case "accepted":
      return "success";
    case "deferred":
      return "warning";
    case "excluded_manual":
    case "excluded_duplicate":
    case "excluded_revoked":
      return "error";
    default:
      return "default";
  }
}

function inputStateTagType(s: string): "success" | "warning" | "error" | "default" {
  switch (s) {
    case "ready":
      return "success";
    case "waiting_for_input":
    case "partially_collected":
      return "warning";
    case "expired":
      return "error";
    default:
      return "default";
  }
}

// ── Master columns ──
const masterColumns = computed<DataTableColumns<dto.DemandInboxRowDTO>>(() => [
  { title: "ID", key: "demandDocumentId", width: 64 },
  { title: t("demandIntake.demandKind"), key: "kind", width: 160, ellipsis: true },
  {
    title: "Profile",
    key: "integrationProfileLabel",
    width: 180,
    ellipsis: { tooltip: true },
  },
  { title: "Source No", key: "sourceDocumentNo", width: 140, ellipsis: true },
  {
    title: t("demandIntake.routing.disposition"),
    key: "routing",
    width: 220,
    render(row) {
      const total = row.totalLineCount ?? 0;
      const ready = row.readyAcceptedCount ?? 0;
      const waiting = row.waitingInputCount ?? 0;
      const deferred = row.deferredCount ?? 0;
      const excluded = row.excludedCount ?? 0;
      const tags: any[] = [
        h(NTag, { size: "tiny", type: "success", round: true, bordered: false }, { default: () => `r ${ready}` }),
      ];
      if (waiting > 0) tags.push(h(NTag, { size: "tiny", type: "warning", round: true, bordered: false }, { default: () => `w ${waiting}` }));
      if (deferred > 0) tags.push(h(NTag, { size: "tiny", round: true, bordered: false }, { default: () => `d ${deferred}` }));
      if (excluded > 0) tags.push(h(NTag, { size: "tiny", round: true, bordered: false }, { default: () => `x ${excluded}` }));
      tags.push(h("span", { style: "color: var(--muted); font-size: 11px;" }, ` /${total}`));
      return h("div", { style: "display: flex; gap: 4px; flex-wrap: wrap; align-items: center;" }, tags);
    },
  },
  {
    title: t("demandInbox.assigned"),
    key: "assignment",
    width: 160,
    render(row) {
      if (row.assigned) {
        return h(NTag, { type: "info", size: "small", round: true, bordered: false }, { default: () => row.assignedWaveLabel || "—" });
      }
      return h("span", { style: "color: var(--muted)" }, t("demandInbox.notAssigned"));
    },
  },
  { title: "Created", key: "createdAt", width: 110, render: (row) => row.createdAt ? row.createdAt.slice(0, 10) : "—" },
]);

const rowProps = (row: dto.DemandInboxRowDTO) => ({
  style:
    selectedDocId.value === row.demandDocumentId
      ? "cursor: pointer; background: var(--accent-surface, rgba(99,102,241,0.08));"
      : "cursor: pointer;",
  onClick: () => selectDoc(row.demandDocumentId),
});

// ── Line columns (detail panel) ──
function lineDirty(line: dto.DemandLineDTO): boolean {
  return dirtyLineIds.value.has(line.id);
}

function getEditedLine(line: dto.DemandLineDTO) {
  return (
    editedLines[line.id] || {
      routingDisposition: line.routingDisposition,
      recipientInputState: line.recipientInputState,
    }
  );
}

const lineColumns = computed<DataTableColumns<dto.DemandLineDTO>>(() => [
  { type: "selection", width: 40 },
  { title: "ID", key: "id", width: 60 },
  { title: "Title", key: "externalTitle", ellipsis: { tooltip: true } },
  { title: "Qty", key: "requestedQuantity", width: 60 },
  {
    title: t("demandIntake.routing.disposition"),
    key: "routingDisposition",
    width: 170,
    render(row) {
      const e = getEditedLine(row);
      return h(NSelect, {
        value: e.routingDisposition,
        options: ROUTING_OPTIONS.value,
        size: "small",
        onUpdateValue: (val: string) => onLineFieldChange(row, "routingDisposition", val),
      });
    },
  },
  {
    title: t("demandInbox.inputState"),
    key: "recipientInputState",
    width: 200,
    render(row) {
      const e = getEditedLine(row);
      return h(NSelect, {
        value: e.recipientInputState,
        options: INPUT_STATE_OPTIONS,
        size: "small",
        onUpdateValue: (val: string) => onLineFieldChange(row, "recipientInputState", val),
      });
    },
  },
  {
    title: t("demandInbox.entitlementAuthority"),
    key: "entitlementAuthority",
    width: 140,
    render(row) {
      return h(
        NTag,
        { size: "tiny", round: true, bordered: false },
        { default: () => row.entitlementAuthority || "—" },
      );
    },
  },
  {
    title: "",
    key: "actions",
    width: 80,
    render(row) {
      if (!lineDirty(row)) return null;
      return h(
        NButton,
        {
          size: "tiny",
          type: "primary",
          loading: updatingLineId.value === row.id,
          onClick: () => saveLine(row),
        },
        { default: () => "Save" },
      );
    },
  },
]);

function onLineFieldChange(
  row: dto.DemandLineDTO,
  field: "routingDisposition" | "recipientInputState",
  val: string,
) {
  const cur =
    editedLines[row.id] ||
    { routingDisposition: row.routingDisposition, recipientInputState: row.recipientInputState };
  editedLines[row.id] = { ...cur, [field]: val };
  // Track dirty
  const e = editedLines[row.id];
  if (
    e.routingDisposition !== row.routingDisposition ||
    e.recipientInputState !== row.recipientInputState
  ) {
    dirtyLineIds.value.add(row.id);
  } else {
    dirtyLineIds.value.delete(row.id);
  }
  // Force reactive update on Set
  dirtyLineIds.value = new Set(dirtyLineIds.value);
}

// ── Loaders ──
async function loadInbox() {
  loading.value = true;
  error.value = "";
  try {
    const assignment = filters.assignment === "all" ? "" : filters.assignment;
    inbox.value = await listDemandInboxRows({
      assignment,
      demandKind: filters.demandKind || "",
    });
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

async function loadLookups() {
  try {
    profiles.value = await listProfiles();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

async function selectDoc(docId: number) {
  selectedDocId.value = docId;
  checkedLineIds.value = [];
  bulkRoutingValue.value = null;
  dirtyLineIds.value = new Set();
  for (const k of Object.keys(editedLines)) delete editedLines[Number(k)];
  linesLoading.value = true;
  try {
    demandLines.value = await listDemandLines(docId);
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    linesLoading.value = false;
  }
}

// ── Save line ──
async function saveLine(line: dto.DemandLineDTO) {
  const e = editedLines[line.id];
  if (!e) return;
  updatingLineId.value = line.id;
  try {
    await updateDemandLineRouting({
      demandLineId: line.id,
      routingDisposition: e.routingDisposition,
      recipientInputState: e.recipientInputState,
      routingReasonCode: line.routingReasonCode || "",
    });
    message.success(t("demandInbox.routingUpdateSuccess"));
    if (selectedDocId.value != null) {
      demandLines.value = await listDemandLines(selectedDocId.value);
    }
    dirtyLineIds.value.delete(line.id);
    delete editedLines[line.id];
    dirtyLineIds.value = new Set(dirtyLineIds.value);
    await loadInbox();
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    updatingLineId.value = null;
  }
}

async function applyBulkRouting() {
  if (!bulkRoutingValue.value || checkedLineIds.value.length === 0) return;
  try {
    await batchUpdateDemandLineRouting({
      updates: (checkedLineIds.value as number[]).map((demandLineId) => ({
        demandLineId,
        routingDisposition: bulkRoutingValue.value as string,
        recipientInputState: "",
        routingReasonCode: "",
      })),
    });
    message.success(t("demandInbox.routingUpdateSuccess"));
    if (selectedDocId.value != null) {
      demandLines.value = await listDemandLines(selectedDocId.value);
    }
    checkedLineIds.value = [];
    bulkRoutingValue.value = null;
    await loadInbox();
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

// ── Read-only navigation: open the assigned wave in workspace ──
function openAssignedWave() {
  if (selectedDoc.value?.assignedWaveId) {
    router.push(`/waves/${selectedDoc.value.assignedWaveId}`);
  }
}

// ── Manual Entry Modal ──
const manualEntry = reactive({
  integrationProfileId: null as number | null,
  sourceDocumentNo: "",
  sourceCustomerRef: "",
  customerProfileId: null as number | null,
  externalTitle: "",
  requestedQuantity: 1,
});

async function submitManualEntry() {
  if (!manualEntry.integrationProfileId || !manualEntry.externalTitle) {
    message.warning("Profile and title are required");
    return;
  }
  submitting.value = true;
  try {
    await importDemandDocument({
      kind: "retail_order",
      captureMode: "manual_entry",
      sourceChannel: "manual",
      sourceDocumentNo: manualEntry.sourceDocumentNo || `MANUAL-${Date.now()}`,
      sourceCustomerRef: manualEntry.sourceCustomerRef,
      customerProfileId: manualEntry.customerProfileId || undefined,
      integrationProfileId: manualEntry.integrationProfileId,
      lines: [
        {
          lineType: "sku_order",
          obligationTriggerKind: "manual_compensation",
          entitlementAuthority: "manual_grant",
          recipientInputState: "ready",
          routingDisposition: "accepted",
          externalTitle: manualEntry.externalTitle,
          requestedQuantity: manualEntry.requestedQuantity,
        },
      ],
    });
    message.success(t("demandInbox.importSuccess"));
    manualModalVisible.value = false;
    manualEntry.sourceDocumentNo = "";
    manualEntry.sourceCustomerRef = "";
    manualEntry.customerProfileId = null;
    manualEntry.externalTitle = "";
    manualEntry.requestedQuantity = 1;
    await loadInbox();
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    submitting.value = false;
  }
}

// ── Import Document Modal (placeholder — currently uses manual entry shape, improvement deferred) ──
const importEntry = reactive({
  integrationProfileId: null as number | null,
  kind: "membership_entitlement",
  sourceDocumentNo: "",
  externalTitle: "",
  requestedQuantity: 1,
});

async function submitImport() {
  if (!importEntry.integrationProfileId || !importEntry.externalTitle) {
    message.warning("Profile and title are required");
    return;
  }
  submitting.value = true;
  try {
    await importDemandDocument({
      kind: importEntry.kind,
      captureMode: "document_import",
      sourceChannel: "import",
      sourceDocumentNo: importEntry.sourceDocumentNo || `IMPORT-${Date.now()}`,
      integrationProfileId: importEntry.integrationProfileId,
      lines: [
        {
          lineType: importEntry.kind === "membership_entitlement" ? "entitlement" : "sku_order",
          obligationTriggerKind: importEntry.kind === "membership_entitlement" ? "loyalty_membership" : "purchase_order",
          entitlementAuthority: importEntry.kind === "membership_entitlement" ? "upstream_platform" : "manual_grant",
          recipientInputState: "ready",
          routingDisposition: "accepted",
          externalTitle: importEntry.externalTitle,
          requestedQuantity: importEntry.requestedQuantity,
        },
      ],
    });
    message.success(t("demandInbox.importSuccess"));
    importModalVisible.value = false;
    importEntry.sourceDocumentNo = "";
    importEntry.externalTitle = "";
    importEntry.requestedQuantity = 1;
    await loadInbox();
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    submitting.value = false;
  }
}

// ── URL query: docId deep link ──
async function applyDocIdQuery() {
  const docIdRaw = route.query.docId;
  if (typeof docIdRaw !== "string") return;
  const docId = Number(docIdRaw);
  if (!Number.isFinite(docId) || docId <= 0) return;
  // Wait until inbox loaded; if not present, still try to load lines directly.
  await selectDoc(docId);
}

watch(
  () => route.query.docId,
  () => {
    void applyDocIdQuery();
  },
);

onMounted(async () => {
  await loadLookups();
  await loadInbox();
  await applyDocIdQuery();
});
</script>

<template>
  <div class="demand-inbox-page">
    <PageHeader
      :title="t('demandInbox.title')"
      :description="t('demandInbox.subtitle')"
    >
      <template #actions>
        <NButton secondary @click="loadInbox">Refresh</NButton>
        <NButton type="primary" secondary @click="importModalVisible = true">
          <template #icon>
            <NIcon><CloudUploadOutline /></NIcon>
          </template>
          {{ t("demandInbox.importDocument") }}
        </NButton>
        <NButton type="primary" @click="manualModalVisible = true">
          <template #icon>
            <NIcon><AddCircleOutline /></NIcon>
          </template>
          {{ t("demandInbox.manualEntry") }}
        </NButton>
      </template>
    </PageHeader>

    <NAlert v-if="error" type="error" class="mb-4" :title="error" closable @close="error = ''" />

    <!-- Filter toolbar -->
    <div class="filter-bar mb-4">
      <NSpace>
        <NSelect
          v-model:value="filters.assignment"
          :options="assignmentOptions"
          style="width: 160px"
          @update:value="loadInbox"
        />
        <NSelect
          v-model:value="filters.demandKind"
          :options="demandKindOptions"
          style="width: 200px"
          @update:value="loadInbox"
        />
        <NSelect
          v-model:value="filters.profileId"
          :options="profileFilterOptions"
          style="width: 240px"
          clearable
        />
      </NSpace>
    </div>

    <!-- Master-Detail Layout -->
    <NGrid :cols="8" :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <!-- Master -->
      <NGridItem :span="5">
        <GlassCard>
          <div class="card-section-title">{{ t("demandInbox.masterTitle") }}</div>
          <NEmpty
            v-if="!loading && filteredInbox.length === 0"
            :description="t('common.empty') || 'No demand documents.'"
            class="empty-block"
          />
          <NDataTable
            v-else
            :columns="masterColumns"
            :data="filteredInbox"
            :loading="loading"
            :pagination="{ pageSize: 20 }"
            :row-props="rowProps"
            :row-key="(row: dto.DemandInboxRowDTO) => row.demandDocumentId"
            size="small"
          />
        </GlassCard>
      </NGridItem>

      <!-- Detail -->
      <NGridItem :span="3">
        <GlassCard>
          <div class="card-section-title">{{ t("demandInbox.detailTitle") }}</div>

          <NEmpty
            v-if="!selectedDoc"
            :description="t('demandInbox.selectDocHint')"
            class="empty-block"
          />

          <template v-else>
            <!-- Section 1: Header -->
            <div class="detail-section">
              <div class="kv-grid">
                <div class="kv-item"><span class="kv-key">Kind</span><span>{{ selectedDoc.kind }}</span></div>
                <div class="kv-item"><span class="kv-key">Capture</span><span>{{ selectedDoc.captureMode }}</span></div>
                <div class="kv-item"><span class="kv-key">Source Channel</span><span>{{ selectedDoc.sourceChannel }}</span></div>
                <div class="kv-item"><span class="kv-key">Surface</span><span>{{ selectedDoc.sourceSurface || "—" }}</span></div>
                <div class="kv-item"><span class="kv-key">Source No</span><span>{{ selectedDoc.sourceDocumentNo || "—" }}</span></div>
                <div class="kv-item"><span class="kv-key">Customer Ref</span><span>{{ selectedDoc.customerProfileId || "—" }}</span></div>
              </div>
            </div>

            <!-- Section 2: Demand Lines -->
            <div class="detail-section">
              <div class="section-subheader">
                <span class="section-subtitle">{{ t("demandInbox.routingEditor") }}</span>
              </div>
              <NAlert type="info" size="small" class="mb-3">
                {{ t("demandInbox.routingHint") }}
              </NAlert>

              <div class="bulk-toolbar mb-2">
                <NSelect
                  v-model:value="bulkRoutingValue"
                  :options="ROUTING_OPTIONS"
                  size="small"
                  placeholder="Bulk routing..."
                  style="width: 180px"
                  :disabled="checkedLineIds.length === 0"
                />
                <NButton
                  size="small"
                  type="primary"
                  :disabled="!bulkRoutingValue || checkedLineIds.length === 0"
                  @click="applyBulkRouting"
                >
                  Apply to {{ checkedLineIds.length }}
                </NButton>
              </div>

              <NSpin v-if="linesLoading" />
              <NEmpty
                v-else-if="demandLines.length === 0"
                description="No lines."
                class="empty-block-sm"
              />
              <NDataTable
                v-else
                v-model:checked-row-keys="checkedLineIds"
                :columns="lineColumns"
                :data="demandLines"
                :pagination="false"
                size="small"
                :row-key="(row: dto.DemandLineDTO) => row.id"
              />
            </div>

            <!-- Section 3: Intake Status (read-only) -->
            <div class="detail-section">
              <div class="section-subheader">
                <span class="section-subtitle">Intake Status</span>
              </div>
              <NSpace align="center">
                <template v-if="selectedDoc.assigned">
                  <NTag type="info" round :bordered="false">
                    {{ t("demandInbox.assigned") }}:
                    {{ selectedDoc.assignedWaveLabel || "—" }}
                  </NTag>
                  <NButton
                    v-if="selectedDoc.assignedWaveId"
                    size="small"
                    quaternary
                    @click="openAssignedWave"
                  >
                    <template #icon>
                      <NIcon><ArrowForwardOutline /></NIcon>
                    </template>
                    Open Wave
                  </NButton>
                </template>
                <NTag v-else round :bordered="false">
                  {{ t("demandInbox.notAssigned") }}
                </NTag>
              </NSpace>
              <NAlert type="info" size="small" class="mt-3">
                Intake into a wave is initiated from inside the Wave Workspace
                (Wave → ① Demand Intake), not here. This page is the upstream
                truth & status board.
              </NAlert>
            </div>
          </template>
        </GlassCard>
      </NGridItem>
    </NGrid>

    <!-- Manual Entry Modal -->
    <NModal v-model:show="manualModalVisible" preset="card" :title="t('demandInbox.manualEntry')" style="width: 480px;">
      <NSpace vertical :size="14">
        <NSelect
          v-model:value="manualEntry.integrationProfileId"
          :options="profileOptions"
          placeholder="Profile"
        />
        <NInput v-model:value="manualEntry.sourceDocumentNo" placeholder="Source document no" />
        <NInput v-model:value="manualEntry.sourceCustomerRef" placeholder="Source customer ref" />
        <NInputNumber v-model:value="manualEntry.customerProfileId" placeholder="Customer profile ID" />
        <NInput v-model:value="manualEntry.externalTitle" placeholder="Demand title" />
        <NInputNumber v-model:value="manualEntry.requestedQuantity" :min="1" placeholder="Quantity" />
        <NSpace justify="end">
          <NButton @click="manualModalVisible = false">Cancel</NButton>
          <NButton type="primary" :loading="submitting" @click="submitManualEntry">
            {{ t("demandIntake.createDemand") }}
          </NButton>
        </NSpace>
      </NSpace>
    </NModal>

    <!-- Import Document Modal -->
    <NModal v-model:show="importModalVisible" preset="card" :title="t('demandInbox.importDocument')" style="width: 480px;">
      <NSpace vertical :size="14">
        <NAlert type="info" size="small">
          Document import (CSV / API) will be available per-Profile. This is the simple inline form for quick imports.
        </NAlert>
        <NSelect
          v-model:value="importEntry.integrationProfileId"
          :options="profileOptions"
          placeholder="Profile"
        />
        <NSelect
          v-model:value="importEntry.kind"
          :options="[
            { label: 'Membership Entitlement', value: 'membership_entitlement' },
            { label: 'Retail Order', value: 'retail_order' },
          ]"
        />
        <NInput v-model:value="importEntry.sourceDocumentNo" placeholder="Source document no" />
        <NInput v-model:value="importEntry.externalTitle" placeholder="Demand title" />
        <NInputNumber v-model:value="importEntry.requestedQuantity" :min="1" placeholder="Quantity" />
        <NSpace justify="end">
          <NButton @click="importModalVisible = false">Cancel</NButton>
          <NButton type="primary" :loading="submitting" @click="submitImport">
            {{ t("demandInbox.importDocument") }}
          </NButton>
        </NSpace>
      </NSpace>
    </NModal>
  </div>
</template>

<style scoped>
.demand-inbox-page {
  display: flex;
  flex-direction: column;
}

.filter-bar {
  padding: 10px 12px;
  background: var(--surface-strong);
  border: 1px solid rgba(148, 163, 184, 0.12);
  border-radius: 10px;
}

:root[data-theme='dark'] .filter-bar {
  background: rgba(15, 23, 42, 0.4);
  border-color: rgba(255, 255, 255, 0.06);
}

.card-section-title {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--muted);
  margin-bottom: 12px;
}

.detail-section {
  padding: 14px 0;
  border-top: 1px solid rgba(148, 163, 184, 0.12);
}

.detail-section:first-child {
  padding-top: 0;
  border-top: none;
}

.section-subheader {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.section-subtitle {
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--text);
}

.kv-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px 16px;
  font-size: 0.8rem;
}

.kv-item {
  display: flex;
  justify-content: space-between;
  gap: 8px;
}

.kv-key {
  color: var(--muted);
  font-size: 0.75rem;
}

.bulk-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
}

.empty-block {
  padding: 48px 0;
}

.empty-block-sm {
  padding: 24px 0;
}

.mb-2 {
  margin-bottom: 8px;
}
.mb-3 {
  margin-bottom: 12px;
}
.mb-4 {
  margin-bottom: 16px;
}
</style>
