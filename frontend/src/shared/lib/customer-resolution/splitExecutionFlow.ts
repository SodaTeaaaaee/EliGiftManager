export interface SplitExecutionRefreshFlow {
  refreshRequired: boolean
  parentRefreshComplete: boolean
  supplementalRefreshComplete: boolean
}

export function createSplitExecutionRefreshFlow(): SplitExecutionRefreshFlow {
  return {
    refreshRequired: false,
    parentRefreshComplete: true,
    supplementalRefreshComplete: true,
  }
}

export function requireSplitExecutionRefresh(): SplitExecutionRefreshFlow {
  return {
    refreshRequired: true,
    parentRefreshComplete: false,
    supplementalRefreshComplete: false,
  }
}

export function completeSplitParentRefresh(
  flow: SplitExecutionRefreshFlow,
): SplitExecutionRefreshFlow {
  return { ...flow, parentRefreshComplete: true }
}

export function completeSplitSupplementalRefresh(
  flow: SplitExecutionRefreshFlow,
): SplitExecutionRefreshFlow {
  return { ...flow, supplementalRefreshComplete: true }
}

export function finishSplitExecutionRefresh(
  flow: SplitExecutionRefreshFlow,
): SplitExecutionRefreshFlow {
  if (!flow.parentRefreshComplete || !flow.supplementalRefreshComplete) return flow
  return createSplitExecutionRefreshFlow()
}

export function canPreviewCustomerSplit(
  flow: SplitExecutionRefreshFlow,
  loading: boolean,
): boolean {
  return !loading && !flow.refreshRequired
}
