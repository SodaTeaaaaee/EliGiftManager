/** Selector payload — discriminated union by `type` field. */
export interface SelectorPayload {
  type: "wave_all" | "platform_all" | "identity_level" | "explicit_override"
  platform?: string
  level?: string
  participant_ids?: number[]
}

/** AllocationPolicyRule (aligned to Go dto.AllocationPolicyRuleDTO). */
export interface AllocationPolicyRule {
  id: number
  wave_id: number
  product_id: number
  selector_payload: SelectorPayload
  product_target_ref: string
  contribution_quantity: number
  rule_kind: string
  priority: number
  active: boolean
  created_at: string
  updated_at: string
}

/** Input for creating a new allocation policy rule. */
export interface CreateAllocationPolicyRuleInput {
  wave_id: number
  product_id: number
  selector_payload: SelectorPayload
  product_target_ref: string
  contribution_quantity: number
  rule_kind: string
  priority: number
  active: boolean
}

/** Input for updating an existing allocation policy rule. Partial fields. */
export interface UpdateAllocationPolicyRuleInput {
  id: number
  product_id?: number
  selector_payload?: SelectorPayload
  product_target_ref?: string
  contribution_quantity?: number
  rule_kind?: string
  priority?: number
  active?: boolean
}

/** Result of a reconcile operation. */
export interface ReconcileResult {
  created: number
  deleted: number
  replayed_count: number
  failures: ReplayFailure[]
}

/** A single replay failure entry. */
export interface ReplayFailure {
  adjustment_id: number
  reason: string
}
