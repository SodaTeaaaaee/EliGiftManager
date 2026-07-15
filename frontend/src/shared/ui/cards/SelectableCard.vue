<script setup lang="ts">
/**
 * SelectableCard — token-styled selectable option card for wizard grids
 * (platform presets, business-surface choices, compact option tiles).
 * Presentational only: parent owns selection via `selected` + `@select`.
 */
const props = withDefaults(
  defineProps<{
    /** Whether this card is the active selection. */
    selected?: boolean
    /** Disables interaction and dims the card. */
    disabled?: boolean
    /** Primary label (already translated). */
    label: string
    /** Optional secondary description (already translated). */
    description?: string
  }>(),
  {
    selected: false,
    disabled: false,
    description: undefined,
  },
)

const emit = defineEmits<{
  select: []
}>()

function handleClick(): void {
  if (props.disabled) return
  emit('select')
}
</script>

<template>
  <button
    type="button"
    class="selectable-card"
    :class="{
      'selectable-card--selected': selected,
      'selectable-card--disabled': disabled,
    }"
    :disabled="disabled"
    :aria-pressed="selected"
    @click="handleClick"
  >
    <span class="selectable-card__label">{{ label }}</span>
    <span v-if="description" class="selectable-card__description">{{ description }}</span>
    <slot />
  </button>
</template>

<style scoped>
.selectable-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  text-align: left;
  padding: var(--space-3);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  background: var(--color-surface);
  cursor: pointer;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}

.selectable-card:hover:not(:disabled) {
  border-color: var(--color-accent);
}

.selectable-card:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.selectable-card--selected {
  border-color: var(--color-accent);
  background: var(--color-accent-subtle);
}

.selectable-card--disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.selectable-card__label {
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.selectable-card__description {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  line-height: var(--line-height-relaxed);
  color: var(--color-text-muted);
}
</style>
