<script setup lang="ts">
/**
 * ContentErrorBoundary — wraps the CONTENT ZONE only (never the whole app;
 * plan 2.1: "错误边界只替换内容区，导航永远可用" fixing the old tree's
 * full-shell replacement). Catches render/setup errors from its default
 * slot via `onErrorCaptured`, shows a token-styled error state with a retry
 * action (which force-remounts the slot via a bumped `:key`) and a
 * copy-to-clipboard button for the raw error details — so a bug report can
 * carry the real stack instead of a screenshot of a friendly sentence.
 *
 * Deliberately does not depend on `useFeedback()` / any global provider: an
 * error boundary must keep working even if something upstream (including a
 * misbehaving provider) is what threw, so all of its own state is local.
 */
import { nextTick, onErrorCaptured, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { WarningOutline, RefreshOutline, CopyOutline } from '@vicons/ionicons5'

const { t } = useI18n({ useScope: 'global' })
const route = useRoute()

/**
 * `retry` fires right before the slot content is force-remounted, so a
 * caller can reset whatever external state actually caused the crash (e.g.
 * refetch the data that came back malformed) before the fresh mount runs.
 */
const emit = defineEmits<{ retry: [] }>()

const hasError = ref(false)
const errorMessage = ref('')
const errorDetail = ref('')
const remountKey = ref(0)
const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | undefined

onErrorCaptured((err, _instance, info) => {
  hasError.value = true
  errorMessage.value = err instanceof Error ? err.message : String(err)
  errorDetail.value = [err instanceof Error ? err.stack : undefined, info ? `Vue lifecycle hook: ${info}` : undefined]
    .filter((line): line is string => Boolean(line))
    .join('\n')
  return false // stop propagation — this boundary owns the failure now
})

/**
 * Clear the latched error on navigation so one content-zone crash cannot
 * permanently replace every subsequent content page until the user hits
 * Retry. Remount via remountKey so the new route's slot mounts cleanly.
 * SideNav lives outside this boundary and is unaffected.
 */
watch(
  () => route.fullPath,
  () => {
    if (!hasError.value) return
    hasError.value = false
    errorMessage.value = ''
    errorDetail.value = ''
    remountKey.value += 1
  },
)

async function retry(): Promise<void> {
  emit('retry')
  hasError.value = false
  errorMessage.value = ''
  errorDetail.value = ''
  await nextTick()
  remountKey.value += 1
}

async function copyDetails(): Promise<void> {
  const detail = [errorMessage.value, errorDetail.value].filter(Boolean).join('\n\n')
  try {
    await navigator.clipboard.writeText(detail)
    copied.value = true
    if (copiedTimer) clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    // Clipboard API unavailable/denied (e.g. non-secure context) — the
    // detail text is still visible on screen in the <details> block below
    // for manual selection/copy.
  }
}
</script>

<template>
  <div class="content-error-boundary">
    <div v-if="hasError" class="content-error-boundary__panel" role="alert">
      <span class="content-error-boundary__icon" aria-hidden="true">
        <WarningOutline />
      </span>
      <h2 class="content-error-boundary__title">{{ t('shellKit.errorBoundary.title') }}</h2>
      <p class="content-error-boundary__description">{{ t('shellKit.errorBoundary.description') }}</p>
      <p v-if="errorMessage" class="content-error-boundary__message tabular-nums">{{ errorMessage }}</p>

      <div class="content-error-boundary__actions">
        <button type="button" class="content-error-boundary__btn content-error-boundary__btn--primary" @click="retry">
          <RefreshOutline class="content-error-boundary__btn-icon" />
          {{ t('shellKit.errorBoundary.retry') }}
        </button>
        <button type="button" class="content-error-boundary__btn" @click="copyDetails">
          <CopyOutline class="content-error-boundary__btn-icon" />
          {{ copied ? t('shellKit.errorBoundary.copied') : t('shellKit.errorBoundary.copyDetails') }}
        </button>
      </div>

      <details v-if="errorDetail" class="content-error-boundary__stack">
        <summary>{{ t('shellKit.errorBoundary.stackToggle') }}</summary>
        <pre class="tabular-nums">{{ errorDetail }}</pre>
      </details>
    </div>

    <div v-else :key="remountKey" class="content-error-boundary__content">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.content-error-boundary {
  height: 100%;
  min-width: 0;
}

.content-error-boundary__content {
  height: 100%;
}

.content-error-boundary__panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: var(--space-2);
  max-width: 32rem;
  margin: var(--space-16) auto;
  padding: var(--space-8) var(--space-6);
}

.content-error-boundary__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  margin-bottom: var(--space-2);
  border-radius: var(--radius-full);
  background: var(--status-error-bg);
  color: var(--status-error-fg);
}

.content-error-boundary__icon :deep(svg) {
  width: 24px;
  height: 24px;
}

.content-error-boundary__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.content-error-boundary__description {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  line-height: var(--line-height-relaxed);
  color: var(--color-text-secondary);
}

.content-error-boundary__message {
  margin: 0;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--status-error-bg);
  color: var(--status-error-fg);
  font-family: var(--font-mono, monospace);
  font-size: var(--font-size-xs);
  word-break: break-word;
}

.content-error-boundary__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-top: var(--space-2);
}

.content-error-boundary__btn {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  height: var(--control-height);
  padding: 0 var(--space-4);
  border-radius: var(--control-radius);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-text-primary);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out);
}

.content-error-boundary__btn:hover {
  background: var(--color-inset);
}

.content-error-boundary__btn:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.content-error-boundary__btn--primary {
  border-color: var(--color-accent);
  background: var(--color-accent);
  color: var(--color-on-accent);
}

.content-error-boundary__btn--primary:hover {
  background: var(--color-accent-hover);
}

.content-error-boundary__btn-icon {
  width: 16px;
  height: 16px;
}

.content-error-boundary__stack {
  width: 100%;
  margin-top: var(--space-3);
  text-align: left;
}

.content-error-boundary__stack summary {
  cursor: pointer;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
}

.content-error-boundary__stack summary:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.content-error-boundary__stack pre {
  margin-top: var(--space-2);
  padding: var(--space-3);
  border-radius: var(--radius-sm);
  background: var(--color-inset);
  color: var(--color-text-secondary);
  font-family: var(--font-mono, monospace);
  font-size: var(--font-size-xs);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 240px;
  overflow-y: auto;
}
</style>
