<script setup lang="ts">
import { ref, computed, onMounted, h } from "vue"
import { useRoute } from "vue-router"
import {
  NDataTable,
  NTag,
  NSpin,
  NAlert,
  NCollapse,
  NCollapseItem,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NDatePicker,
  NButton,
  useMessage,
} from "naive-ui"
import type { DataTableColumns, DataTableRowKey } from "naive-ui"
import {
  listShipmentsByWave,
  getSupplierOrderByWave,
  listLinesBySupplierOrder,
  createShipment,
} from "@/shared/lib/wails/app"
import { dto } from "@/../wailsjs/go/models"

const route = useRoute()
const message = useMessage()
const waveId = computed(() => Number(route.params.waveId))

// ── State ──
const loadingList = ref(true)
const loadingOrder = ref(true)
const submitting = ref(false)
const listError = ref("")
const formError = ref("")
const shipments = ref<dto.ShipmentDTO[]>([])
const supplierOrder = ref<dto.SupplierOrderDTO | null>(null)
const orderLines = ref<dto.SupplierOrderLineDTO[]>([])
const selectedLineKeys = ref<DataTableRowKey[]>([])
const lineQuantities = ref<Record<number, number>>({})
const formCollapsed = ref<string[]>([])

const form = ref({
  shipmentNo: "",
  externalShipmentNo: "",
  carrierCode: "",
  carrierName: "",
  trackingNo: "",
  status: "pending",
  shippedAt: null as number | null,
})
// ── Status options ──
const statusOptions = [
  { label: "待发货", value: "pending" },
  { label: "已发货", value: "shipped" },
  { label: "运输中", value: "in_transit" },
  { label: "已签收", value: "delivered" },
  { label: "异常", value: "exception" },
  { label: "已退回", value: "returned" },
]

const statusColorMap: Record<string, "default" | "info" | "success" | "error" | "warning"> = {
  pending: "default",
  shipped: "info",
  in_transit: "info",
  delivered: "success",
  exception: "error",
  returned: "warning",
}

// ── Shipment list columns ──
const shipmentColumns: DataTableColumns<dto.ShipmentDTO> = [
  { title: "ID", key: "id", width: 60 },
  { title: "发货单号", key: "shipmentNo", ellipsis: { tooltip: true } },
  { title: "外部单号", key: "externalShipmentNo", ellipsis: { tooltip: true } },
  { title: "承运商", key: "carrierName", width: 120 },
  { title: "运单号", key: "trackingNo", ellipsis: { tooltip: true } },
  {
    title: "状态",
    key: "status",
    width: 100,
    render(row) {
      return h(NTag, { type: statusColorMap[row.status] || "default", size: "small" }, { default: () => row.status })
    },
  },
  { title: "发货时间", key: "shippedAt", width: 160 },
  {
    title: "行数",
    key: "lineCount",
    width: 60,
    render(row) {
      return String(row.lines?.length ?? 0)
    },
  },
]

// ── Expanded row (shipment lines) columns ──
const shipmentLineColumns: DataTableColumns<dto.ShipmentLineDTO> = [
  { title: "发货行ID", key: "id", width: 80 },
  { title: "供应商订单行ID", key: "supplierOrderLineId", width: 130 },
  { title: "履约行ID", key: "fulfillmentLineId", width: 100 },
  { title: "数量", key: "quantity", width: 80 },
]
// ── Line selection columns (for create form) ──
const lineSelectionColumns: DataTableColumns<dto.SupplierOrderLineDTO> = [
  { type: "selection" },
  { title: "行号", key: "supplierLineNo", width: 80 },
  { title: "供应商SKU", key: "supplierSku", ellipsis: { tooltip: true } },
  { title: "提交数量", key: "submittedQuantity", width: 100 },
  { title: "履约行ID", key: "fulfillmentLineId", width: 100 },
  {
    title: "本次数量",
    key: "qty",
    width: 120,
    render(row) {
      return h(NInputNumber, {
        value: lineQuantities.value[row.id] ?? row.submittedQuantity,
        min: 1,
        size: "small",
        style: "width: 100px",
        onUpdateValue: (val: number | null) => {
          lineQuantities.value[row.id] = val ?? 1
        },
      })
    },
  },
]

// ── Load functions ──
async function loadShipments() {
  if (!waveId.value) return
  loadingList.value = true
  listError.value = ""
  try {
    shipments.value = await listShipmentsByWave(waveId.value)
  } catch (e: any) {
    listError.value = e?.message || String(e)
  } finally {
    loadingList.value = false
  }
}

