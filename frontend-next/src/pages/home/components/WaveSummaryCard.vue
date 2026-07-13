<script setup lang="ts">
/**
 * WaveSummaryCard — a single "in progress" wave rollup tile (plan 3.1):
 * name + lifecycle-stage badge + relative last-activity time. Purely
 * presentational — the parent (`HomePage.vue`) owns the click -> workspace
 * navigation. No funnel thumbnail (no cheap per-wave overview data source
 * available here — see plan notes).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { StatusBadge } from '@/shared/ui/status'
import { formatGridDate } from '@/shared/ui/data-grid'

const props = defineProps<{
  wave: {
    id: number
    name: string
    /** Treated as a plain string — the glossary's lifecycleStage table resolves label/tone. */
    lifecycleStage: string
    updatedAt: string
  }
}>()

const emit = defineEmits<{ click: [MouseEvent] }>()

const { t } = useI18n({ useScope: 'global' })

const lastActivityText = computed(() =>
  t('taskCenter.lastActivity', { time: formatGridDate(props.wave.updatedAt, 'relative') }),
)

function handleClick(event: MouseEvent): void {
  emit('click', event)
}
</script>

<template>
  <button type="button" class="wave-summary-card" @click="handleClick">
    <span class="wave-summary-card__heading">
      <span class="wave-summary-card__name">{{ wave.name }}</span>
      <StatusBadge dimension="lifecycleStage" :value="wave.lifecycleStage" size="sm" />
    </span>
    <span class="wave-summary-card__meta">{{ lastActivityText }}</span>
  </button>
</template>

<style scoped>
.wave-summary-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-2);
  min-width: 0;
  background: var(--card-bg);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  box-shadow: var(--card-shadow);
  padding: var(--card-padding);
  text-align: left;
  font-family: var(--font-body);
  color: inherit;
  cursor: pointer;
  transition:
    transform var(--duration-fast) var(--ease-out),
    box-shadow var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out);
}

.wave-summary-card:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
  border-color: var(--color-border-strong);
}

.wave-summary-card:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.wave-summary-card:active {
  transform: translateY(0);
}

.wave-summary-card__heading {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-2);
  min-width: 0;
  max-width: 100%;
}

.wave-summary-card__name {
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.wave-summary-card__meta {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}
</style>
