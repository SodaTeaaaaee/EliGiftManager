export type ImportEvidenceReference =
  | { kind: 'disabled' }
  | { kind: 'run'; importRunId: number }

export function resolveImportEvidenceReference(input: {
  importRunId?: number
  evidenceDisabled?: boolean
}): ImportEvidenceReference {
  if (input.evidenceDisabled || !input.importRunId) return { kind: 'disabled' }
  return { kind: 'run', importRunId: input.importRunId }
}