async function loadSupplierOrder() {
  if (!waveId.value) return
  loadingOrder.value = true
  formError.value = ""
  try {
    const o = await getSupplierOrderByWave(waveId.value)
    if (o && o.id > 0) {
      supplierOrder.value = o
      orderLines.value = await listLinesBySupplierOrder(o.id)
    } else {
      supplierOrder.value = null
      orderLines.value = []
    }
  } catch (e: any) {
    formError.value = e?.message || String(e)
  } finally {
    loadingOrder.value = false
  }
}
// ── Submit ──
async function handleSubmit() {
  if (!supplierOrder.value) return
  submitting.value = true
  formError.value = ""
  try {
    const selectedLines = orderLines.value
      .filter((l) => selectedLineKeys.value.includes(l.id))
      .map((l) => ({
        supplierOrderLineId: l.id,
        fulfillmentLineId: l.fulfillmentLineId,
        quantity: lineQuantities.value[l.id] ?? l.submittedQuantity,
      }))

    const shippedAtISO = form.value.shippedAt
      ? new Date(form.value.shippedAt).toISOString()
      : ""

    await createShipment({
      supplierOrderId: supplierOrder.value.id,
      supplierPlatform: supplierOrder.value.supplierPlatform,
      shipmentNo: form.value.shipmentNo,
      externalShipmentNo: form.value.externalShipmentNo,
      carrierCode: form.value.carrierCode,
      carrierName: form.value.carrierName,
      trackingNo: form.value.trackingNo,
      status: form.value.status,
      shippedAt: shippedAtISO,
      basisPayloadSnapshot: "",
      lines: selectedLines,
    })

    message.success("发货单创建成功")
    // Reset form
    form.value = {
      shipmentNo: "",
      externalShipmentNo: "",
      carrierCode: "",
      carrierName: "",
      trackingNo: "",
      status: "pending",
      shippedAt: null,
    }
    selectedLineKeys.value = []
    lineQuantities.value = {}
    formCollapsed.value = []
    await loadShipments()
  } catch (e: any) {
    formError.value = e?.message || String(e)
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadShipments()
  loadSupplierOrder()
})
</script>
<template>
  <div class="wave-shipment-step p-4">
    <h3 class="text-lg font-medium mb-4">发货单列表</h3>

    <n-alert v-if="listError" type="error" class="mb-4" title="加载错误">
      {{ listError }}
    </n-alert>

    <n-spin :show="loadingList">
      <n-data-table
        :columns="shipmentColumns"
        :data="shipments"
        :bordered="true"
        :single-line="false"
        size="small"
        :row-key="(row: dto.ShipmentDTO) => row.id"
        :default-expand-all="false"
      >
        <template #expand="{ rowData }">
          <n-data-table
            :columns="shipmentLineColumns"
            :data="(rowData as dto.ShipmentDTO).lines || []"
            :bordered="false"
            size="small"
            :row-key="(row: dto.ShipmentLineDTO) => row.id"
          />
        </template>
      </n-data-table>
    </n-spin>

    <h3 class="text-lg font-medium mt-6 mb-4">创建发货单</h3>

    <n-alert v-if="formError" type="error" class="mb-4" title="错误">
      {{ formError }}
    </n-alert>
    <n-collapse v-model:expanded-names="formCollapsed">
      <n-collapse-item title="新建发货单" name="create-form">
        <n-spin :show="loadingOrder">
          <n-form label-placement="left" label-width="120">
            <n-form-item label="供应商订单ID">
              <n-input
                :value="supplierOrder ? String(supplierOrder.id) : '—'"
                readonly
              />
            </n-form-item>
            <n-form-item label="发货单号">
              <n-input v-model:value="form.shipmentNo" placeholder="输入发货单号" />
            </n-form-item>
            <n-form-item label="外部单号">
              <n-input v-model:value="form.externalShipmentNo" placeholder="外部发货单号" />
            </n-form-item>
            <n-form-item label="承运商编码">
              <n-input v-model:value="form.carrierCode" placeholder="承运商编码" />
            </n-form-item>
            <n-form-item label="承运商名称">
              <n-input v-model:value="form.carrierName" placeholder="承运商名称" />
            </n-form-item>
            <n-form-item label="运单号">
              <n-input v-model:value="form.trackingNo" placeholder="运单号" />
            </n-form-item>
            <n-form-item label="状态">
              <n-select v-model:value="form.status" :options="statusOptions" />
            </n-form-item>
            <n-form-item label="发货时间">
              <n-date-picker
                v-model:value="form.shippedAt"
                type="datetime"
                clearable
              />
            </n-form-item>
          </n-form>

          <h4 class="text-base font-medium mb-2 mt-4">选择发货行</h4>
          <n-data-table
            :columns="lineSelectionColumns"
            :data="orderLines"
            :bordered="true"
            :single-line="false"
            size="small"
            :row-key="(row: dto.SupplierOrderLineDTO) => row.id"
            v-model:checked-row-keys="selectedLineKeys"
            :row-props="() => ({ style: 'cursor: pointer' })"
          />
          <!-- NDataTable shows selection column automatically when v-model:checked-row-keys is bound -->

          <div class="mt-4">
            <n-button
              type="primary"
              :loading="submitting"
              :disabled="!supplierOrder || selectedLineKeys.length === 0"
              @click="handleSubmit"
            >
              提交发货单
            </n-button>
          </div>
        </n-spin>
      </n-collapse-item>
    </n-collapse>
  </div>
</template>
