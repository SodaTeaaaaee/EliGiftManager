/**
 * Pure, client-side preview transform for FieldMappingEditor. No backend
 * round-trip — the wizard's mapping-step preview and import UIs' default
 * validators both run entirely in the browser.
 *
 * Mirrors `ApplyRow` / `internal/app/template_mapping_service.go`:
 * source mapping first (columns or positions), then defaults overwrite.
 * A destField absent from both resolves to `''` here — callers that need
 * to distinguish "unmapped" from "mapped to empty" check the maps directly.
 */

import type { FieldMappingValue } from './types'

/** One row's resolved values, keyed by destField. */
export interface MappedPreviewRow {
  values: Record<string, string>
}

/**
 * Resolves every destField referenced by columns/positions/defaults for each
 * input row. `rows` are header-keyed maps from `parseTabularFile`. When mode is
 * positional, cells are taken by index over `sourceHeaders` order (fallback:
 * Object.values order).
 *
 * Call forms:
 * - `applyMapping(rows, mappingValue)` — v2 full mapping (mode/positions/hasHeader)
 * - `applyMapping(rows, mappingValue, undefined, sourceHeaders)` — v2 + header order
 * - `applyMapping(rows, columns, defaults)` — legacy header-only back-compat
 */
export function applyMapping(
  rows: Record<string, string>[],
  columnsOrMapping: Record<string, string> | FieldMappingValue,
  defaults?: Record<string, string>,
  sourceHeaders?: string[],
): MappedPreviewRow[] {
  // Back-compat: applyMapping(rows, columns, defaults)
  const mapping: FieldMappingValue =
    defaults !== undefined || !isFieldMappingValue(columnsOrMapping)
      ? {
          columns: (columnsOrMapping as Record<string, string>) ?? {},
          defaults: defaults ?? {},
          mode: 'header',
        }
      : columnsOrMapping

  const mode = mapping.mode === 'positional' ? 'positional' : 'header'
  const columns = mapping.columns ?? {}
  const positions = mapping.positions ?? {}
  const mappingDefaults = mapping.defaults ?? {}
  const transforms = mapping.transforms ?? {}

  const destFields = new Set<string>([
    ...Object.keys(mode === 'positional' ? positions : columns),
    ...Object.keys(mappingDefaults),
  ])

  return rows.map((row) => {
    const values: Record<string, string> = {}
    const orderedCells =
      sourceHeaders && sourceHeaders.length > 0
        ? sourceHeaders.map((header) => row[header] ?? '')
        : Object.values(row)

    for (const destField of destFields) {
      const defaultValue = mappingDefaults[destField]
      if (defaultValue !== undefined && defaultValue !== '') {
        values[destField] = applyPreviewTransforms(defaultValue, transforms[destField])
        continue
      }
      if (mode === 'positional') {
        const idx = positions[destField]
        values[destField] = applyPreviewTransforms(
          typeof idx === 'number' && idx >= 0 && idx < orderedCells.length
            ? (orderedCells[idx] ?? '')
            : '',
          transforms[destField],
        )
        continue
      }
      const sourceColumn = columns[destField]
      const columnValue = sourceColumn !== undefined ? row[sourceColumn] : undefined
      values[destField] = applyPreviewTransforms(columnValue ?? '', transforms[destField])
    }
    return { values }
  })
}

/** Mirrors the backend's supported TemplateMappingRules transforms. */
export function applyPreviewTransforms(value: string, transforms: string[] | undefined): string {
  let next = value
  for (const transform of transforms ?? []) {
    if (transform === 'trim') next = next.trim()
    else if (transform === 'strip_quotes') {
      if (
        next.length >= 2 &&
        ((next.startsWith('"') && next.endsWith('"')) ||
          (next.startsWith("'") && next.endsWith("'")))
      ) {
        next = next.slice(1, -1)
      }
    } else if (transform === 'strip_leading_quote') {
      next = next.startsWith("'") ? next.slice(1) : next
    }
  }
  return next
}

function isFieldMappingValue(value: unknown): value is FieldMappingValue {
  return (
    typeof value === 'object' &&
    value !== null &&
    'columns' in value &&
    'defaults' in value
  )
}

/**
 * Bare-leaf validator registry for client-side mapping preview.
 * Keys are the leaf after the last namespace dot (or the whole key when
 * unprefixed). Returns a stable untranslated reason code, or undefined when
 * valid / no rule applies.
 *
 * Covers demand-intake (`requested_quantity`) and shipment import
 * (`quantity`, reconciliation integers) so ImportWizard / IntakeWizard /
 * ImportFileModal share one mechanism.
 */
type DestFieldValidator = (value: string) => string | undefined

const DEST_FIELD_VALIDATORS: Record<string, DestFieldValidator> = {
  // Demand line quantity — integer (backend setDemandLineField).
  requested_quantity: (value) => (/^-?\d+$/.test(value.trim()) ? undefined : 'invalid_integer'),
  // Shipment quantity — positive integer.
  quantity: (value) => {
    const trimmed = value.trim()
    return /^\d+$/.test(trimmed) && Number(trimmed) > 0 ? undefined : 'invalid_quantity'
  },
  // Reconciliation / id fields — empty or non-negative integer.
  supplierLineNo: optionalInteger,
  supplier_line_no: optionalInteger,
  fulfillment_line_id: optionalInteger,
  lineId: optionalInteger,
  line_id: optionalInteger,
}

function optionalInteger(value: string): string | undefined {
  const trimmed = value.trim()
  return trimmed === '' || /^\d+$/.test(trimmed) ? undefined : 'invalid_integer'
}

/** Strip `ns.` prefix; unprefixed keys pass through. */
export function bareDestFieldLeaf(destField: string): string {
  if (!destField.includes('.')) return destField
  return destField.slice(destField.lastIndexOf('.') + 1)
}

/**
 * Unified dest-field validator used as FieldMappingEditor's default
 * `validate` prop. Looks up the bare leaf in the dest registry.
 */
export function validateDestFieldValue(destField: string, value: string): string | undefined {
  const bare = bareDestFieldLeaf(destField)
  const validator = DEST_FIELD_VALIDATORS[bare] ?? DEST_FIELD_VALIDATORS[destField]
  return validator ? validator(value) : undefined
}
