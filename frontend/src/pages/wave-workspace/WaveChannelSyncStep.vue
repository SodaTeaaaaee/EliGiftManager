<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NAlert, NButton, NCard, NDataTable, NEmpty, NInput, NSelect, NSpace, NTag, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { executeChannelSyncJob, listChannelSyncJobsByWave, listIntegrationProfiles, planChannelClosure, recordChannelClosureDecision, retryChannelSyncJob } from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { t } = useI18n();
const waveId = computed(() => Number(route.params.waveId) || 0);

const loading = ref(false);
const profilesLoading = ref(false);
const planning = ref(false);
const submitting = ref(false);
const profiles = ref<dto.IntegrationProfileSummaryDTO[]>([]);
const jobs = ref<dto.ChannelSyncJobDTO[]>([]);
const selectedProfileId = ref<number | null>(null);
const planResult = ref<dto.PlanChannelClosureResult | null>(null);
const error = ref("");

const manualForms = ref<Record<number, { decisionKind: string; reasonCode: string; note: string; evidenceRef: string; operatorId: string }>>({});

const profileOptions = computed(() =>
  profiles.value.map((profile) => ({
    label: `${profile.profileKey} (${profile.sourceChannel})`,
    value: profile.id,
  })),
);

const decisionKindOptions = computed(() => {
  const base = [
    { label: "mark_sync_unsupported", value: "mark_sync_unsupported" },
    { label: "mark_sync_skipped", value: "mark_sync_skipped" },
  ];
  const selected = profiles.value.find((profile) => profile.id === selectedProfileId.value);
  if (selected?.allowsManualClosure) {
    base.push({ label: "mark_sync_completed_manually", value: "mark_sync_completed_manually" });
  }
  return base;
});

const columns: DataTableColumns<dto.ChannelSyncJobDTO> = [
  { title: "ID", key: "id", width: 60 },
  { title: "Profile", key: "integrationProfileId", width: 100 },
  { title: "Direction", key: "direction", width: 120 },
  {
    title: "Status",
    key: "status",
    width: 120,
    render(row) {
      const type = row.status === "failed" ? "error" : row.status === "success" ? "success" : "default";
      return h(NTag, { type, size: "small", round: true }, { default: () => row.status });
    },
  },
  { title: "Error", key: "errorMessage" },
  {
    title: "Actions",
    key: "actions",
    width: 180,
    render(row) {
      return h(NSpace, { size: "small" }, () => [
        row.status === "pending"
          ? h(NButton, { size: "small", type: "primary", onClick: () => handleExecute(row.id) }, { default: () => "Run" })
          : null,
        row.status === "failed" || row.status === "partial_success"
          ? h(NButton, { size: "small", type: "warning", onClick: () => handleRetry(row.id) }, { default: () => "Retry" })
          : null,
      ]);
    },
  },
];

async function loadProfiles() {
  profilesLoading.value = true;
  try {
    profiles.value = await listIntegrationProfiles();
  } finally {
    profilesLoading.value = false;
  }
}

async function loadJobs() {
  loading.value = true;
  try {
    jobs.value = await listChannelSyncJobsByWave(waveId.value);
  } finally {
    loading.value = false;
  }
}

async function handlePlan() {
  if (!selectedProfileId.value) return;
  planning.value = true;
  error.value = "";
  planResult.value = null;
  try {
    const result = await planChannelClosure({
      waveId: waveId.value,
      integrationProfileId: selectedProfileId.value,
    });
    planResult.value = result;
    if ((result.decision === "manual_closure" || result.decision === "unsupported") && result.items) {
      const forms: Record<number, { decisionKind: string; reasonCode: string; note: string; evidenceRef: string; operatorId: string }> = {};
      for (const item of result.items) {
        forms[item.fulfillmentLineId] = {
          decisionKind: result.decision === "unsupported" ? "mark_sync_unsupported" : "",
          reasonCode: "",
          note: "",
          evidenceRef: "",
          operatorId: "",
        };
      }
      manualForms.value = forms;
    }
    await loadJobs();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    planning.value = false;
  }
}

