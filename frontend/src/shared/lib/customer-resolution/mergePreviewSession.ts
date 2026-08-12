export interface MergePreviewSession {
  sourceProfileId: number
  targetProfileId: number
  candidateId: number | null
  generation: number
}

export interface MergePreviewRequestSession extends MergePreviewSession {
  requestSequence: number
}

export interface AcceptedMergePreviewSession extends MergePreviewRequestSession {
  previewToken: string
}

function candidateId(value?: number | null): number | null {
  return value ?? null
}

export function captureMergePreviewSession(input: {
  sourceProfileId: number | null
  targetProfileId: number | null
  candidateId?: number | null
  generation: number
}): MergePreviewSession | null {
  if (input.sourceProfileId == null || input.targetProfileId == null) return null
  return {
    sourceProfileId: input.sourceProfileId,
    targetProfileId: input.targetProfileId,
    candidateId: candidateId(input.candidateId),
    generation: input.generation,
  }
}

export function mergePreviewSessionIsCurrent(
  session: MergePreviewSession,
  current: {
    sourceProfileId: number | null
    targetProfileId: number | null
    candidateId?: number | null
    generation: number
  },
): boolean {
  return session.sourceProfileId === current.sourceProfileId
    && session.targetProfileId === current.targetProfileId
    && session.candidateId === candidateId(current.candidateId)
    && session.generation === current.generation
}

export function mergePreviewResponseIsCurrent(
  request: MergePreviewRequestSession,
  current: {
    sourceProfileId: number | null
    targetProfileId: number | null
    candidateId?: number | null
    generation: number
    requestSequence: number
  },
  response: {
    sourceProfileId: number
    targetProfileId: number
    candidateId?: number | null
  },
  sourceEntityId: number,
  targetEntityId: number,
): boolean {
  return request.requestSequence === current.requestSequence
    && mergePreviewSessionIsCurrent(request, current)
    && response.sourceProfileId === request.sourceProfileId
    && response.targetProfileId === request.targetProfileId
    && candidateId(response.candidateId) === request.candidateId
    && sourceEntityId === request.sourceProfileId
    && targetEntityId === request.targetProfileId
}

export function acceptMergePreviewSession(
  request: MergePreviewRequestSession,
  previewToken: string,
): AcceptedMergePreviewSession {
  return { ...request, previewToken }
}

export function acceptedMergePreviewIsCurrent(
  accepted: AcceptedMergePreviewSession | null,
  current: {
    sourceProfileId: number | null
    targetProfileId: number | null
    candidateId?: number | null
    generation: number
  },
  preview: {
    sourceProfileId: number
    targetProfileId: number
    candidateId?: number | null
    previewToken: string
  },
): accepted is AcceptedMergePreviewSession {
  return accepted != null
    && accepted.previewToken === preview.previewToken
    && mergePreviewSessionIsCurrent(accepted, current)
    && preview.sourceProfileId === accepted.sourceProfileId
    && preview.targetProfileId === accepted.targetProfileId
    && candidateId(preview.candidateId) === accepted.candidateId
}
