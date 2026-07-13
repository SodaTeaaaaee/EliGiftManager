/**
 * fulfillment-grid/columns — `DataGridColumnSpec<FulfillmentGridRow>[]`
 * factory for the fulfillment lines grid (plan 3.3.2, P3 grid core).
 *
 * Every enum-shaped field renders through a `type: 'status'` spec (→
 * `StatusBadge` via `createColumns`) — NEVER a raw enum string in a `text`
 * cell. `participant`'s `participantBadge` (`row.GiftLevel`, a free label,
 * not a glossary-governed enum — see `internal/app/wave_overview_query_usecase.go:510`)
 * is the one exception rendered as plain text, since it has no fixed value
 * set / glossary dimension to route through.
 *
 * This module only returns the house `DataGridColumnSpec[]` — it does NOT
 * call `createColumns()` itself (that must happen inside the consuming
 * component's `setup()`/computed, per `createColumns`' `useGlossary()`
 * dependency and this codebase's existing convention — see `WavesPage.vue`).
 */
import { h } from 'vue'
import type { DataGridColumnSpec } from '@/shared/ui/data-grid'
import type { FulfillmentGridRow } from './useFulfillmentGrid'

const EMPTY_PLACEHOLDER = '—'

/** Already-resolved translate function — pass the caller's own `useI18n().t` straight through. */
export type FulfillmentGridTranslate = (key: string) => string

export function buildFulfillmentColumns(t: FulfillmentGridTranslate): DataGridColumnSpec<FulfillmentGridRow>[] {
  return [
    {
      type: 'text',
      key: 'participantDisplay',
      title: t('fulfillmentGrid.columns.participant'),
      minWidth: 180,
      getValue: (row) => row.participantDisplay,
      render: (row) => {
        if (!row.participantDisplay) return EMPTY_PLACEHOLDER
        if (!row.participantBadge) return row.participantDisplay
        return h('span', { style: 'display:inline-flex; align-items:center; gap:6px;' }, [
          h('span', row.participantDisplay),
          h(
            'span',
            {
              style:
                'font-size:11px; line-height:1; color:var(--color-text-muted); border:1px solid var(--color-border); border-radius:999px; padding:2px 6px;',
            },
            row.participantBadge,
          ),
        ])
      },
    },
    {
      type: 'text',
      key: 'productDisplay',
      title: t('fulfillmentGrid.columns.product'),
      minWidth: 180,
      getValue: (row) => row.productDisplay,
    },
    {
      type: 'number',
      key: 'quantity',
      title: t('fulfillmentGrid.columns.quantity'),
      width: 90,
      getValue: (row) => row.quantity,
    },
    {
      type: 'status',
      key: 'lineReason',
      title: t('fulfillmentGrid.columns.source'),
      dimension: 'lineReason',
      width: 130,
      getValue: (row) => row.lineReason,
    },
    {
      type: 'status',
      key: 'allocationState',
      title: t('fulfillmentGrid.columns.allocationState'),
      dimension: 'allocationState',
      width: 130,
      getValue: (row) => row.allocationState,
    },
    {
      type: 'status',
      key: 'addressState',
      title: t('fulfillmentGrid.columns.addressState'),
      dimension: 'addressState',
      width: 130,
      getValue: (row) => row.addressState,
    },
    {
      type: 'status',
      key: 'supplierState',
      title: t('fulfillmentGrid.columns.supplierState'),
      dimension: 'supplierState',
      width: 150,
      getValue: (row) => row.supplierState,
    },
    {
      type: 'status',
      key: 'channelSyncState',
      title: t('fulfillmentGrid.columns.channelSyncState'),
      dimension: 'channelSyncState',
      width: 150,
      getValue: (row) => row.channelSyncState,
    },
    {
      type: 'status',
      key: 'reviewRequirement',
      title: t('fulfillmentGrid.columns.reviewRequirement'),
      dimension: 'reviewRequirement',
      width: 140,
      getValue: (row) => row.reviewRequirement,
    },
    // Optional per the P3 grid-core brief. No dedicated
    // `fulfillmentGrid.columns.basisDriftStatus` i18n key exists (foundations'
    // namespace has 10 `columns.*` leaves, none for this) — deliberately
    // reusing `fulfillmentGrid.filters.driftStatus` ("依据漂移状态" /
    // "Basis Drift Status") as the header, since it is the same concept
    // just surfaced as a column instead of a filter control. Flagged in
    // the handoff `deviations` rather than inventing a new key.
    {
      type: 'status',
      key: 'basisDriftStatus',
      title: t('fulfillmentGrid.filters.driftStatus'),
      dimension: 'basisDriftStatus',
      width: 130,
      getValue: (row) => row.basisDriftStatus,
    },
    {
      type: 'text',
      key: 'trackingNo',
      title: t('fulfillmentGrid.columns.trackingNo'),
      minWidth: 150,
      getValue: (row) => row.trackingNo,
    },
  ]
}
