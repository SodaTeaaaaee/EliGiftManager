<script setup lang="ts">
import { NButton } from 'naive-ui'
import type { MergeOperationStatus, MergeOperationViewModel } from '@/shared/lib/customer-resolution'

defineProps<{
  operations: MergeOperationViewModel[]
  statusLabels: Record<MergeOperationStatus, string>
  inspectLabel: string
  undoDryRunLabel: string
  emptyLabel: string
}>()

const emit = defineEmits<{
  inspect: [mergeId: number]
  'undo-dry-run': [mergeId: number]
}>()
</script>

<template>
  <p v-if="operations.length === 0" class="merge-history__empty">{{ emptyLabel }}</p>
  <ol v-else class="merge-history">
    <li v-for="operation in operations" :key="operation.id" class="merge-history__item">
      <div class="merge-history__summary">
        <strong>#{{ operation.id }}</strong>
        <span>{{ operation.sourceProfileId }} → {{ operation.targetProfileId }}</span>
        <span>{{ statusLabels[operation.status] }}</span>
        <time :datetime="operation.createdAt">{{ operation.createdAt }}</time>
      </div>
      <div class="merge-history__actions">
        <NButton size="tiny" @click="emit('inspect', operation.id)">{{ inspectLabel }}</NButton>
        <NButton
          v-if="operation.canRequestUndoDryRun"
          size="tiny"
          type="warning"
          @click="emit('undo-dry-run', operation.id)"
        >
          {{ undoDryRunLabel }}
        </NButton>
      </div>
    </li>
  </ol>
</template>

<style scoped>
.merge-history {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.merge-history__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.merge-history__summary,
.merge-history__actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.merge-history__summary {
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}

.merge-history__summary strong {
  color: var(--color-text-primary);
}

.merge-history__empty {
  margin: 0;
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
}
</style>
