import { describe, expect, test } from 'vitest'
import { parseMappingRules, type FieldMappingValue } from '@/shared/ui/field-mapping'
import {
  buildIntakeTemplatePlan,
  canProceedFromSample,
} from './intakeTemplatePlan'

const catalogHeaderMapping: FieldMappingValue = {
  version: 2,
  mode: 'header',
  hasHeader: true,
  columns: { 'product.name': '手工商品名', 'product.factory_sku': '手工货号' },
  defaults: {},
}

describe('intake sample and template planning', () => {
  test('ZIP header mode proceeds with manually-entered source headers and persists them', () => {
    expect(canProceedFromSample({
      isFactorySurface: true,
      filePath: 'catalog.zip',
      detectedHeaders: [],
      mapping: catalogHeaderMapping,
    })).toBe(true)

    const [plan] = buildIntakeTemplatePlan(catalogHeaderMapping, 'import_product_catalog', 'catalog.zip')
    expect(plan.documentType).toBe('import_product_catalog')
    expect(parseMappingRules(plan.mappingRules).columns).toEqual(catalogHeaderMapping.columns)
  })

  test('does not bind an empty mapping template', () => {
    expect(buildIntakeTemplatePlan({
      version: 2,
      mode: 'header',
      hasHeader: true,
      columns: {},
      defaults: {},
    }, 'import_product_catalog', 'catalog.zip')).toEqual([])
  })

  test('does not plan a shipment template from a catalog ZIP format', () => {
    expect(buildIntakeTemplatePlan(
      catalogHeaderMapping,
      'import_supplier_shipment',
      'catalog.zip',
    )).toEqual([])
  })

  test('plans exactly one document type without dest-prefix fan-out', () => {
    const mixed: FieldMappingValue = {
      ...catalogHeaderMapping,
      columns: {
        'product.name': '商品名',
        'shipment.tracking_no': '物流单号',
        'export.factory_sku': '货号',
      },
    }
    const plan = buildIntakeTemplatePlan(mixed, 'import_product_catalog', 'catalog.csv')
    expect(plan).toHaveLength(1)
    expect(plan[0].documentType).toBe('import_product_catalog')
    expect(parseMappingRules(plan[0].mappingRules).columns).toEqual(mixed.columns)
  })

  test('factory without a file cannot proceed', () => {
    expect(canProceedFromSample({
      isFactorySurface: true,
      filePath: '',
      detectedHeaders: [],
      mapping: catalogHeaderMapping,
    })).toBe(false)
  })
})
