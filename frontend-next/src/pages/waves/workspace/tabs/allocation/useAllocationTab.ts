/**
 * useAllocationTab — data/action composable for `WaveAllocationTab.vue` (P4,
 * plan §3.3 "规则(会员) | 订单映射(零售)"). Owns:
 *
 * - Allocation-policy rule CRUD (list/create/update/delete) + reconcile.
 * - The demand→fulfillment-line mapping run (`mapDemandLines`).
 * - The wave-scoped product list (for the rule editor's product selector +
 *   the rules table's product-name display) and the assigned-demand list
 *   (for the "has any demand been assigned yet" empty-state gate).
 *
 * Mirrors `useFulfillmentGrid`'s shape: an `onMounted` initial load, a
 * `waveId` watch for cross-wave deep links, and every mutation followed by
 * `ctx.refresh()` so the workspace shell (overview/six-bucket/guidance)
 * stays in sync — never a route remount.
 */
import { computed, onMounted, ref, watch, type ComputedRef, type Ref } from 'vue'
import {
  listAllocationPolicyRules,
  createAllocationPolicyRule,
  updateAllocationPolicyRule,
  deleteAllocationPolicyRule,
  reconcileWave,
  mapDemandLines,
  listAssignedDemandsByWave,
  listProductsByWave,
  generateParticipants,
  listWaveParticipantRows,
} from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import type {
  AllocationPolicyRule,
  CreateAllocationPolicyRuleInput,
  UpdateAllocationPolicyRuleInput,
  ReconcileResult,
} from '@/entities/allocation-policy'
import type { Product } from '@/entities/product'
import type { dto } from '@/../wailsjs/go/models'

export interface UseAllocationTabApi {
  waveId: ComputedRef<number>
  rules: Ref<AllocationPolicyRule[]>
  products: Ref<Product[]>
  assignedDemands: Ref<dto.DemandDocumentDTO[]>
  /** `productId -> display name`, for the rules table + rule editor. */
  productNameById: ComputedRef<Map<number, string>>
  loadingRules: Ref<boolean>
  loadingAssigned: Ref<boolean>
  reconciling: Ref<boolean>
  mappingRunning: Ref<boolean>
  lastReconcileResult: Ref<ReconcileResult | null>
  lastMappingResult: Ref<dto.DemandMappingResult | null>
  /** True once the initial load has settled — gates the "no assigned demand" empty state so it never flashes on first paint. */
  ready: Ref<boolean>
  /** Wave-scoped participant snapshot rows (`ListWaveParticipantRows`) — authoritative "have participants been generated" signal, since reconcile/mapping both gate server-side on the same `ListParticipantsByWave` call. */
  participants: Ref<dto.WaveParticipantRowDTO[]>
  participantCount: ComputedRef<number>
  hasParticipants: ComputedRef<boolean>
  loadingParticipants: Ref<boolean>
  generatingParticipants: Ref<boolean>
  /** Count of NEWLY created participants from the most recent `runGenerateParticipants()` call (idempotent upsert by CustomerProfileID) — not the wave's running total. */
  lastGeneratedCount: Ref<number | null>
  loadAll(): Promise<void>
  createRule(input: CreateAllocationPolicyRuleInput): Promise<void>
  updateRule(input: UpdateAllocationPolicyRuleInput): Promise<void>
  removeRule(id: number): Promise<void>
  setRuleActive(rule: AllocationPolicyRule, active: boolean): Promise<void>
  runReconcile(): Promise<ReconcileResult>
  runMapping(): Promise<dto.DemandMappingResult>
  runGenerateParticipants(): Promise<number>
}

