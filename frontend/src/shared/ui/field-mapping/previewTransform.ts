/**
 * Pure, client-side preview transform for FieldMappingEditor. No backend
 * round-trip — the editor's live preview runs entirely in the browser and
 * only mirrors the cheap parts of the backend parse loop
 * (`internal/app/alignment/parse.go`): source cell → joinSources overwrite →
 * defaults overwrite → transform chain. Full parsing (mapEnum tables,
 * parseDate, normalizePhone, splitSkuQuantity row expansion, required drops)
 * belongs to the backend PreviewTemplate use case.
 */

import type { FieldMappingValue } from './types'

/** One row's resolved values, keyed by semantic key. */
export interface MappedPreviewRow {
  values: Record<string, string>
}

/** Unit separator the backend uses between joinAddress source parts. */
export const JOIN_ADDRESS_PART = '\x1f'

function resolveRef(
  row: Record<string, string>,
  ref: string,
  mode: 'header' | 'positional',
  orderedCells: string[],
): string {
  if (mode === 'positional') {
    const idx = Number(ref)
    return Number.isInteger(idx) && idx >= 0 && idx < orderedCells.length ? orderedCells[idx] : ''
  }
  return row[ref] ?? ''
}

/**
 * Resolves every semantic key referenced by columns/positions/joinSources/
 * defaults for each input row. `rows` are header-keyed maps; in positional
 * mode cells are taken by index over `sourceHeaders` order (fallback:
 * Object.values order).
 */
export function applyMapping(
  rows: Record<string, string>[],
  mapping: FieldMappingValue,
  sourceHeaders?: string[],
): MappedPreviewRow[] {
  const mode = mapping.mode === 'positional' ? 'positional' : 'header'
  const columns = mapping.columns ?? {}
  const positions = mapping.positions ?? {}
  const defaults = mapping.defaults ?? {}
  const transforms = mapping.transforms ?? {}
  const joinSources = mapping.joinSources ?? {}

  const destFields = new Set<string>([
    ...Object.keys(mode === 'positional' ? positions : columns),
    ...Object.keys(joinSources),
    ...Object.keys(defaults),
  ])

  return rows.map((row) => {
    const values: Record<string, string> = {}
    const orderedCells =
      sourceHeaders && sourceHeaders.length > 0
        ? sourceHeaders.map((header) => row[header] ?? '')
        : Object.values(row)

    for (const destField of destFields) {
      // Multi-column joins overwrite the plain source cell…
      const refs = joinSources[destField]
      if (refs && refs.length > 0) {
        values[destField] = refs
          .map((ref) => resolveRef(row, ref, mode, orderedCells))
          .join(JOIN_ADDRESS_PART)
        continue
      }
      // …then the fixed default overrides whatever the source produced.
      const defaultValue = defaults[destField]
      if (defaultValue !== undefined && defaultValue !== '') {
        values[destField] = defaultValue
        continue
      }
      if (mode === 'positional') {
        const idx = positions[destField]
        values[destField] =
          typeof idx === 'number' && idx >= 0 && idx < orderedCells.length
            ? (orderedCells[idx] ?? '')
            : ''
        continue
      }
      const sourceColumn = columns[destField]
      values[destField] = sourceColumn !== undefined ? (row[sourceColumn] ?? '') : ''
    }

    for (const destField of destFields) {
      // JoinSources implies joinAddress, mirroring the backend's finishRow.
      const chain = [...(transforms[destField] ?? [])]
      if (joinSources[destField] && !chain.includes('joinAddress')) chain.push('joinAddress')
      values[destField] = applyPreviewTransforms(values[destField], chain)
    }
    return { values }
  })
}

/**
 * Lightweight transform mirroring the backend's cheap rewrites. trim,
 * strip_quotes, and joinAddress run locally; context-dependent transformers
 * (parseDate, mapEnum, normalizePhone) pass through unchanged so the preview
 * never lies about needing the backend for the real value.
 */
export function applyPreviewTransforms(value: string, transforms: string[] | undefined): string {
  let next = value
  for (const transform of transforms ?? []) {
    if (transform === 'trim') {
      next = next.trim()
    } else if (transform === 'strip_quotes') {
      if (
        next.length >= 2 &&
        ((next.startsWith('"') && next.endsWith('"')) ||
          (next.startsWith("'") && next.endsWith("'")))
      ) {
        next = next.slice(1, -1)
      }
    } else if (transform === 'joinAddress') {
      const parts = next.split(JOIN_ADDRESS_PART).map((p) => p.trim()).filter((p) => p !== '')
      next = parts.join('')
    }
  }
  return next
}
