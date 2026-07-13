<script setup lang="ts">
import { computed } from 'vue'
import type { StatusTone } from '@/shared/i18n/glossary'

/**
 * StatCard — label + big tabular-nums value + optional delta, the base
 * tile for the task-center action-card stream (plan 3.1) and the overview
 * six-bucket grouping (plan 3.3.1). When `clickable` (or a click listener is
 * bound) it renders as a real `<button>` for keyboard/AT access, otherwise
 * as a plain `<div>` — never a clickable-looking non-interactive element.
 */
const props = withDefaults(
  defineProps<{
    label: string
    /** Pre-formatted value string — callers own number formatting/locale. */
    value: string
    /** Small caption under the value, e.g. a unit or breakdown hint. */
    caption?: string
    /** Signed delta text, e.g. "+12" / "-3". Rendered with a tone-colored indicator. */
    delta?: string
    /** Tone driving the delta indicator + optional left accent bar. Defaults to 'neutral'. */
    tone?: StatusTone
    /** Force clickable affordance/semantics even without a listener bound (rare). */
    clickable?: boolean
  }>(),
  {
    caption: undefined,
    delta: undefined,
    tone: 'neutral',
    clickable: false,
  },
)

const emit = defineEmits<{ click: [MouseEvent] }>()

const isInteractive = computed(() => props.clickable)

function handleClick(event: MouseEvent) {
  emit('click', event)
}
</script>

<template>
  <component
    :is="isInteractive ? 'button' : 'div'"
    class="stat-card"
    :class="[`stat-card--tone-${tone}`, { 'stat-card--interactive': isInteractive }]"
    :type="isInteractive ? 'button' : undefined"
    @click="isInteractive && handleClick($event)"
  >
    <span class="stat-card__label">{{ label }}</span>
    <span class="stat-card__value tabular-nums">{{ value }}</span>
    <span v-if="delta || caption || $slots.footer" class="stat-card__meta">
      <span v-if="delta" class="stat-card__delta tabular-nums">{{ delta }}</span>
      <span v-if="caption" class="stat-card__caption">{{ caption }}</span>
      <slot name="footer" />
    </span>
  </component>
</template>

<style scoped>
.stat-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  background: var(--card-bg);
  border: 1px solid var(--card-border-color);
  border-left: 3px solid var(--status-neutral-border);
  border-radius: var(--card-radius);
  box-shadow: var(--card-shadow);
  padding: var(--card-padding);
  text-align: left;
  font-family: var(--font-body);
  color: inherit;
  min-width: 0;
}

.stat-card--tone-success {
  border-left-color: var(--status-success-border);
}
.stat-card--tone-warning {
  border-left-color: var(--status-warning-border);
}
.stat-card--tone-error {
  border-left-color: var(--status-error-border);
}
.stat-card--tone-info {
  border-left-color: var(--status-info-border);
}
.stat-card--tone-progress {
  border-left-color: var(--status-progress-border);
}

.stat-card--interactive {
  cursor: pointer;
  transition:
    transform var(--duration-fast) var(--ease-out),
    box-shadow var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out);
}

.stat-card--interactive:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
  border-color: var(--color-border-strong);
}

.stat-card--interactive:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.stat-card--interactive:active {
  transform: translateY(0);
}

.stat-card__label {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}

.stat-card__value {
  font-family: var(--font-display);
  font-size: var(--font-size-3xl);
  font-weight: var(--font-weight-semibold);
  line-height: var(--line-height-tight);
  color: var(--color-text-primary);
}

.stat-card__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.stat-card__delta {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}

.stat-card__caption {
  color: var(--color-text-muted);
}
</style>
