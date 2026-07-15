<script setup lang="ts">
/**
 * Task Center (plan 3.1) — the default landing route. Primary content is
 * the cross-wave action stream (one ActionCard per blocked bucket, sorted
 * by urgency, each a deep link into a pre-filtered wave workspace), plus a
 * secondary "waves in progress" rollup and an onboarding empty state for a
 * brand-new install. Data comes from `getActionCenterSummary()` (bucket
 * counts + deep-link filters) joined with `listWavesFiltered()` (lifecycle
 * stage + last-activity timestamp, which the action-center summary does not
 * carry) by `waveId`.
 */
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { NButton } from "naive-ui";
import { PageHeader } from "@/shared/ui/shell";
import { SectionCard, StatCard } from "@/shared/ui/cards";
import { EmptyState } from "@/shared/ui/empty-state";
import { getActionCenterSummary, listWavesFiltered } from "@/shared/api/bridge";
import { buildWaveFilterLink } from "@/shared/lib/wave-filter-link";
import { useWindowFocusRefresh } from "@/shared/lib/useWindowFocusRefresh";
import type { StatusTone } from "@/shared/i18n/glossary";
import type { dto } from "../../../wailsjs/go/models";
import ActionCard from "./components/ActionCard.vue";
import WaveSummaryCard from "./components/WaveSummaryCard.vue";

interface InProgressWave {
  id: number;
  name: string;
  lifecycleStage: string;
  updatedAt: string;
}

const { t } = useI18n({ useScope: "global" });
const router = useRouter();

// bucketKind (backend snake_case) -> taskCenter.buckets.* i18n key suffix.
const BUCKET_LABEL_KEYS: Record<string, string> = {
  missing_address: "missingAddress",
  waiting_input: "waitingInput",
  mapping_blocked: "mappingBlocked",
  channel_sync_failed: "channelSyncFailed",
  awaiting_manual_closure: "awaitingManualClosure",
  drift_needs_review: "driftReview",
};

// bucketKind -> ActionCard tone. channel_sync_failed is a hard failure (error);
// the rest are attention-needed-but-recoverable (warning). Unknown future
// bucket kinds fall back to warning rather than crashing.
const BUCKET_TONES: Record<string, StatusTone> = {
  missing_address: "warning",
  waiting_input: "info",
  mapping_blocked: "warning",
  channel_sync_failed: "error",
  awaiting_manual_closure: "warning",
  drift_needs_review: "warning",
};

interface BucketStreamCard {
  kind: "bucket";
  key: string;
  waveId: number;
  waveName: string;
  bucketKind: string;
  count: number;
  filter: dto.ActionCenterBucketFilterDTO;
  tone: StatusTone;
}

interface InboxStreamCard {
  kind: "inbox";
  key: string;
  count: number;
  tone: StatusTone;
}

type StreamCard = BucketStreamCard | InboxStreamCard;

const summary = ref<dto.ActionCenterSummaryDTO | null>(null);
const waveMetaMap = ref<Map<number, dto.WaveDTO>>(new Map());
const totalWavesCount = ref(0);
const loading = ref(true);
const hasLoadedOnce = ref(false);

async function loadData(): Promise<void> {
  loading.value = true;
  try {
    const [summaryResult, wavesPage] = await Promise.all([
      getActionCenterSummary(),
      listWavesFiltered({ page: 1, pageSize: 50, sortBy: "updatedAt", sortDesc: true }),
    ]);
    summary.value = summaryResult;
    waveMetaMap.value = new Map(wavesPage.items.map((wave) => [wave.id, wave]));
    totalWavesCount.value = wavesPage.pagination.totalCount;
  } catch {
    summary.value = null;
    waveMetaMap.value = new Map();
    totalWavesCount.value = 0;
  } finally {
    loading.value = false;
    hasLoadedOnce.value = true;
  }
}

onMounted(loadData);
useWindowFocusRefresh(loadData);

const showOnboarding = computed(
  () => hasLoadedOnce.value && totalWavesCount.value === 0 && (!summary.value || summary.value.waves.length === 0),
);

