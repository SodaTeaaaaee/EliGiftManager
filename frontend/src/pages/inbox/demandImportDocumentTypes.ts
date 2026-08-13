/**
 * Demand CSV import document types the inbox file flow may send.
 * Operator-selected on the import modal; never inferred from
 * IntegrationProfile.demandKind (leftover DemandKind is not authoritative).
 */
export const DEMAND_IMPORT_DOCUMENT_TYPES = [
  'import_entitlement',
  'import_sales_order',
] as const

export type DemandImportDocumentType = (typeof DEMAND_IMPORT_DOCUMENT_TYPES)[number]
