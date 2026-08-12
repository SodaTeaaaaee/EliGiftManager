<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSwitch } from 'naive-ui'
import type { CustomerResolutionFeaturePolicyDTO } from '@/entities/customer-resolution'
import { updateCustomerResolutionFeaturePolicy } from '@/shared/api/bridge'
import { useCustomerResolutionFeaturePolicy } from '@/shared/composables/useCustomerResolutionFeaturePolicy'
import {
  CUSTOMER_RESOLUTION_FEATURE_FLAGS,
  isCustomerResolutionFeatureEnabled,
  isCustomerResolutionFeaturePolicyRevisionConflict,
  type CustomerResolutionFeatureFlag,
} from '@/shared/lib/customer-resolution'
import { ErrorBanner } from '@/shared/ui/feedback'

const { t } = useI18n({ useScope: 'global' })
const featurePolicy = useCustomerResolutionFeaturePolicy()
const savingFlag = ref<CustomerResolutionFeatureFlag | null>(null)
const saveError = ref<string | null>(null)

const form = reactive<Record<CustomerResolutionFeatureFlag, boolean>>({
  customerResolutionWritesEnabled: true,
  candidateScanEnabled: true,
  mergeExecutionEnabled: true,
  splitExecutionEnabled: true,
  importEvidenceEnabled: true,
  carrierRegistryWritesEnabled: true,
})

watch(
  () => featurePolicy.policy.value,
  (policy) => {
    if (!policy) return
    for (const flag of CUSTOMER_RESOLUTION_FEATURE_FLAGS) form[flag] = policy[flag]
  },
  { immediate: true },
)

const loadError = computed(() => featurePolicy.error.value)

function effectiveEnabled(flag: CustomerResolutionFeatureFlag): boolean {
  return isCustomerResolutionFeatureEnabled(featurePolicy.policy.value as CustomerResolutionFeaturePolicyDTO | null, flag)
}

async function reload(): Promise<void> {
  saveError.value = null
  await featurePolicy.load(true)
}

async function updateFlag(flag: CustomerResolutionFeatureFlag, value: boolean): Promise<void> {
  const current = featurePolicy.policy.value
  if (!current || savingFlag.value) return

  savingFlag.value = flag
  saveError.value = null
  const previous = form[flag]
  form[flag] = value
  try {
    const next = await updateCustomerResolutionFeaturePolicy({
      expectedRevision: current.revision,
      customerResolutionWritesEnabled: form.customerResolutionWritesEnabled,
      candidateScanEnabled: form.candidateScanEnabled,
      mergeExecutionEnabled: form.mergeExecutionEnabled,
      splitExecutionEnabled: form.splitExecutionEnabled,
      importEvidenceEnabled: form.importEvidenceEnabled,
      carrierRegistryWritesEnabled: form.carrierRegistryWritesEnabled,
      actorRef: 'local_user',
      reason: `settings:${flag}`,
    })
    featurePolicy.replace(next)
  } catch (err) {
    form[flag] = previous
    saveError.value = err instanceof Error ? err.message : String(err)
    if (isCustomerResolutionFeaturePolicyRevisionConflict(err)) await featurePolicy.load(true)
  } finally {
    savingFlag.value = null
  }
}

onMounted(() => {
  void featurePolicy.load()
})
</script>

<template>
  <div class="advanced-policy">
    <ErrorBanner
      v-if="loadError || saveError"
      :message="saveError ? t('settings.featurePolicy.saveFailed') : t('settings.featurePolicy.loadFailed')"
      :detail="saveError ?? loadError ?? undefined"
      @retry="reload"
    />

    <p class="advanced-policy__notice">{{ t('settings.featurePolicy.killSwitchNotice') }}</p>
    <div v-if="featurePolicy.policy.value" class="advanced-policy__list">
      <div
        v-for="flag in CUSTOMER_RESOLUTION_FEATURE_FLAGS"
        :key="flag"
        class="advanced-policy__row"
      >
        <div class="advanced-policy__copy">
          <span class="advanced-policy__label">{{ t(`settings.featurePolicy.flags.${flag}.label`) }}</span>
          <span class="advanced-policy__description">{{ t(`settings.featurePolicy.flags.${flag}.description`) }}</span>
          <code v-if="flag !== 'customerResolutionWritesEnabled' && !effectiveEnabled(flag)" class="advanced-policy__effective">
            {{ t('settings.featurePolicy.effectivelyDisabled') }}
          </code>
        </div>
        <NSwitch
          :value="form[flag]"
          :loading="savingFlag === flag"
          :disabled="featurePolicy.loading.value || savingFlag !== null"
          @update:value="updateFlag(flag, $event)"
        />
      </div>
    </div>
    <p v-else-if="featurePolicy.loading.value" class="advanced-policy__notice">{{ t('common.loading') }}</p>

    <div v-if="featurePolicy.policy.value" class="advanced-policy__revision">
      <span>{{ t('settings.featurePolicy.revision', { revision: featurePolicy.policy.value.revision }) }}</span>
      <span>{{ t('settings.featurePolicy.stableErrors') }}</span>
    </div>
  </div>
</template>

<style scoped>
.advanced-policy,
.advanced-policy__list,
.advanced-policy__copy {
  display: flex;
  flex-direction: column;
}

.advanced-policy {
  gap: var(--space-3);
}

.advanced-policy__notice,
.advanced-policy__revision {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}

.advanced-policy__list {
  gap: var(--space-2);
}

.advanced-policy__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--color-border);
}

.advanced-policy__row:last-child {
  border-bottom: 0;
}

.advanced-policy__copy {
  gap: var(--space-1);
  min-width: 0;
}

.advanced-policy__label {
  color: var(--color-text-primary);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}

.advanced-policy__description,
.advanced-policy__effective {
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}

.advanced-policy__revision {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: var(--space-2);
}
</style>
