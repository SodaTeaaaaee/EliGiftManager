import type {
  CandidateActionPermissions,
  CandidateStatusTone,
  EvidencePolarity,
  EvidenceStrength,
  MergeCandidateStatus,
  MergeCandidateViewModel,
  MergeEvidenceInput,
  MergeEvidenceViewModel,
} from './types'

const polarityRank: Record<EvidencePolarity, number> = {
  blocker: 0,
  negative: 1,
  positive: 2,
}

const strengthRank: Record<EvidenceStrength, number> = {
  hard: 0,
  strong: 1,
  medium: 2,
  weak: 3,
}

export function maskEvidenceValue(type: string, value: string): string {
  const trimmed = value.trim()
  if (!trimmed) return ''

  if (type.includes('email')) {
    const at = trimmed.indexOf('@')
    if (at <= 0) return '***'
    return `${trimmed[0]}***${trimmed.slice(at)}`
  }

  if (type.includes('phone')) {
    const compact = trimmed.replace(/\s+/g, '')
    return compact.length <= 4 ? '***' : `***${compact.slice(-4)}`
  }

  if (type.includes('address')) return '***'
  return trimmed.length <= 4 ? '***' : `***${trimmed.slice(-4)}`
}

export function toEvidenceViewModel(input: MergeEvidenceInput): MergeEvidenceViewModel {
  return {
    id: input.id,
    type: input.type,
    polarity: input.polarity,
    strength: input.strength,
    explanationCode: input.explanationCode,
    displayValue: input.maskedValue ?? maskEvidenceValue(input.type, input.rawValue ?? ''),
    sourceLabel: input.sourceLabel,
    observedAt: input.observedAt,
  }
}

export function sortEvidence(evidence: MergeEvidenceViewModel[]): MergeEvidenceViewModel[] {
  return [...evidence].sort((left, right) => {
    return (
      polarityRank[left.polarity] - polarityRank[right.polarity] ||
      strengthRank[left.strength] - strengthRank[right.strength] ||
      left.explanationCode.localeCompare(right.explanationCode) ||
      left.id.localeCompare(right.id)
    )
  })
}

export function candidateActions(status: MergeCandidateStatus): CandidateActionPermissions {
  return {
    canPreview: status === 'pending' || status === 'reviewing',
    canDismiss: status === 'pending' || status === 'reviewing',
    canRescan: status === 'stale' || status === 'failed' || status === 'expired',
    canViewHistory: status === 'merged',
  }
}

export function candidateStatusTone(status: MergeCandidateStatus): CandidateStatusTone {
  switch (status) {
    case 'merged':
      return 'success'
    case 'executing':
      return 'progress'
    case 'pending':
    case 'reviewing':
      return 'info'
    case 'stale':
    case 'expired':
    case 'superseded':
      return 'warning'
    case 'blocked':
    case 'failed':
      return 'error'
    default:
      return 'neutral'
  }
}

export function buildCandidateViewModel(
  id: number,
  status: MergeCandidateStatus,
  inputs: MergeEvidenceInput[],
): MergeCandidateViewModel {
  const evidence = sortEvidence(inputs.map(toEvidenceViewModel))
  const isBlocked = status === 'blocked' || evidence.some((item) => item.polarity === 'blocker')

  return {
    id,
    status,
    evidence,
    isBlocked,
    isStale: status === 'stale',
    actions: isBlocked
      ? { canPreview: false, canDismiss: status === 'pending' || status === 'reviewing', canRescan: false, canViewHistory: false }
      : candidateActions(status),
  }
}
