<script setup lang="ts">
/**
 * PageHeader — the standard page-top block: small eyebrow "kicker", title,
 * optional description, and a right-aligned actions zone. Used at the top
 * of every top-level page (Task Center, Waves, Inbox, ...) and every wave-
 * workspace step, so page structure reads consistently across the app.
 */
withDefaults(
  defineProps<{
    kicker?: string
    /** Omit to use the `#title` slot for rich content (e.g. an inline editable field). */
    title?: string
    description?: string
  }>(),
  {
    kicker: undefined,
    title: undefined,
    description: undefined,
  },
)
</script>

<template>
  <header class="page-header">
    <div class="page-header__heading">
      <p v-if="kicker || $slots.kicker" class="page-header__kicker">
        <slot name="kicker">{{ kicker }}</slot>
      </p>
      <h1 v-if="title || $slots.title" class="page-header__title">
        <slot name="title">{{ title }}</slot>
      </h1>
      <p v-if="description || $slots.description" class="page-header__description">
        <slot name="description">{{ description }}</slot>
      </p>
    </div>
    <div v-if="$slots.actions" class="page-header__actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-4);
  padding-bottom: var(--space-4);
}

.page-header__heading {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
  flex: 1 1 320px;
}

.page-header__kicker {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-accent);
}

.page-header__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-2xl);
  font-weight: var(--font-weight-semibold);
  line-height: var(--line-height-tight);
  color: var(--color-text-primary);
}

.page-header__description {
  margin: 0;
  max-width: 48em;
  font-family: var(--font-body);
  font-size: var(--font-size-base);
  line-height: var(--line-height-relaxed);
  color: var(--color-text-secondary);
}

.page-header__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
  padding-top: var(--space-1);
}
</style>
