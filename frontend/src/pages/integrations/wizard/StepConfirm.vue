<script setup lang="ts">
/**
 * StepConfirm — the wizard's final step (both modes). Read-only summary of
 * everything the operator configured, plus the derived strategy fields
 * (`deriveProfileDefaults.ts`) rendered via `StatusBadge` — never a raw enum
 * string. `Finish` triggers `state.finish()` from the parent `IntakeWizard`
 * (WizardFrame's `finish` emit), not from inside this component.
 *
 * `persistError` / `bindWarning` render inline so a failed or degraded finish
 * is visible without relying solely on a floating toast.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { StatusBadge } from '@/shared/ui/status'
import { CalloutBar } from '@/shared/ui/guidance'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'
import { destFieldLabelKey, destKeysForDocumentType } from './destFields'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t, te } = useI18n({ useScope: 'global' })

const DEMAND_CAPABILITY_KEYS = [
  'supportsPartialShipment',
  'supportsApiImport',
  'supportsApiExport',
  'requiresCarrierMapping',
  'requiresExternalOrderNo',
  'allowsManualClosure',
] as const

const FACTORY_CAPABILITY_KEYS = [
  'supportsExportSupplierOrder',
  'supportsImportProductCatalog',
  'supportsImportSupplierShipment',
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
  resolvedKind: 'column' | 'position' | 'fixed' | 'unmapped'
  resolvedValue: string
}

const mappingSummary = computed<MappingSummaryRow[]>(() => {
  const mapping = props.state.mapping.value
  const mode = mapping.mode === 'positional' ? 'positional' : 'header'
  const extraKeys = new Set([
    ...Object.keys(mapping.columns ?? {}),
    ...Object.keys(mapping.positions ?? {}),
    ...Object.keys(mapping.defaults ?? {}),
  ])
  const ordered = [...destKeysForDocumentType(props.state.sessionDocumentType.value || props.state.documentType.value)]
  for (const key of extraKeys) {
    if (!ordered.includes(key)) ordered.push(key)
  }
  return ordered
    .map((field) => {
      const fixedValue = mapping.defaults[field]
      if (fixedValue !== undefined && fixedValue !== '') {
        return { field, labelKey: destFieldLabelKey(field), resolvedKind: 'fixed' as const, resolvedValue: fixedValue }
      }
      if (mode === 'positional') {
        const positions = mapping.positions ?? {}
        if (field in positions) {
          return {
            field,
            labelKey: destFieldLabelKey(field),
            resolvedKind: 'position' as const,
            resolvedValue: String(positions[field]),
          }
        }
      } else {
        const column = mapping.columns[field]
        if (column) {
          return { field, labelKey: destFieldLabelKey(field), resolvedKind: 'column' as const, resolvedValue: column }
        }
      }
      return { field, labelKey: destFieldLabelKey(field), resolvedKind: 'unmapped' as const, resolvedValue: '' }
    })
    .filter((row) => row.resolvedKind !== 'unmapped')
})

const bothDemandFiles = computed(
  () => props.state.enableEntitlementImport.value && props.state.enableSalesOrderImport.value,
)

const showStrategy = computed(
  () =>
    !props.state.isRemapMode.value &&
    !props.state.isFactorySurface.value &&
    !props.state.persistedProfile.value &&
    !bothDemandFiles.value,
)
</script>

<template>
  <div class="step-confirm">
    <CalloutBar
      v-if="state.persistError.value"
      tone="error"
      :message="state.persistError.value"
    />
    <CalloutBar
      v-if="state.bindWarning.value"
      tone="warning"
      :message="state.bindWarning.value"
    />

    <CalloutBar
      tone="info"
      :message="t('intakeWizard.confirm.thisFileOnly')"
    />
    <CalloutBar
      tone="info"
      :message="t('intakeWizard.confirm.profileIsPlatform')"
    />
    <CalloutBar
      v-if="state.persistedProfile.value"
      tone="info"
      :message="t('intakeWizard.documentType.loopHint')"
    />

    <section class="step-confirm__section">
      <h4 class="step-confirm__section-title">{{ t('intakeWizard.confirm.profileTitle') }}</h4>
      <p v-if="state.isRemapMode.value || state.persistedProfile.value" class="step-confirm__remap-notice">
        {{ t('intakeWizard.confirm.remapNotice') }}
      </p>
      <dl class="step-confirm__kv">
        <template v-if="!state.isRemapMode.value">
          <dt>{{ t('intakeWizard.profileFields.profileKeyLabel') }}</dt>
          <dd>{{ state.profileKey.value }}</dd>
          <dt>{{ t('intakeWizard.profileFields.sourceChannelLabel') }}</dt>
          <dd>{{ state.sourceChannel.value }}</dd>
          <dt>{{ t('intakeWizard.profileFields.sourceSurfaceLabel') }}</dt>
          <dd>{{ state.sourceSurface.value }}</dd>
          <template v-if="state.isFactorySurface.value">
            <dt>{{ t('intakeWizard.platformKind.factory.label') }}</dt>
            <dd>{{ t('intakeWizard.platformKind.factory.label') }}</dd>
            <dt>{{ t('intakeWizard.profileFields.factorySupplierPlatformLabel') }}</dt>
            <dd>{{ state.factorySupplierPlatform.value || '—' }}</dd>
          </template>
          <dt>{{ t('intakeWizard.confirm.documentType') }}</dt>
          <dd><StatusBadge dimension="documentType" :value="state.documentType.value" size="sm" /></dd>
        </template>
        <template v-else>
          <dt>{{ t('intakeWizard.profileFields.profileKeyLabel') }}</dt>
          <dd>{{ state.profileKey.value }}</dd>
          <dt>{{ t('intakeWizard.confirm.documentType') }}</dt>
          <dd><StatusBadge dimension="documentType" :value="state.documentType.value" size="sm" /></dd>
        </template>
      </dl>
    </section>

    <section v-if="showStrategy" class="step-confirm__section">
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
        <template v-if="state.isFactorySurface.value">
          <template v-for="key in FACTORY_CAPABILITY_KEYS" :key="key">
            <dt>{{ t(`intakeWizard.capabilities.${key}.label`) }}</dt>
            <dd>{{ state.factoryCapabilities[key] ? t('common.yes') : t('common.no') }}</dd>
          </template>
        </template>
        <template v-else>
          <dt>{{ t('intakeWizard.fileKinds.import_entitlement.label') }}</dt>
          <dd>{{ state.enableEntitlementImport.value ? t('common.yes') : t('common.no') }}</dd>
          <dt>{{ t('intakeWizard.fileKinds.import_sales_order.label') }}</dt>
          <dd>{{ state.enableSalesOrderImport.value ? t('common.yes') : t('common.no') }}</dd>
          <template v-for="key in DEMAND_CAPABILITY_KEYS" :key="key">
            <dt>{{ t(`intakeWizard.capabilities.${key}.label`) }}</dt>
            <dd>{{ state.capabilities[key] ? t('common.yes') : t('common.no') }}</dd>
          </template>
        </template>
      </dl>
    </section>

    <section class="step-confirm__section">
      <h4 class="step-confirm__section-title">{{ t('intakeWizard.confirm.mappingTitle') }}</h4>
      <p class="step-confirm__remap-notice">
        {{ t('intakeWizard.mapping.modeLabel') }}:
        {{
          state.mapping.value.mode === 'positional'
            ? t('intakeWizard.mapping.modePositional')
            : t('intakeWizard.mapping.modeHeader')
        }}
        ·
        {{ t('intakeWizard.mapping.hasHeaderLabel') }}:
        {{ state.mapping.value.hasHeader === false ? t('common.no') : t('common.yes') }}
      </p>
      <dl v-if="mappingSummary.length" class="step-confirm__kv">
        <template v-for="row in mappingSummary" :key="row.field">
          <dt>{{ te(row.labelKey) ? t(row.labelKey) : row.field }}</dt>
          <dd>
            <template v-if="row.resolvedKind === 'position'">
              #{{ row.resolvedValue }}
            </template>
            <template v-else>
              {{ row.resolvedValue }}
            </template>
          </dd>
        </template>
      </dl>
      <p v-else class="step-confirm__remap-notice">{{ t('intakeWizard.confirm.mappingEmpty') }}</p>
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

.step-confirm__badge-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}
</style>
