/**
 * FieldMappingEditor kit family — shared type contract (P4 demand-intake
 * wizard's CSV column-mapping step; reusable for shipment / product /
 * carrier template-mapped imports).
 */

/** One destination field the operator can bind a source column/position to. */
export interface FieldMappingDestField {
  /** Stable identifier — dest key as stored in TemplateMappingRules
   *  (unprefixed line fields, or v2 `ns.field` keys). Kept as plain `string`
   *  so this kit stays domain-agnostic. */
  key: string
  /** Already-resolved display label (call `t(...)` before building the list). */
  label: string
  /** Already-resolved tooltip copy, shown next to the label. */
  tooltip?: string
}

/** Mapping mode stored in TemplateMappingRules.mode. */
export type FieldMappingMode = 'header' | 'positional'

/**
 * The v-model shape — mirrors the backend's `TemplateMappingRules` JSON
 * (`internal/app/template_mapping_service.go`):
 *
 * v1 (legacy): `{ columns, defaults }` — treated as version=1, mode=header,
 * hasHeader=true by the backend.
 *
 * v2: adds version/mode/hasHeader/positions/transforms/columnOrder/required.
 * Unprefixed dest keys remain the semantic "line." namespace for demand imports.
 *
 * Resolution order matches the backend: source mapping first (columns or
 * positions), then defaults overwrite. A destField absent from both maps is
 * never set.
 */
export interface FieldMappingValue {
  /** Mapping rules version. Prefer 2 for new templates. */
  version?: number
  /** header = bind by CSV header name; positional = bind by 0-based cell index. */
  mode?: FieldMappingMode
  /** Whether the source sheet's first row is a header row. */
  hasHeader?: boolean
  /** destField → source header name (mode=header). */
  columns: Record<string, string>
  /** destField → 0-based cell index (mode=positional). */
  positions?: Record<string, number>
  /** destField → fixed literal applied to every row (wins over source). */
  defaults: Record<string, string>
  /** destField → ordered transform names (trim, strip_quotes, …). */
  transforms?: Record<string, string[]>
  /** Preferred output column order for export templates. */
  columnOrder?: string[]
  /** Dest keys that must be non-empty after mapping. */
  required?: string[]
}

/**
 * Bare demand-line dest keys (semantic default namespace "line.").
 * Kept in sync with backend `lineDestBare` / IntakeDestField.
 */
const LINE_DEST_BARE = new Set([
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
])

/**
 * Ensure a dest key is stored in the v2 namespaced form.
 * Bare line fields become `line.<field>`; already-namespaced keys are kept.
 */
export function ensureNamespacedDestKey(key: string): string {
  const trimmed = key.trim()
  if (!trimmed || trimmed.includes('.')) return trimmed
  if (LINE_DEST_BARE.has(trimmed)) return `line.${trimmed}`
  return trimmed
}

function mapRecordKeys<T>(record: Record<string, T> | undefined): Record<string, T> {
  const out: Record<string, T> = {}
  for (const [k, v] of Object.entries(record ?? {})) {
    out[ensureNamespacedDestKey(k)] = v
  }
  return out
}

/** Build a v2-ready empty mapping value. */
export function emptyFieldMapping(mode: FieldMappingMode = 'header'): FieldMappingValue {
  return {
    version: 2,
    mode,
    hasHeader: true,
    columns: {},
    positions: {},
    defaults: {},
    columnOrder: [],
  }
}

/**
 * Parse a DocumentTemplate.MappingRules JSON string into a FieldMappingValue.
 * Tolerates missing/legacy shapes; never throws — returns empty mapping on failure.
 */
export function parseMappingRules(raw: string | undefined | null): FieldMappingValue {
  if (!raw || !raw.trim()) return emptyFieldMapping()
  try {
    const parsed = JSON.parse(raw) as Partial<FieldMappingValue> & {
      columns?: Record<string, string>
      defaults?: Record<string, string>
      positions?: Record<string, number>
    }
    const mode: FieldMappingMode = parsed.mode === 'positional' ? 'positional' : 'header'
    return {
      version: typeof parsed.version === 'number' ? parsed.version : 2,
      mode,
      hasHeader: parsed.hasHeader !== false,
      columns: mapRecordKeys(parsed.columns ?? {}),
      positions: mapRecordKeys(parsed.positions ?? {}),
      defaults: mapRecordKeys(parsed.defaults ?? {}),
      transforms: parsed.transforms ? mapRecordKeys(parsed.transforms) : undefined,
      columnOrder: Array.isArray(parsed.columnOrder)
        ? parsed.columnOrder.map(ensureNamespacedDestKey)
        : [],
      required: Array.isArray(parsed.required)
        ? parsed.required.map(ensureNamespacedDestKey)
        : undefined,
    }
  } catch {
    return emptyFieldMapping()
  }
}

/**
 * Serialize a FieldMappingValue to the JSON string stored in
 * DocumentTemplate.MappingRules. Always emits version=2 so new templates
 * get hasHeader/mode/positions support.
 *
 * Boundary: bare line dest keys are rewritten to `line.*` so presets, the
 * editor, and the stored MappingRules share one namespaced form.
 */
export function serializeMappingRules(mapping: FieldMappingValue): string {
  const mode: FieldMappingMode = mapping.mode === 'positional' ? 'positional' : 'header'
  const rules: Record<string, unknown> = {
    version: mapping.version ?? 2,
    mode,
    hasHeader: mapping.hasHeader ?? true,
    defaults: mapRecordKeys(mapping.defaults ?? {}),
  }
  if (mode === 'positional') {
    rules.positions = mapRecordKeys(mapping.positions ?? {})
  } else {
    rules.columns = mapRecordKeys(mapping.columns ?? {})
  }
  if (mapping.transforms && Object.keys(mapping.transforms).length > 0) {
    rules.transforms = mapRecordKeys(mapping.transforms)
  }
  if (mapping.columnOrder && mapping.columnOrder.length > 0) {
    rules.columnOrder = mapping.columnOrder.map(ensureNamespacedDestKey)
  }
  if (mapping.required && mapping.required.length > 0) {
    rules.required = mapping.required.map(ensureNamespacedDestKey)
  }
  return JSON.stringify(rules)
}
