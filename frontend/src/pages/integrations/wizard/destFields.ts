/**
 * Destination field catalogs for MappingRules v2.
 *
 * Backend namespaces (see `internal/app/template_mapping_service.go` +
 * import mappers): document / line / recipient / product / shipment /
 * tracking / carrier / export. Unprefixed keys remain the semantic "line."
 * default for demand-line imports (v1 compat).
 *
 * Kept local to the integrations page tree for the intake wizard; other
 * import UIs can import the shared catalogs from here or re-list their
 * domain subset.
 */
import type { IntakeDestField } from '@/shared/lib/demand-intake/platform-presets'

/** v2 dest-key namespaces accepted by TemplateMappingRules. */
export type MappingDestNamespace =
  | 'line'
  | 'document'
  | 'recipient'
  | 'product'
  | 'shipment'
  | 'tracking'
  | 'carrier'
  | 'export'

/** destField -> camelCase leaf under `intakeWizard.fields.*`. */
const LINE_FIELD_I18N_LEAF: Record<IntakeDestField, string> = {
  line_type: 'lineType',
  obligation_trigger_kind: 'obligationTriggerKind',
  entitlement_authority: 'entitlementAuthority',
  recipient_input_state: 'recipientInputState',
  routing_disposition: 'routingDisposition',
  routing_reason_code: 'routingReasonCode',
  eligibility_context_ref: 'eligibilityContextRef',
  entitlement_code: 'entitlementCode',
  gift_level_snapshot: 'giftLevelSnapshot',
  recipient_input_payload: 'recipientInputPayload',
  external_title: 'externalTitle',
  requested_quantity: 'requestedQuantity',
}

/** Fixed display order for the intake mapping editor's line destFields (unprefixed v1 keys). */
export const INTAKE_DEST_FIELD_ORDER: readonly IntakeDestField[] = [
  'line_type',
  'obligation_trigger_kind',
  'entitlement_authority',
  'recipient_input_state',
  'routing_disposition',
  'routing_reason_code',
  'eligibility_context_ref',
  'entitlement_code',
  'gift_level_snapshot',
  'recipient_input_payload',
  'external_title',
  'requested_quantity',
]

/** document.* keys used by demand import (DocumentFieldDraft). */
export const DOCUMENT_DEST_FIELDS = [
  'document.source_customer_ref',
  'document.source_document_no',
  'document.display_name',
] as const

/** recipient.* keys used by demand import (RecipientAddressDraft). */
export const RECIPIENT_DEST_FIELDS = [
  'recipient.name',
  'recipient.phone',
  'recipient.country',
  'recipient.province',
  'recipient.city',
  'recipient.district',
  'recipient.address_line1',
  'recipient.address_line2',
  'recipient.postal_code',
  'recipient.label',
  'recipient.is_default',
] as const

/** product.* keys used by ImportProductCatalog. */
export const PRODUCT_DEST_FIELDS = [
  'product.supplier_platform',
  'product.factory_sku',
  'product.name',
  'product.supplier_product_ref',
  'product.product_kind',
  'product.extra_data',
] as const

/** shipment.* keys used by MapAndReconcileShipments. */
export const SHIPMENT_DEST_FIELDS = [
  'shipment.third_party_order_no',
  'shipment.external_key',
  'shipment.fulfillment_line_id',
  'shipment.factory_sku',
  'shipment.sku',
  'shipment.supplier_product_ref',
  'shipment.sku_quantity',
  'shipment.spec_quantity',
  'shipment.phone',
  'shipment.recipient_phone',
  'shipment.recipient_name',
  'shipment.name',
  'shipment.tracking_no',
  'shipment.external_shipment_no',
  'shipment.carrier_code',
  'shipment.carrier_name',
  'shipment.quantity',
  'shipment.shipped_at',
] as const

/** export.* keys used by source-tracking export templates (columnOrder-aware). */
export const EXPORT_DEST_FIELDS = [
  'export.third_party_order_no',
  'export.tracking_no',
  'export.carrier_code',
  'export.external_document_no',
  'export.shipment_id',
  'export.recipient',
  'export.phone',
  'export.address',
  'export.factory_sku',
  'export.quantity',
] as const

/** line.* prefixed forms of the 12 demand-line fields (v2 preferred / editor form). */
export const LINE_PREFIXED_DEST_FIELDS = INTAKE_DEST_FIELD_ORDER.map(
  (field) => `line.${field}` as const,
)

/**
 * Full intake-wizard dest list: namespaced line.* + document.* + recipient.*
 * so presets, FieldMappingEditor, and StepConfirm share one key form.
 */
export const INTAKE_V2_DEST_FIELD_ORDER: readonly string[] = [
  ...LINE_PREFIXED_DEST_FIELDS,
  ...DOCUMENT_DEST_FIELDS,
  ...RECIPIENT_DEST_FIELDS,
]

/** `t()`-ready leaf paths for unprefixed line fields. */
export function destFieldLabelKey(field: string): string {
  if (field in LINE_FIELD_I18N_LEAF) {
    return `intakeWizard.fields.${LINE_FIELD_I18N_LEAF[field as IntakeDestField]}.label`
  }
  if (field.startsWith('line.')) {
    const bare = field.slice('line.'.length) as IntakeDestField
    if (bare in LINE_FIELD_I18N_LEAF) {
      return `intakeWizard.fields.${LINE_FIELD_I18N_LEAF[bare]}.label`
    }
  }
  // v2 namespaced fields share a flat i18n tree under intakeWizard.destFields.*
  const leaf = field.replace(/\./g, '_')
  return `intakeWizard.destFields.${leaf}.label`
}

export function destFieldTooltipKey(field: string): string {
  if (field in LINE_FIELD_I18N_LEAF) {
    return `intakeWizard.fields.${LINE_FIELD_I18N_LEAF[field as IntakeDestField]}.tooltip`
  }
  if (field.startsWith('line.')) {
    const bare = field.slice('line.'.length) as IntakeDestField
    if (bare in LINE_FIELD_I18N_LEAF) {
      return `intakeWizard.fields.${LINE_FIELD_I18N_LEAF[bare]}.tooltip`
    }
  }
  const leaf = field.replace(/\./g, '_')
  return `intakeWizard.destFields.${leaf}.tooltip`
}

/** Dest catalog for one session document type — never a union of factory caps. */
export function destKeysForDocumentType(documentType: string): readonly string[] {
  switch (documentType) {
    case 'import_product_catalog':
      return PRODUCT_DEST_FIELDS
    case 'import_supplier_shipment':
      return SHIPMENT_DEST_FIELDS
    case 'export_supplier_order':
      return EXPORT_DEST_FIELDS
    case 'import_entitlement':
    case 'import_sales_order':
    default:
      return INTAKE_V2_DEST_FIELD_ORDER
  }
}
