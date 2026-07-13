<script setup lang="ts">
/**
 * StepConfirm — the wizard's final step (both modes). Read-only summary of
 * everything the operator configured, plus the derived strategy fields
 * (`deriveProfileDefaults.ts`) rendered via `StatusBadge` — never a raw enum
 * string. `Finish` triggers `state.finish()` from the parent `IntakeWizard`
 * (WizardFrame's `finish` emit), not from inside this component.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { StatusBadge } from '@/shared/ui/status'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'
import { INTAKE_DEST_FIELD_ORDER, destFieldLabelKey } from './destFields'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t } = useI18n({ useScope: 'global' })

const CAPABILITY_KEYS = [
  'supportsPartialShipment',
  'supportsApiImport',
  'supportsApiExport',
  'requiresCarrierMapping',
  'requiresExternalOrderNo',
  'allowsManualClosure',
] as const

const STRATEGY_ROWS = [
  { dimension: 'initialAllocationStrategy' as const, get: () => props.state.derivedStrategy.value.initialAllocationStrategy },
  { dimension: 'identityStrategy' as const, get: () => props.state.derivedStrategy.value.identityStrategy },
  { dimension: 'entitlementAuthorityMode' as const, get: () => props.state.derivedStrategy.value.entitlementAuthorityMode },
  { dimension: 'recipientInputMode' as const, get: () => props.state.derivedStrategy.value.recipientInputMode },
  { dimension: 'referenceStrategy' as const, get: () => props.state.derivedStrategy.value.referenceStrategy },
  { dimension: 'trackingSyncMode' as const, get: () => props.state.derivedStrategy.value.trackingSyncMode },
  { dimension: 'closurePolicy' as const, get: () => props.state.derivedStrategy.value.closurePolicy },
]

interface MappingSummaryRow {
  field: string
  labelKey: string
  resolvedKind: 'column' | 'fixed' | 'unmapped'
  resolvedValue: string
}

const mappingSummary = computed<MappingSummaryRow[]>(() =>
  INTAKE_DEST_FIELD_ORDER.map((field) => {
    const fixedValue = props.state.mapping.value.defaults[field]
    if (fixedValue !== undefined && fixedValue !== '') {
      return { field, labelKey: destFieldLabelKey(field), resolvedKind: 'fixed', resolvedValue: fixedValue }
    }
    const column = props.state.mapping.value.columns[field]
    if (column) {
      return { field, labelKey: destFieldLabelKey(field), resolvedKind: 'column', resolvedValue: column }
    }
    return { field, labelKey: destFieldLabelKey(field), resolvedKind: 'unmapped', resolvedValue: '' }
  }),
)
</script>

<template>
  <div class="step-confirm">
    <section class="step-confirm__section">
      <h4 class="step-confirm__section-title">{{ t('intakeWizard.confirm.profileTitle') }}</h4>
      <p v-if="state.isRemapMode.value" class="step-confirm__remap-notice">{{ t('intakeWizard.confirm.remapNotice') }}</p>
      <dl class="step-confirm__kv">
        <template v-if="!state.isRemapMode.value">
          <dt>{{ t('intakeWizard.profileFields.profileKeyLabel') }}</dt>
          <dd>{{ state.profileKey.value }}</dd>
          <dt>{{ t('intakeWizard.profileFields.sourceChannelLabel') }}</dt>
          <dd>{{ state.sourceChannel.value }}</dd>
          <dt>{{ t('intakeWizard.profileFields.sourceSurfaceLabel') }}</dt>
          <dd>{{ state.sourceSurface.value }}</dd>
          <dt>{{ t('statusKit.dimensionNames.demandKind') }}</dt>
          <dd><StatusBadge dimension="demandKind" :value="state.demandKind.value" size="sm" /></dd>
        </template>
        <template v-else>
          <dt>{{ t('intakeWizard.profileFields.profileKeyLabel') }}</dt>
          <dd>{{ state.profileKey.value }}</dd>
        </template>
      </dl>
    </section>

    <section v-if="!state.isRemapMode.value" class="step-confirm__section">
      <h4 class="step-confirm__section-title">{{ t('intakeWizard.confirm.strategyTitle') }}</h4>
      <div class="step-confirm__badge-row">
        <StatusBadge
          v-for="row in STRATEGY_ROWS"
          :key="row.dimension"
          :dimension="row.dimension"
          :value="row.get()"
          size="sm"
          show-dot
        />
      </div>
    </section>

    <section v-if="!state.isRemapMode.value" class="step-confirm__section">
      <h4 class="step-confirm__section-title">{{ t('intakeWizard.confirm.capabilitiesTitle') }}</h4>
      <dl class="step-confirm__kv">
        <template v-for="key in CAPABILITY_KEYS" :key="key">
          <dt>{{ t(`intakeWizard.capabilities.${key}.label`) }}</dt>
          <dd>{{ state.capabilities[key] ? t('common.yes') : t('common.no') }}</dd>
        </template>
      </dl>
    </section>

    <section class="step-confirm__section">
      <h4 class="step-confirm__section-title">{{ t('intakeWizard.confirm.mappingTitle') }}</h4>
      <dl class="step-confirm__kv">
        <template v-for="row in mappingSummary" :key="row.field">
          <dt>{{ t(row.labelKey) }}</dt>
          <dd :class="{ 'step-confirm__unmapped': row.resolvedKind === 'unmapped' }">
            {{ row.resolvedKind === 'unmapped' ? t('intakeWizard.mapping.unmapped') : row.resolvedValue }}
          </dd>
        </template>
      </dl>
    </section>
  </div>
</template>

<style scoped>
.step-confirm {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.step-confirm__section-title {
  margin: 0 0 var(--space-2);
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.step-confirm__remap-notice {
  margin: 0 0 var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.step-confirm__kv {
  display: grid;
  grid-template-columns: minmax(160px, 1fr) minmax(160px, 1.6fr);
  gap: var(--space-1) var(--space-4);
  margin: 0;
}

.step-confirm__kv dt {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.step-confirm__kv dd {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.step-confirm__unmapped {
  color: var(--color-text-muted);
  font-style: italic;
}

.step-confirm__badge-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}
</style>
