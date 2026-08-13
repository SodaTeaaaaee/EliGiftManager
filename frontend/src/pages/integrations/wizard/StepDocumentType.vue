<script setup lang="ts">
/**
 * StepDocumentType — pick exactly one enabled file kind for this mapping session.
 */
import { useI18n } from 'vue-i18n'
import { SelectableCard } from '@/shared/ui/cards'
import { StatusBadge } from '@/shared/ui/status'
import { CalloutBar } from '@/shared/ui/guidance'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t } = useI18n({ useScope: 'global' })

function isConfigured(type: string): boolean {
  return props.state.configuredDocumentTypes.value.includes(type)
}
</script>

<template>
  <div class="step-document-type">
    <CalloutBar
      v-if="state.persistedProfile.value && state.configuredDocumentTypes.value.length"
      tone="info"
      :message="t('intakeWizard.documentType.loopHint')"
    />
    <p class="step-document-type__hint">{{ t('intakeWizard.documentType.pickHint') }}</p>
    <SelectableCard
      v-for="type in state.enabledDocumentTypes.value"
      :key="type"
      class="step-document-type__card"
      :label="t('intakeWizard.documentType.sessionLabel')"
      :selected="state.sessionDocumentType.value === type"
      @select="state.setSessionDocumentType(type)"
    >
      <div class="step-document-type__card-body">
        <StatusBadge dimension="documentType" :value="type" size="sm" />
        <span v-if="isConfigured(type)" class="step-document-type__configured">
          {{ t('intakeWizard.documentType.alreadyConfigured') }}
        </span>
      </div>
    </SelectableCard>
  </div>
</template>

<style scoped>
.step-document-type {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  max-width: 560px;
}

.step-document-type__hint {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.step-document-type__card-body {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.step-document-type__configured {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.step-document-type__card :deep(.selectable-card__label) {
  display: none;
}
</style>
