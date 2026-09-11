/**
 * Pure helpers for the Library templates area: the five semantic dictionary
 * groups (docs/TARGET-DOMAIN-AND-PRODUCT-MODEL.md, 模板配置), the per document
 * type default field set, the "currently in effect" rule, and the sample-file
 * → editor row adapters. No Vue, no i18n — labels are resolved by callers.
 */

import type { SampleFileInfo, TemplateConfig } from '@/entities/models'

/** The five dictionary groups in display order. */
export const SEMANTIC_GROUPS = ['identity', 'source', 'product', 'recipient', 'execution'] as const

export type SemanticGroup = (typeof SEMANTIC_GROUPS)[number]

/**
 * Derive the dictionary group from a semantic key's prefix. `quantity` has
 * no prefix but belongs to 商品与数量; unknown prefixes land in 执行与回写 so
 * every key still renders somewhere.
 */
export function semanticGroupOf(key: string): SemanticGroup {
  const prefix = key.includes('.') ? key.slice(0, key.indexOf('.')) : key
  switch (prefix) {
    case 'customer':
    case 'identity':
    case 'membership':
      return 'identity'
    case 'source':
      return 'source'
    case 'product':
    case 'quantity':
      return 'product'
    case 'recipient':
      return 'recipient'
    case 'tracking':
    case 'shipment':
      return 'execution'
    default:
      return 'execution'
  }
}

/**
 * Semantic keys shown by default in the mapping editor for each input
 * document type; every other key hides behind 「显示全部字段」. Prefix entries
 * ending in `.` match every key under that prefix. Returns null (show all)
 * for unknown document types.
 */
export function defaultVisibleKeysFor(documentType: string, dictionary: string[]): string[] | null {
  const patterns: Record<string, string[]> = {
    membership_list: ['identity.', 'membership.level', 'customer.display_name'],
    order_export: ['source.', 'identity.value', 'product.alias_', 'quantity', 'recipient.'],
    shipment_return: ['tracking.id', 'shipment.', 'product.alias_', 'recipient.', 'source.created_at'],
  }
  const selected = patterns[documentType]
  if (!selected) return null
  return dictionary.filter((key) =>
    selected.some((pattern) =>
      pattern.endsWith('.') || pattern.endsWith('_') ? key.startsWith(pattern) : key === pattern,
    ),
  )
}

function timestamp(value: string | undefined): number {
  if (!value) return 0
  const ms = new Date(value).getTime()
  return Number.isFinite(ms) ? ms : 0
}

/**
 * IDs of the active (non-builtin) templates currently in effect for the
 * document types the backend auto-picks: per (PlatformID, Direction,
 * DocumentType) the greatest UpdatedAt wins, ties break on the greatest ID.
 */
export function inEffectTemplateIDs(templates: TemplateConfig[], autoPickedTypes: Set<string>): Set<number> {
  const best = new Map<string, TemplateConfig>()
  for (const tpl of templates) {
    if (tpl.Builtin || !autoPickedTypes.has(tpl.DocumentType)) continue
    const slot = `${tpl.PlatformID}|${tpl.Direction}|${tpl.DocumentType}`
    const current = best.get(slot)
    if (!current) {
      best.set(slot, tpl)
      continue
    }
    const a = timestamp(tpl.UpdatedAt)
    const b = timestamp(current.UpdatedAt)
    if (a > b || (a === b && tpl.ID > current.ID)) best.set(slot, tpl)
  }
  return new Set([...best.values()].map((tpl) => tpl.ID))
}

/** Editor inputs derived from a raw sample: headers plus header-keyed rows. */
export interface SampleEditorRows {
  sourceHeaders: string[]
  sampleRows: Record<string, string>[]
}

/**
 * Turn raw sample records into what FieldMappingEditor consumes. In header
 * mode Records[0] is the header row and data rows are keyed by header text;
 * in positional mode every record is data and keys are 0-based indexes as
 * strings (the editor already renders those as column numbers).
 */
export function buildSampleEditorRows(
  sample: Pick<SampleFileInfo, 'Records'> | null,
  hasHeader: boolean,
): SampleEditorRows {
  const records = sample?.Records ?? []
  if (records.length === 0) return { sourceHeaders: [], sampleRows: [] }
  const width = records.reduce((max, row) => Math.max(max, row.length), 0)
  if (hasHeader) {
    const sourceHeaders = Array.from({ length: width }, (_, i) => {
      const cell = (records[0][i] ?? '').trim()
      return cell === '' ? String(i) : cell
    })
    const sampleRows = records.slice(1).map((row) =>
      Object.fromEntries(sourceHeaders.map((header, i) => [header, row[i] ?? ''])),
    )
    return { sourceHeaders, sampleRows }
  }
  const sourceHeaders = Array.from({ length: width }, (_, i) => String(i))
  const sampleRows = records.map((row) =>
    Object.fromEntries(sourceHeaders.map((header, i) => [header, row[i] ?? ''])),
  )
  return { sourceHeaders, sampleRows }
}
