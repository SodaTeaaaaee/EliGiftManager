/**
 * LayoutConfig v1 helpers — the render-side half of a template, mirroring
 * `internal/app/alignment/schema.go` (`LayoutConfig`) and stored in
 * TemplateConfig.LayoutJSON. Kept as a tiny pure module so the layout editor
 * and its tests never touch JSON directly.
 */

export type LayoutFormat = 'csv' | 'xlsx'

const LAYOUT_VERSION = 1

/** One selected output column: semantic key plus an optional display header. */
export interface LayoutColumn {
  key: string
  /** Empty string means "fall back to the localized semantic label". */
  header: string
}

export interface LayoutConfigValue {
  version: number
  format: LayoutFormat
  columns: LayoutColumn[]
}

export function emptyLayoutConfig(format: LayoutFormat = 'csv'): LayoutConfigValue {
  return { version: LAYOUT_VERSION, format, columns: [] }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

/**
 * Parse TemplateConfig.LayoutJSON. Empty input, invalid JSON, or a foreign
 * schema version yield an empty v1 layout so the editor starts clean instead
 * of half-rendering foreign data (the backend rejects both on save anyway).
 * Duplicate keys in columnOrder collapse to their first occurrence.
 */
export function parseLayoutConfig(raw: string | undefined | null): LayoutConfigValue {
  if (!raw || !raw.trim()) return emptyLayoutConfig()
  try {
    const parsed = JSON.parse(raw) as unknown
    if (!isRecord(parsed) || parsed.version !== LAYOUT_VERSION) return emptyLayoutConfig()
    const format: LayoutFormat = parsed.format === 'xlsx' ? 'xlsx' : 'csv'
    const headerNames = isRecord(parsed.headerNames) ? parsed.headerNames : {}
    const seen = new Set<string>()
    const columns: LayoutColumn[] = []
    for (const item of Array.isArray(parsed.columnOrder) ? parsed.columnOrder : []) {
      if (typeof item !== 'string') continue
      const key = item.trim()
      if (key === '' || seen.has(key)) continue
      seen.add(key)
      const header = headerNames[key]
      columns.push({ key, header: typeof header === 'string' ? header : '' })
    }
    return { version: LAYOUT_VERSION, format, columns }
  } catch {
    return emptyLayoutConfig()
  }
}

/**
 * Serialize to the JSON string stored in TemplateConfig.LayoutJSON. Always
 * emits version=1; blank headers are omitted so the backend falls back to
 * the semantic key, matching its `omitempty` shape.
 */
export function serializeLayoutConfig(layout: LayoutConfigValue): string {
  const columnOrder: string[] = []
  const headerNames: Record<string, string> = {}
  const seen = new Set<string>()
  for (const column of layout.columns) {
    const key = column.key.trim()
    if (key === '' || seen.has(key)) continue
    seen.add(key)
    columnOrder.push(key)
    const header = column.header.trim()
    if (header !== '') headerNames[key] = header
  }
  const stored: Record<string, unknown> = {
    version: LAYOUT_VERSION,
    format: layout.format === 'xlsx' ? 'xlsx' : 'csv',
    columnOrder,
  }
  if (Object.keys(headerNames).length > 0) stored.headerNames = headerNames
  return JSON.stringify(stored)
}

/** Move the column at `index` by `delta` (-1 up, +1 down); no-op at the edges. */
export function moveLayoutColumn(columns: LayoutColumn[], index: number, delta: -1 | 1): LayoutColumn[] {
  const target = index + delta
  if (index < 0 || index >= columns.length || target < 0 || target >= columns.length) return columns
  const next = [...columns]
  ;[next[index], next[target]] = [next[target], next[index]]
  return next
}

/**
 * Neutral sample cell for the header-preview row. Identifier-like and
 * numeric keys get a realistic-looking value; free-text keys echo the
 * localized label in angle brackets so the preview stays locale-neutral.
 */
export function placeholderSampleFor(key: string, label: string): string {
  const fixed: Record<string, string> = {
    quantity: '1',
    'shipment.quantity': '1',
    'source.line_no': '1',
    'source.document_no': '20260904000001',
    'source.created_at': '2026-09-04 10:00:00',
    'shipment.shipped_at': '2026-09-04 18:30:00',
    'shipment.tracking_no': 'SF1234567890123',
    'shipment.carrier_code': 'SF',
    'tracking.id': 'TRK-0001-A7X9',
    'identity.value': '123456789',
    'identity.type': 'platform_uid',
    'product.alias_id': 'SKU-001',
    'product.factory_sku': 'F-SKU-001',
    'recipient.phone': '13800000000',
    'recipient.postal_code': '100000',
  }
  return fixed[key] ?? `〈${label}〉`
}
