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
  const community = { sourceSurface: 'community', demandKind: 'retail_order' }

  test('demand import excludes factory and community profiles', () => {
    expect(canImportDemand(demand)).toBe(true)
    expect(canImportDemand({ sourceSurface: 'retail', demandKind: '' })).toBe(true)
    expect(canImportDemand(factory)).toBe(false)
    expect(canImportDemand(community)).toBe(false)
    expect(canImportDemand(dualKind)).toBe(true)
  })

  test('demand import keeps leftover membership source platforms', () => {
    expect(canImportDemand(membershipLeftover)).toBe(true)
  })

  test('manual retail entry keeps dual-kind and leftover membership platforms', () => {
    expect(canCreateRetailDemand(demand)).toBe(true)
    expect(canCreateRetailDemand({ sourceSurface: 'retail', demandKind: '' })).toBe(true)
    expect(canCreateRetailDemand(dualKind)).toBe(true)
    expect(canCreateRetailDemand(membershipLeftover)).toBe(true)
    expect(canCreateRetailDemand(factory)).toBe(false)
    expect(canCreateRetailDemand(community)).toBe(false)
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
    const community = { sourceSurface: 'community', demandKind: 'retail_order' }
    const unknown = { sourceSurface: '', demandKind: '' }

    expect(isSourcePlatformProfile(dualKind)).toBe(true)
    expect(isSourcePlatformProfile(membershipLeftover)).toBe(true)
    expect(isSourcePlatformProfile(retail)).toBe(true)
    expect(isSourcePlatformProfile(factory)).toBe(false)
    expect(isSourcePlatformProfile(community)).toBe(false)
    expect(isSourcePlatformProfile(unknown)).toBe(false)
    expect(isFactoryProfile(factory)).toBe(true)

    const { source, factory: factoryGroup } = partitionProfilesForList([
      dualKind,
      membershipLeftover,
      retail,
      factory,
      community,
      unknown,
    ])
    expect(source).toEqual([dualKind, membershipLeftover, retail])
    expect(factoryGroup).toEqual([factory])
  })

  test('omits community and other non-membership/non-retail/non-factory surfaces from source', () => {
    const community = { sourceSurface: 'community', demandKind: 'retail_order' }
    const unknown = { sourceSurface: 'other', demandKind: '' }
    const empty = { sourceSurface: '', demandKind: '' }
    const membership = { sourceSurface: 'membership', demandKind: '' }
    const factory = { sourceSurface: 'factory', demandKind: '' }

    const { source, factory: factoryGroup } = partitionProfilesForList([
      community,
      unknown,
      empty,
      membership,
      factory,
    ])
    expect(source).toEqual([membership])
    expect(factoryGroup).toEqual([factory])
    expect(source).not.toContain(community)
    expect(source).not.toContain(unknown)
    expect(source).not.toContain(empty)
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
    expect(expectedFileKinds(seeded)).toEqual(['import_entitlement', 'import_sales_order'])
    expect(expectedFileKinds(custom)).toEqual(expectedFileKinds(seeded))
  })
})

describe('file-kind readiness', () => {
  test('source platforms list demand kinds unless carrier or tracking flags are set', () => {
    expect(expectedFileKinds({ sourceSurface: 'membership', demandKind: '' })).toEqual([
      'import_entitlement',
      'import_sales_order',
    ])
    expect(expectedFileKinds({
      sourceSurface: 'membership',
      demandKind: 'membership_entitlement',
    })).toEqual(expectedFileKinds({ sourceSurface: 'retail', demandKind: 'retail_order' }))
  })

  test('community is not a source platform and has no expected file kinds', () => {
    const community = { sourceSurface: 'community', demandKind: 'retail_order' }
    expect(isSourcePlatformProfile(community)).toBe(false)
    expect(expectedFileKinds(community)).toEqual([])
  })

  test('import_carrier_mapping is listed only when requiresCarrierMapping is true', () => {
    expect(expectedFileKinds({
      sourceSurface: 'membership',
      demandKind: '',
      requiresCarrierMapping: true,
    })).toEqual([
      'import_entitlement',
      'import_sales_order',
      'import_carrier_mapping',
    ])
    expect(expectedFileKinds({ sourceSurface: 'membership', demandKind: '' })).not.toContain(
      'import_carrier_mapping',
    )
  })

  test('export_source_tracking_update is listed only for document_export tracking', () => {
    expect(expectedFileKinds({
      sourceSurface: 'retail',
      demandKind: 'retail_order',
      trackingSyncMode: 'document_export',
    })).toEqual([
      'import_entitlement',
      'import_sales_order',
      'export_source_tracking_update',
    ])
    for (const trackingSyncMode of ['unsupported', 'api_push', 'manual_confirmation']) {
      expect(expectedFileKinds({
        sourceSurface: 'retail',
        demandKind: 'retail_order',
        trackingSyncMode,
      })).toEqual(['import_entitlement', 'import_sales_order'])
    }
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

  test('factory with no capabilities listed has an empty file-kind list', () => {
    expect(expectedFileKinds({ sourceSurface: 'factory', demandKind: '' })).toEqual([])
    expect(expectedFileKinds({
      sourceSurface: 'factory',
      demandKind: '',
      supportsImportProductCatalog: false,
      supportsImportSupplierShipment: false,
      supportsExportSupplierOrder: false,
    })).toEqual([])
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
