<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from "vue";
import { NAlert, NButton, NCard, NDataTable, NEmpty, NGrid, NGridItem, NInput, NInputNumber, NSelect, NSpace, NTag, NSpin, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import {
  assignDemandToWave,
  importDemandDocument,
  listDemandInboxRows,
  listDemandLines,
  listProfiles,
  listWaves,
  updateDemandLineRouting,
} from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const { t } = useI18n();
const message = useMessage();

const loading = ref(false);
const error = ref("");
const inbox = ref<dto.DemandInboxRowDTO[]>([]);
const profiles = ref<dto.IntegrationProfileDTO[]>([]);
const waves = ref<dto.WaveDTO[]>([]);
const assigningId = ref<number | null>(null);
const selectedWaveByDoc = ref<Record<number, number | null>>({});
const creating = ref(false);

// Routing panel state
const selectedDocId = ref<number | null>(null);
const demandLines = ref<dto.DemandLineDTO[]>([]);
const linesLoading = ref(false);
const updatingLineId = ref<number | null>(null);

const filters = reactive({
  assignment: "all",
  demandKind: "",
});

const manualEntry = reactive({
  integrationProfileId: null as number | null,
  sourceDocumentNo: "",
  sourceCustomerRef: "",
  customerProfileId: null as number | null,
  externalTitle: "",
  requestedQuantity: 1,
});

const assignmentOptions = [
  { label: t("demandIntake.all"), value: "all" },
  { label: t("demandIntake.assigned"), value: "assigned" },
  { label: t("demandIntake.unassigned"), value: "unassigned" },
];

const profileOptions = computed(() =>
  profiles.value.map((profile) => ({
    label: `${profile.profileKey} (${profile.sourceChannel})`,
    value: profile.id,
  })),
);

const waveOptions = computed(() =>
  waves.value.map((wave) => ({
    label: `${wave.waveNo} — ${wave.name}`,
    value: wave.id,
  })),
);

const demandKindOptions = [
  { label: "All", value: "" },
  { label: "Membership", value: "membership_entitlement" },
  { label: "Retail", value: "retail_order" },
];

const routingDispositionOptions = [
  { label: t("demandIntake.routing.accepted"), value: "accepted" },
  { label: t("demandIntake.routing.deferred"), value: "deferred" },
  { label: t("demandIntake.routing.excluded"), value: "excluded_manual" },
];

const recipientInputStateOptions = [
  { label: "not_required", value: "not_required" },
  { label: "waiting_for_input", value: "waiting_for_input" },
  { label: "partially_collected", value: "partially_collected" },
  { label: "ready", value: "ready" },
  { label: "waived", value: "waived" },
  { label: "expired", value: "expired" },
];

function dispositionTagType(disposition: string): "success" | "warning" | "error" | "default" {
  switch (disposition) {
    case "accepted": return "success";
    case "deferred": return "warning";
    case "excluded_manual":
    case "excluded_duplicate":
    case "excluded_revoked": return "error";
    default: return "default";
  }
}

function inputStateTagType(state: string): "success" | "warning" | "error" | "default" {
  switch (state) {
    case "ready": return "success";
    case "waiting_for_input": return "warning";
    case "partially_collected": return "warning";
    case "expired": return "error";
    case "not_required":
    case "waived": return "default";
    default: return "default";
  }
}

const columns = computed<DataTableColumns<dto.DemandInboxRowDTO>>(() => [
  { title: "ID", key: "demandDocumentId", width: 70 },
  { title: t("demandIntake.demandKind"), key: "kind", width: 180 },
  { title: "Profile", key: "integrationProfileLabel", width: 220 },
  { title: "Source", key: "sourceDocumentNo", width: 180 },
  {
    title: t("demandIntake.acceptedReady"),
    key: "readyAcceptedCount",
    width: 100,
    render(row) {
      return h(NTag, { type: "success", size: "small" }, { default: () => String(row.readyAcceptedCount) });
    },
  },
  { title: t("demandIntake.waitingInput"), key: "waitingInputCount", width: 110 },
  { title: t("demandIntake.deferred"), key: "deferredCount", width: 90 },
  { title: t("demandIntake.excluded"), key: "excludedCount", width: 90 },
  {
    title: t("demandIntake.routing.disposition"),
    key: "routingActions",
    width: 120,
    render(row) {
      const isSelected = selectedDocId.value === row.demandDocumentId;
      return h(
        NButton,
        {
          size: "small",
          type: isSelected ? "primary" : "default",
          onClick: () => handleSelectDoc(row.demandDocumentId),
        },
        { default: () => isSelected ? "▲" : t("demandIntake.routing.disposition") },
      );
    },
  },
  {
    title: t("demandIntake.assignToWave"),
    key: "actions",
    width: 250,
    render(row) {
      if (row.assigned) {
        return row.assignedWaveLabel || t("demandIntake.assigned");
      }
      return h(NSpace, { size: "small" }, () => [
        h(NSelect, {
          value: selectedWaveByDoc.value[row.demandDocumentId] ?? null,
          options: waveOptions.value,
          style: "width: 150px",
          onUpdateValue: (value: number | null) => {
            selectedWaveByDoc.value[row.demandDocumentId] = value;
          },
        }),
        h(
          NButton,
          {
            size: "small",
            type: "primary",
            loading: assigningId.value === row.demandDocumentId,
            disabled: !selectedWaveByDoc.value[row.demandDocumentId],
            onClick: () => handleAssign(row),
          },
          { default: () => t("demandIntake.assignToWave") },
        ),
      ]);
    },
  },
]);

const lineColumns = computed<DataTableColumns<dto.DemandLineDTO>>(() => [
  { title: "ID", key: "id", width: 60 },
  { title: "Title", key: "externalTitle", ellipsis: true },
  { title: "Qty", key: "requestedQuantity", width: 60 },
  {
    title: t("demandIntake.routing.disposition"),
    key: "routingDisposition",
    width: 200,
    render(row) {
      return h(NSelect, {
        value: row.routingDisposition,
        options: routingDispositionOptions,
        size: "small",
        style: "width: 180px",
        loading: updatingLineId.value === row.id,
        onUpdateValue: (val: string) => handleUpdateDisposition(row, val),
      });
    },
  },
  {
    title: t("demandIntake.routing.inputState"),
    key: "recipientInputState",
    width: 220,
    render(row) {
      return h(NSpace, { size: "small", align: "center" }, () => [
        h(NTag, {
          type: inputStateTagType(row.recipientInputState),
          size: "small",
        }, { default: () => row.recipientInputState || "—" }),
        h(NSelect, {
          value: row.recipientInputState,
          options: recipientInputStateOptions,
          size: "small",
          style: "width: 160px",
          loading: updatingLineId.value === row.id,
          onUpdateValue: (val: string) => handleUpdateInputState(row, val),
        }),
      ]);
    },
  },
]);

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
  profiles.value = await listProfiles();
  waves.value = await listWaves();
}