export function useAllocationTab(): UseAllocationTabApi {
  const ctx = useWaveWorkspaceContext()

  const rules = ref<AllocationPolicyRule[]>([]) as Ref<AllocationPolicyRule[]>
  const products = ref<Product[]>([]) as Ref<Product[]>
  const assignedDemands = ref<dto.DemandDocumentDTO[]>([]) as Ref<dto.DemandDocumentDTO[]>
  const loadingRules = ref(false)
  const loadingAssigned = ref(false)
  const reconciling = ref(false)
  const mappingRunning = ref(false)
  const lastReconcileResult = ref<ReconcileResult | null>(null) as Ref<ReconcileResult | null>
  const lastMappingResult = ref<dto.DemandMappingResult | null>(null) as Ref<dto.DemandMappingResult | null>
  const ready = ref(false)
  const participants = ref<dto.WaveParticipantRowDTO[]>([]) as Ref<dto.WaveParticipantRowDTO[]>
  const loadingParticipants = ref(false)
  const generatingParticipants = ref(false)
  const lastGeneratedCount = ref<number | null>(null) as Ref<number | null>

  const productNameById = computed<Map<number, string>>(() => {
    const map = new Map<number, string>()
    for (const product of products.value) map.set(product.id, product.name)
    return map
  })

  const participantCount = computed<number>(() => participants.value.length)
  const hasParticipants = computed<boolean>(() => participantCount.value > 0)

  async function loadRules(): Promise<void> {
    loadingRules.value = true
    try {
      rules.value = await listAllocationPolicyRules(ctx.waveId.value)
    } finally {
      loadingRules.value = false
    }
  }

  async function loadProducts(): Promise<void> {
    products.value = await listProductsByWave(ctx.waveId.value)
  }

  async function loadAssignedDemands(): Promise<void> {
    loadingAssigned.value = true
    try {
      assignedDemands.value = await listAssignedDemandsByWave(ctx.waveId.value)
    } finally {
      loadingAssigned.value = false
    }
  }

  /** Mirrors `ListWaveParticipantRows`'s server-side implementation (`internal/app/wave_overview_query_usecase.go:539-543`), which calls the SAME `waveRepo.ListParticipantsByWave` that `ReconcileWave` and `mapDemandLines` gate on — its length is an authoritative "have participants been generated" signal. */
  async function loadParticipants(): Promise<void> {
    loadingParticipants.value = true
    try {
      participants.value = await listWaveParticipantRows(ctx.waveId.value)
    } finally {
      loadingParticipants.value = false
    }
  }

  async function loadAll(): Promise<void> {
    ready.value = false
    await Promise.all([loadRules(), loadProducts(), loadAssignedDemands(), loadParticipants()])
    ready.value = true
  }

  async function createRule(input: CreateAllocationPolicyRuleInput): Promise<void> {
    await createAllocationPolicyRule(input)
    await loadRules()
  }

  async function updateRule(input: UpdateAllocationPolicyRuleInput): Promise<void> {
    await updateAllocationPolicyRule(input)
    await loadRules()
  }

  async function removeRule(id: number): Promise<void> {
    await deleteAllocationPolicyRule(id)
    await loadRules()
  }

  /** In-place active/inactive toggle (rules table's `NSwitch` cell) — a plain boolean field, not a glossary-governed enum. */
  async function setRuleActive(rule: AllocationPolicyRule, active: boolean): Promise<void> {
    await updateAllocationPolicyRule({ id: rule.id, active })
    await loadRules()
  }

  async function runReconcile(): Promise<ReconcileResult> {
    reconciling.value = true
    try {
      const result = await reconcileWave(ctx.waveId.value)
      lastReconcileResult.value = result
      return result
    } finally {
      reconciling.value = false
      await ctx.refresh()
    }
  }

  async function runMapping(): Promise<dto.DemandMappingResult> {
    mappingRunning.value = true
    try {
      const result = await mapDemandLines(ctx.waveId.value)
      lastMappingResult.value = result
      return result
    } finally {
      mappingRunning.value = false
      await ctx.refresh()
    }
  }

  /** `generateParticipants` returns the COUNT OF NEWLY CREATED participants this run (idempotent upsert by CustomerProfileID, `internal/app/use_cases.go:128-158`), NOT the total — `participantCount`/`hasParticipants` (recomputed from `loadParticipants()`) is the running total, `lastGeneratedCount` is the delta for the toast. */
  async function runGenerateParticipants(): Promise<number> {
    generatingParticipants.value = true
    try {
      const created = await generateParticipants(ctx.waveId.value)
      lastGeneratedCount.value = created
      await loadParticipants()
      return created
    } finally {
      generatingParticipants.value = false
      await ctx.refresh()
    }
  }

  // Cross-wave deep link (workspace shell stays mounted, only `route.params.id` changes).
  watch(ctx.waveId, () => void loadAll())

  onMounted(() => {
    void loadAll()
  })

  return {
    waveId: ctx.waveId,
    rules,
    products,
    assignedDemands,
    productNameById,
    loadingRules,
    loadingAssigned,
    reconciling,
    mappingRunning,
    lastReconcileResult,
    lastMappingResult,
    ready,
    participants,
    participantCount,
    hasParticipants,
    loadingParticipants,
    generatingParticipants,
    lastGeneratedCount,
    loadAll,
    createRule,
    updateRule,
    removeRule,
    setRuleActive,
    runReconcile,
    runMapping,
    runGenerateParticipants,
  }
}
