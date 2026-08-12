const blockerTranslationKeys: Record<string, string> = {
  invalid_default_address_selection: 'merge.blocker.invalidDefaultAddressSelection',
  invalid_multiple_default_addresses: 'merge.blocker.invalidMultipleDefaultAddresses',
  invalid_multiple_primary: 'merge.blocker.invalidMultiplePrimary',
  invalid_primary_selection: 'merge.blocker.invalidPrimarySelection',
  multiple_primary_selections: 'merge.blocker.multiplePrimarySelections',
  strong_identity_conflict: 'merge.blocker.strongIdentityConflict',
  unknown_primary_group: 'merge.blocker.unknownPrimaryGroup',
  candidate_expired: 'merge.blocker.candidateExpired',
  candidate_not_pending: 'merge.blocker.candidateNotPending',
  candidate_pair_changed: 'merge.blocker.candidatePairChanged',
  double_pinned_display_name: 'merge.blocker.doublePinnedDisplayName',
  invalid_display_name_resolution: 'merge.blocker.invalidDisplayNameResolution',
  name_episode_collision: 'merge.blocker.nameEpisodeCollision',
  origin_collision: 'merge.blocker.originCollision',
  policy_execution_mode_invalid: 'merge.blocker.policyExecutionModeInvalid',
  policy_not_suggest_only: 'merge.blocker.policyNotSuggestOnly',
  legacy_audit_incomplete: 'merge.blocker.legacyAuditIncomplete',
  merge_not_completed: 'merge.blocker.mergeNotCompleted',
  source_profile_missing: 'merge.blocker.sourceProfileMissing',
  target_profile_missing: 'merge.blocker.targetProfileMissing',
  source_merge_edge_changed: 'merge.blocker.sourceMergeEdgeChanged',
  target_not_active_root: 'merge.blocker.targetNotActiveRoot',
  merge_graph_has_dependents: 'merge.blocker.mergeGraphHasDependents',
  exact_ledger_missing: 'merge.blocker.exactLedgerMissing',
  ledger_entry_incomplete: 'merge.blocker.ledgerEntryIncomplete',
  moved_entity_missing: 'merge.blocker.movedEntityMissing',
  moved_entity_changed: 'merge.blocker.movedEntityChanged',
  demand_document_frozen_after_merge: 'merge.blocker.demandDocumentFrozenAfterMerge',
}

export function mergeBlockerTranslationKey(code: string): string | null {
  return blockerTranslationKeys[code] ?? null
}

export function mergeEventTypeTranslationKey(eventType: string): string | null {
  if (eventType === 'merge_completed') return 'merge.history.eventType.mergeCompleted'
  if (eventType === 'merge_undone') return 'merge.history.eventType.mergeUndone'
  return null
}

export function mergeEventStatusTranslationKey(status: string): string | null {
  if (status === 'completed' || status === 'undone' || status === 'blocked' || status === 'failed') {
    return `merge.history.eventStatus.${status}`
  }
  return null
}
