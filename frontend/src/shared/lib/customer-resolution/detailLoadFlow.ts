export interface DetailLoadFlow<T> {
  requestedId: number | null
  detail: T | null
  error: string | null
  loading: boolean
  generation: number
  requestSequence: number
}

export interface DetailLoadRequest {
  requestedId: number
  generation: number
  requestSequence: number
}

export interface DetailLoadAttempt<T> {
  flow: DetailLoadFlow<T>
  request: DetailLoadRequest
}

export function createDetailLoadFlow<T>(generation = 0): DetailLoadFlow<T> {
  return {
    requestedId: null,
    detail: null,
    error: null,
    loading: false,
    generation,
    requestSequence: 0,
  }
}

export function invalidateDetailLoad<T>(flow: DetailLoadFlow<T>): DetailLoadFlow<T> {
  return createDetailLoadFlow<T>(flow.generation + 1)
}

export function beginDetailLoad<T>(
  flow: DetailLoadFlow<T>,
  requestedId: number,
): DetailLoadAttempt<T> {
  const requestSequence = flow.requestSequence + 1
  const next = {
    ...flow,
    requestedId,
    detail: null,
    error: null,
    loading: true,
    requestSequence,
  }
  return {
    flow: next,
    request: { requestedId, generation: next.generation, requestSequence },
  }
}

export function detailLoadRequestIsCurrent<T>(
  flow: DetailLoadFlow<T>,
  request: DetailLoadRequest,
): boolean {
  return flow.requestedId === request.requestedId
    && flow.generation === request.generation
    && flow.requestSequence === request.requestSequence
}

export function completeDetailLoad<T>(
  flow: DetailLoadFlow<T>,
  request: DetailLoadRequest,
  detail: T,
  resultId: number,
): DetailLoadFlow<T> {
  if (!detailLoadRequestIsCurrent(flow, request)) return flow
  if (resultId !== request.requestedId) {
    return { ...flow, detail: null, error: 'detail_result_id_mismatch', loading: false }
  }
  return { ...flow, detail, error: null, loading: false }
}

export function failDetailLoad<T>(
  flow: DetailLoadFlow<T>,
  request: DetailLoadRequest,
  error: string,
): DetailLoadFlow<T> {
  if (!detailLoadRequestIsCurrent(flow, request)) return flow
  return { ...flow, detail: null, error, loading: false }
}
