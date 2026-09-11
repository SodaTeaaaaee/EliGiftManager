/**
 * FieldMappingEditor kit — shared type contract mirroring the backend's
 * MappingConfig v3 (`internal/app/alignment/schema.go`), stored in
 * TemplateConfig.MappingJSON. Semantic keys come from the backend semantic
 * dictionary; this kit stays domain-agnostic.
 */

/** One destination field the operator can bind a source column/position to. */
export interface FieldMappingDestField {
  /** Semantic dictionary key (e.g. `quantity`, `shipment.tracking_no`). */
  key: string
  /** Already-resolved display label (call `t(...)` before building the list). */
  label: string
  /** Already-resolved tooltip copy, shown next to the label. */
  tooltip?: string
  /** Key of the `FieldMappingGroup` this field renders under; ungrouped when absent. */
  group?: string
}

/** One collapsible section of destination fields (order = display order). */
export interface FieldMappingGroup {
  key: string
  /** Already-resolved section title. */
  label: string
}

/** Mapping mode stored in MappingConfig.mode. */
export type FieldMappingMode = 'header' | 'positional'

/** Schema version of MappingConfig this kit reads and writes. */
const MAPPING_VERSION = 3

/**
 * The v-model shape — a mirror of MappingConfig v3.
 *
 * Resolution order matches the backend parse loop: source cells first
 * (columns or positions), JoinSources overwrite the plain cell, fixed
 * defaults override both, then transform chains run. A semantic key absent
 * from all three maps is never set.
 */
export interface FieldMappingValue {
  /** Mapping schema version. Always 3 for new templates. */
  version?: number
  /** header = bind by source header name; positional = bind by 0-based column index. */
  mode?: FieldMappingMode
  /** Whether the source sheet's first row is a header row. */
  hasHeader?: boolean
  /** Exact worksheet name for spreadsheet sources. */
  sheetName?: string
  /** semantic key → source header name (mode=header). */
  columns: Record<string, string>
  /** semantic key → 0-based column index (mode=positional). */
  positions?: Record<string, number>
  /** semantic key → fixed literal applied to every row (wins over source). */
  defaults: Record<string, string>
  /** semantic key → ordered transformer names (trim, strip_quotes, …). */
  transforms?: Record<string, string[]>
  /** semantic key → external→internal value table used by mapEnum. */
  enumMaps?: Record<string, Record<string, string>>
  /**
   * semantic key → source column refs merged by joinAddress. Header names in
   * header mode, decimal column indexes as strings in positional mode.
   */
  joinSources?: Record<string, string[]>
  /** Semantic key holding the pipe-concatenated multi-product blob to split. */
  splitSkuQuantity?: string
  /** Semantic keys that must be non-empty after mapping. */
  required?: string[]
  /** Semantic keys whose values form the duplicate-detection fingerprint. */
  fingerprint?: string[]
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function copyStrings(source: unknown): Record<string, string> | undefined {
  if (!isRecord(source)) return undefined
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(source)) {
    if (k.trim() !== '' && typeof v === 'string' && v.trim() !== '') out[k] = v
  }
  return Object.keys(out).length > 0 ? out : undefined
}

function copyNumbers(source: unknown): Record<string, number> | undefined {
  if (!isRecord(source)) return undefined
  const out: Record<string, number> = {}
  for (const [k, v] of Object.entries(source)) {
    if (typeof v === 'number' && Number.isFinite(v) && v >= 0) out[k] = Math.floor(v)
  }
  return Object.keys(out).length > 0 ? out : undefined
}

function copyStringLists(source: unknown): Record<string, string[]> | undefined {
  if (!isRecord(source)) return undefined
  const out: Record<string, string[]> = {}
  for (const [k, v] of Object.entries(source)) {
    if (!Array.isArray(v)) continue
    const items = v.filter((item): item is string => typeof item === 'string' && item.trim() !== '')
    if (items.length > 0) out[k] = items
  }
  return Object.keys(out).length > 0 ? out : undefined
}

