<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import {
  NAlert,
  NButton,
  NCollapse,
  NCollapseItem,
  NEmpty,
  NIcon,
  NInput,
  NPopconfirm,
  NSpace,
  NTag,
  useMessage,
} from "naive-ui";
import {
  AddCircleOutline,
  ArrowForwardOutline,
  PlayCircleOutline,
  TrashOutline,
} from "@vicons/ionicons5";
import {
  deleteProfile,
  listProfiles,
  seedDefaultProfiles,
} from "@/shared/lib/wails/app";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

import PageHeader from "@/shared/ui/PageHeader.vue";
import GlassCard from "@/shared/ui/GlassCard.vue";

const router = useRouter();
const { t } = useI18n();
const message = useMessage();

const profiles = ref<dto.IntegrationProfileDTO[]>([]);
const loading = ref(false);
const error = ref("");
const seeding = ref(false);
const search = ref("");

async function load() {
  loading.value = true;
  error.value = "";
  try {
    profiles.value = await listProfiles();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

async function handleSeed() {
  seeding.value = true;
  try {
    await seedDefaultProfiles();
    message.success("Default profiles seeded");
    await load();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    seeding.value = false;
  }
}

async function handleDelete(profile: dto.IntegrationProfileDTO) {
  try {
    await deleteProfile(profile.id);
    message.success(`Deleted: ${profile.profileKey}`);
    await load();
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

function goToDetail(profile: dto.IntegrationProfileDTO) {
  router.push(`/profiles/${profile.id}`);
}

function goToCreate() {
  router.push(`/profiles/0?mode=create`);
}

// Group by sourceChannel + filter by search
const groupedProfiles = computed(() => {
  const kw = search.value.trim().toLowerCase();
  const filtered = kw
    ? profiles.value.filter(
        (p) =>
          p.profileKey.toLowerCase().includes(kw) ||
          p.sourceChannel.toLowerCase().includes(kw) ||
          (p.sourceSurface || "").toLowerCase().includes(kw),
      )
    : profiles.value;

  const groups = new Map<string, dto.IntegrationProfileDTO[]>();
  for (const p of filtered) {
    const key = p.sourceChannel || "(unknown)";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key)!.push(p);
  }
  return Array.from(groups.entries())
    .map(([channel, items]) => ({ channel, items }))
    .sort((a, b) => a.channel.localeCompare(b.channel));
});

function demandKindTagType(kind: string): "info" | "success" | "default" {
  if (kind === "membership_entitlement") return "info";
  if (kind === "retail_order") return "success";
  return "default";
}

function capabilityTags(p: dto.IntegrationProfileDTO): string[] {
  const tags: string[] = [];
  if (p.supportsApiImport) tags.push("api-in");
  if (p.supportsApiExport) tags.push("api-out");
  if (p.supportsPartialShipment) tags.push("partial-ship");
  if (p.requiresCarrierMapping) tags.push("carrier-map");
  if (p.requiresExternalOrderNo) tags.push("ext-order-no");
  if (p.allowsManualClosure) tags.push("manual-closure");
  return tags;
}

onMounted(load);
</script>

<template>
  <div class="profile-management-page">
    <PageHeader
      :title="t('profile.hubTitle') || 'Integration Profiles'"
      :description="t('profile.hubSubtitle') || 'Profile is a business-surface contract; templates and connectors live underneath each profile.'"
    >
      <template #actions>
        <NButton secondary :loading="seeding" @click="handleSeed">
          <template #icon>
            <NIcon><PlayCircleOutline /></NIcon>
          </template>
          Seed Defaults
        </NButton>
        <NButton type="primary" @click="goToCreate">
          <template #icon>
            <NIcon><AddCircleOutline /></NIcon>
          </template>
          New Profile
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

    <div class="filter-bar mb-4">
      <NInput
        v-model:value="search"
        placeholder="Search by profile key / source channel / surface..."
        clearable
        style="max-width: 380px"
      />
    </div>

    <GlassCard>
      <NEmpty
        v-if="!loading && groupedProfiles.length === 0"
        description="No integration profiles yet. Use [Seed Defaults] to create starter set."
        class="empty-block"
      />
      <NCollapse
        v-else
        :default-expanded-names="groupedProfiles.map((g) => g.channel)"
        accordion
      >
        <NCollapseItem
          v-for="group in groupedProfiles"
          :key="group.channel"
          :name="group.channel"
        >
          <template #header>
            <div class="group-header">
              <span class="group-channel">{{ group.channel }}</span>
              <NTag size="small" :bordered="false" round>{{ group.items.length }}</NTag>
            </div>
          </template>

          <ul class="profile-list">
            <li
              v-for="p in group.items"
              :key="p.id"
              class="profile-row"
              @click="goToDetail(p)"
            >
              <div class="profile-row-main">
                <div class="profile-key">{{ p.profileKey }}</div>
                <div class="profile-meta">
                  <NTag
                    size="tiny"
                    :type="demandKindTagType(p.demandKind)"
                    :bordered="false"
                    round
                  >
                    {{ p.demandKind || "—" }}
                  </NTag>
                  <span class="surface" v-if="p.sourceSurface">
                    · {{ p.sourceSurface }}
                  </span>
                  <span
                    v-for="tag in capabilityTags(p)"
                    :key="tag"
                    class="cap-pill"
                  >{{ tag }}</span>
                </div>
              </div>

              <div class="profile-row-actions" @click.stop>
                <NPopconfirm
                  positive-text="Delete"
                  negative-text="Cancel"
                  @positive-click="handleDelete(p)"
                >
                  <template #trigger>
                    <NButton size="tiny" quaternary type="error">
                      <template #icon>
                        <NIcon><TrashOutline /></NIcon>
                      </template>
                    </NButton>
                  </template>
                  Delete profile <strong>{{ p.profileKey }}</strong>?
                </NPopconfirm>
                <NButton size="tiny" quaternary @click="goToDetail(p)">
                  <template #icon>
                    <NIcon><ArrowForwardOutline /></NIcon>
                  </template>
                </NButton>
              </div>
            </li>
          </ul>
        </NCollapseItem>
      </NCollapse>
    </GlassCard>
  </div>
</template>

<style scoped>
.profile-management-page {
  display: flex;
  flex-direction: column;
}

.filter-bar {
  display: flex;
  gap: 8px;
}

.empty-block {
  padding: 48px 0;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.group-channel {
  font-weight: 700;
  color: var(--text);
  text-transform: capitalize;
}

.profile-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.profile-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.profile-row:hover {
  background: var(--accent-surface, rgba(99, 102, 241, 0.08));
}

.profile-row-main {
  flex: 1;
  min-width: 0;
}

.profile-key {
  font-weight: 700;
  font-size: 0.9rem;
  color: var(--text);
}

.profile-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 4px;
  font-size: 0.75rem;
  color: var(--muted);
}

.surface {
  color: var(--muted);
}

.cap-pill {
  background: rgba(148, 163, 184, 0.12);
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 0.7rem;
  color: var(--muted);
}

.profile-row-actions {
  display: flex;
  gap: 4px;
  align-items: center;
}

.mb-4 {
  margin-bottom: 16px;
}
</style>
