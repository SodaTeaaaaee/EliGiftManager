<script setup lang="ts">
import { NButton } from 'naive-ui'
import type { SplitEntityType, SplitSelectionSummary, SplitValidationCode } from '@/shared/lib/customer-resolution'

defineProps<{
  summary: SplitSelectionSummary
  entityLabels: Record<SplitEntityType, string>
  validationLabels?: Partial<Record<SplitValidationCode, string>>
  previewLabel: string
}>()

const emit = defineEmits<{
  preview: []
}>()
</script>

<template>
  <div class="split-summary">
    <dl class="split-summary__counts">
      <div v-for="(count, category) in summary.counts" :key="category">
        <dt>{{ entityLabels[category] }}</dt>
        <dd>{{ count }}</dd>
      </div>
    </dl>

    <ul v-if="summary.validationCodes.length > 0" class="split-summary__errors">
      <li v-for="code in summary.validationCodes" :key="code">
        {{ validationLabels?.[code] ?? code }}
      </li>
    </ul>

    <NButton type="primary" :disabled="!summary.canPreview" @click="emit('preview')">
      {{ previewLabel }}
    </NButton>
  </div>
</template>

<style scoped>
.split-summary {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.split-summary__counts {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin: 0;
}

.split-summary__counts > div {
  min-width: 100px;
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.split-summary__counts dt {
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
}

.split-summary__counts dd {
  margin: var(--space-1) 0 0;
  color: var(--color-text-primary);
  font-weight: var(--font-weight-semibold);
}

.split-summary__errors {
  margin: 0;
  padding: var(--space-2) var(--space-2) var(--space-2) var(--space-5);
  border: 1px solid var(--status-error-border);
  border-radius: var(--radius-sm);
  color: var(--status-error-fg);
  background: var(--status-error-bg);
  font-size: var(--font-size-xs);
}
</style>
