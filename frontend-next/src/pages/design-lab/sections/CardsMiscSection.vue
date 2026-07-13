<script setup lang="ts">
/**
 * Design-lab showcase for the cards/empty-state/funnel/guidance/drawer kit
 * (shared/ui/{cards,empty-state,funnel,guidance,drawer}/**). Composes a mini
 * fake "wave overview" out of every piece in the kit to prove they read as
 * one system at the plan's target look, then shows each component's own
 * variants/states below for isolated review.
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { SectionCard, StatCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { FunnelBar, type FunnelStage } from '@/shared/ui/funnel'
import { GuidanceCard, CalloutBar } from '@/shared/ui/guidance'
import { DetailDrawer } from '@/shared/ui/drawer'

const { t } = useI18n()

const drawerOpen = ref(false)

const funnelStages: FunnelStage[] = [
  { key: 'totalLines', labelKey: 'uiKit.cardsDemo.funnel.totalLines', count: 214 },
  { key: 'addressReady', labelKey: 'uiKit.cardsDemo.funnel.addressReady', count: 198, tone: 'info' },
  { key: 'submittedToSupplier', labelKey: 'uiKit.cardsDemo.funnel.submittedToSupplier', count: 176, tone: 'progress' },
  { key: 'shipmentSynced', labelKey: 'uiKit.cardsDemo.funnel.shipmentSynced', count: 142, tone: 'progress' },
  { key: 'syncedBack', labelKey: 'uiKit.cardsDemo.funnel.syncedBack', count: 118, tone: 'success' },
  { key: 'manualClosure', labelKey: 'uiKit.cardsDemo.funnel.manualClosure', count: 0, tone: 'error' },
]

function handleStageClick(key: string) {
  // eslint-disable-next-line no-console
  console.info('[CardsMiscSection] funnel stage clicked', key)
}

/** Isolated StatCard variants for the review grid below the composition. */
const statCardVariants = [
  { tone: 'neutral' as const, labelKey: 'uiKit.cardsDemo.variants.totalLines', value: '214', delta: undefined, hasCaption: false },
  { tone: 'success' as const, labelKey: 'uiKit.cardsDemo.variants.addressReady', value: '198', delta: '+6', hasCaption: true },
  { tone: 'warning' as const, labelKey: 'uiKit.cardsDemo.variants.addressBlocked', value: '16', delta: '-3', hasCaption: true },
  { tone: 'error' as const, labelKey: 'uiKit.cardsDemo.variants.closureFailed', value: '4', delta: '+4', hasCaption: true },
]

const calloutTones = ['success', 'warning', 'error', 'info', 'progress', 'neutral'] as const
</script>

