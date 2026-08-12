/**
 * useFactoryOrders — data/action composable for `WaveFactoryTab.vue` (P5,
 * plan 3.3.4 first bullet, factory-orders sub-area). Owns:
 *
 * - The wave's supplier-order list (`getSupplierOrderByWave`) plus each
 *   order's lines (`listLinesBySupplierOrder`), fetched in parallel via
 *   `Promise.all` — replacing the old tree's sequential N+1 await loop
 *   (`frontend/src/pages/wave-workspace/WaveExportStep.vue:62-64`).
 * - `regenerate(profileId)` — the profile-scoped (re)export action. Every
 *   call is followed by a full `loadAll()` re-fetch (the backend's
 *   `DeleteDraftsByWave` rebuild only ever touches `draft` orders, so
 *   already-submitted/accepted orders re-fetch unchanged alongside any
 *   freshly created drafts) plus `ctx.refresh()` so the workspace shell
 *   (overview `supplierOrderCount` / undo-boundary notice) stays in sync.
 *
 * Mirrors `useAllocationTab`'s shape: an `onMounted` initial load, a
 * `waveId` watch for cross-wave deep links. Per-order lifecycle mutations
 * (generate file / mark submitted / record acceptance) are NOT owned here —
 * `SupplierOrderCard.vue` and its dialogs call the bridge directly (per
 * `BatchAdjustDialog.vue`'s precedent) and ask the parent to `loadAll()`
 * afterward via a `changed` emit.
 */
import { onMounted, ref, watch, type ComputedRef, type Ref } from 'vue'
import { exportSupplierOrderForProfile, getSupplierOrderByWave, listLinesBySupplierOrder, listProfiles } from '@/shared/api/bridge'
import { useWaveWorkspaceContext } from '@/shared/lib/wave-workspace/useWaveWorkspace'
import type { dto } from '@/../wailsjs/go/models'
import { eligibleFactoryProfiles } from './factoryProfileSelection'

export interface UseFactoryOrdersApi {
  waveId: ComputedRef<number>
  orders: Ref<dto.SupplierOrderDTO[]>
  linesByOrder: Ref<Map<number, dto.SupplierOrderLineDTO[]>>
  loading: Ref<boolean>
  error: Ref<string | null>
  /** True once the initial load has settled — gates the empty state so it never flashes on first paint. */
  ready: Ref<boolean>
  factoryProfiles: Ref<dto.IntegrationProfileDTO[]>
  loadAll(): Promise<void>
  regenerate(factoryProfileId: number): Promise<dto.SupplierOrderDTO[]>
}

export function useFactoryOrders(): UseFactoryOrdersApi {
  const ctx = useWaveWorkspaceContext()

  const orders = ref<dto.SupplierOrderDTO[]>([]) as Ref<dto.SupplierOrderDTO[]>
  const linesByOrder = ref<Map<number, dto.SupplierOrderLineDTO[]>>(new Map()) as Ref<Map<number, dto.SupplierOrderLineDTO[]>>
  const loading = ref(false)
  const error = ref<string | null>(null)
  const ready = ref(false)
  const factoryProfiles = ref<dto.IntegrationProfileDTO[]>([]) as Ref<dto.IntegrationProfileDTO[]>

  async function loadAll(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const fetchedOrders = await getSupplierOrderByWave(ctx.waveId.value)
      const lineLists = await Promise.all(fetchedOrders.map((order) => listLinesBySupplierOrder(order.id)))
      const nextLines = new Map<number, dto.SupplierOrderLineDTO[]>()
      fetchedOrders.forEach((order, index) => nextLines.set(order.id, lineLists[index] ?? []))
      orders.value = fetchedOrders
      linesByOrder.value = nextLines
      const profiles = await listProfiles()
      factoryProfiles.value = eligibleFactoryProfiles(profiles)
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
    } finally {
      loading.value = false
      ready.value = true
    }
  }

  async function regenerate(factoryProfileId: number): Promise<dto.SupplierOrderDTO[]> {
    const result = await exportSupplierOrderForProfile(ctx.waveId.value, factoryProfileId)
    await loadAll()
    await ctx.refresh()
    return result
  }

  // Cross-wave deep link (workspace shell stays mounted, only `route.params.id` changes).
  watch(ctx.waveId, () => void loadAll())

  onMounted(() => {
    void loadAll()
  })

  return { waveId: ctx.waveId, orders, linesByOrder, loading, error, ready, factoryProfiles, loadAll, regenerate }
}
