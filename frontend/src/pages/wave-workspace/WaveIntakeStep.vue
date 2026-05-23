<script setup lang="ts">
import { computed, h, onMounted, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { NAlert, NButton, NDataTable, NEmpty, NSpace, NTag, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import {
  assignDemandToWave,
  listDemandInboxRows,
} from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

import PageHeader from "@/shared/ui/PageHeader.vue";
import GlassCard from "@/shared/ui/GlassCard.vue";

const route = useRoute();
const { t } = useI18n();
const message = useMessage();

const waveId = computed(() => {
  const id = Number(route.params.waveId);
  return Number.isFinite(id) ? id : null;
});

const loading = ref(false);
const error = ref("");
const inbox = ref<dto.DemandInboxRowDTO[]>([]);
const assigningId = ref<number | null>(null);

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
  {
    title: "Action",
    key: "actions",
    width: 250,
    render(row) {
      if (row.assigned) {
        if (row.assignedWaveId === waveId.value) {
          return h(NTag, { type: "info" }, { default: () => "Assigned to this Wave" });
        }
        return row.assignedWaveLabel || t("demandIntake.assigned");
      }
      return h(
        NButton,
        {
          size: "small",
          type: "primary",
          loading: assigningId.value === row.demandDocumentId,
          onClick: () => handleAssignToThisWave(row.demandDocumentId),
        },
        { default: () => "Assign to this Wave" },
      );
    },
  },
]);

async function loadInbox() {
  loading.value = true;
  error.value = "";
  try {
    inbox.value = await listDemandInboxRows({
      assignment: "all",
      demandKind: "",
    });
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

async function handleAssignToThisWave(demandDocumentId: number) {
  if (!waveId.value) return;
  assigningId.value = demandDocumentId;
  try {
    await assignDemandToWave(waveId.value, demandDocumentId);
    message.success("Assigned to wave successfully");
    await loadInbox();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    assigningId.value = null;
  }
}

watch(waveId, () => {
  if (waveId.value) loadInbox();
}, { immediate: true });
</script>

<template>
  <div class="wave-intake-step">
    <PageHeader 
      title="Intake Demands" 
      description="Assign pending demands to this wave." 
    >
      <template #actions>
        <NButton secondary @click="loadInbox">Refresh</NButton>
      </template>
    </PageHeader>

    <NAlert v-if="error" type="error" class="mb-6 rounded-lg" :title="error" />

    <GlassCard>
      <NEmpty v-if="!loading && inbox.length === 0" description="No demands found in the global inbox." class="my-12" />
      <NDataTable
        v-else
        :columns="columns"
        :data="inbox"
        :loading="loading"
        :pagination="false"
        size="large"
      />
    </GlassCard>
  </div>
</template>
