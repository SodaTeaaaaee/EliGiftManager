<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { StatusTone } from '@/shared/i18n/glossary'
import type { FunnelStage } from './types'

/**
 * FunnelBar — the doc-mandated "explainable funnel": raw counts through the
 * fulfillment pipeline, NEVER percentages (plan 3.3.1 / 02-status-and-
 * progress-model.md). Segments are proportional to `count` but a zero-count
 * stage never disappears — it stays visible as a thin, still-clickable
 * sliver so operators can see "this bucket exists and is currently empty"
 * rather than mistaking it for a rendering gap.
 *
 * Colors are neutral-dominant: only give a stage an explicit `tone` when it
 * needs to draw the eye (e.g. a failure bucket in `error`); stages you don't
 * tone default to `neutral` so the bar doesn't read as a rainbow.
 */
const props = defineProps<{ stages: FunnelStage[] }>()

const emit = defineEmits<{ 'stage-click': [key: string] }>()

const { t } = useI18n({ useScope: 'global' })

const total = computed(() => props.stages.reduce((sum, stage) => sum + stage.count, 0))

/** Minimum visual weight for a zero-count stage so it stays a visible sliver
 * instead of collapsing — proportional to the bar's overall scale rather
 * than a fixed pixel amount, so it stays sensible whether the wave has 12
 * lines or 12,000. */
function segmentGrow(stage: FunnelStage): number {
  if (stage.count > 0) return stage.count
  const scale = total.value || props.stages.length || 1
  return Math.max(scale * 0.03, 1)
}

function toneOf(stage: FunnelStage): StatusTone {
  return stage.tone ?? 'neutral'
}

function labelOf(stage: FunnelStage): string {
  return t(stage.labelKey)
}
</script>

<template>
  <div class="funnel-bar" role="group" :aria-label="t('uiKit.funnel.groupLabel')">
    <button
      v-for="stage in stages"
      :key="stage.key"
      type="button"
      class="funnel-bar__segment"
      :class="`funnel-bar__segment--tone-${toneOf(stage)}`"
      :style="{ flexGrow: segmentGrow(stage) }"
      :title="`${labelOf(stage)} · ${stage.count}`"
      :aria-label="t('uiKit.funnel.segmentAriaLabel', { label: labelOf(stage), count: stage.count })"
      @click="emit('stage-click', stage.key)"
    >
      <span class="funnel-bar__count tabular-nums">{{ stage.count }}</span>
      <span class="funnel-bar__label">{{ labelOf(stage) }}</span>
    </button>
  </div>
</template>

<style scoped>
.funnel-bar {
  display: flex;
  align-items: stretch;
  gap: var(--space-1);
  width: 100%;
}

.funnel-bar__segment {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  min-width: 32px;
  flex-basis: 0;
  flex-shrink: 1;
  padding: var(--space-2) var(--space-1);
  border: 1px solid var(--status-neutral-border);
  background: var(--status-neutral-bg);
  color: var(--status-neutral-fg);
  border-radius: var(--radius-sm);
  cursor: pointer;
  overflow: hidden;
  font-family: var(--font-body);
  transition:
    transform var(--duration-fast) var(--ease-out),
    box-shadow var(--duration-fast) var(--ease-out),
    filter var(--duration-fast) var(--ease-out);
}

.funnel-bar__segment:hover {
  filter: brightness(0.97);
  transform: translateY(-1px);
}

.funnel-bar__segment:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.funnel-bar__segment:active {
  transform: translateY(0);
}

.funnel-bar__segment--tone-success {
  border-color: var(--status-success-border);
  background: var(--status-success-bg);
  color: var(--status-success-fg);
}
.funnel-bar__segment--tone-warning {
  border-color: var(--status-warning-border);
  background: var(--status-warning-bg);
  color: var(--status-warning-fg);
}
.funnel-bar__segment--tone-error {
  border-color: var(--status-error-border);
  background: var(--status-error-bg);
  color: var(--status-error-fg);
}
.funnel-bar__segment--tone-info {
  border-color: var(--status-info-border);
  background: var(--status-info-bg);
  color: var(--status-info-fg);
}
.funnel-bar__segment--tone-progress {
  border-color: var(--status-progress-border);
  background: var(--status-progress-bg);
  color: var(--status-progress-fg);
}

.funnel-bar__count {
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  line-height: var(--line-height-tight);
}

.funnel-bar__label {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  line-height: var(--line-height-tight);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}
</style>
