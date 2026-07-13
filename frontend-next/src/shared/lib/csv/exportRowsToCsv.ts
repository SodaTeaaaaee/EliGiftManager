/**
 * exportRowsToCsv — generic client-side CSV export utility (P3 batch action
 * bar). Pure frontend: serializes `rows` through the given column specs into
 * an RFC 4180-quoted CSV string, then triggers a browser download via a
 * throwaway `Blob` + anchor click. No bridge call, no backend involvement.
 *
 * Callers are responsible for routing any status-shaped field through
 * `useGlossary().label(...)` in their `getValue` — this module has no
 * knowledge of glossary dimensions and will otherwise happily serialize a
 * raw enum string.
 */

export interface CsvColumnSpec<T> {
  /** Row field name — used as the default accessor when `getValue` is omitted. */
  key: (keyof T & string) | (string & {})
  /** Already-resolved (localized) column header text. */
  header: string
  /** Full value override — required for any derived/localized cell (e.g. a glossary label). */
  getValue?: (row: T) => string | number | null | undefined
}

/** RFC 4180 field quoting: wrap in quotes (and escape embedded quotes) only when needed. */
function escapeCsvCell(value: string | number | null | undefined): string {
  const raw = value == null ? '' : String(value)
  if (/["\n\r,]/.test(raw)) {
    return `"${raw.replace(/"/g, '""')}"`
  }
  return raw
}

/** Mirrors `data-grid/createColumns.ts`'s `defaultGetValue` — same house convention for a flexible `key`-typed default accessor. */
function defaultGetValue<T>(row: T, key: string): string | number | null | undefined {
  return (row as Record<string, unknown>)[key] as string | number | null | undefined
}

function buildCsv<T>(rows: T[], columns: CsvColumnSpec<T>[]): string {
  const headerLine = columns.map((column) => escapeCsvCell(column.header)).join(',')
  const bodyLines = rows.map((row) =>
    columns
      .map((column) => escapeCsvCell(column.getValue ? column.getValue(row) : defaultGetValue(row, column.key)))
      .join(','),
  )
  // CRLF per RFC 4180; \r\n is also what Excel expects on every platform.
  return [headerLine, ...bodyLines].join('\r\n')
}

/**
 * Serializes `rows` via `columns` and triggers a browser download named
 * `filename`. A UTF-8 BOM is prepended so Excel (Windows) renders CJK
 * content correctly instead of mangling it as an unspecified codepage.
 */
export function exportRowsToCsv<T>(rows: T[], columns: CsvColumnSpec<T>[], filename: string): void {
  const csv = buildCsv(rows, columns)
  const blob = new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  try {
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = filename
    document.body.appendChild(anchor)
    anchor.click()
    document.body.removeChild(anchor)
  } finally {
    URL.revokeObjectURL(url)
  }
}
