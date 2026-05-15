<script setup lang="ts">
import { ref, computed, onMounted, h } from "vue"
import { useRoute } from "vue-router"
import {
  NDataTable,
  NTag,
  NSpin,
  NAlert,
  NSelect,
  NButton,
  NInput,
  NCard,
  NSpace,
  useMessage,
} from "naive-ui"
import type { DataTableColumns } from "naive-ui"
import {
  listIntegrationProfiles,
  listChannelSyncJobsByWave,
  planChannelClosure,
  executeChannelSyncJob,
  retryChannelSyncJob,
  recordChannelClosureDecision,
} from "@/shared/lib/wails/app"
import { dto } from "@/../wailsjs/go/models"

const route = useRoute()
const message = useMessage()
const waveId = computed(() => Number(route.params.waveId))

// ── State ──
const loading = ref(false)
const profilesLoading = ref(true)
const planLoading = ref(false)
const submitLoading = ref(false)

const profiles = ref<dto.IntegrationProfileDTO[]>([])
const selectedProfileId = ref<number | null>(null)
const jobs = ref<dto.ChannelSyncJobDTO[]>([])
const planResult = ref<dto.PlanChannelClosureResult | null>(null)
const error = ref("")

// Manual decision form data (reactive Record, not Map)
interface ManualFormData {
  decisionKind: string
  reasonCode: string
  note: string
  evidenceRef: string
  operatorId: string
}
const manualForms = ref<Record<number, ManualFormData>>({})

// ── Options ──
const decisionKindOptions = [
  { label: "标记为不支持同步", value: "mark_sync_unsupported" },
  { label: "标记为跳过同步", value: "mark_sync_skipped" },
  { label: "标记为已手动完成", value: "mark_sync_completed_manually" },
]

const jobStatusColor: Record<string, "default" | "info" | "success" | "warning" | "error"> = {
  pending: "default",
  running: "info",
  success: "success",
  partial_success: "warning",
  failed: "error",
}

const itemStatusColor: Record<string, "default" | "success" | "error"> = {
  pending: "default",
  success: "success",
  failed: "error",
}

// ── Profile select options ──
const profileOptions = computed(() =>
  profiles.value.map((p) => ({
    label: `${p.profileKey} (${p.sourceChannel})`,
    value: p.id,
  })),
)

// ── Job table columns ──
const jobColumns: DataTableColumns<dto.ChannelSyncJobDTO> = [
  { type: "expand" },
  { title: "ID", key: "id", width: 60 },
  { title: "Profile ID", key: "integrationProfileId", width: 100 },
  { title: "方向", key: "direction", width: 100 },
  {
    title: "状态",
    key: "status",
    width: 120,
    render(row) {
      return h(NTag, { type: jobStatusColor[row.status] || "default", size: "small" }, { default: () => row.status })
    },
  },
  { title: "创建时间", key: "createdAt", width: 160, ellipsis: { tooltip: true } },
  { title: "错误信息", key: "errorMessage", ellipsis: { tooltip: true } },
  {
    title: "操作",
    key: "actions",
    width: 160,
    render(row) {
      const buttons: any[] = []
      if (row.status === "pending") {
        buttons.push(
          h(NButton, { size: "small", type: "primary", onClick: () => handleExecute(row.id) }, { default: () => "执行" }),
        )
      }
      if (row.status === "failed" || row.status === "partial_success") {
        buttons.push(
          h(NButton, { size: "small", type: "warning", onClick: () => handleRetry(row.id) }, { default: () => "重试" }),
        )
      }
      return h(NSpace, { size: "small" }, { default: () => buttons })
    },
  },
]

// ── Item sub-table columns ──
const itemColumns: DataTableColumns<dto.ChannelSyncItemDTO> = [
  { title: "ID", key: "id", width: 60 },
  { title: "履约行ID", key: "fulfillmentLineId", width: 100 },
  { title: "运单号", key: "trackingNo", ellipsis: { tooltip: true } },
  { title: "承运商", key: "carrierCode", width: 100 },
  {
    title: "状态",
    key: "status",
    width: 100,
    render(row) {
      return h(NTag, { type: itemStatusColor[row.status] || "default", size: "small" }, { default: () => row.status })
    },
  },
  { title: "错误信息", key: "errorMessage", ellipsis: { tooltip: true } },
]

