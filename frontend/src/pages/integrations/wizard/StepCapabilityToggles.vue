<script setup lang="ts">
/**
 * StepCapabilityToggles — wizard step 4 (both create and remap modes,
 * though it only visibly matters in create mode; remap mode still mounts it
 * so the operator can review the existing profile's capabilities, but they
 * are read-only there — see `:disabled="state.isRemapMode.value"`).
 *
 * Demand surfaces: the 6 boolean capability switches + optional connectorKey.
 * Factory surface: the 3 factory capability switches (export supplier order /
 * import product catalog / import supplier shipment) instead of demand caps.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSwitch, NSelect, NFormItem } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { SelectableCard } from '@/shared/ui/cards'
import { StatusBadge } from '@/shared/ui/status'
import { listConnectorCapabilities } from '@/shared/api/bridge'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'
import type { IntakeProfileCapabilities } from '@/shared/lib/demand-intake/platform-presets'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t } = useI18n({ useScope: 'global' })

const DEMAND_CAPABILITY_KEYS: (keyof IntakeProfileCapabilities)[] = [
  'supportsPartialShipment',
  'supportsApiImport',
  'supportsApiExport',
  'requiresCarrierMapping',
  'requiresExternalOrderNo',
  'allowsManualClosure',
]

const FACTORY_CAPABILITY_KEYS = [
  'supportsExportSupplierOrder',
  'supportsImportProductCatalog',
  'supportsImportSupplierShipment',
] as const

const connectorKeys = ref<string[]>([])

onMounted(async () => {
  try {
    const caps = await listConnectorCapabilities()
    connectorKeys.value = Object.keys(caps)
  } catch {
    connectorKeys.value = []
  }
})

const connectorOptions = computed<SelectOption[]>(() => connectorKeys.value.map((key) => ({ label: key, value: key })))

const hasConnector = computed(() => props.state.connectorKey.value.trim().length > 0)
const isFactory = computed(() => props.state.isFactorySurface.value)

const TRACKING_SYNC_MODE_OPTIONS = ['api_push', 'document_export', 'manual_confirmation', 'unsupported']
</script>

<template>
  <div class="step-capability-toggles">
    <template v-if="isFactory">
      <p class="step-capability-toggles__intro">{{ t('intakeWizard.capabilities.factoryIntro') }}</p>
      <div
        v-for="key in FACTORY_CAPABILITY_KEYS"
        :key="key"
        class="step-capability-toggles__row"
      >
        <div class="step-capability-toggles__label-group">
          <span class="step-capability-toggles__label">{{ t(`intakeWizard.capabilities.${key}.label`) }}</span>
          <span class="step-capability-toggles__hint">{{ t(`intakeWizard.capabilities.${key}.hint`) }}</span>
        </div>
        <NSwitch v-model:value="state.factoryCapabilities[key]" :disabled="state.isRemapMode.value" />
      </div>
    </template>

    <template v-else>
      <div v-for="key in DEMAND_CAPABILITY_KEYS" :key="key" class="step-capability-toggles__row">
        <div class="step-capability-toggles__label-group">
          <span class="step-capability-toggles__label">{{ t(`intakeWizard.capabilities.${key}.label`) }}</span>
          <span class="step-capability-toggles__hint">{{ t(`intakeWizard.capabilities.${key}.hint`) }}</span>
        </div>
        <NSwitch v-model:value="state.capabilities[key]" :disabled="state.isRemapMode.value" />
      </div>
    </template>

    <NFormItem :label="t('intakeWizard.capabilities.connectorKeyLabel')" :show-feedback="false">
      <NSelect
        v-model:value="state.connectorKey.value"
        :options="connectorOptions"
        clearable
        filterable
        tag
        :placeholder="t('intakeWizard.capabilities.connectorKeyPlaceholder')"
        :disabled="state.isRemapMode.value"
      />
    </NFormItem>

    <NFormItem v-if="hasConnector" :label="t('statusKit.dimensionNames.trackingSyncMode')" :show-feedback="false">
      <div class="step-capability-toggles__sync-mode-options">
        <SelectableCard
          v-for="mode in TRACKING_SYNC_MODE_OPTIONS"
          :key="mode"
          class="step-capability-toggles__sync-card"
          :label="mode"
          :selected="state.trackingSyncModeOverride.value === mode"
          :disabled="state.isRemapMode.value"
          @select="state.trackingSyncModeOverride.value = mode"
        >
          <StatusBadge dimension="trackingSyncMode" :value="mode" size="sm" />
        </SelectableCard>
      </div>
    </NFormItem>
  </div>
</template>

<style scoped>
.step-capability-toggles {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  max-width: 560px;
}

.step-capability-toggles__intro {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.step-capability-toggles__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--card-border-color);
}

.step-capability-toggles__label-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}

.step-capability-toggles__label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.step-capability-toggles__hint {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.step-capability-toggles__sync-mode-options {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  width: 100%;
}

.step-capability-toggles__sync-card {
  min-width: 140px;
  flex: 1 1 140px;
}

.step-capability-toggles__sync-card :deep(.selectable-card__label) {
  display: none;
}
</style>
