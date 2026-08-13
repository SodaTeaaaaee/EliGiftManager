import { describe, expect, test } from 'vitest'
import {
  CARRIER_DEST_FIELDS,
  EXPORT_DEST_FIELDS,
  INTAKE_V2_DEST_FIELD_ORDER,
  PRODUCT_DEST_FIELDS,
  SHIPMENT_DEST_FIELDS,
  TRACKING_DEST_FIELDS,
  destKeysForDocumentType,
} from './destFields'

const ADDED_EXPORT_DEST_FIELDS = [
  'export.recipient_name',
  'export.supplier_sku',
  'export.supplier_line_no',
  'export.product_id',
  'export.fulfillment_line_id',
] as const

describe('destKeysForDocumentType', () => {
  test('import_carrier_mapping uses carrier dests only', () => {
    const keys = destKeysForDocumentType('import_carrier_mapping')
    expect(keys).toEqual(CARRIER_DEST_FIELDS)
    expect(keys.some((key) => key.startsWith('line.'))).toBe(false)
    expect(keys.some((key) => key.startsWith('document.'))).toBe(false)
    expect(keys.some((key) => key.startsWith('recipient.'))).toBe(false)
  })

  test('export_source_tracking_update concatenates tracking then export dests', () => {
    const keys = destKeysForDocumentType('export_source_tracking_update')
    for (const key of TRACKING_DEST_FIELDS) {
      expect(keys).toContain(key)
    }
    for (const key of EXPORT_DEST_FIELDS) {
      expect(keys).toContain(key)
    }
    expect(keys.slice(0, TRACKING_DEST_FIELDS.length)).toEqual([...TRACKING_DEST_FIELDS])
    expect(keys.slice(TRACKING_DEST_FIELDS.length)).toEqual([...EXPORT_DEST_FIELDS])
  })

  test('export_supplier_order includes the added export dest keys', () => {
    const keys = destKeysForDocumentType('export_supplier_order')
    for (const key of ADDED_EXPORT_DEST_FIELDS) {
      expect(keys).toContain(key)
    }
  })

  test('import_entitlement and import_sales_order keep the intake v2 catalog', () => {
    expect(destKeysForDocumentType('import_entitlement')).toEqual(INTAKE_V2_DEST_FIELD_ORDER)
    expect(destKeysForDocumentType('import_sales_order')).toEqual(INTAKE_V2_DEST_FIELD_ORDER)
  })

  test('import_product_catalog and import_supplier_shipment catalogs are unchanged', () => {
    expect(destKeysForDocumentType('import_product_catalog')).toEqual(PRODUCT_DEST_FIELDS)
    expect(destKeysForDocumentType('import_supplier_shipment')).toEqual(SHIPMENT_DEST_FIELDS)
  })
})

describe('EXPORT_DEST_FIELDS', () => {
  test('matches the backend exportDests list', () => {
    expect([...EXPORT_DEST_FIELDS]).toEqual([
      'export.third_party_order_no',
      'export.tracking_no',
      'export.carrier_code',
      'export.external_document_no',
      'export.shipment_id',
      'export.recipient',
      'export.recipient_name',
      'export.phone',
      'export.address',
      'export.factory_sku',
      'export.supplier_sku',
      'export.quantity',
      'export.supplier_line_no',
      'export.product_id',
      'export.fulfillment_line_id',
    ])
    for (const key of ADDED_EXPORT_DEST_FIELDS) {
      expect(EXPORT_DEST_FIELDS).toContain(key)
    }
  })
})
