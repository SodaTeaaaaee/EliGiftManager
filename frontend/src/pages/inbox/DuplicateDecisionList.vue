<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton } from 'naive-ui'
import { StatusBadge } from '@/shared/ui/status'
import { useFeedback } from '@/shared/ui/feedback'
import { decideDuplicate } from '@/shared/api/bridge'
import type { DuplicateObservation } from '@/entities/models'

/**
 * Receipt-embedded duplicate decision list. Each undecided observation gets
 * the operator ask: keep the duplicate record (accept — no new fulfillment
 * responsibility) or claim a new responsibility (reject — the stored input is
 * replayed as a new fact). Decisions mutate the observation in place so the
 * receipt immediately reflects the chosen verdict, and emit `decided` so the
 * hosting page can refresh its row list.
 */
const props = defineProps<{
  duplicates: DuplicateObservation[]
}>()

const emit = defineEmits<{
  decided: [observationID: number, accept: boolean]
}>()

const { t } = useI18n()
const feedback = useFeedback()
const decidingId = ref<number | null>(null)

async function decide(dup: DuplicateObservation, accept: boolean) {
  if (decidingId.value !== null) return
  decidingId.value = dup.ID
  try {
    await decideDuplicate(dup.ID, accept)
    dup.Decided = true
    dup.Verdict = accept ? 'record_only' : 'new_responsibility'
    feedback.success(t('inbox.decideDuplicateSuccess'))
    emit('decided', dup.ID, accept)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    decidingId.value = null
  }
}
</script>

<template>
  <ul class="duplicate-decision-list">
    <li
      v-for="dup in props.duplicates"
      :key="dup.ID"
      class="duplicate-decision-list__row"
    >
      <StatusBadge dimension="duplicateVerdict" :value="dup.Verdict || 'record_only'" />
      <span class="duplicate-decision-list__reason">{{ dup.Reason || '—' }}</span>
      <span v-if="!dup.Decided" class="duplicate-decision-list__actions">
        <NButton
          size="tiny"
          secondary
          :loading="decidingId === dup.ID"
          :disabled="decidingId !== null && decidingId !== dup.ID"
          @click="decide(dup, true)"
        >
          {{ t('inbox.acceptDuplicate') }}
        </NButton>
        <NButton
          size="tiny"
          type="warning"
          secondary
          :disabled="decidingId !== null"
          @click="decide(dup, false)"
        >
          {{ t('inbox.rejectDuplicate') }}
        </NButton>
      </span>
      <span v-else class="duplicate-decision-list__decided">
        {{ t('inbox.duplicateDecided') }}
      </span>
    </li>
  </ul>
</template>

<style scoped>
.duplicate-decision-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: var(--font-size-sm);
}

.duplicate-decision-list__row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.duplicate-decision-list__reason {
  color: var(--color-text-secondary);
  min-width: 0;
}

.duplicate-decision-list__actions {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.duplicate-decision-list__decided {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}
</style>
