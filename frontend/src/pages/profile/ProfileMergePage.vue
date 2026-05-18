<template>
  <div class="page">
    <div class="page-header">
      <h1>{{ t("merge.title") }}</h1>
      <p>{{ t("merge.subtitle") }}</p>
    </div>

    <n-space vertical>
      <n-select
        v-model:value="sourceId"
        :options="profileOptions"
        :placeholder="t('merge.sourceProfile')"
        style="width: 360px"
      />
      <n-text depth="3" style="font-size:0.8rem">→ {{ t("merge.sourceHint") }}</n-text>

      <n-select
        v-model:value="targetId"
        :options="profileOptions"
        :placeholder="t('merge.targetProfile')"
        style="width: 360px"
      />
      <n-text depth="3" style="font-size:0.8rem">→ {{ t("merge.targetHint") }}</n-text>

      <n-button
        type="error"
        :disabled="!sourceId || !targetId || sourceId === targetId"
        @click="showConfirm = true"
      >
        {{ t("merge.execute") }}
      </n-button>
    </n-space>

    <!-- Confirm Dialog -->
    <n-modal v-model:show="showConfirm" :title="t('merge.confirmTitle')">
      <n-card style="width:480px" :bordered="false" role="dialog">
        <p>{{ t("merge.confirmDesc") }}</p>
        <p><strong>{{ t("merge.sourceProfile") }}:</strong> {{ profileLabel(sourceId) }}</p>
        <p><strong>{{ t("merge.targetProfile") }}:</strong> {{ profileLabel(targetId) }}</p>
        <template #footer>
          <n-space justify="end">
            <n-button @click="showConfirm = false">{{ t("common.cancel") }}</n-button>
            <n-button type="error" :loading="merging" @click="doMerge">{{ t("merge.execute") }}</n-button>
          </n-space>
        </template>
      </n-card>
    </n-modal>

    <!-- Result -->
    <n-card v-if="result" title="Merge Result" style="margin-top:20px; max-width:480px">
      <n-space vertical>
        <n-text>{{ t("merge.result.identityCount") }}: {{ result.migratedIdentityCount }}</n-text>
        <n-text>{{ t("merge.result.addressCount") }}: {{ result.migratedAddressCount }}</n-text>
        <n-text>{{ t("merge.result.demandDocs") }}: {{ result.updatedDemandDocs }}</n-text>
        <n-text>{{ t("merge.result.participants") }}: {{ result.updatedParticipants }}</n-text>
        <n-text>{{ t("merge.result.fulfillmentLines") }}: {{ result.updatedFulfillmentLines }}</n-text>
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import {
  NButton, NSelect, NSpace, NModal, NCard, NText, useMessage,
} from "naive-ui";
import { useI18n } from "@/shared/i18n";
import { listProfiles, mergeProfiles } from "@/shared/lib/wails/app";
import type { MergeProfilesResult } from "@/entities/merge";

const { t } = useI18n();
const message = useMessage();

const profileOptions = ref<{ label: string; value: number }[]>([]);
const sourceId = ref<number | null>(null);
const targetId = ref<number | null>(null);
const showConfirm = ref(false);
const merging = ref(false);
const result = ref<MergeProfilesResult | null>(null);

function profileLabel(id: number | null): string {
  const p = profileOptions.value.find(o => o.value === id);
  return p?.label ?? String(id ?? "");
}

async function doMerge() {
  if (!sourceId.value || !targetId.value) return;
  merging.value = true;
  try {
    result.value = await mergeProfiles({
      sourceProfileId: sourceId.value,
      targetProfileId: targetId.value,
    });
    showConfirm.value = false;
    message.success(t("merge.success"));
  } finally {
    merging.value = false;
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
.page { padding: 24px; max-width: 800px; }
.page-header { margin-bottom: 20px; }
.page-header h1 { font-size: 1.5rem; font-weight: 700; margin: 0; }
.page-header p { color: var(--text-muted); margin: 4px 0 0; }
</style>
