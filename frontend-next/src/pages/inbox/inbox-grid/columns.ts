/**
 * inbox-grid/columns — `DataGridColumnSpec<DemandInboxRow>[]` factory for the
 * demand-inbox grid (plan P4). This module only returns the house column
 * spec array — it does NOT call `createColumns()` itself (see
 * `fulfillment-grid/columns.ts`'s identical convention).
 *
 * `kind` and `captureMode` are glossary-governed enums here (routed through
 * `StatusBadge` via the `demandKind` and `captureMode` dimensions
 * respectively). `captureMode`'s authoritative 3-value wire-string set —
 * `document_import` / `api_ingest` / `manual_entry` — comes from
 * `internal/domain/enums.go:34-38` (do not re-derive by guessing from usage
 * sites). `sourceChannel` remains a fixed-value-set-but-ungoverned backend
 * string (e.g. `"manual"`) — no `GlossaryDimension` exists for it in
 * `shared/i18n/glossary.ts`. It renders as plain text; flagged in the P4
 * handoff `deviations` rather than silently treated as glossary-governed.
 */
import type { DataGridColumnSpec } from '@/shared/ui/data-grid'
import type { DemandInboxRow } from '@/entities/demand'

/** Already-resolved translate function — pass the caller's own `useI18n().t` straight through. */
export type InboxGridTranslate = (key: string) => string

export function buildInboxColumns(t: InboxGridTranslate): DataGridColumnSpec<DemandInboxRow>[] {
  return [
    {
      type: 'status',
      key: 'kind',
      title: t('inbox.columns.kind'),
      dimension: 'demandKind',
      width: 150,
      getValue: (row) => row.kind,
    },
    {
      type: 'text',
      key: 'integrationProfileLabel',
      title: t('inbox.columns.profile'),
      minWidth: 160,
      sortable: false,
      getValue: (row) => row.integrationProfileLabel,
    },
    {
      type: 'status',
      key: 'captureMode',
      title: t('inbox.columns.captureMode'),
      dimension: 'captureMode',
      width: 130,
      getValue: (row) => row.captureMode,
    },
    {
      type: 'text',
      key: 'sourceChannel',
      title: t('inbox.columns.sourceChannel'),
      width: 130,
      getValue: (row) => row.sourceChannel,
    },
    {
      type: 'text',
      key: 'sourceDocumentNo',
      title: t('inbox.columns.sourceDoc'),
      minWidth: 150,
      getValue: (row) => row.sourceDocumentNo,
    },
    {
      type: 'number',
      key: 'totalLineCount',
      title: t('inbox.columns.lineCount'),
      width: 90,
      sortable: false,
      getValue: (row) => row.totalLineCount,
    },
    {
      type: 'number',
      key: 'readyAcceptedCount',
      title: t('inbox.columns.ready'),
      width: 90,
      sortable: false,
      getValue: (row) => row.readyAcceptedCount,
    },
    {
      type: 'number',
      key: 'waitingInputCount',
      title: t('inbox.columns.waiting'),
      width: 90,
      sortable: false,
      getValue: (row) => row.waitingInputCount,
    },
    {
      type: 'number',
      key: 'deferredCount',
      title: t('inbox.columns.deferred'),
      width: 90,
      sortable: false,
      getValue: (row) => row.deferredCount,
    },
    {
      type: 'number',
      key: 'excludedCount',
      title: t('inbox.columns.excluded'),
      width: 90,
      sortable: false,
      getValue: (row) => row.excludedCount,
    },
    {
      type: 'text',
      key: 'assignedWaveLabel',
      title: t('inbox.columns.assignedWave'),
      minWidth: 150,
      sortable: false,
      getValue: (row) => (row.assigned ? row.assignedWaveLabel : undefined),
    },
    {
      type: 'date',
      key: 'createdAt',
      title: t('inbox.columns.createdAt'),
      width: 150,
      format: 'datetime',
      getValue: (row) => row.createdAt,
    },
  ]
}
