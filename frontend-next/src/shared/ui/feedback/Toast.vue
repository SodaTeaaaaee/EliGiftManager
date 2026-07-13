<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ToastRecord } from './types'

const props = defineProps<{
  toast: ToastRecord
}>()

const emit = defineEmits<{
  dismiss: [id: string]
  'toggle-detail': [id: string]
  'pointer-enter': [id: string]
  'pointer-leave': [id: string]
}>()

const { t } = useI18n()

const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const isAlert = computed(() => props.toast.kind === 'error')

const iconPath = computed(() => {
  switch (props.toast.kind) {
    case 'success':
      return 'M8 12.5l2.5 2.5L16 9'
    case 'error':
      return 'M12 8v5M12 16h.01'
    default:
      return 'M12 8h.01M11.5 11h1v5h-1'
  }
})

async function copyDetail(): Promise<void> {
  if (!props.toast.detail) return
  try {
    await navigator.clipboard.writeText(props.toast.detail)
    copied.value = true
    if (copiedTimer) clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => {
      copied.value = false
    }, 1800)
  } catch {
    // Clipboard API unavailable (e.g. insecure context) — silently ignore,
    // the detail text is still visible/selectable on screen.
  }
}
</script>

<template>
  <div
    class="toast"
    :class="[`toast--${toast.kind}`]"
    :role="isAlert ? 'alert' : 'status'"
    :aria-live="isAlert ? 'assertive' : 'polite'"
    @pointerenter="emit('pointer-enter', toast.id)"
    @pointerleave="emit('pointer-leave', toast.id)"
  >
    <span class="toast__icon" aria-hidden="true">
      <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="9" />
        <path :d="iconPath" />
      </svg>
    </span>

    <div class="toast__body">
      <p class="toast__message">{{ toast.message }}</p>

      <button
        v-if="toast.detail"
        type="button"
        class="toast__detail-toggle"
        @click="emit('toggle-detail', toast.id)"
      >
        {{ toast.detailExpanded ? t('feedback.toast.hideDetail') : t('feedback.toast.showDetail') }}
      </button>

      <div v-if="toast.detail && toast.detailExpanded" class="toast__detail">
        <pre class="toast__detail-text">{{ toast.detail }}</pre>
        <button type="button" class="toast__copy" @click="copyDetail">
          {{ copied ? t('feedback.toast.detailCopied') : t('feedback.toast.copyDetail') }}
        </button>
      </div>
    </div>

    <button
      type="button"
      class="toast__close"
      :aria-label="t('feedback.toast.dismiss')"
      @click="emit('dismiss', toast.id)"
    >
      <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
        <path d="M6 6l12 12M18 6L6 18" />
      </svg>
    </button>
  </div>
</template>

<style scoped>
.toast {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  width: 100%;
  min-width: var(--toast-min-width);
  max-width: var(--toast-max-width);
  padding: var(--toast-padding);
  border-radius: var(--toast-radius);
  background: var(--toast-bg);
  box-shadow: var(--toast-shadow);
  border: 1px solid var(--color-border);
  font-family: var(--font-body);
  color: var(--color-text-primary);
  pointer-events: auto;
}

.toast--success .toast__icon {
  color: var(--status-success-fg);
}
.toast--error .toast__icon {
  color: var(--status-error-fg);
}
.toast--info .toast__icon {
  color: var(--status-info-fg);
}

.toast__icon {
  flex: none;
  display: flex;
  align-items: center;
  padding-top: 1px;
}

.toast__body {
  flex: 1;
  min-width: 0;
}

.toast__message {
  margin: 0;
  font-size: var(--font-size-sm);
  line-height: var(--line-height-normal);
  word-break: break-word;
}

.toast__detail-toggle {
  margin-top: var(--space-1);
  border: none;
  background: none;
  padding: 0;
  font-size: var(--font-size-xs);
  color: var(--color-accent);
  cursor: pointer;
  font-family: var(--font-body);
}

.toast__detail-toggle:hover {
  color: var(--color-accent-hover);
}

.toast__detail {
  margin-top: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--color-inset);
  border: 1px solid var(--color-border);
}

.toast__detail-text {
  margin: 0 0 var(--space-2);
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  white-space: pre-wrap;
  word-break: break-word;
}

.toast__copy {
  border: 1px solid var(--color-border-strong);
  background: var(--color-surface);
  color: var(--color-text-secondary);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  padding: 2px var(--space-2);
  cursor: pointer;
  font-family: var(--font-body);
  transition: border-color var(--duration-fast) var(--ease-out);
}

.toast__copy:hover {
  border-color: var(--color-accent);
  color: var(--color-text-primary);
}

.toast__close {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  background: none;
  color: var(--color-text-muted);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}

.toast__close:hover {
  background: var(--color-inset);
  color: var(--color-text-primary);
}

.toast__close:focus-visible,
.toast__detail-toggle:focus-visible,
.toast__copy:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}
</style>
