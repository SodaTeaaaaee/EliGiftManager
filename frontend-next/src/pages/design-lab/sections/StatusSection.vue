<script setup lang="ts">
/**
 * Design-lab showcase for the status-rendering kit (shared/ui/status/**).
 * Doubles as the terminology review page: every glossary dimension x value
 * is rendered here in both badge sizes, plus StatusDot and StatusLegend, so
 * a product owner can review labels/descriptions/tones in one place.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { glossaryTables, type GlossaryDimension } from '@/shared/i18n/glossary'
import StatusBadge from '@/shared/ui/status/StatusBadge.vue'
import StatusDot from '@/shared/ui/status/StatusDot.vue'
import StatusLegend from '@/shared/ui/status/StatusLegend.vue'

const { t } = useI18n()

/** Realistic fulfillment-domain sample subject per dimension — CJK + Latin mixed. */
const sampleSubjects: Record<GlossaryDimension, string> = {
  lifecycleStage: '2026-07 会员波 · July Membership Wave',
  routingDisposition: 'Order #RT-20458 · 類想 Ayaka',
  recipientInputState: '宮子 Miyako · 収货地址补充',
  addressState: '东京都渋谷区 1-2-3 · Shibuya Studio',
  supplierState: 'SO-2026-0710 · 印刷工坊 Printcraft',
  channelSyncState: 'Shopify JP Store · 渠道回填',
  allocationState: 'Line #42 · Elissia 限定徽章',
  shipmentStatus: 'SF1234567890 · 顺丰速运',
  demandKind: '会员周期礼 · Membership Cycle Gift',
  adjustmentKind: 'Line #88 · 追加徽章 Bonus Badge',
  productKind: 'Elissia 亚克力立牌 · Acrylic Standee',
  driftSummary: '2026-07 会员波 vs 上游事实 Upstream Facts',
  reviewRequirement: 'Line #12 · 复查要求 Review Requirement',
  lineReason: 'Line #12 · 来源 Line Reason',
  basisDriftStatus: 'Line #12 · 依据漂移状态 Basis Drift Status',
  allocationSelectorType: 'Rule #7 · 会员波匹配范围 Selector Scope',
  demandMappingBlockedReason: 'Demand Line #56 · 映射阻塞原因 Mapping Blocked Reason',
}

const dimensions = computed(() =>
  (Object.keys(glossaryTables) as GlossaryDimension[]).map((dimension) => ({
    dimension,
    title: t(`statusKit.dimensionNames.${dimension}`),
    subject: sampleSubjects[dimension],
    values: Object.keys(glossaryTables[dimension]),
  })),
)
</script>

<template>
  <section class="status-section">
    <header class="status-section__header">
      <h2 class="status-section__title">{{ t('statusKit.demo.title') }}</h2>
      <p class="status-section__subtitle">{{ t('statusKit.demo.subtitle') }}</p>
    </header>

    <div class="status-section__grid">
      <article v-for="group in dimensions" :key="group.dimension" class="status-card">
        <header class="status-card__header">
          <h3 class="status-card__title">{{ group.title }}</h3>
          <p class="status-card__subject">{{ group.subject }}</p>
        </header>

        <table class="status-card__table">
          <thead>
            <tr>
              <th>{{ t('statusKit.demo.sampleColumn') }}</th>
              <th>{{ t('statusKit.demo.sizeSm') }}</th>
              <th>{{ t('statusKit.demo.sizeMd') }}</th>
              <th>{{ t('statusKit.demo.dotPlain') }}</th>
              <th>{{ t('statusKit.demo.dotLabeled') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="value in group.values" :key="value">
              <td class="status-card__value-key tabular-nums">{{ value }}</td>
              <td><StatusBadge :dimension="group.dimension" :value="value" size="sm" show-dot /></td>
              <td><StatusBadge :dimension="group.dimension" :value="value" size="md" /></td>
              <td><StatusDot :dimension="group.dimension" :value="value" /></td>
              <td><StatusDot :dimension="group.dimension" :value="value" show-label /></td>
            </tr>
          </tbody>
        </table>

        <footer class="status-card__legend">
          <p class="status-card__legend-title">{{ t('statusKit.demo.legendSectionTitle') }}</p>
          <StatusLegend :dimension="group.dimension" :show-title="false" />
        </footer>
      </article>
    </div>
  </section>
</template>

<style scoped>
.status-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.status-section__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.status-section__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.status-section__subtitle {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.status-section__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(420px, 1fr));
  gap: var(--space-5);
}

.status-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  background: var(--card-bg);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  padding: var(--card-padding);
  box-shadow: var(--card-shadow);
}

.status-card__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.status-card__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.status-card__subject {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.status-card__table {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
}

.status-card__table th {
  text-align: left;
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
  padding: var(--space-1) var(--space-2);
  border-bottom: 1px solid var(--color-border);
}

.status-card__table td {
  padding: var(--space-2);
  border-bottom: 1px solid var(--color-border);
  vertical-align: middle;
}

.status-card__value-key {
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
  white-space: nowrap;
}

.status-card__legend {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding-top: var(--space-2);
  border-top: 1px dashed var(--color-border);
}

.status-card__legend-title {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}
</style>
