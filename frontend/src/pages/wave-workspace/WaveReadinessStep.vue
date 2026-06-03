<script setup lang="ts">
import { computed, h, inject, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NAlert,
  NButton,
  NDataTable,
  NEmpty,
  NIcon,
  NSpace,
  NSpin,
  NTabPane,
  NTabs,
  NTag,
  useMessage,
} from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import {
  ArrowForwardOutline,
  CheckmarkCircleOutline,
  WarningOutline,
} from "@vicons/ionicons5";
import { listWaveFulfillmentRows } from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";
import { waveWorkspaceSnapshotKey } from "@/shared/model/wave-injection-keys";

import PageHeader from "@/shared/ui/PageHeader.vue";
import GlassCard from "@/shared/ui/GlassCard.vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const message = useMessage();
const snapshot = inject(waveWorkspaceSnapshotKey)

const waveId = computed(() => {
  const id = Number(route.params.waveId);
  return Number.isFinite(id) ? id : null;
});

const fulfillmentRows = ref<dto.WaveFulfillmentRowDTO[]>([]);
const loading = ref(false);
const error = ref("");
const activeTab = ref("address");

async function load() {
  if (!waveId.value) return;
  loading.value = true;
  error.value = "";
  try {
    fulfillmentRows.value = await listWaveFulfillmentRows(waveId.value);
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

onMounted(load);

// ── Derived: address-missing rows ──
const addressMissingRows = computed(() =>
  fulfillmentRows.value.filter(
    (r) => r.addressState === "missing" || r.addressState === "invalid",
  ),
);

const overview = computed(() => snapshot?.value?.overview);

const blockingTotal = computed(() => {
  const ov = overview.value;
  const addrMissing = addressMissingRows.value.length;
  if (!ov)
    return addrMissing;
  return (ov.addressMissingCount || addrMissing) +
    (ov.acceptedWaitingForInput || 0) +
    (ov.mappingBlockedCount || 0);
});

const allReady = computed(
  () => blockingTotal.value === 0 && fulfillmentRows.value.length > 0,
);

// ── Column factories ──
const baseColumns = computed<DataTableColumns<dto.WaveFulfillmentRowDTO>>(
  () => [
    { title: "Line ID", key: "fulfillmentLineId", width: 80 },
    {
      title: "Participant",
      key: "participantDisplay",
      width: 200,
      ellipsis: { tooltip: true },
    },
    {
      title: "Product",
      key: "productDisplay",
      width: 220,
      ellipsis: { tooltip: true },
    },
    { title: "Qty", key: "quantity", width: 70 },
  ],
);

const addressColumns = computed<DataTableColumns<dto.WaveFulfillmentRowDTO>>(
  () => [
    ...baseColumns.value,
    {
      title: "Address",
      key: "addressState",
      width: 120,
      render(row) {
        const state = row.addressState;
        const tType =
          state === "missing"
            ? "error"
            : state === "invalid"
              ? "warning"
              : "success";
        return h(
          NTag,
          { type: tType, size: "tiny", round: true, bordered: false },
          { default: () => state },
        );
      },
    },
    {
      title: "",
      key: "actions",
      width: 140,
      render(row) {
        const customerId = row.customerProfileId;
        return h(
          NButton,
          {
            size: "tiny",
            secondary: true,
            onClick: () => {
              if (customerId) {
                router.push(`/customers/${customerId}`);
              } else {
                router.push(`/customers`);
              }
            },
          },
          { default: () => t("readiness.addressFix") },
        );
      },
    },
  ],
);

function proceedToExecution() {
  if (waveId.value) router.push(`/waves/${waveId.value}/export`);
}

function openInbox() {
  router.push("/demand-inbox?assignment=assigned");
}
</script>

<template>
  <div class="wave-readiness-step">
    <PageHeader
      :title="t('readiness.title')"
      :description="t('readiness.subtitle')"
    >
      <template #actions>
        <NButton secondary @click="load">Refresh</NButton>
        <NButton
          type="primary"
          :disabled="blockingTotal > 0"
          @click="proceedToExecution"
        >
          <template #icon>
            <NIcon><ArrowForwardOutline /></NIcon>
          </template>
          {{ t("readiness.proceed") }}
        </NButton>
      </template>
    </PageHeader>

    <NAlert
      v-if="error"
      type="error"
      class="mb-4"
      :title="error"
      closable
      @close="error = ''"
    />

    <!-- Top status banner -->
    <NAlert v-if="allReady" type="success" :show-icon="true" class="mb-4">
      <template #icon>
        <NIcon><CheckmarkCircleOutline /></NIcon>
      </template>
      {{ t("readiness.allReady") }}
    </NAlert>
    <NAlert
      v-else-if="blockingTotal > 0"
      type="warning"
      :show-icon="true"
      class="mb-4"
    >
      <template #icon>
        <NIcon><WarningOutline /></NIcon>
      </template>
      {{ t("readiness.blockingCount") }}: <strong>{{ blockingTotal }}</strong>
    </NAlert>

    <NSpin :show="loading">
      <GlassCard>
        <NTabs v-model:value="activeTab" type="line" animated>
          <!-- Tab 1: Address -->
          <NTabPane name="address" :tab="t('readiness.tabAddress')">
            <div class="tab-header">
              <div class="tab-subtitle">
                {{ t("readiness.addressMissingTitle") }}
              </div>
              <NTag
                :type="addressMissingRows.length > 0 ? 'error' : 'success'"
                size="small"
                round
                :bordered="false"
              >
                {{ addressMissingRows.length }}
              </NTag>
            </div>
            <NAlert
              v-if="addressMissingRows.length > 0"
              type="warning"
              class="mb-3"
            >
              {{ t("readiness.addressMissingHint") }}
            </NAlert>
            <NEmpty
              v-if="addressMissingRows.length === 0"
              description="All fulfillment lines have an address."
              class="empty-block"
            />
            <NDataTable
              v-else
              :columns="addressColumns"
              :data="addressMissingRows"
              :pagination="{ pageSize: 20 }"
              size="small"
            />
          </NTabPane>

          <!-- Tab 2: Recipient Input -->
          <NTabPane name="input" :tab="t('readiness.tabInput')">
            <div class="tab-header">
              <div class="tab-subtitle">
                {{ t("readiness.inputWaitingTitle") }}
              </div>
              <NTag
                :type="
                  (overview?.acceptedWaitingForInput ?? 0) > 0
                    ? 'warning'
                    : 'success'
                "
                size="small"
                round
                :bordered="false"
              >
                {{ overview?.acceptedWaitingForInput ?? 0 }}
              </NTag>
            </div>
            <NAlert type="info" class="mb-3">
              {{ t("readiness.inputWaitingHint") }}
              Recipient input state is tracked at the DemandLine level. Use the
              global Demand Inbox to update.
            </NAlert>
            <NEmpty
              v-if="(overview?.acceptedWaitingForInput ?? 0) === 0"
              description="All recipient inputs ready."
              class="empty-block"
            />
            <NSpace v-else>
              <NButton type="primary" secondary @click="openInbox">
                <template #icon>
                  <NIcon><ArrowForwardOutline /></NIcon>
                </template>
                Open Demand Inbox
              </NButton>
            </NSpace>
          </NTabPane>

          <!-- Tab 3: Validation Report -->
          <NTabPane name="validation" :tab="t('readiness.tabValidation')">
            <div class="tab-header">
              <div class="tab-subtitle">
                {{ t("readiness.validationTitle") }}
              </div>
            </div>

            <ul class="check-list">
              <li
                class="check-item"
                :class="
                  addressMissingRows.length === 0 ? 'is-ok' : 'is-fail'
                "
              >
                <NIcon class="check-icon">
                  <CheckmarkCircleOutline
                    v-if="addressMissingRows.length === 0"
                  />
                  <WarningOutline v-else />
                </NIcon>
                <span class="check-text">
                  Address completeness — {{ addressMissingRows.length }}
                  missing
                </span>
              </li>
              <li
                class="check-item"
                :class="
                  (overview?.acceptedWaitingForInput ?? 0) === 0
                    ? 'is-ok'
                    : 'is-fail'
                "
              >
                <NIcon class="check-icon">
                  <CheckmarkCircleOutline
                    v-if="(overview?.acceptedWaitingForInput ?? 0) === 0"
                  />
                  <WarningOutline v-else />
                </NIcon>
                <span class="check-text">
                  Recipient input collection —
                  {{ overview?.acceptedWaitingForInput ?? 0 }} waiting
                </span>
              </li>
              <li
                class="check-item"
                :class="
                  (overview?.mappingBlockedCount ?? 0) === 0
                    ? 'is-ok'
                    : 'is-fail'
                "
              >
                <NIcon class="check-icon">
                  <CheckmarkCircleOutline
                    v-if="(overview?.mappingBlockedCount ?? 0) === 0"
                  />
                  <WarningOutline v-else />
                </NIcon>
                <span class="check-text">
                  Demand mapping —
                  {{ overview?.mappingBlockedCount ?? 0 }} blocked
                </span>
              </li>
              <li
                class="check-item"
                :class="
                  (overview?.fulfillmentReadyCount ?? 0) > 0
                    ? 'is-ok'
                    : 'is-fail'
                "
              >
                <NIcon class="check-icon">
                  <CheckmarkCircleOutline
                    v-if="(overview?.fulfillmentReadyCount ?? 0) > 0"
                  />
                  <WarningOutline v-else />
                </NIcon>
                <span class="check-text">
                  Fulfillment lines ready —
                  {{ overview?.fulfillmentReadyCount ?? 0 }} of
                  {{ overview?.fulfillmentCount ?? 0 }}
                </span>
              </li>
            </ul>

            <NSpace class="mt-6">
              <NButton
                type="primary"
                :disabled="blockingTotal > 0"
                @click="proceedToExecution"
              >
                <template #icon>
                  <NIcon><ArrowForwardOutline /></NIcon>
                </template>
                {{ t("readiness.proceed") }}
              </NButton>
            </NSpace>
          </NTabPane>
        </NTabs>
      </GlassCard>
    </NSpin>
  </div>
</template>

<style scoped>
.wave-readiness-step {
  display: flex;
  flex-direction: column;
}

.tab-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.tab-subtitle {
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--text);
}

.empty-block {
  padding: 32px 0;
}

.check-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.check-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: 10px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  font-size: 0.85rem;
}

.check-item.is-ok {
  background: rgba(34, 197, 94, 0.05);
  border-color: rgba(34, 197, 94, 0.25);
}

.check-item.is-fail {
  background: rgba(234, 179, 8, 0.05);
  border-color: rgba(234, 179, 8, 0.25);
}

.check-icon {
  font-size: 1.2rem;
}

.check-text {
  color: var(--text);
}

.mb-3 {
  margin-bottom: 12px;
}
.mb-4 {
  margin-bottom: 16px;
}
.mt-6 {
  margin-top: 24px;
}
</style>
