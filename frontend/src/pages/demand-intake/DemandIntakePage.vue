<template>
  <div class="demand-intake-page">
    <h1 class="text-xl font-medium mb-4">需求导入</h1>

    <n-space vertical size="large">
      <n-card title="创建演示需求单">
        <n-space vertical>
          <n-button
            type="primary"
            @click="createDemoDemand"
            :loading="loading"
          >
            创建一条最低限度的需求文档
          </n-button>

          <n-alert v-if="result" type="success" title="创建成功">
            <p>需求单 ID: {{ result.id }}</p>
            <p>Kind: {{ result.kind }}</p>
            <p>SourceDocumentNo: {{ result.sourceDocumentNo }}</p>
          </n-alert>

          <n-alert v-if="error" type="error" :title="error" />

          <n-card v-if="result" title="返回的 DTO 字段" size="small">
            <n-data-table
              :columns="detailColumns"
              :data="[result]"
              :pagination="false"
              size="small"
            />
          </n-card>
        </n-space>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { NCard, NButton, NSpace, NAlert, NDataTable } from "naive-ui";
import { importDemandDocument } from "@/shared/lib/wails/app";
import { dto } from "@/../wailsjs/go/models";

const loading = ref(false);
const result = ref<dto.DemandDocumentDTO | null>(null);
const error = ref<string | null>(null);

const detailColumns = [
  { title: "ID", key: "id", width: 60 },
  { title: "Kind", key: "kind", width: 180 },
  { title: "CaptureMode", key: "captureMode", width: 120 },
  { title: "SourceChannel", key: "sourceChannel" },
  { title: "SourceDocumentNo", key: "sourceDocumentNo" },
];

async function createDemoDemand() {
  loading.value = true;
  error.value = null;
  result.value = null;

  try {
    const input = {
      kind: "membership_entitlement",
      captureMode: "manual_entry",
      sourceChannel: "demo",
      sourceDocumentNo: "DEMO-" + Date.now(),
      lines: [
        {
          lineType: "entitlement_rule",
          obligationTriggerKind: "periodic_membership",
          entitlementAuthority: "local_policy",
          routingDisposition: "accepted",
          externalTitle: "演示权益",
          requestedQuantity: 1,
        },
      ],
    };
    result.value = await importDemandDocument(input);
  } catch (e: any) {
    error.value = e?.message ?? String(e);
  } finally {
    loading.value = false;
  }
}
</script>
