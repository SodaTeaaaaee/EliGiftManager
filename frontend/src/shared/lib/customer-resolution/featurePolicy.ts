import type { CustomerResolutionFeaturePolicyDTO } from '@/entities/customer-resolution'

export type CustomerResolutionFeatureFlag =
  | 'customerResolutionWritesEnabled'
  | 'candidateScanEnabled'
  | 'mergeExecutionEnabled'
  | 'splitExecutionEnabled'
  | 'importEvidenceEnabled'
  | 'carrierRegistryWritesEnabled'

export const CUSTOMER_RESOLUTION_FEATURE_FLAGS: CustomerResolutionFeatureFlag[] = [
  'customerResolutionWritesEnabled',
  'candidateScanEnabled',
  'mergeExecutionEnabled',
  'splitExecutionEnabled',
  'importEvidenceEnabled',
  'carrierRegistryWritesEnabled',
]

export interface CustomerResolutionWriteAccess {
  canCreateProfile: boolean
  canEditProfile: boolean
  canDeleteProfile: boolean
  canManageIdentities: boolean
  canManageAddresses: boolean
}

const STABLE_ERROR_CODES: Record<CustomerResolutionFeatureFlag, string> = {
  customerResolutionWritesEnabled: 'customer_resolution_writes_disabled',
  candidateScanEnabled: 'candidate_scan_disabled',
  mergeExecutionEnabled: 'merge_execution_disabled',
  splitExecutionEnabled: 'split_execution_disabled',
  importEvidenceEnabled: 'import_evidence_disabled',
  carrierRegistryWritesEnabled: 'carrier_registry_writes_disabled',
}

export function isCustomerResolutionFeatureEnabled(
  policy: CustomerResolutionFeaturePolicyDTO | null,
  flag: CustomerResolutionFeatureFlag,
): boolean {
  if (!policy) return false
  if (flag === 'customerResolutionWritesEnabled') return policy.customerResolutionWritesEnabled
  return policy.customerResolutionWritesEnabled && policy[flag]
}

export function customerResolutionWriteAccess(
  policy: CustomerResolutionFeaturePolicyDTO | null,
): CustomerResolutionWriteAccess {
  const enabled = isCustomerResolutionFeatureEnabled(policy, 'customerResolutionWritesEnabled')
  return {
    canCreateProfile: enabled,
    canEditProfile: enabled,
    canDeleteProfile: enabled,
    canManageIdentities: enabled,
    canManageAddresses: enabled,
  }
}

export function customerResolutionFeatureDisabledCode(flag: CustomerResolutionFeatureFlag): string {
  return STABLE_ERROR_CODES[flag]
}

export function isCustomerResolutionFeaturePolicyRevisionConflict(error: unknown): boolean {
  const message = error instanceof Error ? error.message : String(error)
  return message.includes('customer_resolution_feature_policy_revision_conflict')
}

export function isCustomerResolutionFeatureDisabledError(error: unknown): boolean {
  const message = error instanceof Error ? error.message : String(error)
  return Object.values(STABLE_ERROR_CODES).some((code) => message.includes(code))
}
