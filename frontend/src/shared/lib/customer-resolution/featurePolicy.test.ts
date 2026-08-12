import { describe, expect, it } from 'vitest'
import type { CustomerResolutionFeaturePolicyDTO } from '@/entities/customer-resolution'
import {
  customerResolutionFeatureDisabledCode,
  customerResolutionWriteAccess,
  isCustomerResolutionFeatureDisabledError,
  isCustomerResolutionFeatureEnabled,
  isCustomerResolutionFeaturePolicyRevisionConflict,
} from './featurePolicy'

function policy(overrides: Partial<CustomerResolutionFeaturePolicyDTO> = {}): CustomerResolutionFeaturePolicyDTO {
  return {
    revision: 1,
    customerResolutionWritesEnabled: true,
    candidateScanEnabled: true,
    mergeExecutionEnabled: true,
    splitExecutionEnabled: true,
    importEvidenceEnabled: true,
    carrierRegistryWritesEnabled: true,
    actorRef: '',
    reason: '',
    updatedAt: '2026-07-15T00:00:00Z',
    ...overrides,
  } as CustomerResolutionFeaturePolicyDTO
}

describe('customer resolution feature policy', () => {
  it('applies the master kill switch before every child flag', () => {
    const disabled = policy({ customerResolutionWritesEnabled: false })
    expect(isCustomerResolutionFeatureEnabled(disabled, 'customerResolutionWritesEnabled')).toBe(false)
    for (const flag of [
      'candidateScanEnabled',
      'mergeExecutionEnabled',
      'splitExecutionEnabled',
      'importEvidenceEnabled',
      'carrierRegistryWritesEnabled',
    ] as const) {
      expect(isCustomerResolutionFeatureEnabled(disabled, flag)).toBe(false)
    }
  })

  it('keeps child flags independent while the master switch is enabled', () => {
    const mixed = policy({ splitExecutionEnabled: false })
    expect(isCustomerResolutionFeatureEnabled(mixed, 'mergeExecutionEnabled')).toBe(true)
    expect(isCustomerResolutionFeatureEnabled(mixed, 'splitExecutionEnabled')).toBe(false)
  })

  it('recognizes stable backend error codes', () => {
    expect(customerResolutionFeatureDisabledCode('carrierRegistryWritesEnabled')).toBe('carrier_registry_writes_disabled')
    expect(isCustomerResolutionFeatureDisabledError(new Error('carrier_registry_writes_disabled: disabled by operator'))).toBe(true)
    expect(isCustomerResolutionFeaturePolicyRevisionConflict(new Error('customer_resolution_feature_policy_revision_conflict: stale revision'))).toBe(true)
  })

  it('gates every customer-profile write surface with the master switch', () => {
    expect(customerResolutionWriteAccess(policy({ customerResolutionWritesEnabled: false }))).toEqual({
      canCreateProfile: false,
      canEditProfile: false,
      canDeleteProfile: false,
      canManageIdentities: false,
      canManageAddresses: false,
    })
  })
})
