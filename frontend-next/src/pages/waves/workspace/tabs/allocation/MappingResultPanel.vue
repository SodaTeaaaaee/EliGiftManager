<script setup lang="ts">
/**
 * MappingResultPanel — renders the outcome of one `mapDemandLines` run
 * (`WaveAllocationTab.vue`'s mapping section, P4 plan §3.3). `blockedLines[].reason`
 * is a fixed 2-value enum (`wave_product_missing` / `address_unavailable` —
 * both hardcoded server-side in `use_cases.go`'s demand→fulfillment mapping),
 * so it renders through `StatusBadge` + the `demandMappingBlockedReason`
 * glossary dimension, never a raw string.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { CalloutBar } from '@/shared/ui/guidance'
import { StatusBadge } from '@/shared/ui/status'
import type { dto } from '@/../wailsjs/go/models'

const props = defineProps<{
  result: dto.DemandMappingResult | null
}>()

const { t } = useI18n({ useScope: 'global' })

const createdCount = computed(() => props.result?.createdLines.length ?? 0)
const blockedLines = computed(() => props.result?.blockedLines ?? [])
</script>

<template>
  <div v-if="result" class="mapping-result-panel">
    <CalloutBar tone="success" :message="t('allocation.mapping.createdLines', { count: createdCount })" />
    <CalloutBar
      v-if="blockedLines.length > 0"
      tone="warning"
      :message="t('allocation.mapping.blockedLines', { count: blockedLines.length })"
    />

    <ul v-if="blockedLines.length > 0" class="mapping-result-panel__blocked-list">
      <li v-for="line in blockedLines" :key="line.demandLineId" class="mapping-result-panel__blocked-item">
        <span class="mapping-result-panel__blocked-title">{{ line.demandLineTitle }}</span>
        <StatusBadge dimension="demandMappingBlockedReason" :value="line.reason" size="sm" />
      </li>
    </ul>
  </div>
</template>

<style scoped>
.mapping-result-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.mapping-result-panel__blocked-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin: 0;
  padding: 0;
  list-style: none;
}

.mapping-result-panel__blocked-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--card-border-color);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.mapping-result-panel__blocked-title {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}
</style>
