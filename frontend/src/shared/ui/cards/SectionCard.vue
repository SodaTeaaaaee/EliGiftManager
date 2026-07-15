<script setup lang="ts">
/**
 * SectionCard — the general-purpose page-skeleton card: title/description
 * header zone, optional actions zone (top-right), and a default content
 * zone. Used to wrap any page section (a form, a table, a chart) with
 * consistent card chrome. Consumes only Layer 3 `--card-*` tokens.
 */
withDefaults(
  defineProps<{
    /** Optional — omit to use the `#title` slot for rich content (e.g. an icon + text). */
    title?: string
    description?: string
    /** Removes the card border/shadow for a flatter, nested composition. */
    flat?: boolean
  }>(),
  {
    title: undefined,
    description: undefined,
    flat: false,
  },
)
</script>

<template>
  <section class="section-card" :class="{ 'section-card--flat': flat }">
    <header v-if="title || description || $slots.title || $slots.actions || $slots.description" class="section-card__header">
      <div class="section-card__heading">
        <h2 v-if="title || $slots.title" class="section-card__title">
          <slot name="title">{{ title }}</slot>
        </h2>
        <p v-if="description || $slots.description" class="section-card__description">
          <slot name="description">{{ description }}</slot>
        </p>
      </div>
      <div v-if="$slots.actions" class="section-card__actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="section-card__content">
      <slot />
    </div>
    <footer v-if="$slots.footer" class="section-card__footer">
      <slot name="footer" />
    </footer>
  </section>
</template>

<style scoped>
.section-card {
  display: flex;
  flex-direction: column;
  gap: var(--card-gap);
  background: var(--card-bg);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  box-shadow: var(--card-shadow);
  padding: var(--card-padding);
}

.section-card--flat {
  border-color: transparent;
  box-shadow: none;
  background: transparent;
  padding: 0;
}

.section-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
}

.section-card__heading {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}

.section-card__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  line-height: var(--line-height-tight);
  color: var(--color-text-primary);
}

.section-card__description {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  line-height: var(--line-height-normal);
  color: var(--color-text-secondary);
}

.section-card__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
}

.section-card__content {
  min-width: 0;
}

.section-card__footer {
  border-top: 1px solid var(--color-border);
  padding-top: var(--card-gap);
}
</style>
