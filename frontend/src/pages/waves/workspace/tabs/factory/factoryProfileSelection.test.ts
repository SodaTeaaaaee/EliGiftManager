import { describe, expect, test } from 'vitest'
import { eligibleFactoryProfiles, factoryGenerationDecision } from './factoryProfileSelection'

const profiles = [
  { id: 1, profileKey: 'demand', sourceSurface: 'retail', supportsExportSupplierOrder: true, factorySupplierPlatform: 'factory-a' },
  { id: 2, profileKey: 'disabled', sourceSurface: 'factory', supportsExportSupplierOrder: false, factorySupplierPlatform: 'factory-a' },
  { id: 3, profileKey: 'factory-b', sourceSurface: 'factory', supportsExportSupplierOrder: true, factorySupplierPlatform: 'factory-b' },
  { id: 4, profileKey: 'factory-a', sourceSurface: 'factory', supportsExportSupplierOrder: true, factorySupplierPlatform: 'factory-a' },
]

describe('factory supplier-order profile selection', () => {
  test('never offers demand profiles or factory profiles without export capability', () => {
    expect(eligibleFactoryProfiles(profiles).map((profile) => profile.id)).toEqual([4, 3])
  })

  test('keeps other platforms eligible after one platform already has orders', () => {
    expect(eligibleFactoryProfiles(profiles).map((profile) => profile.id)).toEqual([4, 3])
  })

  test('distinguishes independent platform generation from an explicit rebuild', () => {
    const orders = [{ supplierPlatform: 'factory-a', factoryIntegrationProfileId: 4 }]
    expect(factoryGenerationDecision(profiles[2], orders)).toEqual({ kind: 'new_platform' })
    expect(factoryGenerationDecision(profiles[3], orders)).toEqual({ kind: 'rebuild_profile' })
  })

  test('blocks a replacement profile on the same platform to avoid duplicate orders', () => {
    const replacement = { ...profiles[3], id: 5, profileKey: 'factory-a-next' }
    expect(factoryGenerationDecision(replacement, [
      { supplierPlatform: 'factory-a', factoryIntegrationProfileId: 4 },
    ])).toEqual({ kind: 'profile_conflict', existingProfileId: 4 })
  })
})
