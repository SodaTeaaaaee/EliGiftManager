/**
 * AllocationPolicy entity types.
 *
 * Unlike other entity files (which re-export generated DTO shapes), this file
 * defines its own interfaces because the Wails codegen represents
 * `selector_payload` as `number[]`, whereas the actual runtime value is a
 * typed discriminated union (`SelectorPayload`). The entity types here are
 * the canonical frontend representation; callers MUST use these instead of
 * the generated `dto.AllocationPolicyRuleDTO` / `dto.CreateAllocationPolicyRuleInput` / etc.
 */

/** Selector payload — discriminated union by `type` field. */
export interface SelectorPayload {
  type: "wave_all" | "platform_all" | "identity_level" | "explicit_override"
  platform?: string
  level?: string
  participant_ids?: number[]
}

/**
 * AllocationPolicyRule (aligned to Go dto.AllocationPolicyRuleDTO).
 * Container fields use camelCase to match the DTO json tags; the nested
 * `selectorPayload` keys (`type`/`platform`/`level`/`participant_ids`) stay
 * as-is because they mirror the domain SelectorPayload JSON stored in the DB.
 */
export interface AllocationPolicyRule {
  id: number
  waveId: number
  productId: number
  selectorPayload: SelectorPayload
  productTargetRef: string
  contributionQuantity: number
  ruleKind: string
  priority: number
  active: boolean
  createdAt: string
  updatedAt: string
}

/** Input for creating a new allocation policy rule. */
export interface CreateAllocationPolicyRuleInput {
  waveId: number
  productId: number
  selectorPayload: SelectorPayload
  productTargetRef: string
  contributionQuantity: number
  ruleKind: string
  priority: number
  active: boolean
}

/** Input for updating an existing allocation policy rule. Partial fields. */
export interface UpdateAllocationPolicyRuleInput {
  id: number
  productId?: number
  selectorPayload?: SelectorPayload
  productTargetRef?: string
  contributionQuantity?: number
  ruleKind?: string
  priority?: number
  active?: boolean
}

/** Result of a reconcile operation. */
export interface ReconcileResult {
  created: number
  deleted: number
  replayedCount: number
  failures: ReplayFailure[]
}

/** A single replay failure entry. */
export interface ReplayFailure {
  adjustmentId: number
  reason: string
}
