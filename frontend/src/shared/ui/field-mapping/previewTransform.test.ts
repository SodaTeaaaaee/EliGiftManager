import { describe, expect, test } from 'vitest'
import { documentTypeForDemandKind } from '@/pages/integrations/wizard/deriveProfileDefaults'
import { applyMapping, applyPreviewTransforms } from './previewTransform'
import { parseMappingRules, serializeMappingRules, type FieldMappingValue } from './types'

describe('profile document type derivation', () => {
  test('retail demand uses sales-order import instead of entitlement fallback', () => {
    expect(documentTypeForDemandKind('retail_order')).toBe('import_sales_order')
    expect(documentTypeForDemandKind('membership_entitlement')).toBe('import_entitlement')
  })
})

describe('mapping preview contract', () => {
  test('applies backend-compatible transforms in order', () => {
    expect(applyPreviewTransforms("  '00123'  ", ['trim', 'strip_quotes'])).toBe('00123')
    expect(applyPreviewTransforms("'00999", ['strip_leading_quote'])).toBe('00999')
  })

  test('preview applies transforms to source values and defaults', () => {
    const mapping: FieldMappingValue = {
      version: 2,
      mode: 'header',
      hasHeader: true,
      columns: { 'shipment.tracking_no': '物流单号' },
      defaults: { 'shipment.external_shipment_no': '  demo  ' },
      transforms: {
        'shipment.tracking_no': ['strip_leading_quote'],
        'shipment.external_shipment_no': ['trim'],
      },
    }
    const [row] = applyMapping([{ 物流单号: "'123456789012345678" }], mapping)
    expect(row.values['shipment.tracking_no']).toBe('123456789012345678')
    expect(row.values['shipment.external_shipment_no']).toBe('demo')
  })

  test('required and transforms survive template JSON round-trip', () => {
    const raw = serializeMappingRules({
      version: 2,
      mode: 'header',
      hasHeader: true,
      columns: { tracking_no: '物流单号*' },
      defaults: {},
      transforms: { tracking_no: ['trim'] },
      required: ['tracking_no'],
      sheetName: '发货信息',
    })
    const parsed = parseMappingRules(raw)
    expect(parsed.transforms?.tracking_no).toEqual(['trim'])
    expect(parsed.required).toEqual(['tracking_no'])
    expect(parsed.sheetName).toBe('发货信息')
  })

  test('catalog image layout survives template JSON round-trip', () => {
    const parsed = parseMappingRules(serializeMappingRules({
      version: 2,
      mode: 'header',
      hasHeader: true,
      columns: { 'product.name': '商品名称' },
      defaults: {},
      imageLayout: {
        enabled: true,
        matchField: 'product.name',
        coverDir: '主图',
        detailDir: '详情图',
        namePattern: '{match}#{nn}',
        tabularGlob: '*.csv',
        imageExts: ['png', 'jpg', 'jpeg'],
      },
    }))
    expect(parsed.imageLayout?.coverDir).toBe('主图')
    expect(parsed.imageLayout?.imageExts).toEqual(['png', 'jpg', 'jpeg'])
  })
})
