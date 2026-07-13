/**
 * The 12 canonical `IntakeDestField` values, in a fixed display order, plus
 * their i18n leaf under `intakeWizard.fields.*`. Kept local to the
 * integrations page tree (not `shared/lib/demand-intake/`) since it is only
 * ever needed by the intake wizard's mapping step — the FieldMappingEditor
 * kit itself stays domain-agnostic (see `shared/ui/field-mapping/types.ts`).
 */
import type { IntakeDestField } from '@/shared/lib/demand-intake/platform-presets'

/** destField -> the camelCase leaf name under `intakeWizard.fields.*`. */
const DEST_FIELD_I18N_LEAF: Record<IntakeDestField, string> = {
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

/** Fixed display order for the mapping editor's destField list. */
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

/** `t()`-ready leaf paths: `intakeWizard.fields.<camelCase>.label` / `.tooltip`. */
export function destFieldLabelKey(field: IntakeDestField): string {
  return `intakeWizard.fields.${DEST_FIELD_I18N_LEAF[field]}.label`
}

export function destFieldTooltipKey(field: IntakeDestField): string {
  return `intakeWizard.fields.${DEST_FIELD_I18N_LEAF[field]}.tooltip`
}
