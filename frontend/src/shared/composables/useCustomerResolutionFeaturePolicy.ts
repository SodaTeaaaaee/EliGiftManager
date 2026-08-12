import { readonly, ref } from 'vue'
import type { CustomerResolutionFeaturePolicyDTO } from '@/entities/customer-resolution'
import { getCustomerResolutionFeaturePolicy } from '@/shared/api/bridge'
import {
  isCustomerResolutionFeatureEnabled,
  type CustomerResolutionFeatureFlag,
} from '@/shared/lib/customer-resolution/featurePolicy'

const policy = ref<CustomerResolutionFeaturePolicyDTO | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
let pendingLoad: Promise<CustomerResolutionFeaturePolicyDTO | null> | null = null

async function load(force = false): Promise<CustomerResolutionFeaturePolicyDTO | null> {
  if (policy.value && !force) return policy.value
  if (pendingLoad) return pendingLoad

  loading.value = true
  error.value = null
  pendingLoad = getCustomerResolutionFeaturePolicy()
    .then((next) => {
      policy.value = next
      return next
    })
    .catch((err) => {
      error.value = err instanceof Error ? err.message : String(err)
      return null
    })
    .finally(() => {
      loading.value = false
      pendingLoad = null
    })
  return pendingLoad
}

function replace(next: CustomerResolutionFeaturePolicyDTO): void {
  policy.value = next
  error.value = null
}

function isEnabled(flag: CustomerResolutionFeatureFlag): boolean {
  return isCustomerResolutionFeatureEnabled(policy.value, flag)
}

export function useCustomerResolutionFeaturePolicy() {
  return {
    policy: readonly(policy),
    loading: readonly(loading),
    error: readonly(error),
    load,
    replace,
    isEnabled,
  }
}
