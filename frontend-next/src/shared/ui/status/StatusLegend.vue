<script setup lang="ts">
/**
 * Renders every value of a single glossary dimension as a badge + its
 * one-sentence description — the "what do these mean" reference. Content-
 * only (no popover chrome of its own) so a caller can slot it directly into
 * an `NPopover`/`NTooltip` trigger next to a `DataGrid` status column header,
 * or render it inline (e.g. the design-lab terminology review page).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { glossaryTables, useGlossary, type GlossaryDimension } from '@/shared/i18n/glossary'
import StatusBadge from './StatusBadge.vue'

const props = withDefaults(
  defineProps<{
    dimension: GlossaryDimension
    /** Show the dimension name as a heading above the list. */
    showTitle?: boolean
  }>(),
  {
    showTitle: true,
  },
)

const { t } = useI18n()
const { desc } = useGlossary()

const dimensionTitle = computed(() => t(`statusKit.dimensionNames.${props.dimension}`))

const values = computed(() => Object.keys(glossaryTables[props.dimension]) as string[])
</script>

<template>
  <div class="status-legend">
    <p v-if="showTitle" class="status-legend__title">
      {{ t('statusKit.legend.title', { dimension: dimensionTitle }) }}
    </p>
    <p v-if="values.length === 0" class="status-legend__empty">
      {{ t('statusKit.legend.empty') }}
    </p>
    <ul v-else class="status-legend__list">
      <li v-for="v in values" :key="v" class="status-legend__row">
        <StatusBadge :dimension="dimension" :value="v" size="sm" show-dot />
        <span class="status-legend__desc">{{ desc(dimension, v) }}</span>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.status-legend {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-width: 220px;
  max-width: 360px;
}

.status-legend__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.status-legend__empty {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.status-legend__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.status-legend__row {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
}

.status-legend__desc {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  line-height: var(--line-height-normal);
}
</style>
