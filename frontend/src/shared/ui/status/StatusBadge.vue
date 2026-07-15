<script setup lang="ts">
/**
 * The ONLY legal way to render a domain status pill. Resolves label / desc /
 * tone through `useGlossary()` — never accepts a pre-resolved color or copy.
 * Unknown `(dimension, value)` pairs fall back to `neutral` + the raw value
 * string (via `useGlossary()`), so this component never throws.
 */
import { computed } from 'vue'
import { NTooltip } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useGlossary, type GlossaryDimension, type GlossaryDimensionValueMap } from '@/shared/i18n/glossary'

type Size = 'sm' | 'md'

const props = withDefaults(
  defineProps<{
    /** The glossary dimension this value belongs to, e.g. `'lifecycleStage'`. */
    dimension: GlossaryDimension
    /** The raw enum value. Unknown values render as neutral + raw string. */
    value: GlossaryDimensionValueMap[GlossaryDimension] | (string & {})
    size?: Size
    /** Prefix the label with a small tone-colored dot. */
    showDot?: boolean
  }>(),
  {
    size: 'md',
    showDot: false,
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
      <span
        class="status-badge"
        :class="[`status-badge--${size}`, `status-badge--${resolvedTone}`]"
        role="status"
        :aria-label="ariaLabel"
      >
        <span v-if="showDot" class="status-badge__dot" aria-hidden="true" />
        <span class="status-badge__label">{{ resolvedLabel }}</span>
      </span>
    </template>
    {{ resolvedDesc }}
  </NTooltip>
</template>

<style scoped>
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--statusbadge-gap);
  padding: var(--statusbadge-padding-y) var(--statusbadge-padding-x);
  border-radius: var(--statusbadge-radius);
  font-family: var(--font-body);
  font-size: var(--statusbadge-font-size);
  font-weight: var(--statusbadge-font-weight);
  line-height: var(--line-height-tight);
  white-space: nowrap;
  border: 1px solid transparent;
  cursor: default;
  transition:
    background-color var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out);
}

.status-badge--sm {
  font-size: 0.6875rem;
  padding-block: 1px;
}

.status-badge__dot {
  width: var(--statusbadge-dot-size);
  height: var(--statusbadge-dot-size);
  border-radius: var(--radius-full);
  background: currentColor;
  flex-shrink: 0;
}

.status-badge__label {
  overflow: hidden;
  text-overflow: ellipsis;
}

.status-badge--success {
  color: var(--status-success-fg);
  background: var(--status-success-bg);
  border-color: var(--status-success-border);
}
.status-badge--warning {
  color: var(--status-warning-fg);
  background: var(--status-warning-bg);
  border-color: var(--status-warning-border);
}
.status-badge--error {
  color: var(--status-error-fg);
  background: var(--status-error-bg);
  border-color: var(--status-error-border);
}
.status-badge--info {
  color: var(--status-info-fg);
  background: var(--status-info-bg);
  border-color: var(--status-info-border);
}
.status-badge--progress {
  color: var(--status-progress-fg);
  background: var(--status-progress-bg);
  border-color: var(--status-progress-border);
}
.status-badge--neutral {
  color: var(--status-neutral-fg);
  background: var(--status-neutral-bg);
  border-color: var(--status-neutral-border);
}
</style>
