/**
 * FieldMappingEditor kit family — shared type contract (P4 demand-intake
 * wizard's CSV column-mapping step; reusable later for e.g. a P5
 * shipment-CSV mapping UI).
 */

/** One destination field the operator can bind a source CSV column to. */
export interface FieldMappingDestField {
  /** Stable identifier — for the demand-intake wizard, one of the 12 canonical
   *  snake_case destFields from `@/shared/lib/demand-intake/platform-presets`
   *  (`IntakeDestField`). Kept as plain `string` here so this kit stays
   *  domain-agnostic and reusable outside demand-intake. */
  key: string
  /** Already-resolved display label (call `t(...)` before building the list). */
  label: string
  /** Already-resolved tooltip copy, shown next to the label. */
  tooltip?: string
}

/**
 * The v-model shape — mirrors the backend's `TemplateMappingRules` JSON
 * shape byte-for-byte (`internal/app/template_mapping_service.go`):
 * `columns` maps a destField key to the source CSV header bound to it;
 * `defaults` maps a destField key to a fixed literal value applied to every
 * row (used for fields with no natural per-row column, e.g. a constant
 * `line_type`). A destField present in BOTH maps resolves to the `defaults`
 * value — the backend applies column mappings first, then defaults
 * afterward, so defaults win. Only destFields explicitly present in one of
 * these maps get submitted; a destField absent from both is simply never
 * set (not an empty string).
 */
export interface FieldMappingValue {
  columns: Record<string, string>
  defaults: Record<string, string>
}
