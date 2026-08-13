<script setup lang="ts">
/**
 * IntegrationCard — one added platform. Builtin shortcut and custom create
 * render through this same card; readiness lives on the shared detail drawer.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { IntegrationProfile } from '@/entities/profile'
import { isFactoryProfile } from './profileAvailability'

const props = defineProps<{ profile: IntegrationProfile }>()

defineEmits<{ click: [] }>()

const { t } = useI18n({ useScope: 'global' })

const title = computed(() => props.profile.sourceChannel || props.profile.profileKey)
const showKey = computed(() =>
  Boolean(props.profile.sourceChannel) && props.profile.sourceChannel !== props.profile.profileKey,
)
const surfaceLabel = computed(() =>
  isFactoryProfile(props.profile)
    ? t('integrations.card.surfaceFactory')
    : t('integrations.card.surfaceSource'),
)
</script>

<template>
  <button type="button" class="integration-card" @click="$emit('click')">
    <header class="integration-card__header">
      <h3 class="integration-card__title">{{ title }}</h3>
      <span class="integration-card__surface-label">{{ surfaceLabel }}</span>
    </header>
    <p v-if="showKey" class="integration-card__channel">{{ profile.profileKey }}</p>
    <p v-else-if="profile.sourceSurface === 'factory' && profile.factorySupplierPlatform" class="integration-card__channel">
      {{ profile.factorySupplierPlatform }}
    </p>
  </button>
</template>

<style scoped>
.integration-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  text-align: left;
  padding: var(--card-padding);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  background: var(--card-bg);
  box-shadow: var(--card-shadow);
  cursor: pointer;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    transform var(--duration-fast) var(--ease-out);
}

.integration-card:hover {
  border-color: var(--color-accent);
}

.integration-card:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.integration-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.integration-card__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.integration-card__channel {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.integration-card__surface-label {
  flex-shrink: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}
</style>
