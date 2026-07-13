<script setup lang="ts">
/**
 * StepBusinessSurface — wizard step 2 (create mode only). A simple two-card
 * choice confirming/overriding `demandKind`, independent of whatever the
 * platform preset pre-filled — an operator onboarding e.g. a merch-only
 * Bilibili shop, not a membership programme, can still pick "Retail Order"
 * here even though the `bilibili` preset defaults to membership.
 */
import { useI18n } from 'vue-i18n'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'
import type { DemandKind } from './deriveProfileDefaults'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t } = useI18n({ useScope: 'global' })

const OPTIONS: { value: DemandKind; key: string }[] = [
  { value: 'membership_entitlement', key: 'membership' },
  { value: 'retail_order', key: 'retail' },
]

function select(value: DemandKind): void {
  props.state.demandKind.value = value
}
</script>

<template>
  <div class="step-business-surface">
    <button
      v-for="option in OPTIONS"
      :key="option.value"
      type="button"
      class="step-business-surface__card"
      :class="{ 'step-business-surface__card--active': state.demandKind.value === option.value }"
      @click="select(option.value)"
    >
      <span class="step-business-surface__label">{{ t(`intakeWizard.businessSurface.${option.key}.label`) }}</span>
      <span class="step-business-surface__desc">{{ t(`intakeWizard.businessSurface.${option.key}.description`) }}</span>
    </button>
  </div>
</template>

<style scoped>
.step-business-surface {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--space-3);
  max-width: 640px;
}

.step-business-surface__card {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  text-align: left;
  padding: var(--space-4);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  background: var(--color-surface);
  cursor: pointer;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}

.step-business-surface__card:hover {
  border-color: var(--color-accent);
}

.step-business-surface__card--active {
  border-color: var(--color-accent);
  background: var(--color-accent-subtle);
}

.step-business-surface__label {
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.step-business-surface__desc {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  line-height: var(--line-height-relaxed);
  color: var(--color-text-muted);
}
</style>
