<script setup lang="ts">
import type { NicknameEpisodeViewModel } from '@/shared/lib/customer-resolution'

defineProps<{
  episodes: NicknameEpisodeViewModel[]
  labels: {
    currentDisplayName: string
    observedCount: (count: number) => string
    sourceFallback: string
    empty: string
  }
}>()
</script>

<template>
  <p v-if="episodes.length === 0" class="nickname-timeline__empty">{{ labels.empty }}</p>
  <ol v-else class="nickname-timeline">
    <li v-for="episode in episodes" :key="episode.observationIds.join('-')" class="nickname-timeline__episode">
      <span class="nickname-timeline__marker" aria-hidden="true" />
      <div class="nickname-timeline__content">
        <div class="nickname-timeline__heading">
          <strong>{{ episode.displayValue }}</strong>
          <span v-if="episode.isCurrentDisplayName" class="nickname-timeline__current">
            {{ labels.currentDisplayName }}
          </span>
        </div>
        <span class="nickname-timeline__meta">
          {{ episode.sourceLabel || episode.sourceNamespace || labels.sourceFallback }}
          · {{ episode.firstSeenAt }} → {{ episode.lastSeenAt }}
          · {{ labels.observedCount(episode.observationCount) }}
        </span>
      </div>
    </li>
  </ol>
</template>

<style scoped>
.nickname-timeline {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}

.nickname-timeline__episode {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: var(--space-3);
}

.nickname-timeline__marker {
  width: 8px;
  height: 8px;
  margin-top: 0.4rem;
  border-radius: var(--radius-full);
  background: var(--color-accent);
}

.nickname-timeline__content,
.nickname-timeline__heading {
  display: flex;
  gap: var(--space-2);
}

.nickname-timeline__content {
  flex-direction: column;
}

.nickname-timeline__heading {
  align-items: center;
  flex-wrap: wrap;
  color: var(--color-text-primary);
  font-size: var(--font-size-sm);
}

.nickname-timeline__current {
  padding: 1px var(--space-2);
  border-radius: var(--radius-full);
  color: var(--status-info-fg);
  background: var(--status-info-bg);
  font-size: var(--font-size-xs);
}

.nickname-timeline__meta,
.nickname-timeline__empty {
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
}

.nickname-timeline__empty {
  margin: 0;
}
</style>