<template>
  <section class="cards-misc-section">
    <header class="cards-misc-section__header">
      <h2 class="cards-misc-section__title">{{ t('uiKit.cardsDemo.heading') }}</h2>
      <p class="cards-misc-section__subtitle">{{ t('uiKit.cardsDemo.subheading') }}</p>
    </header>

    <!-- Composition: a fake wave-overview page built entirely from this kit family. -->
    <SectionCard :title="t('uiKit.cardsDemo.heading')" :description="t('uiKit.cardsDemo.subheading')">
      <template #actions>
        <button type="button" class="cards-misc-section__drawer-trigger" @click="drawerOpen = true">
          {{ t('uiKit.cardsDemo.drawer.openLabel') }}
        </button>
      </template>

      <div class="cards-misc-section__stats">
        <StatCard
          :label="t('uiKit.cardsDemo.stats.totalLines')"
          value="214"
          tone="neutral"
        />
        <StatCard
          :label="t('uiKit.cardsDemo.stats.addressReady')"
          value="198"
          delta="+6"
          :caption="t('uiKit.cardsDemo.stats.deltaCaption')"
          tone="success"
        />
        <StatCard
          :label="t('uiKit.cardsDemo.stats.submittedToSupplier')"
          value="176"
          delta="+11"
          :caption="t('uiKit.cardsDemo.stats.deltaCaption')"
          tone="progress"
          clickable
          @click="drawerOpen = true"
        />
        <StatCard
          :label="t('uiKit.cardsDemo.stats.closureFailed')"
          value="4"
          delta="+4"
          :caption="t('uiKit.cardsDemo.stats.deltaCaption')"
          tone="error"
        />
      </div>

      <FunnelBar :stages="funnelStages" @stage-click="handleStageClick" />

      <GuidanceCard
        :title="t('uiKit.cardsDemo.guidance.title')"
        :reason="t('uiKit.cardsDemo.guidance.reason')"
        :primary-label="t('uiKit.cardsDemo.guidance.primary')"
      >
        <template #secondary>
          <li><button type="button">{{ t('uiKit.cardsDemo.guidance.secondaryFix') }}</button></li>
          <li><button type="button">{{ t('uiKit.cardsDemo.guidance.secondaryRule') }}</button></li>
        </template>
      </GuidanceCard>

      <CalloutBar
        tone="warning"
        :message="t('uiKit.cardsDemo.callout.message')"
        :action-label="t('uiKit.cardsDemo.callout.action')"
      />

      <SectionCard flat :title="t('uiKit.cardsDemo.flatCard.title')" :description="t('uiKit.cardsDemo.flatCard.description')">
        <EmptyState
          size="sm"
          scene="wave-empty"
          :title="t('uiKit.cardsDemo.emptyState.title')"
          :description="t('uiKit.cardsDemo.emptyState.description')"
        >
          <button type="button" class="cards-misc-section__empty-action">
            {{ t('uiKit.cardsDemo.emptyState.action') }}
          </button>
        </EmptyState>
      </SectionCard>
    </SectionCard>

    <DetailDrawer v-model:show="drawerOpen" :title="t('uiKit.cardsDemo.drawer.title')" size="md">
      <div class="cards-misc-section__drawer-row">
        <span class="cards-misc-section__drawer-label">{{ t('uiKit.cardsDemo.drawer.participantLabel') }}</span>
        <span class="cards-misc-section__drawer-value">{{ t('uiKit.cardsDemo.drawer.participantValue') }}</span>
      </div>
      <div class="cards-misc-section__drawer-row">
        <span class="cards-misc-section__drawer-label">{{ t('uiKit.cardsDemo.drawer.productLabel') }}</span>
        <span class="cards-misc-section__drawer-value">{{ t('uiKit.cardsDemo.drawer.productValue') }}</span>
      </div>
      <div class="cards-misc-section__drawer-row">
        <span class="cards-misc-section__drawer-label">{{ t('uiKit.cardsDemo.drawer.quantityLabel') }}</span>
        <span class="cards-misc-section__drawer-value tabular-nums">{{ t('uiKit.cardsDemo.drawer.quantityValue') }}</span>
      </div>
      <p class="cards-misc-section__drawer-note">{{ t('uiKit.cardsDemo.drawer.note') }}</p>
      <template #footer>
        <button type="button" class="cards-misc-section__drawer-confirm" @click="drawerOpen = false">
          {{ t('uiKit.cardsDemo.drawer.confirm') }}
        </button>
      </template>
    </DetailDrawer>

    <!-- Isolated variant review: StatCard tones + clickable state. -->
    <SectionCard :title="t('uiKit.cardsDemo.reviewGrids.statCardTitle')" :description="t('uiKit.cardsDemo.reviewGrids.statCardDescription')">
      <div class="cards-misc-section__variant-grid">
        <StatCard
          v-for="variant in statCardVariants"
          :key="variant.labelKey"
          :label="t(variant.labelKey)"
          :value="variant.value"
          :delta="variant.delta"
          :caption="variant.hasCaption ? t('uiKit.cardsDemo.variants.vsLastWeek') : undefined"
          :tone="variant.tone"
        />
        <StatCard
          :label="t('uiKit.cardsDemo.variants.clickableTile')"
          value="32"
          tone="info"
          clickable
          @click="handleStageClick('review')"
        />
      </div>
    </SectionCard>

    <!-- Isolated variant review: CalloutBar across all 6 tones. -->
    <SectionCard :title="t('uiKit.cardsDemo.reviewGrids.calloutBarTitle')">
      <div class="cards-misc-section__callout-stack">
        <CalloutBar
          v-for="tone in calloutTones"
          :key="tone"
          :tone="tone"
          :message="`${t(`uiKit.cardsDemo.variants.toneNames.${tone}`)} — ${t('uiKit.cardsDemo.callout.message')}`"
          :action-label="t('uiKit.cardsDemo.callout.action')"
        />
      </div>
    </SectionCard>

    <!-- Isolated variant review: EmptyState sm/md, with and without icon slot. -->
    <SectionCard :title="t('uiKit.cardsDemo.reviewGrids.emptyStateTitle')" :description="t('uiKit.cardsDemo.reviewGrids.emptyStateDescription')">
      <div class="cards-misc-section__empty-grid">
        <EmptyState
          size="md"
          :title="t('uiKit.cardsDemo.emptyState.title')"
          :description="t('uiKit.cardsDemo.emptyState.description')"
        >
          <button type="button" class="cards-misc-section__empty-action">
            {{ t('uiKit.cardsDemo.emptyState.action') }}
          </button>
        </EmptyState>
        <EmptyState
          size="sm"
          :title="t('uiKit.cardsDemo.emptyState.title')"
          :description="t('uiKit.cardsDemo.emptyState.description')"
        />
      </div>
    </SectionCard>
  </section>
</template>

<style scoped>
.cards-misc-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.cards-misc-section__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.cards-misc-section__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.cards-misc-section__subtitle {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.cards-misc-section__stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--space-3);
}

.cards-misc-section__variant-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--space-3);
}

.cards-misc-section__callout-stack {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.cards-misc-section__empty-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: var(--space-4);
}

.cards-misc-section__empty-grid > :first-child {
  border: 1px solid var(--color-border);
  border-radius: var(--card-radius);
}

.cards-misc-section__empty-grid > :last-child {
  border: 1px dashed var(--color-border);
  border-radius: var(--card-radius);
}

.cards-misc-section__empty-action,
.cards-misc-section__drawer-confirm {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: var(--control-height-sm);
  padding: 0 var(--space-3);
  border-radius: var(--control-radius);
  border: 1px solid var(--color-accent);
  background: transparent;
  color: var(--color-accent);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}

.cards-misc-section__empty-action:hover,
.cards-misc-section__drawer-confirm:hover {
  background: var(--color-accent);
  color: var(--color-on-accent);
}

.cards-misc-section__empty-action:focus-visible,
.cards-misc-section__drawer-confirm:focus-visible,
.cards-misc-section__drawer-trigger:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.cards-misc-section__drawer-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: var(--control-height-sm);
  padding: 0 var(--space-3);
  border-radius: var(--control-radius);
  border: 1px solid var(--color-border);
  background: var(--color-surface-raised);
  color: var(--color-text-secondary);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}

.cards-misc-section__drawer-trigger:hover {
  border-color: var(--color-border-strong);
  color: var(--color-text-primary);
}

.cards-misc-section__drawer-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-3);
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--color-border);
}

.cards-misc-section__drawer-label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.cards-misc-section__drawer-value {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
  text-align: right;
}

.cards-misc-section__drawer-note {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  line-height: var(--line-height-relaxed);
  color: var(--color-text-muted);
}
</style>
