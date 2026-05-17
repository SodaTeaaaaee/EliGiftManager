<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NAlert, NButton, NCard, NDataTable, NDrawer, NDrawerContent, NEmpty, NFormItem, NInput, NInputNumber, NModal, NPopconfirm, NSelect, NSpace, NSwitch, NTag, useMessage } from "naive-ui";
import type { DataTableColumns } from "naive-ui";
import { createAllocationPolicyRule, deleteAllocationPolicyRule, generateParticipants, listAllocationPolicyRules, listProductMasters, listProductsByWave, listWaveParticipantRows, reconcileWave, snapshotProductsForWave, updateAllocationPolicyRule } from "@/shared/lib/wails/app";
import type { AllocationPolicyRule, CreateAllocationPolicyRuleInput, UpdateAllocationPolicyRuleInput, SelectorPayload, ReconcileResult } from "@/entities/allocation-policy";
import { useI18n } from "@/shared/i18n";
import { dto } from "@/../wailsjs/go/models";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const { t } = useI18n();
const waveId = computed(() => Number(route.params.waveId) || 0);

const rules = ref<AllocationPolicyRule[]>([]);
const participants = ref<dto.WaveParticipantRowDTO[]>([]);
const productOptions = ref<Array<{ label: string; value: number }>>([]);
const loading = ref(false);
const reconciling = ref(false);
const reconcileResult = ref<ReconcileResult | null>(null);

const drawerVisible = ref(false);
const editingRule = ref<AllocationPolicyRule | null>(null);
const saving = ref(false);

const catalogModalVisible = ref(false);
const catalogMasters = ref<any[]>([]);
const catalogCheckedKeys = ref<Array<string | number>>([]);

const form = reactive<{
  product_id: number | null;
  selector_payload: SelectorPayload;
  product_target_ref: string;
  contribution_quantity: number;
  rule_kind: string;
  priority: number;
  active: boolean;
}>({
  product_id: null,
  selector_payload: { type: "wave_all" },
  product_target_ref: "",
  contribution_quantity: 1,
  rule_kind: "standard",
  priority: 0,
  active: true,
});

const selectorTypeOptions = [
  { label: t("allocation.selectorTypeOptions.wave_all"), value: "wave_all" },
  { label: t("allocation.selectorTypeOptions.platform_all"), value: "platform_all" },
  { label: t("allocation.selectorTypeOptions.identity_level"), value: "identity_level" },
  { label: t("allocation.selectorTypeOptions.explicit_override"), value: "explicit_override" },
];

const ruleKindOptions = [
  { label: t("allocation.ruleKindOptions.standard"), value: "standard" },
  { label: t("allocation.ruleKindOptions.supplement"), value: "supplement" },
  { label: t("allocation.ruleKindOptions.replacement"), value: "replacement" },
];

const participantOptions = computed(() =>
  participants.value.map((row) => ({
    label: `${row.displayName} · ${row.identityPlatform}`,
    value: row.waveParticipantSnapshotId,
  })),
);

function selectorTypeText(value: string) {
  const map: Record<string, string> = {
    wave_all: t("allocation.selectorTypeOptions.wave_all"),
    platform_all: t("allocation.selectorTypeOptions.platform_all"),
    identity_level: t("allocation.selectorTypeOptions.identity_level"),
    explicit_override: t("allocation.selectorTypeOptions.explicit_override"),
  };
  return map[value] || value;
}

function ruleKindText(value: string) {
  const map: Record<string, string> = {
    standard: t("allocation.ruleKindOptions.standard"),
    supplement: t("allocation.ruleKindOptions.supplement"),
    replacement: t("allocation.ruleKindOptions.replacement"),
  };
  return map[value] || value;
}

const columns = computed<DataTableColumns<AllocationPolicyRule>>(() => [
  { title: t("allocation.columns.id"), key: "id", width: 60 },
  { title: t("allocation.columns.product"), key: "product_id", width: 100 },
  { title: t("allocation.columns.selector"), key: "selector_payload", width: 180, render: (row) => selectorTypeText(row.selector_payload.type) },
  { title: t("allocation.columns.targetRef"), key: "product_target_ref" },
  { title: t("allocation.columns.qty"), key: "contribution_quantity", width: 80 },
  { title: t("allocation.columns.priority"), key: "priority", width: 80 },
  {
    title: t("allocation.columns.status"),
    key: "active",
    width: 90,
    render: (row) =>
      h(
        NTag,
        { type: row.active ? "success" : "default", size: "small", round: true },
        { default: () => (row.active ? t("allocation.statusOptions.active") : t("allocation.statusOptions.inactive")) },
      ),
  },
  {
    title: t("allocation.columns.actions"),
    key: "actions",
    width: 150,
    render(row) {
      return h(NSpace, { size: "small" }, () => [
        h(NButton, { size: "small", onClick: () => openEditDrawer(row) }, { default: () => t("allocation.editRule") }),
        h(
          NPopconfirm,
          { onPositiveClick: () => handleDelete(row) },
          {
            trigger: () => h(NButton, { size: "small", type: "error" }, { default: () => t("common.error") }),
            default: () => `${t("allocation.editRule")}?`,
          },
        ),
      ]);
    },
  },
]);

