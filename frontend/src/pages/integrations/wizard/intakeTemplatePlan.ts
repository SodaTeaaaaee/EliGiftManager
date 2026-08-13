import { serializeMappingRules, type FieldMappingValue } from '@/shared/ui/field-mapping'

export interface IntakeTemplatePlanItem {
  documentType: string
  format: string
  mappingRules: string
}

export function mappingIsConfigured(mapping: FieldMappingValue): boolean {
  if (mapping.mode === 'positional') {
    return Object.keys(mapping.positions ?? {}).length > 0 || Object.keys(mapping.defaults).length > 0
  }
  return Object.keys(mapping.columns).length > 0 || Object.keys(mapping.defaults).length > 0
}

export function canProceedFromSample(input: {
  isFactorySurface: boolean
  filePath: string
  detectedHeaders: string[]
  mapping: FieldMappingValue
}): boolean {
  const hasFile = input.filePath.trim().length > 0 || input.detectedHeaders.length > 0
  if (!hasFile) return false
  return mappingIsConfigured(input.mapping) || input.detectedHeaders.length > 0
}

export function intakeFormatFromPath(filePath: string): string {
  const lower = filePath.toLowerCase()
  if (lower.endsWith('.zip')) return 'zip'
  if (lower.endsWith('.xlsx')) return 'xlsx'
  if (lower.endsWith('.xls')) return 'xls'
  return 'csv'
}

export function formatSupportsDocumentType(format: string, documentType: string): boolean {
  if (documentType === 'export_supplier_order') return format === 'csv' || format === 'xlsx'
  if (documentType === 'import_product_catalog') return ['csv', 'xls', 'xlsx', 'zip'].includes(format)
  return ['csv', 'xls', 'xlsx'].includes(format)
}

/**
 * Plans at most one template for the session document type. Does not project
 * dest prefixes into extra templates — one sample maps to one file kind.
 */
export function buildIntakeTemplatePlan(
  mapping: FieldMappingValue,
  documentType: string,
  filePath: string,
): IntakeTemplatePlanItem[] {
  const format = intakeFormatFromPath(filePath)
  if (!formatSupportsDocumentType(format, documentType)) return []
  if (!mappingIsConfigured(mapping)) return []
  return [{
    documentType,
    format,
    mappingRules: serializeMappingRules(mapping),
  }]
}
