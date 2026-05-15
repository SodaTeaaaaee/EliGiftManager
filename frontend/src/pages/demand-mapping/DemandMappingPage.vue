<script setup lang="ts">
import { ref, computed, onMounted, h } from "vue"
import { useRoute } from "vue-router"
import {
  NDataTable,
  NTag,
  NButton,
  NCard,
  NSpin,
  NSpace,
  useMessage,
} from "naive-ui"
import type { DataTableColumns, DataTableExpandedRowKeys } from "naive-ui"
import {
  listAssignedDemandsByWave,
  listDemandDocuments,
  listDemandLines,
  assignDemandToWave,
  generateParticipants,
} from "@/shared/lib/wails/app"
import { dto } from "@/../wailsjs/go/models"

const route = useRoute()
const message = useMessage()
const waveId = computed(() => Number(route.params.waveId))

// ── State ──
const assignedDocs = ref<dto.DemandDocumentDTO[]>([])
const allDocs = ref<dto.DemandDocumentDTO[]>([])
const loadingAssigned = ref(true)
const loadingAll = ref(true)
const linesCache = ref<Record<number, dto.DemandLineDTO[]>>({})
const linesLoading = ref<Record<number, boolean>>({})
const expandedRowKeys = ref<DataTableExpandedRowKeys>([])
const generatingParticipants = ref(false)
const assigningId = ref<number | null>(null)

const assignedIdSet = computed(() => new Set(assignedDocs.value.map((d) => d.id)))
const availableDocs = computed(() =>
  allDocs.value.filter((d) => !assignedIdSet.value.has(d.id)),
)

// ── Loaders ──
async function loadAssigned() {
  if (!waveId.value) return
  loadingAssigned.value = true
  try {
    assignedDocs.value = await listAssignedDemandsByWave(waveId.value)
  } catch (e: any) {
    message.error(`加载已分配需求失败: ${e?.message || e}`)
  } finally {
    loadingAssigned.value = false
  }
}

async function loadAll() {
  loadingAll.value = true
  try {
    allDocs.value = await listDemandDocuments()
  } catch (e: any) {
    message.error(`加载需求列表失败: ${e?.message || e}`)
  } finally {
    loadingAll.value = false
  }
}

async function loadBoth() {
  await Promise.all([loadAssigned(), loadAll()])
}

async function loadLines(docId: number) {
  if (linesCache.value[docId]) return
  linesLoading.value[docId] = true
  try {
    linesCache.value[docId] = await listDemandLines(docId)
  } catch (e: any) {
    message.error(`加载需求行失败: ${e?.message || e}`)
  } finally {
    linesLoading.value[docId] = false
  }
}

// ── Actions ──
async function handleAssign(docId: number) {
  assigningId.value = docId
  try {
    await assignDemandToWave(waveId.value, docId)
    message.success("分配成功")
    await loadBoth()
  } catch (e: any) {
    const msg = String(e?.message || e)
    if (/unique|duplicate|already/i.test(msg)) {
      message.warning("该需求已分配到此波次")
      await loadBoth()
    } else {
      message.error(`分配失败: ${msg}`)
    }
  } finally {
    assigningId.value = null
  }
}

async function handleGenerate() {
  generatingParticipants.value = true
  try {
    const count = await generateParticipants(waveId.value)
    message.success(`已生成 ${count} 个参与者`)
  } catch (e: any) {
    message.error(`生成参与者失败: ${e?.message || e}`)
  } finally {
    generatingParticipants.value = false
  }
}

// ── Kind tag type ──
function kindTagType(kind: string): "info" | "success" | "default" {
  if (kind === "membership_entitlement") return "info"
  if (kind === "retail_order") return "success"
  return "default"
}

// ── Routing disposition tag type ──
function dispositionTagType(
  d: string,
): "success" | "warning" | "error" | "default" {
  if (d === "accepted") return "success"
  if (d === "deferred") return "warning"
  if (d.startsWith("excluded")) return "error"
  return "default"
}

// ── Disposition summary ──
function dispositionSummary(lines: dto.DemandLineDTO[]) {
  const counts = { accepted: 0, deferred: 0, excluded: 0, pending: 0 }
  for (const l of lines) {
    if (l.routingDisposition === "accepted") counts.accepted++
    else if (l.routingDisposition === "deferred") counts.deferred++
    else if (l.routingDisposition.startsWith("excluded")) counts.excluded++
    else counts.pending++
  }
  return counts
}

