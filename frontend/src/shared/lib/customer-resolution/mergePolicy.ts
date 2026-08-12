import type { LegacyMergeSettings, MergePolicyViewModel } from './types'

export function migrateLegacyMergeSettings(settings: LegacyMergeSettings): MergePolicyViewModel {
  const candidateDetectionEnabled = settings.autoMergeCrossPlatform

  return {
    schemaVersion: 2,
    candidateDetectionEnabled,
    email: {
      mode: settings.autoMergeByEmail ? 'legacy_raw_exact' : 'off',
      enabled: settings.autoMergeByEmail,
      dormant: settings.autoMergeByEmail && !candidateDetectionEnabled,
    },
    phone: {
      mode: settings.autoMergeByPhone ? 'legacy_raw_exact' : 'off',
      enabled: settings.autoMergeByPhone,
      dormant: settings.autoMergeByPhone && !candidateDetectionEnabled,
    },
    executionMode: 'suggest_only',
    migratedFromLegacy: true,
  }
}

export function setCandidateDetectionEnabled(
  policy: MergePolicyViewModel,
  enabled: boolean,
): MergePolicyViewModel {
  return {
    ...policy,
    candidateDetectionEnabled: enabled,
    email: { ...policy.email, dormant: policy.email.enabled && !enabled },
    phone: { ...policy.phone, dormant: policy.phone.enabled && !enabled },
  }
}

export function isMergePolicyRevisionConflict(error: unknown): boolean {
  const message = error instanceof Error ? error.message : String(error)
  return message.toLowerCase().includes('merge policy revision conflict')
}
