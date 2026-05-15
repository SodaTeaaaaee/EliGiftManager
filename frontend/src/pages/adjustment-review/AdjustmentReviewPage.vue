<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch, h } from "vue";
import { useRoute } from "vue-router";
import { NTag, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { listAdjustmentsByWave, recordAdjustment } from "@/shared/lib/wails/app";
import { dto } from "@/../wailsjs/go/models";

const route = useRoute();
const message = useMessage();
const waveId = computed(() => Number(route.params.waveId) || 0);

// ── List state ──

const adjustments = ref<dto.FulfillmentAdjustmentDTO[]>([]);
const loading = ref(true);
const error = ref("");

async function loadAdjustments() {
  if (!waveId.value) return;
  loading.value = true;
  error.value = "";
  try {
    adjustments.value = await listAdjustmentsByWave(waveId.value);
  } catch (e: any) {
    error.value = e?.message ?? String(e);
    message.error(`加载调整记录失败: ${error.value}`);
  } finally {
    loading.value = false;
  }
}

onMounted(loadAdjustments);

// ── Table columns ──

const kindColorMap: Record<string, string> = {
  add: "success",
  reduce: "warning",
  compensation: "info",
  remove: "error",
};

const columns: DataTableColumns<dto.FulfillmentAdjustmentDTO> = [
  { title: "ID", key: "id", width: 60 },
  {
    title: "目标类型",
    key: "targetKind",
    width: 120,
    render(row) {
      const label = row.targetKind === "fulfillment_line" ? "履约行" : "参与者";
      return h(NTag, { size: "small", bordered: false }, { default: () => label });
    },
  },
  {
    title: "目标ID",
    key: "targetId",
    width: 100,
    render(row) {
      return row.targetKind === "fulfillment_line"
        ? String(row.fulfillmentLineId ?? "-")
        : String(row.waveParticipantSnapshotId ?? "-");
    },
  },
  {
    title: "调整类型",
    key: "adjustmentKind",
    width: 120,
    render(row) {
      const color = kindColorMap[row.adjustmentKind] ?? "default";
      return h(
        NTag,
        { size: "small", type: color as any, bordered: false },
        { default: () => row.adjustmentKind },
      );
    },
  },
  {
    title: "数量变化",
    key: "quantityDelta",
    width: 100,
    render(row) {
      const v = row.quantityDelta;
      const prefix = v > 0 ? "+" : "";
      return `${prefix}${v}`;
    },
  },
  { title: "原因码", key: "reasonCode", width: 120, ellipsis: { tooltip: true } },
  { title: "备注", key: "note", ellipsis: { tooltip: true } },
  { title: "创建时间", key: "createdAt", width: 160 },
];

// ── Drawer state ──

const drawerVisible = ref(false);
const submitting = ref(false);

interface AdjustmentForm {
  targetKind: string;
  fulfillmentLineId: number | null;
  waveParticipantSnapshotId: number | null;
  adjustmentKind: string;
  quantityDelta: number;
  reasonCode: string;
  note: string;
  evidenceRef: string;
  operatorId: string;
}

const defaultForm = (): AdjustmentForm => ({
  targetKind: "fulfillment_line",
  fulfillmentLineId: null,
  waveParticipantSnapshotId: null,
  adjustmentKind: "add",
  quantityDelta: 1,
  reasonCode: "",
  note: "",
  evidenceRef: "",
  operatorId: "",
});

const form = reactive<AdjustmentForm>(defaultForm());

const adjustmentKindOptions = computed(() => {
  if (form.targetKind === "participant") {
    return [{ label: "compensation（补偿）", value: "compensation" }];
  }
  return [
    { label: "add（新增）", value: "add" },
    { label: "reduce（减少）", value: "reduce" },
    { label: "remove（移除）", value: "remove" },
  ];
});

watch(
  () => form.targetKind,
  () => {
    form.adjustmentKind =
      form.targetKind === "participant" ? "compensation" : "add";
  },
);

watch(
  () => form.adjustmentKind,
  (kind) => {
    if (kind === "remove") form.quantityDelta = 0;
  },
);

function openDrawer() {
  Object.assign(form, defaultForm());
  drawerVisible.value = true;
}

async function handleSubmit() {
  submitting.value = true;
  try {
    const payload: any = {
      waveId: waveId.value,
      targetKind: form.targetKind,
      adjustmentKind: form.adjustmentKind,
      quantityDelta: form.adjustmentKind === "remove" ? 0 : form.quantityDelta,
      reasonCode: form.reasonCode,
      note: form.note,
      evidenceRef: form.evidenceRef,
      operatorId: form.operatorId,
    };
    if (form.targetKind === "fulfillment_line") {
      payload.fulfillmentLineId = form.fulfillmentLineId;
    } else {
      payload.waveParticipantSnapshotId = form.waveParticipantSnapshotId;
    }
    await recordAdjustment(payload);
    message.success("调整记录已创建");
    drawerVisible.value = false;
    await loadAdjustments();
  } catch (e: any) {
    message.error(`提交失败: ${e?.message ?? String(e)}`);
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="p-4">
    <n-card title="履约调整审核">
      <template #header-extra>
        <n-button type="primary" @click="openDrawer">新增调整</n-button>
      </template>

      <n-data-table
        :columns="columns"
        :data="adjustments"
        :loading="loading"
        :bordered="false"
        :single-line="false"
        size="small"
      />
    </n-card>

    <n-drawer v-model:show="drawerVisible" :width="480" placement="right">
      <n-drawer-content title="新增履约调整" closable>
        <n-form label-placement="top">
          <n-form-item label="目标类型">
            <n-radio-group v-model:value="form.targetKind">
              <n-radio-button value="fulfillment_line">履约行</n-radio-button>
              <n-radio-button value="participant">参与者</n-radio-button>
            </n-radio-group>
          </n-form-item>

          <n-form-item v-if="form.targetKind === 'fulfillment_line'" label="履约行 ID">
            <n-input-number
              v-model:value="form.fulfillmentLineId"
              :min="1"
              placeholder="输入履约行 ID"
              class="w-full"
            />
          </n-form-item>

          <n-form-item v-if="form.targetKind === 'participant'" label="参与者快照 ID">
            <n-input-number
              v-model:value="form.waveParticipantSnapshotId"
              :min="1"
              placeholder="输入参与者快照 ID"
              class="w-full"
            />
          </n-form-item>

          <n-form-item label="调整类型">
            <n-select
              v-model:value="form.adjustmentKind"
              :options="adjustmentKindOptions"
            />
          </n-form-item>

          <n-form-item v-if="form.adjustmentKind !== 'remove'" label="数量变化">
            <n-input-number
              v-model:value="form.quantityDelta"
              placeholder="数量"
              class="w-full"
            />
          </n-form-item>

          <n-form-item label="原因码">
            <n-input v-model:value="form.reasonCode" placeholder="原因码" />
          </n-form-item>

          <n-form-item label="备注">
            <n-input
              v-model:value="form.note"
              type="textarea"
              placeholder="备注信息"
              :rows="3"
            />
          </n-form-item>

          <n-form-item label="证据引用">
            <n-input v-model:value="form.evidenceRef" placeholder="证据引用" />
          </n-form-item>

          <n-form-item label="操作人 ID">
            <n-input v-model:value="form.operatorId" placeholder="操作人 ID" />
          </n-form-item>
        </n-form>

        <template #footer>
          <n-button
            type="primary"
            :loading="submitting"
            @click="handleSubmit"
          >
            提交
          </n-button>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>