function resetForm() {
  form.product_id = null;
  form.selector_payload = { type: "wave_all" };
  form.product_target_ref = "";
  form.contribution_quantity = 1;
  form.rule_kind = "standard";
  form.priority = 0;
  form.active = true;
}

function openCreateDrawer() {
  editingRule.value = null;
  resetForm();
  drawerVisible.value = true;
}

function openEditDrawer(rule: AllocationPolicyRule) {
  editingRule.value = rule;
  form.product_id = rule.product_id;
  form.selector_payload = { ...rule.selector_payload };
  form.product_target_ref = rule.product_target_ref;
  form.contribution_quantity = rule.contribution_quantity;
  form.rule_kind = rule.rule_kind;
  form.priority = rule.priority;
  form.active = rule.active;
  drawerVisible.value = true;
}

async function loadData() {
  loading.value = true;
  try {
    const [rulesResult, participantsResult, productsResult] = await Promise.all([
      listAllocationPolicyRules(waveId.value),
      listWaveParticipantRows(waveId.value),
      listProductsByWave(waveId.value),
    ]);
    rules.value = rulesResult;
    participants.value = participantsResult;
    productOptions.value = productsResult.map((product) => ({
      label: `${product.name} (${product.factorySku})`,
      value: product.id,
    }));
  } finally {
    loading.value = false;
  }
}

async function handleSave() {
  if (!form.product_id) {
    message.warning(t("allocation.selectProductWarning"));
    return;
  }
  saving.value = true;
  try {
    if (editingRule.value) {
      const input: UpdateAllocationPolicyRuleInput = {
        id: editingRule.value.id,
        product_id: form.product_id,
        selector_payload: form.selector_payload,
        product_target_ref: form.product_target_ref,
        contribution_quantity: form.contribution_quantity,
        rule_kind: form.rule_kind,
        priority: form.priority,
        active: form.active,
      };
      await updateAllocationPolicyRule(input);
    } else {
      const input: CreateAllocationPolicyRuleInput = {
        wave_id: waveId.value,
        product_id: form.product_id,
        selector_payload: form.selector_payload,
        product_target_ref: form.product_target_ref,
        contribution_quantity: form.contribution_quantity,
        rule_kind: form.rule_kind,
        priority: form.priority,
        active: form.active,
      };
      await createAllocationPolicyRule(input);
    }
    drawerVisible.value = false;
    await loadData();
  } finally {
    saving.value = false;
  }
}

async function handleDelete(rule: AllocationPolicyRule) {
  await deleteAllocationPolicyRule(rule.id);
  await loadData();
}

async function handleReconcile() {
  reconciling.value = true;
  reconcileResult.value = null;
  try {
    await generateParticipants(waveId.value);
    reconcileResult.value = await reconcileWave(waveId.value);
    await loadData();
  } finally {
    reconciling.value = false;
  }
}

async function openCatalogModal() {
  catalogModalVisible.value = true;
  catalogMasters.value = await listProductMasters();
}

async function doAddFromCatalog() {
  await snapshotProductsForWave({
    waveId: waveId.value,
    masterIds: catalogCheckedKeys.value.map((value) => Number(value)),
  });
  catalogModalVisible.value = false;
  await loadData();
}

onMounted(loadData);
</script>

