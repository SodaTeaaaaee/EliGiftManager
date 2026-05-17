<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { NAlert, NButton, NCard, NDataTable, NEmpty, NGrid, NGridItem, NInput, NInputNumber, NSelect, NSpace, NTag, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { assignDemandToWave, importDemandDocument, listDemandInboxRows, listProfiles, listWaves } from "@/shared/lib/wails/app";
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