async function handleExecute(jobId: number) {
  await executeChannelSyncJob(jobId);
  await loadJobs();
}

async function handleRetry(jobId: number) {
  await retryChannelSyncJob(jobId);
  await loadJobs();
}

async function handleSubmitDecisions() {
  if (!selectedProfileId.value || !planResult.value) return;
  submitting.value = true;
  try {
    const entries = Object.entries(manualForms.value)
      .filter(([, form]) => form.decisionKind)
      .map(([lineId, form]) => ({
        fulfillmentLineId: Number(lineId),
        decisionKind: form.decisionKind,
        reasonCode: form.reasonCode,
        note: form.note,
        evidenceRef: form.evidenceRef,
        operatorId: form.operatorId,
      }));
    await recordChannelClosureDecision({
      waveId: waveId.value,
      integrationProfileId: selectedProfileId.value,
      entries,
    });
    message.success(t("sync.createJob"));
    planResult.value = null;
    await loadJobs();
  } finally {
    submitting.value = false;
  }
}

onMounted(async () => {
  await loadProfiles();
  await loadJobs();
});
</script>

<template>
  <div class="wave-channel-sync-step">
    <div class="mb-6">
      <div class="app-kicker">{{ t("wave.sync") }}</div>
      <h2 class="app-title mt-2">{{ t("sync.title") }}</h2>
      <p class="app-copy mt-3">{{ t("sync.subtitle") }}</p>
    </div>

    <NAlert v-if="error" type="error" class="mb-4" :title="error" />

    <NCard :title="t('sync.planning')" class="mb-4">
      <NSpace align="center">
        <NSelect
          v-model:value="selectedProfileId"
          :options="profileOptions"
          :loading="profilesLoading"
          style="width: 320px"
        />
        <NButton type="primary" :loading="planning" :disabled="!selectedProfileId" @click="handlePlan">
          {{ t("sync.createJob") }}
        </NButton>
      </NSpace>
    </NCard>

    <NCard v-if="planResult && (planResult.decision === 'manual_closure' || planResult.decision === 'unsupported')" class="mb-4">
      <div
        v-for="item in planResult.items"
        :key="item.fulfillmentLineId"
        class="mb-4 rounded border border-gray-200 p-3"
      >
        <div class="mb-2 font-medium">Fulfillment Line #{{ item.fulfillmentLineId }}</div>
        <NSpace vertical>
          <NSelect
            v-model:value="manualForms[item.fulfillmentLineId].decisionKind"
            :options="decisionKindOptions"
          />
          <NInput v-model:value="manualForms[item.fulfillmentLineId].reasonCode" placeholder="Reason Code" />
          <NInput v-model:value="manualForms[item.fulfillmentLineId].note" type="textarea" :rows="2" placeholder="Note" />
          <NInput v-model:value="manualForms[item.fulfillmentLineId].evidenceRef" placeholder="Evidence Ref" />
          <NInput v-model:value="manualForms[item.fulfillmentLineId].operatorId" placeholder="Operator ID" />
        </NSpace>
      </div>
      <NButton type="primary" :loading="submitting" @click="handleSubmitDecisions">
        {{ t("sync.createJob") }}
      </NButton>
    </NCard>

    <NCard :title="t('sync.jobs')">
      <NEmpty v-if="!loading && jobs.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="columns"
        :data="jobs"
        :loading="loading"
        :pagination="false"
        size="small"
        :row-key="(row: dto.ChannelSyncJobDTO) => row.id"
      />
    </NCard>

    <div class="mt-4 flex justify-between">
      <NButton @click="router.push(`/waves/${waveId}/shipment`)">{{ t("wave.prevStep") }}</NButton>
      <NButton secondary @click="router.push(`/waves`)">{{ t("wave.returnToQueue") }}</NButton>
    </div>
  </div>
</template>
