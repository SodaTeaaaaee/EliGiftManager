import { isCustomerResolutionFeatureDisabledError } from './featurePolicy'

export type CandidateDismissErrorKind = 'feature_disabled' | 'changed' | 'other'

const CANDIDATE_CHANGED_ERROR = 'merge candidate evidence or policy version changed'

export function classifyCandidateDismissError(error: unknown): CandidateDismissErrorKind {
  if (isCustomerResolutionFeatureDisabledError(error)) return 'feature_disabled'
  const message = (error instanceof Error ? error.message : String(error)).toLowerCase()
  if (message.includes(CANDIDATE_CHANGED_ERROR)) return 'changed'
  return 'other'
}
