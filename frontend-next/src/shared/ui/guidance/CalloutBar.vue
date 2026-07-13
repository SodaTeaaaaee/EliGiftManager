<script setup lang="ts">
import type { StatusTone } from '@/shared/i18n/glossary'

/**
 * CalloutBar — the "advisory gating" strip (plan 3.3.2's consultative
 * ValidateStepAccess prompts): a tone + message + optional action link,
 * rendered as a thin banner. NOT a hard blocker by itself — the caller
 * decides whether the surrounding action is actually disabled; this
 * component only communicates the advisory.
 */
withDefaults(
  defineProps<{
    tone?: StatusTone
    message: string
    actionLabel?: string
  }>(),
  {
    tone: 'info',
    actionLabel: undefined,
  },
)

const emit = defineEmits<{ action: [] }>()
</script>

<template>
  <div class="callout-bar" :class="`callout-bar--tone-${tone}`" role="status">
    <span class="callout-bar__dot" aria-hidden="true" />
    <p class="callout-bar__message">{{ message }}</p>
    <button v-if="actionLabel" type="button" class="callout-bar__action" @click="emit('action')">
      {{ actionLabel }}
    </button>
  </div>
</template>

<style scoped>
.callout-bar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  border: 1px solid var(--status-neutral-border);
  background: var(--status-neutral-bg);
  color: var(--status-neutral-fg);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
}

.callout-bar--tone-success {
  border-color: var(--status-success-border);
  background: var(--status-success-bg);
  color: var(--status-success-fg);
}
.callout-bar--tone-warning {
  border-color: var(--status-warning-border);
  background: var(--status-warning-bg);
  color: var(--status-warning-fg);
}
.callout-bar--tone-error {
  border-color: var(--status-error-border);
  background: var(--status-error-bg);
  color: var(--status-error-fg);
}
.callout-bar--tone-info {
  border-color: var(--status-info-border);
  background: var(--status-info-bg);
  color: var(--status-info-fg);
}
.callout-bar--tone-progress {
  border-color: var(--status-progress-border);
  background: var(--status-progress-bg);
  color: var(--status-progress-fg);
}

.callout-bar__dot {
  flex-shrink: 0;
  width: var(--statusbadge-dot-size);
  height: var(--statusbadge-dot-size);
  border-radius: var(--radius-full);
  background: currentColor;
}

.callout-bar__message {
  flex: 1 1 auto;
  margin: 0;
  line-height: var(--line-height-normal);
}

.callout-bar__action {
  flex-shrink: 0;
  background: none;
  border: none;
  padding: 0;
  font: inherit;
  font-weight: var(--font-weight-semibold);
  color: currentColor;
  text-decoration: underline;
  text-underline-offset: 2px;
  cursor: pointer;
}

.callout-bar__action:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}
</style>
