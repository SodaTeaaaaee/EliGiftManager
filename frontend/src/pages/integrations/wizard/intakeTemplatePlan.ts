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
  // A factory profile may be created without a template. finish() deliberately
  // skips unconfigured templates instead of binding empty mapping rules.
  if (input.isFactorySurface && !hasFile) return true
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

function allowedDestPrefixes(documentType: string): readonly string[] {
  switch (documentType) {
    case 'import_product_catalog': return ['product.']
    case 'import_supplier_shipment': return ['shipment.']
    case 'export_supplier_order': return ['export.']
    case 'import_entitlement':
    case 'import_sales_order': return ['line.', 'document.', 'recipient.']
    case 'import_carrier_mapping': return ['carrier.']
    case 'export_source_tracking_update': return ['tracking.', 'export.']
    default: return []
  }
}

function destAllowed(dest: string, documentType: string): boolean {
  const key = dest.trim()
  if (!key) return false
  if (allowedDestPrefixes(documentType).some((prefix) => key.startsWith(prefix))) return true
  return (documentType === 'import_entitlement' || documentType === 'import_sales_order') && !key.includes('.')
}

function filterRecord<T>(record: Record<string, T> | undefined, keep: (dest: string) => boolean): Record<string, T> {
  return Object.fromEntries(Object.entries(record ?? {}).filter(([key]) => keep(key)))
}

export function mappingForDocumentType(mapping: FieldMappingValue, documentType: string): FieldMappingValue | null {
  const keep = (dest: string) => destAllowed(dest, documentType)
  const columns = filterRecord(mapping.columns, keep)
  const positions = filterRecord(mapping.positions, keep)
  const hasSource = mapping.mode === 'positional'
    ? Object.keys(positions).length > 0
    : Object.keys(columns).length > 0
  if (!hasSource) return null

  return {
    version: mapping.version ?? 2,
    mode: mapping.mode,
    hasHeader: mapping.hasHeader,
    columns,
    positions,
    defaults: filterRecord(mapping.defaults, keep),
    transforms: mapping.transforms ? filterRecord(mapping.transforms, keep) : undefined,
    columnOrder: (mapping.columnOrder ?? []).filter(keep),
    required: mapping.required?.filter(keep),
    imageLayout: documentType === 'import_product_catalog' ? mapping.imageLayout : undefined,
    sheetName: mapping.sheetName,
  }
}

export function formatSupportsDocumentType(format: string, documentType: string): boolean {
  if (documentType === 'export_supplier_order') return format === 'csv' || format === 'xlsx'
  if (documentType === 'import_product_catalog') return ['csv', 'xls', 'xlsx', 'zip'].includes(format)
  return ['csv', 'xls', 'xlsx'].includes(format)
}

/**
 * Produces only executable template bindings. A catalog ZIP mapping therefore
 * creates a catalog template but is never copied into shipment/order bindings.
 */
export function buildIntakeTemplatePlan(
  mapping: FieldMappingValue,
  documentTypes: string[],
  filePath: string,
): IntakeTemplatePlanItem[] {
  const format = intakeFormatFromPath(filePath)
  return documentTypes.flatMap((documentType) => {
    if (!formatSupportsDocumentType(format, documentType)) return []
    const projected = mappingForDocumentType(mapping, documentType)
    if (!projected) return []
    return [{ documentType, format, mappingRules: serializeMappingRules(projected) }]
  })
}