async function handleSelectDoc(docId: number) {
  if (selectedDocId.value === docId) {
    selectedDocId.value = null;
    demandLines.value = [];
    return;
  }
  selectedDocId.value = docId;
  linesLoading.value = true;
  try {
    demandLines.value = await listDemandLines(docId);
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    linesLoading.value = false;
  }
}

async function handleUpdateDisposition(line: dto.DemandLineDTO, newDisposition: string) {
  updatingLineId.value = line.id;
  try {
    await updateDemandLineRouting({
      demandLineId: line.id,
      routingDisposition: newDisposition,
      recipientInputState: line.recipientInputState,
      routingReasonCode: line.routingReasonCode || "",
    });
    message.success(t("demandIntake.routing.updateSuccess"));
    if (selectedDocId.value !== null) {
      demandLines.value = await listDemandLines(selectedDocId.value);
    }
    await loadInbox();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    updatingLineId.value = null;
  }
}

async function handleUpdateInputState(line: dto.DemandLineDTO, newState: string) {
  updatingLineId.value = line.id;
  try {
    await updateDemandLineRouting({
      demandLineId: line.id,
      routingDisposition: line.routingDisposition,
      recipientInputState: newState,
      routingReasonCode: line.routingReasonCode || "",
    });
    message.success(t("demandIntake.routing.updateSuccess"));
    if (selectedDocId.value !== null) {
      demandLines.value = await listDemandLines(selectedDocId.value);
    }
    await loadInbox();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    updatingLineId.value = null;
  }
}

async function handleAssign(row: dto.DemandInboxRowDTO) {
  const waveId = selectedWaveByDoc.value[row.demandDocumentId];
  if (!waveId) return;
  assigningId.value = row.demandDocumentId;
  try {
    await assignDemandToWave(waveId, row.demandDocumentId);
    message.success(`${t("demandIntake.assignToWave")} OK`);
    await loadInbox();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    assigningId.value = null;
  }
}

async function handleManualEntry() {
  if (!manualEntry.integrationProfileId || !manualEntry.externalTitle) {
    message.warning("Profile and title are required");
    return;
  }
  creating.value = true;
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
    message.success(t("demandIntake.createDemand"));
    manualEntry.sourceDocumentNo = "";
    manualEntry.sourceCustomerRef = "";
    manualEntry.customerProfileId = null;
    manualEntry.externalTitle = "";
    manualEntry.requestedQuantity = 1;
    await loadInbox();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    creating.value = false;
  }
}

onMounted(async () => {
  await loadLookups();
  await loadInbox();
});
</script>

<template>
  <div class="demand-intake-page">
    <div class="mb-6">
      <div class="app-kicker">{{ t("nav.demandIntake") }}</div>
      <h1 class="app-title mt-2">{{ t("demandIntake.title") }}</h1>
      <p class="app-copy mt-3">{{ t("demandIntake.subtitle") }}</p>
    </div>

    <NAlert type="info" class="mb-4">
      {{ t("demandIntake.manualEntryOnly") }}
    </NAlert>
    <NAlert v-if="error" type="error" class="mb-4" :title="error" />

    <NGrid :cols="2" :x-gap="16" :y-gap="16">
      <NGridItem>
        <NCard :title="t('demandIntake.inbox')">
          <NSpace class="mb-4">
            <NSelect v-model:value="filters.assignment" :options="assignmentOptions" @update:value="loadInbox" />
            <NSelect v-model:value="filters.demandKind" :options="demandKindOptions" @update:value="loadInbox" />
          </NSpace>
          <NEmpty v-if="!loading && inbox.length === 0" :description="t('common.empty')" />
          <NDataTable
            v-else
            :columns="columns"
            :data="inbox"
            :loading="loading"
            :pagination="false"
            size="small"
          />
        </NCard>

        <NCard v-if="selectedDocId !== null" :title="t('demandIntake.routing.disposition')" class="mt-4">
          <NSpin v-if="linesLoading" />
          <NEmpty v-else-if="demandLines.length === 0" :description="t('common.empty')" />
          <NDataTable
            v-else
            :columns="lineColumns"
            :data="demandLines"
            :pagination="false"
            size="small"
          />
        </NCard>
      </NGridItem>

      <NGridItem>
        <NCard :title="t('demandIntake.manualEntry')">
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
            <NButton type="primary" :loading="creating" @click="handleManualEntry">
              {{ t("demandIntake.createDemand") }}
            </NButton>
          </NSpace>
        </NCard>
      </NGridItem>
    </NGrid>
  </div>
</template>
