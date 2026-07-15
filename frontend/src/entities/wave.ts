/**
 * Wave entity types — domain enums defined here; DTO shapes re-exported
 * from generated Wails models (wailsjs/go/models.ts) where available.
 *
 * `WaveParticipantSnapshot` is NOT in the generated models — it represents
 * a snapshot concept used only in the frontend domain layer.
 */
import type { dto } from '@/../wailsjs/go/models'
import type {
  LifecycleStage as DomainLifecycleStage,
  SnapshotType as DomainSnapshotType,
  WaveType as DomainWaveType,
} from '@/shared/api/generated/enums'

/** Classification of wave composition. */
export type WaveType = DomainWaveType

/** Lifecycle phase of a wave. */
export type LifecycleStage = DomainLifecycleStage

/** Role of a participant within this wave snapshot. */
export type SnapshotType = DomainSnapshotType

/** All wave allocations — policy rules, contribution sums, final results. */
export type InitialAllocationStrategy =
  | 'policy_driven'
  | 'demand_driven'
  | 'mixed_strategy'

/** WaveDTO — re-exported from generated model. */
export type Wave = dto.WaveDTO

/**
 * A snapshot of one customer/participant within a wave,
 * capturing their identity and gift entitlement at wave creation time.
 * NOTE: No generated DTO counterpart exists; defined manually.
 */
export interface WaveParticipantSnapshot {
  id: number
  waveId: number
  customerProfileId: number
  snapshotType: SnapshotType
  identityPlatform: string
  identityValue: string
  displayName: string
  giftLevel: string
  sourceDocumentRefs: number[] | null
  sourceProfileRefs: number[] | null
  extraData: string | null
  createdAt: string
}

/** Wave with its participant snapshots eagerly loaded. */
export interface WaveWithParticipants extends dto.WaveDTO {
  participants: WaveParticipantSnapshot[]
}
