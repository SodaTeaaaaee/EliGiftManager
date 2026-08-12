<script setup lang="ts">
import { computed } from 'vue'
import { candidateStatusTone, type MergeCandidateStatus } from '@/shared/lib/customer-resolution'

const props = defineProps<{
  status: MergeCandidateStatus
  label: string
}>()

const tone = computed(() => candidateStatusTone(props.status))
</script>

<template>
  <span class="candidate-status" :class="`candidate-status--${tone}`" role="status">
    {{ label }}
  </span>
</template>

<style scoped>
.candidate-status {
  display: inline-flex;
  align-items: center;
  padding: 1px var(--space-2);
  border: 1px solid var(--status-neutral-border);
  border-radius: var(--radius-full);
  background: var(--status-neutral-bg);
  color: var(--status-neutral-fg);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
}

.candidate-status--info {
  color: var(--status-info-fg);
  background: var(--status-info-bg);
  border-color: var(--status-info-border);
}

.candidate-status--warning {
  color: var(--status-warning-fg);
  background: var(--status-warning-bg);
  border-color: var(--status-warning-border);
}

.candidate-status--error {
  color: var(--status-error-fg);
  background: var(--status-error-bg);
  border-color: var(--status-error-border);
}

.candidate-status--progress {
  color: var(--status-progress-fg);
  background: var(--status-progress-bg);
  border-color: var(--status-progress-border);
}

.candidate-status--success {
  color: var(--status-success-fg);
  background: var(--status-success-bg);
  border-color: var(--status-success-border);
}
</style>
