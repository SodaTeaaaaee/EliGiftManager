<script setup lang="ts">
/**
 * ErrorBanner — page/section-level error strip. Distinct from toasts (which
 * are transient and float above content): this is meant to sit inline at
 * the top of a page/section and stay until the underlying error is resolved
 * or the surface refreshes. Exposes a `retry` slot for the caller's own
 * retry control; falls back to a default retry button that emits `retry`.
 */
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    message: string
    detail?: string
  }>(),
  {
    detail: undefined,
  },
)

const emit = defineEmits<{
  retry: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="error-banner" role="alert">
    <span class="error-banner__icon" :aria-label="t('feedback.errorBanner.iconLabel')">
      <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <circle cx="12" cy="12" r="9" />
        <path d="M12 8v5M12 16h.01" />
      </svg>
    </span>

    <div class="error-banner__body">
      <p class="error-banner__message">{{ props.message }}</p>
      <p v-if="props.detail" class="error-banner__detail">{{ props.detail }}</p>
    </div>

    <div class="error-banner__actions">
      <slot name="retry">
        <button type="button" class="error-banner__retry" @click="emit('retry')">
          {{ t('feedback.errorBanner.retry') }}
        </button>
      </slot>
    </div>
  </div>
</template>

<style scoped>
.error-banner {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  border: 1px solid var(--status-error-border);
  background: var(--status-error-bg);
  color: var(--status-error-fg);
  font-family: var(--font-body);
}

.error-banner__icon {
  flex: none;
  display: flex;
  align-items: center;
  padding-top: 1px;
}

.error-banner__body {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.error-banner__message {
  margin: 0;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  line-height: var(--line-height-normal);
}

.error-banner__detail {
  margin: 0;
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  opacity: 0.85;
  word-break: break-word;
}

.error-banner__actions {
  flex: none;
}

.error-banner__retry {
  border: 1px solid var(--status-error-border);
  background: var(--color-surface-raised);
  color: var(--status-error-fg);
  border-radius: var(--radius-sm);
  padding: var(--space-1) var(--space-3);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  font-family: var(--font-body);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out);
}

.error-banner__retry:hover {
  background: var(--status-error-border);
  color: var(--color-text-inverse);
}

.error-banner__retry:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}
</style>
