/**
 * useReconciliationIndex — client-side lookup index for the shipment-backfill
 * CSV import wizard (plan 3.3.4 second bullet / P5 shipment-backfill).
 *
 * `ImportShipments` requires internal DB ids (`supplierOrderLineId` +
 * `fulfillmentLineId`) per entry, but the factory return file can only ever
 * carry the reconciliation key `GenerateSupplierOrderFile` already embeds
 * per line — the line's own id ("行 ID"), or the (batchNo, supplierLineNo)
 * pair ("批次号+行号"). This composable fetches every supplier order in the
 * wave (`getSupplierOrderByWave`) plus every one of their lines
 * (`listLinesBySupplierOrder`) ONCE and builds two lookup maps so
 * `ImportWizard.vue` can resolve either reconciliation key to the mandatory
 * internal ids entirely client-side, before ever calling `importShipments`.
 * The factory-facing CSV itself never needs to contain our DB ids — only the
 * resolved Wails wire payload (an internal contract, never seen by the
 * factory) does.
 */
import { ref } from 'vue'
import { getSupplierOrderByWave, listLinesBySupplierOrder } from '@/shared/api/bridge'
import type { dto } from '@/../wailsjs/go/models'

/** One supplier order line, flattened with its parent order's `batchNo` for the fallback reconciliation key. */
export interface ReconciliationLine {
  lineId: number
  fulfillmentLineId: number
  supplierOrderId: number
  batchNo: string
  supplierLineNo: number | null
  supplierSku: string
  submittedQuantity: number
}

export interface ReconciliationIndex {
  orders: dto.SupplierOrderDTO[]
  byLineId: Map<number, ReconciliationLine>
  byBatchAndLineNo: Map<string, ReconciliationLine>
}

const EMPTY_INDEX: ReconciliationIndex = { orders: [], byLineId: new Map(), byBatchAndLineNo: new Map() }

/** Stable composite key for the (batchNo, supplierLineNo) reconciliation map. */
export function batchLineNoKey(batchNo: string, supplierLineNo: number): string {
  return `${batchNo.trim()}::${supplierLineNo}`
}

export function useReconciliationIndex() {
  const index = ref<ReconciliationIndex>(EMPTY_INDEX)
  const loading = ref(false)
  const loadError = ref(false)

  async function load(waveId: number): Promise<void> {
    loading.value = true
    loadError.value = false
    try {
      const orders = await getSupplierOrderByWave(waveId)
      const lineLists = await Promise.all(orders.map((order) => listLinesBySupplierOrder(order.id)))

      const byLineId = new Map<number, ReconciliationLine>()
      const byBatchAndLineNo = new Map<string, ReconciliationLine>()

      orders.forEach((order, orderIndex) => {
        for (const line of lineLists[orderIndex]) {
          const entry: ReconciliationLine = {
            lineId: line.id,
            fulfillmentLineId: line.fulfillmentLineId,
            supplierOrderId: order.id,
            batchNo: order.batchNo,
            supplierLineNo: line.supplierLineNo ?? null,
            supplierSku: line.supplierSku,
            submittedQuantity: line.submittedQuantity,
          }
          byLineId.set(line.id, entry)
          if (line.supplierLineNo != null) {
            byBatchAndLineNo.set(batchLineNoKey(order.batchNo, line.supplierLineNo), entry)
          }
        }
      })

      index.value = { orders, byLineId, byBatchAndLineNo }
    } catch {
      index.value = EMPTY_INDEX
      loadError.value = true
    } finally {
      loading.value = false
    }
  }

  return { index, loading, loadError, load }
}
