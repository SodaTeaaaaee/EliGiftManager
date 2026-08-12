import type {
  MergeOperationInput,
  MergeOperationViewModel,
  SplitEntityType,
  SplitSelectionContext,
  SplitSelectionItem,
  SplitSelectionSummary,
  SplitValidationCode,
  UndoDryRunInput,
  UndoDryRunViewModel,
} from './types'

function timestamp(value: string): number {
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? 0 : parsed
}

export function buildMergeHistory(operations: MergeOperationInput[]): MergeOperationViewModel[] {
  return [...operations]
    .sort((left, right) => timestamp(right.createdAt) - timestamp(left.createdAt) || right.id - left.id)
    .map((operation) => ({
      ...operation,
      canInspect: true,
      canRequestUndoDryRun: operation.status === 'completed' && operation.undoneAt == null,
    }))
}

export function buildUndoDryRun(input: UndoDryRunInput): UndoDryRunViewModel {
  const totalRestoreCount = Object.values(input.restoreCounts).reduce((sum, count) => sum + count, 0)
  return {
    ...input,
    totalRestoreCount,
    canConfirm: input.eligible && input.blockers.length === 0,
  }
}

function emptyCounts(): Record<SplitEntityType, number> {
  return { identity: 0, address: 0, name_observation: 0, demand_document: 0 }
}

export function summarizeSplitSelection(
  items: SplitSelectionItem[],
  context: SplitSelectionContext,
): SplitSelectionSummary {
  const selected = items.filter((item) => item.selected)
  const counts = emptyCounts()
  for (const item of selected) counts[item.entityType] += 1

  const validationCodes: SplitValidationCode[] = []
  if (selected.length === 0) validationCodes.push('selection_required')

  const hasProfileAnchor =
    counts.identity > 0 || context.allowIdentitylessProfile === true || context.newProfileDisplayName.trim().length > 0
  if (selected.length > 0 && !hasProfileAnchor) validationCodes.push('profile_anchor_required')

  if (selected.some((item) => item.entityType === 'demand_document' && item.frozen)) {
    validationCodes.push('frozen_document_selected')
  }

  if (selected.filter((item) => item.entityType === 'address' && item.isDefault).length > 1) {
    validationCodes.push('multiple_default_addresses')
  }

  const primaryGroups = new Map<string, number>()
  for (const item of selected) {
    if (item.entityType !== 'identity' || !item.isPrimary) continue
    const group = item.primaryGroup ?? 'default'
    primaryGroups.set(group, (primaryGroups.get(group) ?? 0) + 1)
  }
  if ([...primaryGroups.values()].some((count) => count > 1)) {
    validationCodes.push('multiple_primary_identities')
  }

  return {
    counts,
    selectedCount: selected.length,
    validationCodes,
    canPreview: selected.length > 0 && validationCodes.length === 0,
  }
}
