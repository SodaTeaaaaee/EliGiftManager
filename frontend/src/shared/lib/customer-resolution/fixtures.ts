import type {
  CustomerNameObservationInput,
  LegacyMergeSettings,
  MergeEvidenceInput,
  MergeOperationInput,
  SplitSelectionItem,
  UndoDryRunInput,
} from './types'

export const legacyMergeSettingsFixture: LegacyMergeSettings = {
  autoMergeCrossPlatform: true,
  autoMergeByEmail: true,
  autoMergeByPhone: false,
}

export const mergeEvidenceFixture: MergeEvidenceInput[] = [
  {
    id: 'phone-1',
    type: 'normalized_phone',
    polarity: 'positive',
    strength: 'medium',
    explanationCode: 'merge.evidence.same_phone',
    maskedValue: '***8000',
  },
]

export const nicknameObservationsFixture: CustomerNameObservationInput[] = [
  {
    id: 1,
    kind: 'platform_nickname',
    displayValue: '星莓',
    sourceNamespace: 'bilibili:demo',
    sourceLabel: 'Bilibili',
    firstSeenAt: '2026-01-01T00:00:00Z',
    lastSeenAt: '2026-01-01T00:00:00Z',
    observationCount: 1,
  },
]

export const mergeHistoryFixture: MergeOperationInput[] = [
  {
    id: 1,
    sourceProfileId: 12,
    targetProfileId: 7,
    status: 'completed',
    triggerType: 'manual',
    createdAt: '2026-01-02T00:00:00Z',
  },
]

export const undoDryRunFixture: UndoDryRunInput = {
  mergeId: 1,
  eligible: true,
  blockers: [],
  restoreCounts: { identities: 1, addresses: 1, nameObservations: 2, demandDocuments: 0 },
}

export const splitSelectionFixture: SplitSelectionItem[] = [
  { entityType: 'identity', entityId: 1, selected: true, isPrimary: true, primaryGroup: 'bilibili:uid' },
  { entityType: 'name_observation', entityId: 2, selected: true },
]