<template>
  <div class="membership-allocation-page">
    <div class="mb-6">
      <div class="app-kicker">{{ t("wave.allocation") }}</div>
      <h2 class="app-title mt-2">{{ t("allocation.title") }}</h2>
      <p class="app-copy mt-3">{{ t("allocation.subtitle") }}</p>
    </div>

    <NAlert v-if="reconcileResult && reconcileResult.failures.length > 0" type="warning" class="mb-4">
      {{ t("allocation.replayFailures") }}: {{ reconcileResult.failures.length }}
    </NAlert>

    <NCard class="mb-4" :title="t('allocation.participantContext')">
      <NEmpty v-if="participants.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="[
          { title: t('allocation.participantColumns.participant'), key: 'displayName' },
          { title: t('allocation.participantColumns.platform'), key: 'identityPlatform', width: 120 },
          { title: t('allocation.participantColumns.type'), key: 'snapshotType', width: 120 },
          { title: t('allocation.participantColumns.giftLevel'), key: 'giftLevel', width: 120 },
          { title: t('allocation.participantColumns.readyLines'), key: 'readyFulfillmentCount', width: 100 },
        ]"
        :data="participants"
        :pagination="false"
        size="small"
      />
    </NCard>

    <NCard :title="t('allocation.rules')">
      <template #header-extra>
        <NSpace>
          <NButton @click="openCreateDrawer">{{ t("allocation.addRule") }}</NButton>
          <NButton @click="openCatalogModal">{{ t("allocation.catalog") }}</NButton>
          <NButton type="primary" :loading="reconciling" @click="handleReconcile">
            {{ t("allocation.execute") }}
          </NButton>
        </NSpace>
      </template>

      <NEmpty v-if="!loading && rules.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="columns"
        :data="rules"
        :loading="loading"
        :pagination="false"
        size="small"
      />
    </NCard>

    <div class="mt-4 flex justify-between">
      <NButton @click="router.push(`/waves/${waveId}`)">{{ t("wave.prevStep") }}</NButton>
      <NSpace>
        <NButton secondary @click="router.push(`/waves/${waveId}`)">{{ t("wave.backToOverview") }}</NButton>
        <NButton type="primary" @click="router.push(`/waves/${waveId}`)">{{ t("wave.nextStep") }}</NButton>
      </NSpace>
    </div>

    <NDrawer v-model:show="drawerVisible" :width="520" placement="right">
      <NDrawerContent :title="editingRule ? 'Edit Rule' : 'Create Rule'" closable>
        <NSpace vertical :size="16">
          <NFormItem :label="t('allocation.product')">
            <NSelect v-model:value="form.product_id" :options="productOptions" filterable />
          </NFormItem>
          <NFormItem :label="t('allocation.selectorType')">
            <NSelect
              :value="form.selector_payload.type"
              :options="selectorTypeOptions"
              @update:value="(value) => form.selector_payload = { type: value as SelectorPayload['type'] }"
            />
          </NFormItem>
          <NFormItem v-if="form.selector_payload.type === 'platform_all'" :label="t('allocation.allocationPlatform')">
            <NInput v-model:value="form.selector_payload.platform" />
          </NFormItem>
          <template v-if="form.selector_payload.type === 'identity_level'">
            <NFormItem :label="t('allocation.allocationPlatform')">
              <NInput v-model:value="form.selector_payload.platform" />
            </NFormItem>
            <NFormItem :label="t('allocation.allocationLevel')">
              <NInput v-model:value="form.selector_payload.level" />
            </NFormItem>
          </template>
          <NFormItem v-if="form.selector_payload.type === 'explicit_override'" :label="t('allocation.participants')">
            <NSelect
              multiple
              :value="form.selector_payload.participant_ids || []"
              :options="participantOptions"
              @update:value="(value) => form.selector_payload.participant_ids = value as number[]"
            />
          </NFormItem>
          <NFormItem :label="t('allocation.targetRef')">
            <NInput v-model:value="form.product_target_ref" />
          </NFormItem>
          <NFormItem :label="t('allocation.quantity')">
            <NInputNumber v-model:value="form.contribution_quantity" class="w-full" />
          </NFormItem>
          <NFormItem :label="t('allocation.ruleKind')">
            <NSelect v-model:value="form.rule_kind" :options="ruleKindOptions" />
          </NFormItem>
          <NFormItem :label="t('allocation.priority')">
            <NInputNumber v-model:value="form.priority" :min="0" class="w-full" />
          </NFormItem>
          <NFormItem :label="t('allocation.active')">
            <NSwitch v-model:value="form.active" />
          </NFormItem>
          <NButton type="primary" :loading="saving" @click="handleSave">
            {{ t("common.save") }}
          </NButton>
        </NSpace>
      </NDrawerContent>
    </NDrawer>

    <NModal v-model:show="catalogModalVisible" preset="card" :title="t('allocation.snapshotProducts')" style="width: 680px">
      <NDataTable
        :columns="[
          { type: 'selection' as const },
          { title: t('allocation.catalogColumns.id'), key: 'id', width: 60 },
          { title: t('allocation.catalogColumns.name'), key: 'name' },
          { title: t('allocation.catalogColumns.factorySku'), key: 'factorySku', width: 140 },
        ]"
        :data="catalogMasters"
        :row-key="(row: any) => row.id"
        v-model:checked-row-keys="catalogCheckedKeys"
        size="small"
      />
      <template #footer>
        <NSpace justify="end">
          <NButton @click="catalogModalVisible = false">{{ t("common.cancel") }}</NButton>
          <NButton type="primary" @click="doAddFromCatalog">{{ t("common.save") }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
