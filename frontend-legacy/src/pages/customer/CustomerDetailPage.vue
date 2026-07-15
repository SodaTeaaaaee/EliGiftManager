<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NAlert,
  NButton,
  NDataTable,
  NEmpty,
  NIcon,
  NSpace,
  NSpin,
  NTag,
} from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { ArrowBackOutline } from "@vicons/ionicons5";
import {
  getCustomerProfile,
  listAddressesByProfile,
} from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

import GlassCard from "@/shared/ui/GlassCard.vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const customerId = computed(() => Number(route.params.id) || 0);

const profile = ref<dto.CustomerProfileDTO | null>(null);
const addresses = ref<dto.CustomerAddressDTO[]>([]);
const loading = ref(false);
const error = ref("");

async function load() {
  if (!customerId.value) return;
  loading.value = true;
  error.value = "";
  try {
    profile.value = await getCustomerProfile(customerId.value);
    addresses.value = await listAddressesByProfile(customerId.value);
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

watch(customerId, () => {
  void load();
});

onMounted(load);

function goBack() {
  router.push("/customers");
}

// ── Identity columns ──
const identityColumns = computed<DataTableColumns<dto.CustomerIdentityDTO>>(() => [
  { title: "ID", key: "id", width: 60 },
  { title: "Platform", key: "identityPlatform", width: 140 },
  { title: "Type", key: "identityType", width: 140 },
  {
    title: "Identity Value",
    key: "identityValue",
    ellipsis: { tooltip: true },
  },
  {
    title: "Primary",
    key: "isPrimary",
    width: 80,
    render(row) {
      return row.isPrimary
        ? h(
            NTag,
            { size: "tiny", type: "success", round: true, bordered: false },
            { default: () => t("customerDetail.primaryIdentity") },
          )
        : "—";
    },
  },
  { title: "Created", key: "createdAt", width: 110, render: (r) => r.createdAt?.slice(0, 10) || "—" },
]);

// ── Address columns ──
const addressColumns = computed<DataTableColumns<dto.CustomerAddressDTO>>(() => [
  { title: "ID", key: "id", width: 60 },
  { title: "Label", key: "label", width: 120 },
  { title: "Recipient", key: "recipientName", width: 120 },
  { title: "Phone", key: "phone", width: 130 },
  {
    title: "Address",
    key: "addr",
    ellipsis: { tooltip: true },
    render(row) {
      const parts = [
        row.country,
        row.province,
        row.city,
        row.district,
        row.addressLine1,
        row.addressLine2,
      ].filter(Boolean);
      return parts.join(" / ");
    },
  },
  {
    title: "Flags",
    key: "flags",
    width: 140,
    render(row) {
      const tags: any[] = [];
      if (row.isDefault)
        tags.push(
          h(
            NTag,
            { size: "tiny", type: "info", round: true, bordered: false },
            { default: () => "default" },
          ),
        );
      if (row.isTest)
        tags.push(
          h(
            NTag,
            { size: "tiny", round: true, bordered: false },
            { default: () => "test" },
          ),
        );
      if (row.validationStatus && row.validationStatus !== "unvalidated")
        tags.push(
          h(
            NTag,
            {
              size: "tiny",
              type: row.validationStatus === "valid" ? "success" : "warning",
              round: true,
              bordered: false,
            },
            { default: () => row.validationStatus },
          ),
        );
      return h("div", { style: "display:flex;gap:4px;flex-wrap:wrap;" }, tags);
    },
  },
]);

const profileTypeTagType = computed<"info" | "success" | "warning" | "default">(() => {
  switch (profile.value?.profileType) {
    case "member":
      return "info";
    case "buyer":
      return "success";
    case "mixed":
      return "warning";
    default:
      return "default";
  }
});
</script>

<template>
  <div class="customer-detail-page">
    <!-- Header -->
    <div class="detail-header">
      <NButton text @click="goBack">
        <template #icon>
          <NIcon><ArrowBackOutline /></NIcon>
        </template>
        {{ t("customerDetail.backToList") }}
      </NButton>

      <div class="detail-header-main">
        <div class="app-kicker">{{ t("nav.customers") }}</div>
        <h1 class="detail-title">{{ profile?.displayName || "—" }}</h1>
        <div class="detail-subtitle">
          <NTag
            v-if="profile"
            :type="profileTypeTagType"
            size="small"
            round
            :bordered="false"
          >
            {{ profile.profileType }}
          </NTag>
          <span class="detail-subtitle-text">ID: {{ profile?.id ?? "—" }}</span>
        </div>
      </div>
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
        <!-- Section 1: Identities -->
        <GlassCard>
          <div class="section-header">
            <div class="section-title">
              {{ t("customerDetail.sectionIdentities") }}
              <NTag size="tiny" :bordered="false" round>
                {{ profile?.identities?.length ?? 0 }}
              </NTag>
            </div>
          </div>
          <NEmpty
            v-if="!profile?.identities?.length"
            :description="t('customerDetail.noIdentities')"
            class="empty-block"
          />
          <NDataTable
            v-else
            :columns="identityColumns"
            :data="profile.identities"
            :pagination="false"
            size="small"
          />
        </GlassCard>

        <!-- Section 2: Addresses -->
        <GlassCard>
          <div class="section-header">
            <div class="section-title">
              {{ t("customerDetail.sectionAddresses") }}
              <NTag size="tiny" :bordered="false" round>{{ addresses.length }}</NTag>
            </div>
          </div>
          <NEmpty
            v-if="addresses.length === 0"
            :description="t('customerDetail.noAddresses')"
            class="empty-block"
          />
          <NDataTable
            v-else
            :columns="addressColumns"
            :data="addresses"
            :pagination="false"
            size="small"
          />
        </GlassCard>

        <!-- Section 3: Wave History (placeholder — backend may not yet expose this) -->
        <GlassCard>
          <div class="section-header">
            <div class="section-title">{{ t("customerDetail.sectionWaves") }}</div>
          </div>
          <NEmpty
            :description="t('customerDetail.noWaves')"
            class="empty-block"
          >
            <template #extra>
              <span class="muted-hint">
                Wave participation history is derived from FulfillmentLine ↔ CustomerProfile.
                Cross-reference query is not yet exposed in the bridge.
              </span>
            </template>
          </NEmpty>
        </GlassCard>

        <!-- Section 4: Demand History (placeholder) -->
        <GlassCard>
          <div class="section-header">
            <div class="section-title">{{ t("customerDetail.sectionDemands") }}</div>
          </div>
          <NEmpty
            :description="t('customerDetail.noDemands')"
            class="empty-block"
          >
            <template #extra>
              <NSpace align="center" justify="center">
                <NButton
                  size="small"
                  secondary
                  @click="router.push(`/demand-inbox`)"
                >
                  Open Demand Inbox
                </NButton>
              </NSpace>
            </template>
          </NEmpty>
        </GlassCard>

        <!-- Extra Data -->
        <GlassCard v-if="profile?.extraData">
          <div class="section-title">Extra Data</div>
          <pre class="json-preview">{{ profile.extraData }}</pre>
        </GlassCard>
      </div>
    </NSpin>
  </div>
</template>

<style scoped>
.customer-detail-page {
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
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  font-size: 0.85rem;
}

.detail-subtitle-text {
  color: var(--muted);
  font-family: monospace;
  font-size: 0.75rem;
}

.detail-sections {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}

.section-title {
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--muted);
  display: flex;
  align-items: center;
  gap: 8px;
}

.empty-block {
  padding: 32px 0;
}

.muted-hint {
  color: var(--muted);
  font-size: 0.75rem;
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

.mb-4 {
  margin-bottom: 16px;
}
</style>
