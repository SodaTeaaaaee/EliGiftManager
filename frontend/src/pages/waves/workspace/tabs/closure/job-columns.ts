/**
 * closure/job-columns — `DataGridColumnSpec<ClosureJobRow>[]` factory for
 * `WaveClosureTab.vue`'s backfill-jobs table (P5, plan §3.3.4 third bullet).
 *
 * `status` is the ONLY glossary-governed enum column here — it renders via
 * `type: 'status'` + the `channelSyncJobStatus` dimension. `direction`
 * (`ChannelSyncJob.Direction`, `internal/infra/persistence/enums.go:156-160`)
 * currently has exactly one server-defined value (`push_tracking`) and is
 * NOT registered as a glossary dimension by foundations — it stays a plain
 * text column, matching the house "free-text field with no branching logic"
 * exception (see `allocation/rule-columns.ts`'s `ruleKind` column).
 *
 * `outputFilePath` is NOT a typed DTO field — it's parsed client-side out of
 * `ChannelSyncJobDTO.responsePayload` by `useClosureTab.ts`'s
 * `parseOutputFilePath` and precomputed onto each `ClosureJobRow` before
 * this factory ever sees it; this column only decides how to DISPLAY the
 * already-resolved path (or a pending/unavailable placeholder) plus the
 * "open containing folder" affordance (`revealInFolder`, sensei-approved
 * reuse of the factory tab's file-reveal capability for parity).
 */
import { h } from 'vue'
import { NButton } from 'naive-ui'
import type { DataGridColumnSpec } from '@/shared/ui/data-grid'
import type { dto } from '@/../wailsjs/go/models'

const EMPTY_PLACEHOLDER = '—'

export interface ClosureJobRow extends dto.ChannelSyncJobDTO {
  profileLabel: string
  outputFilePath: string | null
}

export type ClosureJobTranslate = (key: string, params?: Record<string, unknown>) => string

export interface ClosureJobColumnCallbacks {
  isRunning(jobId: number): boolean
  isRetrying(jobId: number): boolean
  onRun(row: ClosureJobRow): void
  onRetry(row: ClosureJobRow): void
  onViewItems(row: ClosureJobRow): void
  onOpenFolder(path: string): void
}

export function buildClosureJobColumns(
  t: ClosureJobTranslate,
  callbacks: ClosureJobColumnCallbacks,
): DataGridColumnSpec<ClosureJobRow>[] {
  return [
    {
      type: 'number',
      key: 'id',
      title: t('waveWorkspace.closure.jobs.columns.id'),
      width: 80,
      getValue: (row) => row.id,
    },
    {
      type: 'text',
      key: 'profileLabel',
      title: t('waveWorkspace.closure.jobs.columns.profile'),
      minWidth: 140,
      getValue: (row) => row.profileLabel,
    },
    {
      type: 'text',
      key: 'direction',
      title: t('waveWorkspace.closure.jobs.columns.direction'),
      width: 130,
      sortable: false,
      getValue: (row) => row.direction || EMPTY_PLACEHOLDER,
    },
    {
      type: 'status',
      key: 'status',
      title: t('waveWorkspace.closure.jobs.columns.status'),
      dimension: 'channelSyncJobStatus',
      width: 130,
      getValue: (row) => row.status,
    },
    {
      type: 'text',
      key: 'outputFilePath',
      title: t('waveWorkspace.closure.jobs.columns.outputPath'),
      minWidth: 280,
      sortable: false,
      // Custom `render` already handles truncation/tooltip on the inner
      // span — disable naive-ui's own ellipsis wrapper so it doesn't
      // double-wrap the button+path flex row.
      ellipsis: false,
      render: (row) => {
        if (row.status === 'pending' || row.status === 'running') {
          return h('span', { style: 'color: var(--color-text-muted);' }, t('waveWorkspace.closure.jobs.outputPathPending'))
        }
        if (!row.outputFilePath) {
          return h('span', { style: 'color: var(--color-text-muted);' }, t('waveWorkspace.closure.jobs.outputPathUnavailable'))
        }
        return h('div', { style: 'display:flex; align-items:center; gap:8px; min-width:0;' }, [
          h(
            'span',
            {
              style:
                'overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-family: var(--font-mono, monospace); font-size: var(--font-size-xs);',
              title: row.outputFilePath,
            },
            row.outputFilePath,
          ),
          h(
            NButton,
            { size: 'tiny', quaternary: true, onClick: () => callbacks.onOpenFolder(row.outputFilePath as string) },
            // Reuses the factory tab's generic "open containing folder" copy
            // (no closure-specific i18n key exists, and this sub-area cannot
            // edit locale files) — the string is generic UI chrome, not
            // factory-domain-specific.
            { default: () => t('waveWorkspace.factory.generateFile.openFolder') },
          ),
        ])
      },
    },
    {
      type: 'text',
      key: 'errorMessage',
      title: t('waveWorkspace.closure.jobs.columns.error'),
      minWidth: 160,
      getValue: (row) => row.errorMessage || EMPTY_PLACEHOLDER,
    },
    {
      type: 'actions',
      key: 'rowActions',
      title: t('waveWorkspace.closure.jobs.columns.actions'),
      width: 220,
      render: (row) => {
        const buttons = []
        if (row.status === 'pending') {
          buttons.push(
            h(
              NButton,
              { size: 'small', quaternary: true, loading: callbacks.isRunning(row.id), onClick: () => callbacks.onRun(row) },
              { default: () => t('waveWorkspace.closure.jobs.actions.run') },
            ),
          )
        }
        if (row.status === 'failed' || row.status === 'partial_success') {
          buttons.push(
            h(
              NButton,
              { size: 'small', quaternary: true, loading: callbacks.isRetrying(row.id), onClick: () => callbacks.onRetry(row) },
              { default: () => t('waveWorkspace.closure.jobs.actions.retry') },
            ),
          )
        }
        if (row.items.length > 0) {
          buttons.push(
            h(
              NButton,
              { size: 'small', quaternary: true, onClick: () => callbacks.onViewItems(row) },
              // No dedicated "view items" i18n key exists for this sub-area
              // — composed from the generic `common.more` + the real item
              // count rather than inventing new locale-file content.
              { default: () => `${t('common.more')} (${row.items.length})` },
            ),
          )
        }
        return h('div', { style: 'display:flex; gap:8px; justify-content:flex-end;' }, buttons)
      },
    },
  ]
}