// ── Columns ──
const docColumns: DataTableColumns<dto.DemandDocumentDTO> = [
  { type: "expand" },
  { title: "ID", key: "id", width: 60 },
  {
    title: "Kind",
    key: "kind",
    width: 180,
    render(row) {
      return h(NTag, { type: kindTagType(row.kind), size: "small" }, () => row.kind)
    },
  },
  { title: "渠道", key: "sourceChannel", width: 120 },
  { title: "单据号", key: "sourceDocumentNo", ellipsis: { tooltip: true } },
  { title: "客户档案ID", key: "customerProfileId", width: 110 },
  { title: "创建时间", key: "createdAt", width: 180 },
]

const availableColumns: DataTableColumns<dto.DemandDocumentDTO> = [
  ...docColumns,
  {
    title: "操作",
    key: "actions",
    width: 80,
    render(row) {
      return h(
        NButton,
        {
          size: "small",
          type: "primary",
          loading: assigningId.value === row.id,
          disabled: assigningId.value !== null,
          onClick: () => handleAssign(row.id),
        },
        () => "分配",
      )
    },
  },
]

const lineColumns: DataTableColumns<dto.DemandLineDTO> = [
  { title: "ID", key: "id", width: 60 },
  { title: "行号", key: "sourceLineNo", width: 60 },
  { title: "类型", key: "lineType", width: 100 },
  { title: "标题", key: "externalTitle", ellipsis: { tooltip: true } },
  { title: "数量", key: "requestedQuantity", width: 60 },
  {
    title: "路由处置",
    key: "routingDisposition",
    width: 140,
    render(row) {
      return h(
        NTag,
        { type: dispositionTagType(row.routingDisposition), size: "small" },
        () => row.routingDisposition,
      )
    },
  },
]

// ── Expand handling ──
function handleExpandChange(keys: DataTableExpandedRowKeys) {
  expandedRowKeys.value = keys
  for (const key of keys) {
    loadLines(key as number)
  }
}

function renderExpand(row: dto.DemandDocumentDTO) {
  const docId = row.id
  if (linesLoading.value[docId]) {
    return h(NSpin, { size: "small", style: "padding: 12px" })
  }
  const lines = linesCache.value[docId]
  if (!lines || lines.length === 0) {
    return h("div", { style: "padding: 12px; color: #999" }, "无需求行")
  }
  const summary = dispositionSummary(lines)
  return h("div", { style: "padding: 8px 0" }, [
    h(NSpace, { style: "margin-bottom: 8px" }, () => [
      h(NTag, { type: "success", size: "small" }, () => `accepted: ${summary.accepted}`),
      h(NTag, { type: "warning", size: "small" }, () => `deferred: ${summary.deferred}`),
      h(NTag, { type: "error", size: "small" }, () => `excluded: ${summary.excluded}`),
      h(NTag, { size: "small" }, () => `pending: ${summary.pending}`),
    ]),
    h(NDataTable, {
      columns: lineColumns,
      data: lines,
      size: "small",
      bordered: false,
      rowKey: (r: dto.DemandLineDTO) => r.id,
    }),
  ])
}

onMounted(() => loadBoth())
</script>

<template>
  <div class="p-4 flex flex-col gap-4">
    <NCard title="已分配需求">
      <template #header-extra>
        <NButton
          type="primary"
          size="small"
          :loading="generatingParticipants"
          @click="handleGenerate"
        >
          生成参与者
        </NButton>
      </template>
      <NDataTable
        :columns="docColumns"
        :data="assignedDocs"
        :loading="loadingAssigned"
        :row-key="(row: dto.DemandDocumentDTO) => row.id"
        :expanded-row-keys="expandedRowKeys"
        :render-expand="renderExpand"
        size="small"
        @update:expanded-row-keys="handleExpandChange"
      />
    </NCard>

    <NCard title="可分配需求">
      <NDataTable
        :columns="availableColumns"
        :data="availableDocs"
        :loading="loadingAll"
        :row-key="(row: dto.DemandDocumentDTO) => row.id"
        size="small"
      />
    </NCard>
  </div>
</template>
