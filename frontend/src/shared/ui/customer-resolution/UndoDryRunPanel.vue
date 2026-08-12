<script setup lang="ts">
import { NButton } from 'naive-ui'
import type { UndoDryRunViewModel } from '@/shared/lib/customer-resolution'

defineProps<{
  result: UndoDryRunViewModel
  labels: {
    identities: string
    addresses: string
    nameObservations: string
    demandDocuments: string
    confirm: string
    blocked: string
  }
  blockerLabels?: Record<string, string>
  disabled?: boolean
  disabledMessage?: string
}>()

const emit = defineEmits<{
  confirm: [mergeId: number]
}>()
</script>

<template>
  <div class="undo-dry-run">
    <dl class="undo-dry-run__counts">
      <div><dt>{{ labels.identities }}</dt><dd>{{ result.restoreCounts.identities }}</dd></div>
      <div><dt>{{ labels.addresses }}</dt><dd>{{ result.restoreCounts.addresses }}</dd></div>
      <div><dt>{{ labels.nameObservations }}</dt><dd>{{ result.restoreCounts.nameObservations }}</dd></div>
      <div><dt>{{ labels.demandDocuments }}</dt><dd>{{ result.restoreCounts.demandDocuments }}</dd></div>
    </dl>

    <section v-if="result.blockers.length > 0" class="undo-dry-run__blockers">
      <strong>{{ labels.blocked }}</strong>
      <ul>
        <li v-for="(blocker, index) in result.blockers" :key="`${blocker.code}-${blocker.entityId ?? index}`">
          {{ blockerLabels?.[blocker.code] ?? blocker.code }}
        </li>
      </ul>
    </section>

    <p v-if="disabled && disabledMessage" class="undo-dry-run__disabled">{{ disabledMessage }}</p>

    <NButton
      type="warning"
      :disabled="!result.canConfirm || disabled"
      @click="emit('confirm', result.mergeId)"
    >
      {{ labels.confirm }}
    </NButton>
  </div>
</template>

<style scoped>
.undo-dry-run {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.undo-dry-run__counts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: var(--space-2);
  margin: 0;
}

.undo-dry-run__counts > div {
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.undo-dry-run__counts dt {
  color: var(--color-text-muted);
  font-size: var(--font-size-xs);
}

.undo-dry-run__counts dd {
  margin: var(--space-1) 0 0;
  color: var(--color-text-primary);
  font-weight: var(--font-weight-semibold);
}

.undo-dry-run__blockers {
  padding: var(--space-2);
  border: 1px solid var(--status-error-border);
  border-radius: var(--radius-sm);
  color: var(--status-error-fg);
  background: var(--status-error-bg);
  font-size: var(--font-size-xs);
}

.undo-dry-run__disabled {
  margin: 0;
  color: var(--status-warning-fg);
  font-size: var(--font-size-xs);
}

.undo-dry-run__blockers ul {
  margin-bottom: 0;
}
</style>
