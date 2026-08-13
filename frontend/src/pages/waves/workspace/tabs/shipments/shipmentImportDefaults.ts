/**
 * Factory-return shipment import always uses this document type.
 * `mapAndReconcileShipments` resolves the profile's default binding for it;
 * this entry never sends mappingRules.
 */
import { getDefaultTemplateForProfile } from '@/shared/api/bridge'

export const SUPPLIER_SHIPMENT_IMPORT_DOCUMENT_TYPE = 'import_supplier_shipment' as const

export type ShipmentImportBindingState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'loaded'; templateKey: string; hasHeader: boolean }
  | { status: 'missing' }
  | { status: 'error'; message: string }

/**
 * Read `hasHeader` from a DocumentTemplate `mappingRules` JSON string.
 * True unless the parsed object has `hasHeader === false`. Never throws.
 */
export function hasHeaderFromMappingRules(raw: string | undefined | null): boolean {
  if (raw == null || !String(raw).trim()) return true
  try {
    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed !== 'object' || parsed == null) return true
    return (parsed as { hasHeader?: unknown }).hasHeader !== false
  } catch {
    return true
  }
}

export async function loadShipmentImportDefaultBinding(
  profileId: number,
): Promise<Extract<ShipmentImportBindingState, { status: 'loaded' | 'missing' | 'error' }>> {
  try {
    const tmpl = await getDefaultTemplateForProfile(profileId, SUPPLIER_SHIPMENT_IMPORT_DOCUMENT_TYPE)
    if (tmpl?.templateKey) {
      return {
        status: 'loaded',
        templateKey: tmpl.templateKey,
        hasHeader: hasHeaderFromMappingRules(tmpl.mappingRules),
      }
    }
    return { status: 'missing' }
  } catch (err) {
    return { status: 'error', message: err instanceof Error ? err.message : String(err) }
  }
}
