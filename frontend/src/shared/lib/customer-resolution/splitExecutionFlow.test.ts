import { describe, expect, it } from 'vitest'
import {
  canPreviewCustomerSplit,
  completeSplitParentRefresh,
  completeSplitSupplementalRefresh,
  finishSplitExecutionRefresh,
  requireSplitExecutionRefresh,
} from './splitExecutionFlow'

describe('customer split refresh protocol', () => {
  it('locks preview after execute failure until parent and supplemental refresh both complete', () => {
    const failed = requireSplitExecutionRefresh()
    const parentOnly = completeSplitParentRefresh(failed)
    const both = completeSplitSupplementalRefresh(parentOnly)

    expect(canPreviewCustomerSplit(failed, false)).toBe(false)
    expect(canPreviewCustomerSplit(finishSplitExecutionRefresh(parentOnly), false)).toBe(false)
    expect(canPreviewCustomerSplit(finishSplitExecutionRefresh(both), false)).toBe(true)
  })

  it('keeps preview locked while refreshed data is still loading', () => {
    const both = completeSplitSupplementalRefresh(completeSplitParentRefresh(requireSplitExecutionRefresh()))
    expect(canPreviewCustomerSplit(finishSplitExecutionRefresh(both), true)).toBe(false)
  })
})