/** Sorted by urgency: highest wave.totalBlockedCount first, then largest bucket.count within each wave. Inbox card is appended last. */
const streamCards = computed<StreamCard[]>(() => {
  if (!summary.value) return [];
  const cards: StreamCard[] = [];

  const sortedWaves = [...summary.value.waves].sort((a, b) => b.totalBlockedCount - a.totalBlockedCount);
  for (const wave of sortedWaves) {
    const sortedBuckets = [...wave.buckets].sort((a, b) => b.count - a.count);
    for (const bucket of sortedBuckets) {
      cards.push({
        kind: "bucket",
        key: `${wave.waveId}:${bucket.bucketKind}`,
        waveId: wave.waveId,
        waveName: wave.waveName,
        bucketKind: bucket.bucketKind,
        count: bucket.count,
        filter: bucket.filter,
        tone: BUCKET_TONES[bucket.bucketKind] ?? "warning",
      });
    }
  }

  if (summary.value.inboxPendingIntakeCount > 0) {
    cards.push({
      kind: "inbox",
      key: "inbox",
      count: summary.value.inboxPendingIntakeCount,
      tone: "info",
    });
  }

  return cards;
});

const inProgressWaves = computed<InProgressWave[]>(() => {
  if (!summary.value) return [];
  return summary.value.waves.map((wave) => {
    const meta = waveMetaMap.value.get(wave.waveId);
    return {
      id: wave.waveId,
      name: wave.waveName,
      lifecycleStage: meta?.lifecycleStage ?? "",
      updatedAt: meta?.updatedAt ?? "",
    };
  });
});

function cardLabel(card: StreamCard): string {
  if (card.kind === "inbox") return t("taskCenter.inbox.pendingIntake", { count: card.count });
  const key = BUCKET_LABEL_KEYS[card.bucketKind];
  return key ? t(`taskCenter.buckets.${key}`) : card.bucketKind;
}

function handleCardClick(card: StreamCard): void {
  if (card.kind === "bucket") {
    router.push(buildWaveFilterLink(card.waveId, card.filter));
    return;
  }
  router.push({ name: "inbox" });
}

function handleWaveClick(waveId: number): void {
  router.push({ name: "wave-workspace", params: { id: waveId } });
}

function handleOnboardingCta(): void {
  router.push("/waves");
}
</script>

<template>
  <div class="home-page">
    <PageHeader :title="t('taskCenter.title')" :description="t('taskCenter.subtitle')">
      <template #actions>
        <NButton size="small" :loading="loading" @click="loadData">{{ t("taskCenter.refresh") }}</NButton>
      </template>
    </PageHeader>

    <EmptyState v-if="showOnboarding" :title="t('taskCenter.onboarding.title')" :description="t('taskCenter.onboarding.description')">
      <NButton type="primary" @click="handleOnboardingCta">{{ t("taskCenter.onboarding.cta") }}</NButton>
    </EmptyState>

    <template v-else>
      <SectionCard :title="t('taskCenter.actionStream.title')">
        <div v-if="loading && !hasLoadedOnce" class="home-page__loading-grid">
          <StatCard v-for="n in 4" :key="n" :label="t('common.loading')" value="—" />
        </div>
        <EmptyState v-else-if="!streamCards.length" size="sm" :title="t('taskCenter.actionStream.empty')" />
        <div v-else class="home-page__action-grid">
          <ActionCard
            v-for="card in streamCards"
            :key="card.key"
            :bucket-label="cardLabel(card)"
            :wave-name="card.kind === 'bucket' ? card.waveName : undefined"
            :count="card.count"
            :tone="card.tone"
            @click="handleCardClick(card)"
          />
        </div>
      </SectionCard>

      <SectionCard :title="t('taskCenter.inProgress.title')">
        <div v-if="loading && !hasLoadedOnce" class="home-page__loading-grid">
          <StatCard v-for="n in 3" :key="n" :label="t('common.loading')" value="—" />
        </div>
        <EmptyState v-else-if="!inProgressWaves.length" size="sm" :title="t('taskCenter.inProgress.empty')" />
        <div v-else class="home-page__wave-grid">
          <WaveSummaryCard v-for="wave in inProgressWaves" :key="wave.id" :wave="wave" @click="handleWaveClick(wave.id)" />
        </div>
      </SectionCard>
    </template>
  </div>
</template>

<style scoped>
.home-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.home-page__loading-grid,
.home-page__action-grid,
.home-page__wave-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--space-3);
}
</style>
