<script setup lang="ts">
/**
 * IntegrationCard — one integration profile summarized as a clickable card
 * (plan P4 integrations page's per-group card grid). Purely presentational:
 * no bridge calls, no state of its own — `IntegrationsPage.vue` owns the
 * `listProfiles()` fetch and passes each `IntegrationProfile` down.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { StatusBadge } from '@/shared/ui/status'
import type { IntegrationProfile } from '@/entities/profile'

const props = defineProps<{ profile: IntegrationProfile }>()

defineEmits<{ click: [] }>()

const { t } = useI18n({ useScope: 'global' })

const CAPABILITY_KEYS = [
  'supportsPartialShipment',
  'supportsApiImport',
  'supportsApiExport',
  'requiresCarrierMapping',
  'requiresExternalOrderNo',
  'allowsManualClosure',
] as const

const enabledCapabilityCount = computed(() => CAPABILITY_KEYS.filter((key) => props.profile[key]).length)
</script>

<template>
  <button type="button" class="integration-card" @click="$emit('click')">
    <header class="integration-card__header">
      <h3 class="integration-card__title">{{ profile.profileKey }}</h3>
      <StatusBadge dimension="demandKind" :value="profile.demandKind" size="sm" />
    </header>
    <p class="integration-card__channel">{{ profile.sourceChannel || '—' }} · {{ profile.sourceSurface || '—' }}</p>
    <dl class="integration-card__zones">
      <div class="integration-card__zone">
        <dt>{{ t('integrations.card.connector') }}</dt>
        <dd>{{ profile.connectorKey || '—' }}</dd>
      </div>
      <div class="integration-card__zone">
        <dt>{{ t('integrations.card.capabilities') }}</dt>
        <dd>{{ enabledCapabilityCount }}/{{ CAPABILITY_KEYS.length }}</dd>
      </div>
      <div class="integration-card__zone">
        <dt>{{ t('integrations.card.templates') }}</dt>
        <dd>{{ t('integrations.actions.manageBindings') }}</dd>
      </div>
    </dl>
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

.integration-card__zones {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-2);
  margin: var(--space-2) 0 0;
}

.integration-card__zone {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.integration-card__zone dt {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.integration-card__zone dd {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
