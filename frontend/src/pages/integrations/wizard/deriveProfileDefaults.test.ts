import { describe, expect, test } from 'vitest'
import {
  buildCustomCreateProfileInput,
  suggestProfileKey,
} from './deriveProfileDefaults'

describe('custom create defaults', () => {
  test('source uses empty demandKind so both file kinds can bind', () => {
    const input = buildCustomCreateProfileInput({
      displayName: 'Workshop',
      profileKey: 'workshop_1',
      surface: 'source',
      factorySupplierPlatform: '',
    })
    expect(input.sourceSurface).toBe('membership')
    expect(input.demandKind).toBe('')
    expect(input.sourceChannel).toBe('Workshop')
    expect(input.connectorKey).toBe('eli.local_export')
    expect(input.trackingSyncMode).toBe('document_export')
    expect(input.supportsImportProductCatalog).toBe(false)
  })

  test('factory requires supplier label and turns catalog/shipment/export on', () => {
    const input = buildCustomCreateProfileInput({
      displayName: '柔造车间',
      profileKey: 'custom_factory',
      surface: 'factory',
      factorySupplierPlatform: 'rouzao-shop',
    })
    expect(input.sourceSurface).toBe('factory')
    expect(input.demandKind).toBe('')
    expect(input.factorySupplierPlatform).toBe('rouzao-shop')
    expect(input.supportsImportProductCatalog).toBe(true)
    expect(input.supportsImportSupplierShipment).toBe(true)
    expect(input.supportsExportSupplierOrder).toBe(true)
    expect(input.trackingSyncMode).toBe('unsupported')
  })

  test('suggestProfileKey slugs ascii and leaves non-latin empty', () => {
    expect(suggestProfileKey('My Shop 2026')).toBe('my_shop_2026')
    expect(suggestProfileKey('柔造')).toBe('')
  })
})
