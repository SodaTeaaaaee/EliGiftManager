/** Classification of wave composition. */
export type WaveType = "membership" | "retail" | "mixed";

/** Lifecycle phase of a wave. */
export type LifecycleStage =
  | "intake"
  | "allocation"
  | "review"
  | "execution"
  | "syncing_back"
  | "awaiting_manual_closure"
  | "closed";

/**
 * WaveDTO (aligned to Go dto.WaveDTO).
 * Fields match generated wailsjs/models.ts — Go is the authority.
 */
export interface Wave {
  id: number;
  waveNo: string;
  name: string;
  waveType: string;
  lifecycleStage: string;
  progressSnapshot: string;
  notes: string;
  levelTags: string;
  createdAt: string;
  updatedAt: string;
}

/** Role of a participant within this wave snapshot. */
export type SnapshotType = "member" | "buyer" | "mixed";

/**
 * A snapshot of one customer/participant within a wave,
 * capturing their identity and gift entitlement at wave creation time.
 */
export interface WaveParticipantSnapshot {
  id: number;
  waveId: number;
  customerProfileId: number;
  snapshotType: SnapshotType;
  identityPlatform: string;
  identityValue: string;
  displayName: string;
  giftLevel: string;
  sourceDocumentRefs: number[] | null;
  sourceProfileRefs: number[] | null;
  extraData: string | null;
  createdAt: string;
}

/** All wave allocations — policy rules, contribution sums, final results. */
export type InitialAllocationStrategy =
  | "policy_driven"
  | "demand_driven"
  | "mixed_strategy";

/** Wave with its participant snapshots eagerly loaded. */
export interface WaveWithParticipants extends Wave {
  participants: WaveParticipantSnapshot[];
}
