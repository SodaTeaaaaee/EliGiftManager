<script setup lang="ts">
import { computed } from 'vue'
import { sortEvidence, type EvidencePolarity, type MergeEvidenceViewModel } from '@/shared/lib/customer-resolution'

const props = defineProps<{
  evidence: MergeEvidenceViewModel[]
  explanationLabels?: Record<string, string>
  polarityLabels?: Partial<Record<EvidencePolarity, string>>
  emptyLabel?: string
}>()

const sortedEvidence = computed(() => sortEvidence(props.evidence))

function explanation(item: MergeEvidenceViewModel): string {
  return props.explanationLabels?.[item.explanationCode] ?? item.explanationCode
}
</script>

<template>
  <p v-if="sortedEvidence.length === 0" class="merge-evidence__empty">{{ emptyLabel }}</p>
  <ul v-else class="merge-evidence">
    <li
      v-for="item in sortedEvidence"
      :key="item.id"
      class="merge-evidence__item"
      :class="`merge-evidence__item--${item.polarity}`"
    >
      <span class="merge-evidence__polarity">{{ polarityLabels?.[item.polarity] ?? item.polarity }}</span>
      <span class="merge-evidence__explanation">{{ explanation(item) }}</span>
      <span v-if="item.displayValue" class="merge-evidence__value">{{ item.displayValue }}</span>
      <span v-if="item.sourceLabel" class="merge-evidence__source">{{ item.sourceLabel }}</span>
    </li>
  </ul>
</template>

<style scoped>
.merge-evidence {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.merge-evidence__item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
}

.merge-evidence__item--blocker {
  color: var(--status-error-fg);
  background: var(--status-error-bg);
  border-color: var(--status-error-border);
}

.merge-evidence__item--negative {
  color: var(--status-warning-fg);
  background: var(--status-warning-bg);
  border-color: var(--status-warning-border);
}

.merge-evidence__polarity,
.merge-evidence__value {
  font-weight: var(--font-weight-semibold);
}

.merge-evidence__source {
  grid-column: 2 / -1;
  color: var(--color-text-muted);
}

.merge-evidence__empty {
  margin: 0;
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
}
</style>
