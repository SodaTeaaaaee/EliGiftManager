export type MergeExecutionMode = 'suggest_only' | 'auto_hard_identity'

export type EvidenceRuleMode = 'off' | 'legacy_raw_exact' | 'normalized' | 'normalized_verified'

export interface LegacyMergeSettings {
  autoMergeCrossPlatform: boolean
  autoMergeByEmail: boolean
  autoMergeByPhone: boolean
}

export interface MergePolicyRuleViewModel {
  mode: EvidenceRuleMode
  enabled: boolean
  dormant: boolean
}

export interface MergePolicyViewModel {
  schemaVersion: 2
  candidateDetectionEnabled: boolean
  email: MergePolicyRuleViewModel
  phone: MergePolicyRuleViewModel
  executionMode: MergeExecutionMode
  migratedFromLegacy: true
}

export type MergeCandidateStatus =
  | 'pending'
  | 'reviewing'
  | 'blocked'
  | 'stale'
  | 'dismissed'
  | 'superseded'
  | 'expired'
  | 'executing'
  | 'merged'
  | 'failed'

export type EvidencePolarity = 'positive' | 'negative' | 'blocker'
export type EvidenceStrength = 'hard' | 'strong' | 'medium' | 'weak'

export interface MergeEvidenceInput {
  id: string
  type: string
  polarity: EvidencePolarity
  strength: EvidenceStrength
  explanationCode: string
  maskedValue?: string
  rawValue?: string
  sourceLabel?: string
  observedAt?: string
}

export interface MergeEvidenceViewModel {
  id: string
  type: string
  polarity: EvidencePolarity
  strength: EvidenceStrength
  explanationCode: string
  displayValue: string
  sourceLabel?: string
  observedAt?: string
}

export interface CandidateActionPermissions {
  canPreview: boolean
  canDismiss: boolean
  canRescan: boolean
  canViewHistory: boolean
}

export interface MergeCandidateViewModel {
  id: number
  status: MergeCandidateStatus
  evidence: MergeEvidenceViewModel[]
  isBlocked: boolean
  isStale: boolean
  actions: CandidateActionPermissions
}

export type CandidateStatusTone = 'neutral' | 'info' | 'warning' | 'error' | 'progress' | 'success'

export interface CustomerNameObservationInput {
  id: number
  kind: string
  displayValue: string
  normalizedValue?: string
  sourceNamespace?: string
  sourceLabel?: string
  originProfileId?: number
  firstSeenAt: string
  lastSeenAt: string
  observationCount: number
}

export interface NicknameEpisodeViewModel extends CustomerNameObservationInput {
  observationIds: number[]
  isCurrentDisplayName: boolean
}

export type DisplayNameMode = 'auto' | 'pinned'

export interface DisplayNameEditState {
  mode: DisplayNameMode
  persistedMode: DisplayNameMode
  draftName: string
  pinnedDraftName: string
  persistedName: string
  autoName: string
  rowVersion: number
  dirty: boolean
}

export type DisplayNameEditEvent =
  | { type: 'edit_name'; value: string }
  | { type: 'select_mode'; mode: DisplayNameMode }
  | { type: 'replace_auto_name'; value: string }
  | { type: 'reset' }
  | { type: 'saved'; name: string; mode: DisplayNameMode; rowVersion: number }

export type MergeOperationStatus = 'completed' | 'undone' | 'blocked' | 'failed'

export interface MergeOperationInput {
  id: number
  sourceProfileId: number
  targetProfileId: number
  status: MergeOperationStatus
  triggerType: 'manual' | 'policy' | 'migration'
  createdAt: string
  undoneAt?: string
}

export interface MergeOperationViewModel extends MergeOperationInput {
  canInspect: boolean
  canRequestUndoDryRun: boolean
}

export interface UndoBlocker {
  code: string
  entityType?: string
  entityId?: number
}

export interface UndoDryRunInput {
  mergeId: number
  eligible: boolean
  blockers: UndoBlocker[]
  restoreCounts: {
    identities: number
    addresses: number
    nameObservations: number
    demandDocuments: number
  }
}

export interface UndoDryRunViewModel extends UndoDryRunInput {
  canConfirm: boolean
  totalRestoreCount: number
}

export type SplitEntityType = 'identity' | 'address' | 'name_observation' | 'demand_document'

export interface SplitSelectionItem {
  entityType: SplitEntityType
  entityId: number
  selected: boolean
  frozen?: boolean
  isPrimary?: boolean
  primaryGroup?: string
  isDefault?: boolean
}

export interface SplitSelectionContext {
  newProfileDisplayName: string
  allowIdentitylessProfile?: boolean
}

export type SplitValidationCode =
  | 'selection_required'
  | 'profile_anchor_required'
  | 'frozen_document_selected'
  | 'multiple_default_addresses'
  | 'multiple_primary_identities'

export interface SplitSelectionSummary {
  counts: Record<SplitEntityType, number>
  selectedCount: number
  validationCodes: SplitValidationCode[]
  canPreview: boolean
}
