/**
 * allocation/rule-columns — `DataGridColumnSpec<AllocationPolicyRule>[]`
 * factory for `WaveAllocationTab.vue`'s rules table.
 *
 * `selectorPayload.type` is a fixed 4-value union (`internal/domain/models.go`
 * `SelectorPayload`) — rendered as a `type: 'status'` column through the
 * `allocationSelectorType` glossary dimension, never a raw string. `ruleKind`
 * is genuinely free text server-side (a passthrough field with no branching
 * logic — see `allocation_policy_usecase.go`), so it stays a plain text
 * column per the house "free-text label" exception. `active` is a plain
 * boolean (not enum-shaped) — rendered as a real `NSwitch` control (in-place
 * toggle), not a status pill.
 */
import { h } from 'vue'
import { NButton, NPopconfirm, NSwitch } from 'naive-ui'
import type { DataGridColumnSpec } from '@/shared/ui/data-grid'
import type { AllocationPolicyRule } from '@/entities/allocation-policy'

const EMPTY_PLACEHOLDER = '—'

export type AllocationRuleTranslate = (key: string, params?: Record<string, unknown>) => string

export interface AllocationRuleColumnCallbacks {
  productNameById: Map<number, string>
  onToggleActive(rule: AllocationPolicyRule, active: boolean): void
  onEdit(rule: AllocationPolicyRule): void
  onDelete(rule: AllocationPolicyRule): void
}

export function buildAllocationRuleColumns(
  t: AllocationRuleTranslate,
  callbacks: AllocationRuleColumnCallbacks,
): DataGridColumnSpec<AllocationPolicyRule>[] {
  return [
    {
      type: 'status',
      key: 'selectorType',
      title: t('allocation.rules.selectorType'),
      dimension: 'allocationSelectorType',
      width: 140,
      getValue: (row) => row.selectorPayload?.type,
    },
    {
      type: 'text',
      key: 'product',
      title: t('allocation.rules.product'),
      minWidth: 160,
      getValue: (row) => callbacks.productNameById.get(row.productId) ?? `#${row.productId}`,
    },
    {
      type: 'text',
      key: 'ruleKind',
      title: t('allocation.rules.ruleName'),
      width: 130,
      getValue: (row) => row.ruleKind || EMPTY_PLACEHOLDER,
    },
    {
      type: 'number',
      key: 'priority',
      title: t('allocation.rules.priority'),
      width: 90,
      getValue: (row) => row.priority,
    },
    {
      type: 'number',
      key: 'contributionQuantity',
      title: t('allocation.rules.quantity'),
      width: 90,
      getValue: (row) => row.contributionQuantity,
    },
    {
      type: 'actions',
      key: 'active',
      title: t('allocation.rules.active'),
      width: 90,
      render: (row) =>
        h(NSwitch, {
          value: row.active,
          size: 'small',
          'onUpdate:value': (value: boolean) => callbacks.onToggleActive(row, value),
        }),
    },
    {
      type: 'actions',
      key: 'rowActions',
      title: '',
      width: 140,
      render: (row) =>
        h('div', { style: 'display:flex; gap:8px; justify-content:flex-end;' }, [
          h(
            NButton,
            { size: 'small', quaternary: true, onClick: () => callbacks.onEdit(row) },
            { default: () => t('common.edit') },
          ),
          h(
            NPopconfirm,
            {
              positiveText: t('common.confirm'),
              negativeText: t('common.cancel'),
              onPositiveClick: () => callbacks.onDelete(row),
            },
            {
              trigger: () =>
                h(NButton, { size: 'small', quaternary: true, type: 'error' }, { default: () => t('common.delete') }),
              default: () => t('allocation.rules.deleteConfirm'),
            },
          ),
        ]),
    },
  ]
}
