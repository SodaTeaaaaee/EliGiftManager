<script setup lang="ts">
/**
 * A minimal tone-colored dot for dense grids/nav where a full `StatusBadge`
 * pill would be too heavy (e.g. a status column repeated on every row of a
 * `DataGrid`, or a nav item summary). Same glossary resolution as
 * `StatusBadge` — never a bare color.
 */
import { computed } from 'vue'
import { NTooltip } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useGlossary, type GlossaryDimension, type GlossaryDimensionValueMap } from '@/shared/i18n/glossary'

const props = withDefaults(
  defineProps<{
    dimension: GlossaryDimension
    value: GlossaryDimensionValueMap[GlossaryDimension] | (string & {})
    /** Render the resolved label text next to the dot. */
    showLabel?: boolean
  }>(),
  {
    showLabel: false,
  },
)

const { t } = useI18n()
const { label, desc, tone } = useGlossary()

const resolvedLabel = computed(() => label(props.dimension, props.value))
const resolvedDesc = computed(() => desc(props.dimension, props.value))
const resolvedTone = computed(() => tone(props.dimension, props.value))
const ariaLabel = computed(() =>
  t('statusKit.badge.ariaLabel', { label: resolvedLabel.value, desc: resolvedDesc.value }),
)
</script>

<template>
  <NTooltip trigger="hover" :disabled="!resolvedDesc" :delay="200">
    <template #trigger>
      <span class="status-dot-wrap" role="status" :aria-label="ariaLabel">
        <span class="status-dot" :class="`status-dot--${resolvedTone}`" aria-hidden="true" />
        <span v-if="showLabel" class="status-dot__label">{{ resolvedLabel }}</span>
      </span>
    </template>
    {{ resolvedDesc }}
  </NTooltip>
</template>

<style scoped>
.status-dot-wrap {
  display: inline-flex;
  align-items: center;
  gap: var(--statusbadge-gap);
  cursor: default;
}

.status-dot {
  width: var(--statusbadge-dot-size);
  height: var(--statusbadge-dot-size);
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.status-dot__label {
  font-family: var(--font-body);
  font-size: var(--statusbadge-font-size);
  color: var(--color-text-secondary);
  white-space: nowrap;
}

.status-dot--success {
  background: var(--status-success-fg);
  box-shadow: 0 0 0 2px var(--status-success-bg);
}
.status-dot--warning {
  background: var(--status-warning-fg);
  box-shadow: 0 0 0 2px var(--status-warning-bg);
}
.status-dot--error {
  background: var(--status-error-fg);
  box-shadow: 0 0 0 2px var(--status-error-bg);
}
.status-dot--info {
  background: var(--status-info-fg);
  box-shadow: 0 0 0 2px var(--status-info-bg);
}
.status-dot--progress {
  background: var(--status-progress-fg);
  box-shadow: 0 0 0 2px var(--status-progress-bg);
}
.status-dot--neutral {
  background: var(--status-neutral-fg);
  box-shadow: 0 0 0 2px var(--status-neutral-bg);
}
</style>
