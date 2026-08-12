import { describe, expect, test } from 'vitest'
import {
  canImportDemand,
  canImportProductCatalog,
  canImportSupplierShipment,
  canCreateRetailDemand,
} from './profileAvailability'

describe('specialized profile availability', () => {
  const demand = { sourceSurface: 'retail', demandKind: 'retail_order' }
  const factory = {
    sourceSurface: 'factory',
    demandKind: '',
    supportsImportProductCatalog: true,
    supportsImportSupplierShipment: false,
  }

  test('demand import excludes factory profiles', () => {
    expect(canImportDemand(demand)).toBe(true)
    expect(canImportDemand(factory)).toBe(false)
  })

  test('manual retail entry excludes membership profiles', () => {
    expect(canCreateRetailDemand(demand)).toBe(true)
    expect(canCreateRetailDemand({ sourceSurface: 'membership', demandKind: 'membership_entitlement' })).toBe(false)
  })

  test('factory import selectors require the matching capability', () => {
    expect(canImportProductCatalog(factory)).toBe(true)
    expect(canImportSupplierShipment(factory)).toBe(false)
  })
})
