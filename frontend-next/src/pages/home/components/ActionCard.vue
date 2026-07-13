<script setup lang="ts">
/**
 * ActionCard — a single clickable action-stream tile (plan 3.1): a thin
 * presentational wrapper over `StatCard`. Purely presentational — it does
 * not know about routing or the deep-link filter mapping; the parent
 * (`HomePage.vue`) owns the click -> navigation decision.
 */
import { computed } from 'vue'
import { StatCard } from '@/shared/ui/cards'
import type { StatusTone } from '@/shared/i18n/glossary'

const props = withDefaults(
  defineProps<{
    /** Wave display name — shown as the caption for bucket cards, omitted for the inbox card. */
    waveName?: string
    /** Already-translated bucket label, e.g. t('taskCenter.buckets.missingAddress'). */
    bucketLabel: string
    count: number
    tone?: StatusTone
  }>(),
  {
    waveName: undefined,
    tone: 'neutral',
  },
)

const emit = defineEmits<{ click: [MouseEvent] }>()

const formattedCount = computed(() => props.count.toLocaleString())

function handleClick(event: MouseEvent): void {
  emit('click', event)
}
</script>

<template>
  <StatCard
    :label="bucketLabel"
    :value="formattedCount"
    :caption="waveName"
    :tone="tone"
    clickable
    @click="handleClick"
  />
</template>
