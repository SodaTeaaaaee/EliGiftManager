<script setup lang="ts">
/**
 * Design-lab showcase for the status-rendering kit (shared/ui/status/**).
 * Doubles as the terminology review page: every glossary dimension x value
 * is rendered here in both badge sizes, plus StatusDot and StatusLegend.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { glossaryTables, type GlossaryDimension } from '@/shared/i18n/glossary'
import StatusBadge from '@/shared/ui/status/StatusBadge.vue'
import StatusDot from '@/shared/ui/status/StatusDot.vue'
import StatusLegend from '@/shared/ui/status/StatusLegend.vue'

const { t } = useI18n()

const sampleSubjects: Record<GlossaryDimension, string> = {
  identityType: 'UID12345678 · Bilibili UID',
  platformKind: 'Bilibili / Rozao Factory',
  inputFactKind: 'Membership Cycle Gift',
  templateDirection: 'Input Parse / Output Export',
  waveCloseResult: 'W-2024-01 · Close Result',
  entitlementSelectorType: 'Rule #1 · Platform Level',
  fulfillmentSourceKind: 'Entitlement Instance #42',
  blockReason: 'Unaligned Product / Missing Address',
  supplierOrderStatus: 'SO-2024-001 · Supplier Order',
  duplicateVerdict: 'Duplicate Observation #10',
  writebackStatus: 'Writeback #99 · Bilibili',
  workState: 'Fulfillment Result #128',
}

const dimensions = computed(() =>
  (Object.keys(glossaryTables) as GlossaryDimension[]).map((dimension) => ({
    dimension,
    title: t(`glossary.${dimension}.${Object.keys(glossaryTables[dimension])[0]}.label`) || dimension,
    subject: sampleSubjects[dimension] || dimension,
    values: Object.keys(glossaryTables[dimension]),
  })),
)
</script>

<template>
  <div class="status-section">
    <div v-for="item in dimensions" :key="item.dimension" class="status-section__dimension">
      <div class="status-section__header">
        <h4 class="status-section__title">{{ item.dimension }}</h4>
        <span class="status-section__subject">{{ item.subject }}</span>
      </div>

      <div class="status-section__values">
        <div v-for="val in item.values" :key="val" class="status-section__value-row">
          <StatusBadge :dimension="item.dimension" :value="val" size="md" />
          <StatusBadge :dimension="item.dimension" :value="val" size="sm" />
          <StatusDot :dimension="item.dimension" :value="val" />
        </div>
      </div>

      <div class="status-section__legend">
        <StatusLegend :dimension="item.dimension" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.status-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.status-section__dimension {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.status-section__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-section__title {
  margin: 0;
  font-weight: var(--font-weight-semibold);
}

.status-section__subject {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.status-section__values {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
}

.status-section__value-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.status-section__legend {
  margin-top: var(--space-2);
}
</style>
