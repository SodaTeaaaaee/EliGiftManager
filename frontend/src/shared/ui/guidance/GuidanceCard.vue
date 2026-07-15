<script setup lang="ts">
/**
 * GuidanceCard — the overview's single "suggested next step" card (plan
 * 3.3.1): backend-driven `suggestedNextStep` + a human-language `reason`
 * sentence (resolved through the glossary/i18n layer by the caller — this
 * component only renders pre-resolved strings, it does not know about
 * guidance codes). Exactly one primary CTA; optional secondary links for
 * "or do X instead" escape hatches.
 */
withDefaults(
  defineProps<{
    /** Small eyebrow heading, e.g. "建议下一步" / "Suggested next step". */
    title?: string
    /** The human-readable reason sentence — never a bare code/enum. */
    reason: string
    /** Primary CTA label. Omit and use the `#primary` slot for a router-link/custom element instead. */
    primaryLabel?: string
  }>(),
  {
    title: undefined,
    primaryLabel: undefined,
  },
)

const emit = defineEmits<{ 'primary-click': [] }>()
</script>

<template>
  <div class="guidance-card">
    <div class="guidance-card__body">
      <p v-if="title" class="guidance-card__eyebrow">{{ title }}</p>
      <p class="guidance-card__reason">{{ reason }}</p>
      <ul v-if="$slots.secondary" class="guidance-card__secondary">
        <slot name="secondary" />
      </ul>
    </div>
    <div class="guidance-card__action">
      <slot name="primary">
        <button v-if="primaryLabel" type="button" class="guidance-card__cta" @click="emit('primary-click')">
          {{ primaryLabel }}
        </button>
      </slot>
    </div>
  </div>
</template>

<style scoped>
.guidance-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-4);
  background: var(--color-accent-subtle);
  border: 1px solid var(--color-accent);
  border-radius: var(--card-radius);
  padding: var(--card-padding);
}

.guidance-card__body {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
  flex: 1 1 260px;
}

.guidance-card__eyebrow {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-accent);
}

.guidance-card__reason {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-base);
  line-height: var(--line-height-normal);
  color: var(--color-text-primary);
}

.guidance-card__secondary {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1) var(--space-4);
  margin: var(--space-1) 0 0;
  padding: 0;
  list-style: none;
  font-size: var(--font-size-sm);
}

.guidance-card__secondary :deep(a),
.guidance-card__secondary :deep(button) {
  color: var(--color-text-secondary);
  text-decoration: underline;
  text-underline-offset: 2px;
  background: none;
  border: none;
  padding: 0;
  font: inherit;
  cursor: pointer;
}

.guidance-card__secondary :deep(a:hover),
.guidance-card__secondary :deep(button:hover) {
  color: var(--color-accent);
}

.guidance-card__action {
  flex-shrink: 0;
}

.guidance-card__cta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: var(--control-height);
  padding: 0 var(--space-4);
  border-radius: var(--control-radius);
  border: none;
  background: var(--color-accent);
  color: var(--color-on-accent);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    transform var(--duration-fast) var(--ease-out);
}

.guidance-card__cta:hover {
  background: var(--color-accent-hover);
}

.guidance-card__cta:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.guidance-card__cta:active {
  transform: translateY(1px);
}
</style>
