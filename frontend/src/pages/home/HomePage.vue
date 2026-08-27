<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NButton, NDataTable, NSpin } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard, StatCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import type { StatusTone } from '@/shared/i18n/glossary'
import { getHomeBuckets } from '@/shared/api/bridge'
import type { HomeBuckets, Wave } from '@/entities/models'

const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const buckets = ref<HomeBuckets>({
  Unassigned: 0,
  DuplicateAsk: 0,
  AlignmentConflict: 0,
  IdentityUnattached: 0,
  PendingRevisions: 0,
  BlockedResults: 0,
  WritebackFailed: 0,
  ResidualClose: 0,
  RevisionFrozenConflicts: 0,
  RecentWaves: [],
})

async function loadData() {
  loading.value = true
  try {
    buckets.value = await getHomeBuckets()
  } catch (err) {
    console.error('Failed to load home buckets:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadData()
})

interface HomeBucketCard {
  key: string
  label: string
  value: string
  caption: string
  tone: StatusTone
  to: string
  /** Extra warning line rendered in the card footer (empty = none). */
  warning: string
}

const bucketCards = computed<HomeBucketCard[]>(() => [
  {
    key: 'unassigned',
    label: t('home.buckets.unassigned'),
    value: String(buckets.value.Unassigned),
    caption: t('home.buckets.unassignedDesc'),
    tone: buckets.value.Unassigned > 0 ? 'warning' : 'neutral',
    to: '/inbox',
    warning: '',
  },
  {
    key: 'duplicateAsk',
    label: t('home.buckets.duplicateAsk'),
    value: String(buckets.value.DuplicateAsk),
    caption: t('home.buckets.duplicateAskDesc'),
    tone: buckets.value.DuplicateAsk > 0 ? 'warning' : 'neutral',
    to: '/inbox',
    warning: '',
  },
  {
    key: 'alignmentConflict',
    label: t('home.buckets.alignmentConflict'),
    value: String(buckets.value.AlignmentConflict),
    caption: t('home.buckets.alignmentConflictDesc'),
    tone: buckets.value.AlignmentConflict > 0 ? 'error' : 'neutral',
    to: '/inbox',
    warning: '',
  },
  {
    key: 'identityUnattached',
    label: t('home.buckets.identityUnattached'),
    value: String(buckets.value.IdentityUnattached),
    caption: t('home.buckets.identityUnattachedDesc'),
    tone: buckets.value.IdentityUnattached > 0 ? 'error' : 'neutral',
    to: '/inbox',
    warning: '',
  },
  {
    key: 'pendingRevisions',
    label: t('home.buckets.pendingRevisions'),
    value: String(buckets.value.PendingRevisions),
    caption:
      buckets.value.RevisionFrozenConflicts > 0
        ? t('home.buckets.revisionFrozenConflictDesc')
        : t('home.buckets.pendingRevisionsDesc'),
    tone:
      buckets.value.PendingRevisions === 0
        ? buckets.value.RevisionFrozenConflicts > 0
          ? 'warning'
          : 'neutral'
        : buckets.value.RevisionFrozenConflicts > 0
          ? 'error'
          : 'warning',
    to: '/inbox',
    warning:
      buckets.value.RevisionFrozenConflicts > 0
        ? t('home.buckets.revisionFrozenConflict', {
            n: buckets.value.RevisionFrozenConflicts,
          })
        : '',
  },
  {
    key: 'blockedResults',
    label: t('home.buckets.blockedResults'),
    value: String(buckets.value.BlockedResults),
    caption: t('home.buckets.blockedResultsDesc'),
    tone: buckets.value.BlockedResults > 0 ? 'error' : 'neutral',
    to: '/waves',
    warning: '',
  },
  {
    key: 'writebackFailed',
    label: t('home.buckets.writebackFailed'),
    value: String(buckets.value.WritebackFailed),
    caption: t('home.buckets.writebackFailedDesc'),
    tone: buckets.value.WritebackFailed > 0 ? 'error' : 'neutral',
    to: '/waves',
    warning: '',
  },
  {
    key: 'residualClose',
    label: t('home.buckets.residualClose'),
    value: String(buckets.value.ResidualClose),
    caption: t('home.buckets.residualCloseDesc'),
    tone: buckets.value.ResidualClose > 0 ? 'info' : 'neutral',
    to: '/waves',
    warning: '',
  },
])

function navigateTo(path: string) {
  void router.push(path)
}

function openWave(wave: Wave) {
  void router.push(`/waves/${wave.ID}/results`)
}

const waveColumns = [
  {
    title: t('waves.waveNo'),
    key: 'WaveNo',
    width: 140,
  },
  {
    title: t('waves.name'),
    key: 'Name',
  },
  {
    title: t('waves.status'),
    key: 'CloseResult',
    width: 130,
    render(row: Wave) {
      return h(StatusBadge, {
        dimension: 'waveCloseResult',
        value: row.CloseResult || 'open',
      })
    },
  },
  {
    title: t('waves.notes'),
    key: 'Notes',
    ellipsis: { tooltip: true },
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 120,
    render(row: Wave) {
      return h(
        NButton,
        {
          size: 'small',
          type: 'primary',
          quaternary: true,
          onClick: () => openWave(row),
        },
        { default: () => t('waves.open') },
      )
    },
  },
]
</script>

<template>
  <div class="home-page">
    <PageHeader :title="t('home.title')" :description="t('home.subtitle')">
      <template #actions>
        <NButton size="small" :loading="loading" @click="loadData">
          {{ t('common.refresh') }}
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="loading && !buckets.RecentWaves.length">
      <div class="home-page__buckets">
        <StatCard
          v-for="b in bucketCards"
          :key="b.key"
          :label="b.label"
          :value="b.value"
          :caption="b.caption"
          :tone="b.tone"
          :clickable="true"
          @click="navigateTo(b.to)"
        >
          <template v-if="b.warning" #footer>
            <span
              class="home-page__bucket-warning"
              :title="t('home.buckets.revisionFrozenConflictHint')"
            >
              {{ b.warning }}
            </span>
          </template>
        </StatCard>
      </div>

      <SectionCard :title="t('home.recentWaves')">
        <div v-if="!buckets.RecentWaves || buckets.RecentWaves.length === 0" class="home-page__empty">
          <EmptyState :title="t('home.noWaves')" size="sm" />
        </div>
        <div v-else class="home-page__table">
          <NDataTable
            :columns="waveColumns"
            :data="buckets.RecentWaves"
            :row-key="(row: Wave) => row.ID"
            size="small"
          />
        </div>
      </SectionCard>
    </NSpin>
  </div>
</template>

<style scoped>
.home-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.home-page__buckets {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.home-page__empty {
  padding: var(--space-4) 0;
}

.home-page__table {
  width: 100%;
}

.home-page__bucket-warning {
  color: var(--status-error-fg);
  font-weight: var(--font-weight-medium);
}
</style>
