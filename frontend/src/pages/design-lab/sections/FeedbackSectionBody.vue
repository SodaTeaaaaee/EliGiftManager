<script setup lang="ts">
/**
 * Actual demo content for the feedback kit showcase. Split out from
 * `FeedbackSection.vue` because `useFeedback()` must be called from a
 * component instance mounted *under* `<FeedbackProvider>` — provide/inject
 * resolves against the real component tree (the receiving component is the
 * injected content's parent instance), not the lexical template location,
 * so this body has to be a distinct child component of the provider rather
 * than living in the same file as the `<FeedbackProvider>` tag itself.
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import DisconnectedBanner from '@/shared/ui/feedback/DisconnectedBanner.vue'
import ErrorBanner from '@/shared/ui/feedback/ErrorBanner.vue'
import { useFeedback } from '@/shared/ui/feedback/context'
import type { BridgeState } from '@/shared/api/health'

const { t } = useI18n()
const feedback = useFeedback()

const retryCount = ref(0)
const simulatedState = ref<BridgeState>('unavailable')

function fireSuccess(): void {
  feedback.success(t('feedback.demo.successMessage'))
}

function fireInfo(): void {
  feedback.info(t('feedback.demo.infoMessage'))
}

function fireError(): void {
  feedback.error(t('feedback.demo.errorMessage'), t('feedback.demo.errorDetail'))
}

function logUndo(): void {
  feedback.receipt({ kind: 'undo', summary: t('feedback.demo.sampleUndo') })
}

function logRedo(): void {
  feedback.receipt({ kind: 'redo', summary: t('feedback.demo.sampleRedo') })
}

function logAction(): void {
  feedback.receipt({ kind: 'action', summary: t('feedback.demo.sampleAction') })
}

function toggleSimulatedDisconnect(): void {
  simulatedState.value = simulatedState.value === 'unavailable' ? 'available' : 'unavailable'
}

function onBannerRetry(): void {
  retryCount.value += 1
}
</script>

<template>
  <section class="feedback-section">
    <header class="feedback-section__header">
      <h2 class="feedback-section__title">{{ t('feedback.demo.title') }}</h2>
      <p class="feedback-section__subtitle">{{ t('feedback.demo.subtitle') }}</p>
    </header>

    <article class="feedback-card">
      <h3 class="feedback-card__title">{{ t('feedback.demo.toastGroupTitle') }}</h3>
      <div class="feedback-card__row">
        <button type="button" class="demo-button demo-button--success" @click="fireSuccess">
          {{ t('feedback.demo.triggerSuccess') }}
        </button>
        <button type="button" class="demo-button demo-button--info" @click="fireInfo">
          {{ t('feedback.demo.triggerInfo') }}
        </button>
        <button type="button" class="demo-button demo-button--error" @click="fireError">
          {{ t('feedback.demo.triggerError') }}
        </button>
      </div>
    </article>

    <article class="feedback-card">
      <h3 class="feedback-card__title">{{ t('feedback.demo.receiptGroupTitle') }}</h3>
      <div class="feedback-card__row">
        <button type="button" class="demo-button" @click="logUndo">{{ t('feedback.demo.logUndo') }}</button>
        <button type="button" class="demo-button" @click="logRedo">{{ t('feedback.demo.logRedo') }}</button>
        <button type="button" class="demo-button" @click="logAction">{{ t('feedback.demo.logAction') }}</button>
      </div>
      <p class="feedback-card__hint">{{ t('feedback.receiptTray.buttonLabel') }} — {{ t('common.more') }} ↘</p>
    </article>

    <article class="feedback-card">
      <h3 class="feedback-card__title">{{ t('feedback.demo.bannerGroupTitle') }}</h3>
      <ErrorBanner
        :message="t('feedback.demo.bannerMessage')"
        :detail="t('feedback.demo.bannerDetail')"
        @retry="onBannerRetry"
      />
      <p class="feedback-card__hint">{{ t('common.retry') }}: {{ retryCount }}</p>
    </article>

    <article class="feedback-card">
      <h3 class="feedback-card__title">{{ t('feedback.demo.disconnectedGroupTitle') }}</h3>
      <div class="feedback-card__row">
        <button type="button" class="demo-button" @click="toggleSimulatedDisconnect">
          {{ simulatedState === 'unavailable' ? t('feedback.demo.toggleConnected') : t('feedback.demo.toggleDisconnected') }}
        </button>
      </div>
      <DisconnectedBanner :state-override="simulatedState" />
    </article>

    <article class="feedback-card">
      <h3 class="feedback-card__title">{{ t('feedback.demo.liveDisconnectedGroupTitle') }}</h3>
      <p class="feedback-card__hint">{{ t('feedback.demo.liveDisconnectedHint') }}</p>
      <DisconnectedBanner />
    </article>
  </section>
</template>

<style scoped>
.feedback-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.feedback-section__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.feedback-section__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.feedback-section__subtitle {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.feedback-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  background: var(--card-bg);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  padding: var(--card-padding);
  box-shadow: var(--card-shadow);
}

.feedback-card__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.feedback-card__row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.feedback-card__hint {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.demo-button {
  height: var(--control-height);
  padding: 0 var(--space-4);
  border-radius: var(--control-radius);
  border: 1px solid var(--control-border-color);
  background: var(--control-bg);
  color: var(--color-text-primary);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}

.demo-button:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.demo-button:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.demo-button--success {
  border-color: var(--status-success-border);
  color: var(--status-success-fg);
}
.demo-button--success:hover {
  background: var(--status-success-bg);
  border-color: var(--status-success-fg);
  color: var(--status-success-fg);
}

.demo-button--info {
  border-color: var(--status-info-border);
  color: var(--status-info-fg);
}
.demo-button--info:hover {
  background: var(--status-info-bg);
  border-color: var(--status-info-fg);
  color: var(--status-info-fg);
}

.demo-button--error {
  border-color: var(--status-error-border);
  color: var(--status-error-fg);
}
.demo-button--error:hover {
  background: var(--status-error-bg);
  border-color: var(--status-error-fg);
  color: var(--status-error-fg);
}
</style>
