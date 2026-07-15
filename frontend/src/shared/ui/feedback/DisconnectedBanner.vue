<script setup lang="ts">
/**
 * DisconnectedBanner — global banner surfacing `useBridgeHealth()`'s
 * `unavailable` state (the Wails runtime is unreachable). Rechecks for real
 * on click by calling the bridge's own `isWailsRuntimeAvailable()` guard,
 * which synchronously re-derives and updates the shared bridge-health state.
 *
 * `stateOverride` is design-lab/testing-only: it lets a showcase page force
 * a particular visual state without an actual Wails runtime. Real call
 * sites should omit it and let the component read the live bridge state.
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { isWailsRuntimeAvailable } from '@/shared/api/bridge'
import { type BridgeState, useBridgeHealth } from '@/shared/api/health'

const props = withDefaults(
  defineProps<{
    stateOverride?: BridgeState
  }>(),
  {
    stateOverride: undefined,
  },
)

const { t } = useI18n()
const { bridgeState } = useBridgeHealth()

const checking = ref(false)

function effectiveState(): BridgeState {
  return props.stateOverride ?? bridgeState.value
}

async function recheck(): Promise<void> {
  if (checking.value || props.stateOverride !== undefined) return
  checking.value = true
  // Real Wails IPC calls resolve near-instantly; this floor keeps the
  // "checking…" label perceivable rather than flashing for a single frame.
  await new Promise((resolve) => setTimeout(resolve, 260))
  isWailsRuntimeAvailable()
  checking.value = false
}

onMounted(() => {
  if (props.stateOverride === undefined) isWailsRuntimeAvailable()
})
</script>

<template>
  <Transition name="disconnected-banner-fade">
    <div v-if="effectiveState() === 'unavailable'" class="disconnected-banner" role="alert">
      <span class="disconnected-banner__icon" aria-hidden="true">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 8.5a15 15 0 0 1 18 0" />
          <path d="M6.2 12a10 10 0 0 1 11.6 0" />
          <path d="M9.5 15.5a5 5 0 0 1 5 0" />
          <path d="M12 19h.01" />
          <path d="M3 3l18 18" />
        </svg>
      </span>

      <div class="disconnected-banner__body">
        <p class="disconnected-banner__title">{{ t('feedback.disconnectedBanner.title') }}</p>
        <p class="disconnected-banner__description">{{ t('feedback.disconnectedBanner.description') }}</p>
      </div>

      <button
        type="button"
        class="disconnected-banner__recheck"
        :disabled="checking"
        @click="recheck"
      >
        {{ checking ? t('feedback.disconnectedBanner.checking') : t('feedback.disconnectedBanner.recheck') }}
      </button>
    </div>
  </Transition>
</template>

<style scoped>
.disconnected-banner {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  border: 1px solid var(--status-warning-border);
  background: var(--status-warning-bg);
  color: var(--status-warning-fg);
  font-family: var(--font-body);
}

.disconnected-banner__icon {
  flex: none;
  display: flex;
  align-items: center;
  padding-top: 1px;
}

.disconnected-banner__body {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.disconnected-banner__title {
  margin: 0;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  line-height: var(--line-height-normal);
}

.disconnected-banner__description {
  margin: 0;
  font-size: var(--font-size-xs);
  line-height: var(--line-height-normal);
  opacity: 0.9;
}

.disconnected-banner__recheck {
  flex: none;
  border: 1px solid var(--status-warning-border);
  background: var(--color-surface-raised);
  color: var(--status-warning-fg);
  border-radius: var(--radius-sm);
  padding: var(--space-1) var(--space-3);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  font-family: var(--font-body);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out),
    opacity var(--duration-fast) var(--ease-out);
}

.disconnected-banner__recheck:hover:not(:disabled) {
  background: var(--status-warning-border);
  color: var(--color-text-inverse);
}

.disconnected-banner__recheck:disabled {
  opacity: 0.6;
  cursor: default;
}

.disconnected-banner__recheck:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.disconnected-banner-fade-enter-active,
.disconnected-banner-fade-leave-active {
  transition:
    opacity var(--duration-slow) var(--ease-out),
    transform var(--duration-slow) var(--ease-out);
}
.disconnected-banner-fade-enter-from,
.disconnected-banner-fade-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
