/**
 * shipments/history-columns — `DataGridColumnSpec<dto.ShipmentDTO>[]`
 * factory for `ShipmentHistory.vue`'s per-wave shipment table (P5, plan
 * §3.3.4 second bullet). Mirrors `closure/job-columns.ts`'s
 * builder-function-with-callbacks shape.
 *
 * `status` is the only glossary-governed column — it renders through
 * `type: 'status'` + the pre-existing `shipmentStatus` dimension (owned by
 * an earlier P, not this sub-area). NOTE: `shipmentStatus`'s glossary table
 * (`shared/i18n/glossary.ts`) currently only lists
 * pending/shipped/in_transit/delivered/exception/returned — it does NOT yet
 * have an entry for `"voided"` (`domain.ShipmentStatusVoided`,
 * `internal/domain/enums.go:152`, the value `VoidShipment` writes).
 * `StatusBadge`'s own contract is to gracefully fall back to a neutral tone
 * + the raw value string for an unregistered `(dimension, value)` pair, so
 * this never throws — but a voided shipment's badge reads "voided" instead
 * of a translated label until foundations adds that glossary entry. Flagged
 * in deviations; not fixed here (glossary.ts is a shared file this unit may
 * not edit).
 */
import { h } from 'vue'
import { NButton } from 'naive-ui'
import type { DataGridColumnSpec } from '@/shared/ui/data-grid'
import type { dto } from '@/../wailsjs/go/models'

const EMPTY_PLACEHOLDER = '—'

export type ShipmentHistoryTranslate = (key: string, params?: Record<string, unknown>) => string

export interface ShipmentHistoryColumnCallbacks {
  onCorrect(row: dto.ShipmentDTO): void
  onVoid(row: dto.ShipmentDTO): void
}

export function buildShipmentHistoryColumns(
  t: ShipmentHistoryTranslate,
  callbacks: ShipmentHistoryColumnCallbacks,
): DataGridColumnSpec<dto.ShipmentDTO>[] {
  return [
    {
      type: 'text',
      key: 'shipmentNo',
      title: t('waveWorkspace.shipments.history.columns.shipmentNo'),
      minWidth: 140,
    },
    {
      type: 'text',
      key: 'supplierPlatform',
      title: t('waveWorkspace.shipments.history.columns.supplierPlatform'),
      minWidth: 120,
    },
    {
      type: 'text',
      key: 'externalShipmentNo',
      title: t('waveWorkspace.shipments.history.columns.externalShipmentNo'),
      minWidth: 140,
    },
    {
      type: 'text',
      key: 'carrier',
      title: t('waveWorkspace.shipments.history.columns.carrier'),
      minWidth: 160,
      getValue: (row) => [row.carrierName, row.carrierCode].filter(Boolean).join(' · ') || EMPTY_PLACEHOLDER,
    },
    {
      type: 'text',
      key: 'trackingNo',
      title: t('waveWorkspace.shipments.history.columns.trackingNo'),
      minWidth: 160,
    },
    {
      type: 'status',
      key: 'status',
      title: t('waveWorkspace.shipments.history.columns.status'),
      dimension: 'shipmentStatus',
      width: 130,
    },
    {
      type: 'date',
      key: 'shippedAt',
      title: t('waveWorkspace.shipments.history.columns.shippedAt'),
      format: 'datetime',
      width: 170,
    },
    {
      type: 'actions',
      key: 'rowActions',
      title: t('waveWorkspace.shipments.history.columns.actions'),
      width: 160,
      render: (row) => {
        const voided = row.status === 'voided'
        return h('div', { style: 'display:flex; gap:8px; justify-content:flex-end;' }, [
          h(
            NButton,
            { size: 'small', quaternary: true, disabled: voided, onClick: () => callbacks.onCorrect(row) },
            { default: () => t('waveWorkspace.shipments.history.actions.correct') },
          ),
          h(
            NButton,
            { size: 'small', quaternary: true, type: 'error', disabled: voided, onClick: () => callbacks.onVoid(row) },
            { default: () => t('waveWorkspace.shipments.history.actions.void') },
          ),
        ])
      },
    },
  ]
}
