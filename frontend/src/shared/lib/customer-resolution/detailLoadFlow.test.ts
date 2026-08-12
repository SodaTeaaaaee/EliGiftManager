import { describe, expect, it } from 'vitest'
import {
  beginDetailLoad,
  completeDetailLoad,
  createDetailLoadFlow,
  failDetailLoad,
  invalidateDetailLoad,
} from './detailLoadFlow'

describe('detail load retry flow', () => {
  it('retains the requested id after failure so retry targets the same bridge call', () => {
    const attempt = beginDetailLoad(createDetailLoadFlow<{ id: number }>(), 42)
    const failed = failDetailLoad(attempt.flow, attempt.request, 'temporary failure')

    expect(failed.requestedId).toBe(42)
    expect(failed.error).toBe('temporary failure')
    expect(beginDetailLoad(failed, failed.requestedId!).request.requestedId).toBe(42)
  })

  it('replaces the error with the successfully loaded detail', () => {
    const attempt = beginDetailLoad(createDetailLoadFlow<{ id: number }>(), 7)
    const failed = failDetailLoad(attempt.flow, attempt.request, 'temporary failure')
    const retry = beginDetailLoad(failed, 7)
    const loaded = completeDetailLoad(retry.flow, retry.request, { id: 7 }, 7)

    expect(loaded.detail).toEqual({ id: 7 })
    expect(loaded.error).toBeNull()
    expect(loaded.loading).toBe(false)
  })

  it('ignores an A response after B becomes the current request', () => {
    const requestA = beginDetailLoad(createDetailLoadFlow<{ id: number }>(), 1)
    const requestB = beginDetailLoad(requestA.flow, 2)
    const afterA = completeDetailLoad(requestB.flow, requestA.request, { id: 1 }, 1)

    expect(afterA).toBe(requestB.flow)
    expect(afterA.requestedId).toBe(2)
    expect(afterA.loading).toBe(true)
  })

  it('rejects a response whose entity id does not match the requested id', () => {
    const attempt = beginDetailLoad(createDetailLoadFlow<{ id: number }>(), 7)
    const mismatched = completeDetailLoad(attempt.flow, attempt.request, { id: 8 }, 8)

    expect(mismatched.detail).toBeNull()
    expect(mismatched.error).toBe('detail_result_id_mismatch')
  })

  it('invalidates an in-flight response when the detail context closes', () => {
    const attempt = beginDetailLoad(createDetailLoadFlow<{ id: number }>(), 7)
    const closed = invalidateDetailLoad(attempt.flow)

    expect(completeDetailLoad(closed, attempt.request, { id: 7 }, 7)).toBe(closed)
  })
})
