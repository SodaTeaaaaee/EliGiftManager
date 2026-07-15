<script setup lang="ts">
/**
 * StepBusinessSurface — wizard step 2 (create mode only). Three-card choice
 * for the integration's business surface: membership, retail, or factory.
 * Selecting a surface updates `sourceSurface` (+ demandKind for demand faces)
 * so factory onboarding is a first-class path, not a free-text sourceSurface.
 */
import { useI18n } from 'vue-i18n'
import { SelectableCard } from '@/shared/ui/cards'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'
import type { BusinessSurfaceChoice } from './useIntakeWizardState'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t } = useI18n({ useScope: 'global' })

const OPTIONS: { value: BusinessSurfaceChoice; key: string }[] = [
  { value: 'membership', key: 'membership' },
  { value: 'retail', key: 'retail' },
  { value: 'factory', key: 'factory' },
]

function select(value: BusinessSurfaceChoice): void {
  props.state.setBusinessSurface(value)
}
</script>

<template>
  <div class="step-business-surface">
    <SelectableCard
      v-for="option in OPTIONS"
      :key="option.value"
      :label="t(`intakeWizard.businessSurface.${option.key}.label`)"
      :description="t(`intakeWizard.businessSurface.${option.key}.description`)"
      :selected="state.businessSurface.value === option.value"
      @select="select(option.value)"
    />
  </div>
</template>

<style scoped>
.step-business-surface {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--space-3);
  max-width: 720px;
}
</style>
