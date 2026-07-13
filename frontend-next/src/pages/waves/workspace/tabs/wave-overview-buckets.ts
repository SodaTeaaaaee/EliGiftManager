/**
 * wave-overview-buckets — `bucketKind` -> i18n label key / `StatCard` tone
 * maps for the overview tab's six-bucket row (plan 3.3.1, P2 unit B).
 *
 * Deliberately redeclared here rather than imported from
 * `pages/home/HomePage.vue` (which owns an identical table for the
 * task-center action stream): the P2 foundations contract forbids either
 * page unit from editing `HomePage.vue`, and importing from it would create
 * a cross-unit dependency that breaks parallel-safety. The `bucketKind` set
 * is the same (`ActionCenterWaveBucketDTO.bucketKind`), so both tables must
 * be kept in lockstep by hand if the backend ever adds a bucket kind.
 *
 * Labels resolve through the EXISTING `taskCenter.buckets.*` i18n keys
 * (zh-CN.ts / en-US.ts) — this module intentionally does NOT introduce an
 * `overview.buckets.*` namespace.
 */
import type { StatusTone } from '@/shared/i18n/glossary'

/** `ActionCenterWaveBucketDTO.bucketKind` (backend snake_case) -> `taskCenter.buckets.*` i18n key suffix. */
export const BUCKET_LABEL_KEYS: Record<string, string> = {
  missing_address: 'missingAddress',
  waiting_input: 'waitingInput',
  mapping_blocked: 'mappingBlocked',
  channel_sync_failed: 'channelSyncFailed',
  awaiting_manual_closure: 'awaitingManualClosure',
  // Manual override — does NOT mechanically camelCase
  // (`drift_needs_review` would naively become `driftNeedsReview`, but the
  // actual i18n key is `driftReview`).
  drift_needs_review: 'driftReview',
}

/**
 * `bucketKind` -> `StatCard` tone. `channel_sync_failed` is a hard failure
 * (error); the rest are attention-needed-but-recoverable (warning/info).
 * Mirrors `HomePage.vue`'s `BUCKET_TONES` table.
 */
export const BUCKET_TONES: Record<string, StatusTone> = {
  missing_address: 'warning',
  waiting_input: 'info',
  mapping_blocked: 'warning',
  channel_sync_failed: 'error',
  awaiting_manual_closure: 'warning',
  drift_needs_review: 'warning',
}