function copyEnumMaps(source: unknown): Record<string, Record<string, string>> | undefined {
  if (!isRecord(source)) return undefined
  const out: Record<string, Record<string, string>> = {}
  for (const [k, v] of Object.entries(source)) {
    const table = copyStrings(v)
    if (table) out[k] = table
  }
  return Object.keys(out).length > 0 ? out : undefined
}

function copyKeyList(source: unknown): string[] | undefined {
  if (!Array.isArray(source)) return undefined
  const items = source.filter((item): item is string => typeof item === 'string' && item.trim() !== '')
  return items.length > 0 ? items : undefined
}

/** Build a v3-ready empty mapping value. */
export function emptyFieldMapping(mode: FieldMappingMode = 'header'): FieldMappingValue {
  return {
    version: MAPPING_VERSION,
    mode,
    hasHeader: mode === 'header',
    columns: {},
    positions: {},
    defaults: {},
  }
}

/**
 * Parse a TemplateConfig.MappingJSON string into a FieldMappingValue.
 * Tolerates missing input; invalid JSON or a foreign schema version yields an
 * empty v3 mapping — the backend treats both as hard errors, so the editor
 * starts from a clean slate instead of half-rendering foreign data.
 */
export function parseMappingRules(raw: string | undefined | null): FieldMappingValue {
  if (!raw || !raw.trim()) return emptyFieldMapping()
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    if (!isRecord(parsed) || parsed.version !== MAPPING_VERSION) return emptyFieldMapping()
    const mode: FieldMappingMode = parsed.mode === 'positional' ? 'positional' : 'header'
    return {
      version: MAPPING_VERSION,
      mode,
      hasHeader: mode === 'header',
      sheetName: typeof parsed.sheetName === 'string' && parsed.sheetName.trim() !== ''
        ? parsed.sheetName
        : undefined,
      columns: copyStrings(parsed.columns) ?? {},
      positions: copyNumbers(parsed.positions),
      defaults: copyStrings(parsed.defaults) ?? {},
      transforms: copyStringLists(parsed.transforms),
      enumMaps: copyEnumMaps(parsed.enumMaps),
      joinSources: copyStringLists(parsed.joinSources),
      splitSkuQuantity: typeof parsed.splitSkuQuantity === 'string' &&
          parsed.splitSkuQuantity.trim() !== ''
        ? parsed.splitSkuQuantity
        : undefined,
      required: copyKeyList(parsed.required),
      fingerprint: copyKeyList(parsed.fingerprint),
    }
  } catch {
    return emptyFieldMapping()
  }
}

/**
 * Serialize a FieldMappingValue to the JSON string stored in
 * TemplateConfig.MappingJSON. Always emits version=3; empty optional maps are
 * omitted so stored configs stay as small as the backend's omitempty shape.
 */
export function serializeMappingRules(mapping: FieldMappingValue): string {
  const mode: FieldMappingMode = mapping.mode === 'positional' ? 'positional' : 'header'
  const rules: Record<string, unknown> = {
    version: MAPPING_VERSION,
    mode,
    hasHeader: mode === 'header',
  }
  if (mode === 'positional') {
    const positions = copyNumbers(mapping.positions)
    if (positions) rules.positions = positions
  } else {
    const columns = copyStrings(mapping.columns)
    if (columns) rules.columns = columns
  }
  const defaults = copyStrings(mapping.defaults)
  if (defaults) rules.defaults = defaults
  const transforms = copyStringLists(mapping.transforms)
  if (transforms) rules.transforms = transforms
  const enumMaps = copyEnumMaps(mapping.enumMaps)
  if (enumMaps) rules.enumMaps = enumMaps
  const joinSources = copyStringLists(mapping.joinSources)
  if (joinSources) rules.joinSources = joinSources
  if (mapping.splitSkuQuantity?.trim()) rules.splitSkuQuantity = mapping.splitSkuQuantity.trim()
  const required = copyKeyList(mapping.required)
  if (required) rules.required = required
  const fingerprint = copyKeyList(mapping.fingerprint)
  if (fingerprint) rules.fingerprint = fingerprint
  if (mapping.sheetName?.trim()) rules.sheetName = mapping.sheetName.trim()
  return JSON.stringify(rules)
}
