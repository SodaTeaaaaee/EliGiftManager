<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useRoute } from "vue-router"
import {
  NButton,
  NAlert,
  NSpin,
  NDescriptions,
  NDescriptionsItem,
  NDataTable,
  NTag,
  useMessage,
} from "naive-ui"
import type { DataTableColumns } from "naive-ui"
import {
  getSupplierOrderByWave,
  listLinesBySupplierOrder,
  exportSupplierOrder,
} from "@/shared/lib/wails/app"
import { dto } from "@/../wailsjs/go/models"

const route = useRoute()
const message = useMessage()
const waveId = computed(() => Number(route.params.waveId))

const loading = ref(true)
const exporting = ref(false)
const error = ref("")
const order = ref<dto.SupplierOrderDTO | null>(null)
const lines = ref<dto.SupplierOrderLineDTO[]>([])

const hasOrder = computed(() => order.value && order.value.id > 0)
const isDraft = computed(() => hasOrder.value && order.value!.status === "draft")

const columns: DataTableColumns<dto.SupplierOrderLineDTO> = [
  { title: "行号", key: "supplierLineNo", width: 80 },
  { title: "供应商SKU", key: "supplierSku", ellipsis: { tooltip: true } },
  { title: "提交数量", key: "submittedQuantity", width: 100 },
  { title: "接受数量", key: "acceptedQuantity", width: 100 },
  { title: "状态", key: "status", width: 100 },
  { title: "履约行ID", key: "fulfillmentLineId", width: 100 },
]

async function loadOrder() {
  if (!waveId.value) return
  loading.value = true
  error.value = ""
  try {
    const o = await getSupplierOrderByWave(waveId.value)
    if (o && o.id > 0) {
      order.value = o
      lines.value = await listLinesBySupplierOrder(o.id)
    } else {
      order.value = null
      lines.value = []
    }
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

async function handleExport() {
  exporting.value = true
  error.value = ""
  try {
    await exportSupplierOrder(waveId.value)
    await loadOrder()
    message.success("导出成功")
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  loadOrder()
})
</script>

<template>
  <div class="wave-export-step p-4">
    <n-alert
      v-if="isDraft"
      type="warning"
      class="mb-4"
      title="草稿已存在"
    >
      当前波次已有导出草稿，重新导出将替代当前草稿。
    </n-alert>
    <n-alert
      v-else-if="hasOrder && !isDraft"
      type="info"
      class="mb-4"
      title="已有导出记录"
    >
      当前波次存在状态为「{{ order!.status }}」的供应商订单，重新导出将创建新的草稿订单。
    </n-alert>

    <n-alert v-if="error" type="error" class="mb-4" title="错误">
      {{ error }}
    </n-alert>

    <div class="mb-4">
      <n-button
        type="primary"
        :loading="exporting"
        :disabled="loading"
        @click="handleExport"
      >
        {{ isDraft ? "重新导出（覆盖草稿）" : hasOrder ? "重新导出" : "导出供应商订单" }}
      </n-button>
    </div>

    <n-spin :show="loading">
      <template v-if="hasOrder && order">
        <n-descriptions bordered :column="2" class="mb-4" label-placement="left">
          <n-descriptions-item label="ID">{{ order.id }}</n-descriptions-item>
          <n-descriptions-item label="供应商平台">{{ order.supplierPlatform }}</n-descriptions-item>
          <n-descriptions-item label="批次号">{{ order.batchNo }}</n-descriptions-item>
          <n-descriptions-item label="提交方式">{{ order.submissionMode }}</n-descriptions-item>
          <n-descriptions-item label="外部订单号">{{ order.externalOrderNo }}</n-descriptions-item>
          <n-descriptions-item label="状态">
            <n-tag :type="order.status === 'draft' ? 'warning' : 'success'" size="small">
              {{ order.status }}
            </n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="创建时间">{{ order.createdAt }}</n-descriptions-item>
        </n-descriptions>

        <n-data-table
          :columns="columns"
          :data="lines"
          :bordered="true"
          :single-line="false"
          size="small"
          :row-key="(row: dto.SupplierOrderLineDTO) => row.id"
        />
      </template>
      <template v-else-if="!loading && !error">
        <div class="text-center text-gray-400 py-8">
          当前波次尚无导出记录，点击上方按钮开始导出
        </div>
      </template>
    </n-spin>
  </div>
</template>
