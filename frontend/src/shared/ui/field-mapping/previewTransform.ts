/**
 * Pure, client-side preview transform for FieldMappingEditor. No backend
 * round-trip — the wizard's mapping-step preview and the demand-intake
 * default validator both run entirely in the browser.
 *
 * `applyMapping` mirrors `MapCSVRowToDemandLine` /
 * `internal/app/template_mapping_service.go`'s field-resolution order
 * exactly: column mappings are resolved first, then `defaults` are applied
 * on top and WIN over a column-mapped value for the same destField (the
 * backend's "Apply defaults" loop runs after "Apply column mappings" and
 * unconditionally overwrites). A destField absent from both `columns` and
 * `defaults` resolves to `''` here — callers that need to distinguish
 * "unmapped" from "mapped to an empty value" should check `columns`/
 * `defaults` directly (as `FieldMappingEditor` does for its "unmapped" cell
 * state), not infer it from this output.
 */

/** One row's resolved values, keyed by destField. */
export interface MappedPreviewRow {
  values: Record<string, string>
}

/**
 * Resolves every destField referenced by `columns` or `defaults` for each
 * input row. `rows` are raw CSV rows (header -> cell value, as parsed by
 * `parseCSVFile` / `CSVFilePreviewDTO.rows`).
 */
export function applyMapping(
  rows: Record<string, string>[],
  columns: Record<string, string>,
  defaults: Record<string, string>,
): MappedPreviewRow[] {
  const destFields = new Set<string>([...Object.keys(columns), ...Object.keys(defaults)])

  return rows.map((row) => {
    const values: Record<string, string> = {}
    for (const destField of destFields) {
      const defaultValue = defaults[destField]
      if (defaultValue !== undefined && defaultValue !== '') {
        values[destField] = defaultValue
        continue
      }
      const sourceColumn = columns[destField]
      const columnValue = sourceColumn !== undefined ? row[sourceColumn] : undefined
      values[destField] = columnValue ?? ''
    }
    return { values }
  })
}

/**
 * The demand-intake domain's validation rule, mirroring
 * `setDemandLineField`'s only hard rule (`internal/app/template_mapping_service.go`):
 * every destField is free text EXCEPT `requested_quantity`, which must parse
 * as an integer (`strconv.Atoi`). Returns a stable, untranslated reason code
 * (`undefined` when valid) — callers resolve it to display copy themselves
 * (e.g. `intakeWizard.mapping.invalidValue`).
 *
 * FieldMappingEditor accepts this as its default `validate` prop; pass a
 * different function to reuse the editor for a domain with different rules
 * (e.g. a future P5 shipment-CSV mapping UI).
 */
export function validateDestFieldValue(destField: string, value: string): string | undefined {
  if (destField !== 'requested_quantity') return undefined
  return /^-?\d+$/.test(value.trim()) ? undefined : 'invalid_integer'
}
