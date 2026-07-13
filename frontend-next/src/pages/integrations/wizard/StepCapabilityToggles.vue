<script setup lang="ts">
/**
 * StepCapabilityToggles — wizard step 4 (both create and remap modes,
 * though it only visibly matters in create mode; remap mode still mounts it
 * so the operator can review the existing profile's capabilities, but they
 * are read-only there — see `:disabled="state.isRemapMode.value"`).
 *
 * The 6 boolean capability switches, plus an OPTIONAL `connectorKey` picked
 * from `listConnectorCapabilities()`'s registered keys. `trackingSyncMode`
 * only becomes an explicit choice once a connector is picked — see
 * `deriveProfileDefaults.ts`'s doc comment for why this can't be silently
 * guessed once a real connector enters the picture.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSwitch, NSelect, NFormItem } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { StatusBadge } from '@/shared/ui/status'
import { listConnectorCapabilities } from '@/shared/api/bridge'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'
import type { IntakeProfileCapabilities } from '@/shared/lib/demand-intake/platform-presets'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t } = useI18n({ useScope: 'global' })

const CAPABILITY_KEYS: (keyof IntakeProfileCapabilities)[] = [
  'supportsPartialShipment',
  'supportsApiImport',
  'supportsApiExport',
  'requiresCarrierMapping',
  'requiresExternalOrderNo',
  'allowsManualClosure',
]

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

const TRACKING_SYNC_MODE_OPTIONS = ['api_push', 'document_export', 'manual_confirmation', 'unsupported']
</script>

<template>
  <div class="step-capability-toggles">
    <div v-for="key in CAPABILITY_KEYS" :key="key" class="step-capability-toggles__row">
      <div class="step-capability-toggles__label-group">
        <span class="step-capability-toggles__label">{{ t(`intakeWizard.capabilities.${key}.label`) }}</span>
        <span class="step-capability-toggles__hint">{{ t(`intakeWizard.capabilities.${key}.hint`) }}</span>
      </div>
      <NSwitch v-model:value="state.capabilities[key]" :disabled="state.isRemapMode.value" />
    </div>

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
        <button
          v-for="mode in TRACKING_SYNC_MODE_OPTIONS"
          :key="mode"
          type="button"
          class="step-capability-toggles__sync-mode-option"
          :class="{ 'step-capability-toggles__sync-mode-option--active': state.trackingSyncModeOverride.value === mode }"
          :disabled="state.isRemapMode.value"
          @click="state.trackingSyncModeOverride.value = mode"
        >
          <StatusBadge dimension="trackingSyncMode" :value="mode" size="sm" />
        </button>
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
}

.step-capability-toggles__sync-mode-option {
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  padding: 2px;
  cursor: pointer;
}

.step-capability-toggles__sync-mode-option--active {
  border-color: var(--color-accent);
}
</style>
