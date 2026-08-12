import { describe, expect, test } from 'vitest'
import { parseMappingRules, type FieldMappingValue } from '@/shared/ui/field-mapping'
import {
  buildIntakeTemplatePlan,
  canProceedFromSample,
  mappingForDocumentType,
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

    const [plan] = buildIntakeTemplatePlan(catalogHeaderMapping, ['import_product_catalog'], 'catalog.zip')
    expect(parseMappingRules(plan.mappingRules).columns).toEqual(catalogHeaderMapping.columns)
  })

  test('does not bind an empty mapping template', () => {
    expect(buildIntakeTemplatePlan({
      version: 2,
      mode: 'header',
      hasHeader: true,
      columns: {},
      defaults: {},
    }, ['import_product_catalog'], 'catalog.zip')).toEqual([])
  })

  test('does not copy a catalog ZIP format or product mapping to supplier-order templates', () => {
    const plan = buildIntakeTemplatePlan(
      catalogHeaderMapping,
      ['import_product_catalog', 'import_supplier_shipment', 'export_supplier_order'],
      'catalog.zip',
    )
    expect(plan.map((item) => [item.documentType, item.format])).toEqual([
      ['import_product_catalog', 'zip'],
    ])
  })

  test('projects each document namespace independently', () => {
    expect(mappingForDocumentType({
      ...catalogHeaderMapping,
      columns: {
        'product.name': '商品名',
        'shipment.tracking_no': '物流单号',
        'export.factory_sku': '货号',
      },
    }, 'import_supplier_shipment')?.columns).toEqual({ 'shipment.tracking_no': '物流单号' })
  })
})