// ── Load functions ──
async function loadProfiles() {
  profilesLoading.value = true
  try {
    profiles.value = await listIntegrationProfiles()
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    profilesLoading.value = false
  }
}

async function loadJobs() {
  if (!waveId.value) return
  loading.value = true
  error.value = ""
  try {
    jobs.value = await listChannelSyncJobsByWave(waveId.value)
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}

// ── Plan ──
async function handlePlan() {
  if (!selectedProfileId.value || !waveId.value) return

  // Guard: block if existing jobs for this profile are pending/running
  const activeJob = jobs.value.find(
    (j) =>
      j.integrationProfileId === selectedProfileId.value &&
      (j.status === "pending" || j.status === "running"),
  )
  if (activeJob) {
    message.warning("该 Profile 已有进行中的任务，请等待完成后再规划")
    return
  }

  planLoading.value = true
  planResult.value = null
  error.value = ""
  try {
    const result = await planChannelClosure({
      waveId: waveId.value,
      integrationProfileId: selectedProfileId.value,
    })
    planResult.value = result

    // Initialize manual forms if manual_closure or unsupported (both need closure decisions)
    if ((result.decision === "manual_closure" || result.decision === "unsupported") && result.items) {
      const forms: Record<number, ManualFormData> = {}
      for (const item of result.items) {
        forms[item.fulfillmentLineId] = {
          decisionKind: result.decision === "unsupported" ? "mark_sync_unsupported" : "",
          reasonCode: "",
          note: "",
          evidenceRef: "",
          operatorId: "",
        }
      }
      manualForms.value = forms
    }

    await loadJobs()
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    planLoading.value = false
  }
}

// ── Execute ──
async function handleExecute(jobId: number) {
  const job = jobs.value.find((j) => j.id === jobId)
  if (!job || job.status !== "pending") {
    message.warning("只能执行 pending 状态的任务")
    return
  }
  try {
    await executeChannelSyncJob(jobId)
    message.success("执行完成")
    await loadJobs()
  } catch (e: any) {
    message.error(e?.message || String(e))
  }
}

// ── Retry ──
async function handleRetry(jobId: number) {
  const job = jobs.value.find((j) => j.id === jobId)
  if (!job || (job.status !== "failed" && job.status !== "partial_success")) {
    message.warning("只能重试 failed 或 partial_success 状态的任务")
    return
  }
  try {
    await retryChannelSyncJob(jobId)
    message.success("重试完成")
    await loadJobs()
  } catch (e: any) {
    message.error(e?.message || String(e))
  }
}

// ── Submit manual decisions ──
async function handleSubmitDecisions() {
  if (!selectedProfileId.value || !waveId.value) return
  if (!planResult.value || (planResult.value.decision !== "manual_closure" && planResult.value.decision !== "unsupported")) return

  const entries = Object.entries(manualForms.value)
    .filter(([, form]) => form.decisionKind)
    .map(([lineId, form]) => ({
      fulfillmentLineId: Number(lineId),
      decisionKind: form.decisionKind,
      reasonCode: form.reasonCode,
      note: form.note,
      evidenceRef: form.evidenceRef,
      operatorId: form.operatorId,
    }))

  if (entries.length === 0) {
    message.warning("请至少填写一条决策")
    return
  }

  submitLoading.value = true
  try {
    await recordChannelClosureDecision({
      waveId: waveId.value,
      integrationProfileId: selectedProfileId.value,
      entries,
    })
    message.success("决策提交成功")
    planResult.value = null
    await loadJobs()
  } catch (e: any) {
    message.error(e?.message || String(e))
  } finally {
    submitLoading.value = false
  }
}

onMounted(() => {
  loadProfiles()
  loadJobs()
})
</script>

<template>
  <div class="wave-channel-sync-step p-4">
    <n-alert v-if="error" type="error" class="mb-4" title="错误">
      {{ error }}
    </n-alert>

    <!-- Section 1: Profile selector + Plan -->
    <n-card title="渠道回填规划" size="small" class="mb-4">
      <n-space align="center">
        <n-select
          v-model:value="selectedProfileId"
          :options="profileOptions"
          :loading="profilesLoading"
          placeholder="选择集成配置"
          style="width: 320px"
        />
        <n-button
          type="primary"
          :loading="planLoading"
          :disabled="!selectedProfileId"
          @click="handlePlan"
        >
          规划关闭
        </n-button>
      </n-space>
    </n-card>

    <!-- Section 2: Plan result -->
    <template v-if="planResult">
      <!-- create_job -->
      <n-card v-if="planResult.decision === 'create_job'" title="同步任务已创建" size="small" class="mb-4">
        <p class="mb-2">系统已自动创建同步任务，请在下方任务列表中执行。</p>
        <n-button
          v-if="planResult.job"
          type="primary"
          size="small"
          @click="handleExecute(planResult.job.id)"
        >
          立即执行
        </n-button>
      </n-card>

      <!-- manual_closure -->
      <n-card v-else-if="planResult.decision === 'manual_closure'" title="需要手动关闭" size="small" class="mb-4">
        <p class="mb-3">以下履约行需要手动决策关闭方式：</p>
        <div
          v-for="item in planResult.items"
          :key="item.fulfillmentLineId"
          class="mb-4 p-3 border border-gray-200 rounded"
        >
          <p class="mb-2 font-medium">履约行 #{{ item.fulfillmentLineId }}</p>
          <n-space vertical>
            <n-select
              v-model:value="manualForms[item.fulfillmentLineId].decisionKind"
              :options="decisionKindOptions"
              placeholder="选择决策类型"
              style="width: 280px"
            />
            <n-input
              v-model:value="manualForms[item.fulfillmentLineId].reasonCode"
              placeholder="原因代码"
            />
            <n-input
              v-model:value="manualForms[item.fulfillmentLineId].note"
              placeholder="备注"
              type="textarea"
              :rows="2"
            />
            <n-input
              v-model:value="manualForms[item.fulfillmentLineId].evidenceRef"
              placeholder="证据引用"
            />
            <n-input
              v-model:value="manualForms[item.fulfillmentLineId].operatorId"
              placeholder="操作人ID"
            />
          </n-space>
        </div>
        <n-button
          type="primary"
          :loading="submitLoading"
          @click="handleSubmitDecisions"
        >
          提交决策
        </n-button>
      </n-card>

      <!-- unsupported — still allow manual closure decisions -->
      <n-card v-else-if="planResult.decision === 'unsupported'" title="不支持自动同步" size="small" class="mb-4">
        <n-alert type="warning" class="mb-3" title="当前集成配置不支持自动同步">
          以下履约行仍可手动记录关闭决策，以便统计与闭环。
        </n-alert>
        <div
          v-for="item in planResult.items"
          :key="item.fulfillmentLineId"
          class="mb-4 p-3 border border-gray-200 rounded"
        >
          <p class="mb-2 font-medium">履约行 #{{ item.fulfillmentLineId }}</p>
          <n-space vertical>
            <n-select
              v-model:value="manualForms[item.fulfillmentLineId].decisionKind"
              :options="decisionKindOptions"
              placeholder="选择决策类型"
              style="width: 280px"
            />
            <n-input
              v-model:value="manualForms[item.fulfillmentLineId].reasonCode"
              placeholder="原因代码"
            />
            <n-input
              v-model:value="manualForms[item.fulfillmentLineId].note"
              placeholder="备注"
              type="textarea"
              :rows="2"
            />
          </n-space>
        </div>
        <n-button
          type="primary"
          :loading="submitLoading"
          @click="handleSubmitDecisions"
        >
          提交决策
        </n-button>
      </n-card>
    </template>

    <!-- Section 3: Existing jobs list -->
    <h3 class="text-lg font-medium mb-4">同步任务列表</h3>
    <n-spin :show="loading">
      <n-data-table
        :columns="jobColumns"
        :data="jobs"
        :bordered="true"
        :single-line="false"
        size="small"
        :row-key="(row: dto.ChannelSyncJobDTO) => row.id"
      >
        <template #expand="{ rowData }">
          <n-data-table
            :columns="itemColumns"
            :data="(rowData as dto.ChannelSyncJobDTO).items || []"
            :bordered="false"
            size="small"
            :row-key="(row: dto.ChannelSyncItemDTO) => row.id"
          />
        </template>
      </n-data-table>
    </n-spin>
  </div>
</template>
