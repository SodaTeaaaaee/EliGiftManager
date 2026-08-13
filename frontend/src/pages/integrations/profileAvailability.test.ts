import { describe, expect, test } from 'vitest'
import {
  canImportDemand,
  canImportProductCatalog,
  canImportSupplierShipment,
  canCreateRetailDemand,
  expectedFileKinds,
  hasDefaultFileKindBinding,
  installableBuiltins,
  isBuiltinProfileKey,
  isFactoryProfile,
  isSourcePlatformProfile,
  partitionProfilesForList,
  BILIBILI_BUILTIN_PROFILE_KEY,
  MEMBERSHIP_DEFAULT_PROFILE_KEY,
  ROUZAO_BUILTIN_PROFILE_KEY,
} from './profileAvailability'

describe('specialized profile availability', () => {
  const demand = { sourceSurface: 'retail', demandKind: 'retail_order' }
  const factory = {
    sourceSurface: 'factory',
    demandKind: '',
    supportsImportProductCatalog: true,
    supportsImportSupplierShipment: false,
  }
  const dualKind = { sourceSurface: 'membership', demandKind: '' }
  const membershipLeftover = {
    sourceSurface: 'membership',
    demandKind: 'membership_entitlement',
  }

  test('demand import excludes factory profiles', () => {
    expect(canImportDemand(demand)).toBe(true)
    expect(canImportDemand(factory)).toBe(false)
    expect(canImportDemand(dualKind)).toBe(true)
  })

  test('demand import keeps leftover membership source platforms', () => {
    expect(canImportDemand(membershipLeftover)).toBe(true)
  })

  test('manual retail entry keeps dual-kind and leftover membership platforms', () => {
    expect(canCreateRetailDemand(demand)).toBe(true)
    expect(canCreateRetailDemand(dualKind)).toBe(true)
    expect(canCreateRetailDemand(membershipLeftover)).toBe(true)
    expect(canCreateRetailDemand(factory)).toBe(false)
  })

  test('factory import selectors require the matching capability', () => {
    expect(canImportProductCatalog(factory)).toBe(true)
    expect(canImportSupplierShipment(factory)).toBe(false)
  })
})

describe('integration list grouping', () => {
  test('groups by source platform vs factory, not demandKind', () => {
    const dualKind = { sourceSurface: 'membership', demandKind: '' }
    const membershipLeftover = {
      sourceSurface: 'membership',
      demandKind: 'membership_entitlement',
    }
    const retail = { sourceSurface: 'retail', demandKind: 'retail_order' }
    const factory = { sourceSurface: 'factory', demandKind: '' }

    expect(isSourcePlatformProfile(dualKind)).toBe(true)
    expect(isSourcePlatformProfile(membershipLeftover)).toBe(true)
    expect(isSourcePlatformProfile(retail)).toBe(true)
    expect(isSourcePlatformProfile(factory)).toBe(false)
    expect(isFactoryProfile(factory)).toBe(true)

    const { source, factory: factoryGroup } = partitionProfilesForList([
      dualKind,
      membershipLeftover,
      retail,
      factory,
    ])
    expect(source).toEqual([dualKind, membershipLeftover, retail])
    expect(factoryGroup).toEqual([factory])
  })
})

describe('installable builtins', () => {
  test('hides a builtin once its profileKey is present', () => {
    const remaining = installableBuiltins([
      { profileKey: BILIBILI_BUILTIN_PROFILE_KEY },
      { profileKey: MEMBERSHIP_DEFAULT_PROFILE_KEY },
    ])
    expect(remaining.map((item) => item.installKey)).toEqual(['rouzao'])
    expect(isBuiltinProfileKey(BILIBILI_BUILTIN_PROFILE_KEY)).toBe(true)
    expect(isBuiltinProfileKey(ROUZAO_BUILTIN_PROFILE_KEY)).toBe(true)
    expect(isBuiltinProfileKey(MEMBERSHIP_DEFAULT_PROFILE_KEY)).toBe(false)
  })

  test('does not treat membership_default as an installable builtin', () => {
    expect(installableBuiltins([]).map((item) => item.installKey)).toEqual(['bilibili', 'rouzao'])
    expect(installableBuiltins([{ profileKey: MEMBERSHIP_DEFAULT_PROFILE_KEY }]).map((item) => item.installKey))
      .toEqual(['bilibili', 'rouzao'])
  })

  test('file-kind list does not depend on whether the profile was a builtin shortcut', () => {
    const seeded = {
      sourceSurface: 'membership' as const,
      demandKind: 'membership_entitlement',
    }
    const custom = { sourceSurface: 'membership' as const, demandKind: '' }
    expect(expectedFileKinds(seeded)).toEqual(expectedFileKinds(custom))
  })
})

describe('file-kind readiness', () => {
  test('source platforms list both demand kinds plus carrier and tracking', () => {
    expect(expectedFileKinds({ sourceSurface: 'membership', demandKind: '' })).toEqual([
      'import_entitlement',
      'import_sales_order',
      'import_carrier_mapping',
      'export_source_tracking_update',
    ])
    expect(expectedFileKinds({
      sourceSurface: 'membership',
      demandKind: 'membership_entitlement',
    })).toEqual(expectedFileKinds({ sourceSurface: 'retail', demandKind: 'retail_order' }))
  })

  test('factory platforms list kinds implied by capabilities', () => {
    expect(expectedFileKinds({
      sourceSurface: 'factory',
      demandKind: '',
      supportsImportProductCatalog: true,
      supportsImportSupplierShipment: false,
      supportsExportSupplierOrder: true,
    })).toEqual(['import_product_catalog', 'export_supplier_order'])
  })

  test('default binding marks a file kind ready', () => {
    const bindings = [
      { documentType: 'import_entitlement', isDefault: false },
      { documentType: 'import_sales_order', isDefault: true },
    ]
    expect(hasDefaultFileKindBinding(bindings, 'import_entitlement')).toBe(false)
    expect(hasDefaultFileKindBinding(bindings, 'import_sales_order')).toBe(true)
  })
})
