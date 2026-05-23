<script setup lang="ts">
import { computed, onMounted, reactive, ref, h } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NAlert, NButton, NCard, NDataTable, NDrawer, NDrawerContent, NEmpty, NFormItem, NInput, NInputNumber, NModal, NPopconfirm, NSelect, NSpace, NSwitch, NTag, NScrollbar, NList, NListItem, useMessage } from "naive-ui";
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

// Participant Search and Filter State
const searchKeyword = ref("");
const platformFilter = ref<string | null>(null);

const filteredParticipants = computed(() => {
  return participants.value.filter((p) => {
    // If route.params.demandKind is set, ensure participant has that demand kind
    const demandKindParam = route.params.demandKind as string;
    if (demandKindParam && (!p.demandKinds || !p.demandKinds.includes(demandKindParam))) {
      return false;
    }

    const matchesKeyword = !searchKeyword.value || 
      p.displayName.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
      p.identityValue.toLowerCase().includes(searchKeyword.value.toLowerCase());
    const matchesPlatform = !platformFilter.value || p.identityPlatform === platformFilter.value;
    return matchesKeyword && matchesPlatform;
  });
});

const platformOptions = computed(() => {
  const platforms = new Set(participants.value.map((p) => p.identityPlatform));
  return Array.from(platforms).map((p) => ({ label: p, value: p }));
});

const participantsPagination = reactive({
  page: 1,
  pageSize: 10,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    participantsPagination.page = page;
  },
  onUpdatePageSize: (pageSize: number) => {
    participantsPagination.pageSize = pageSize;
    participantsPagination.page = 1;
  }
});

const rulesPagination = reactive({
  page: 1,
  pageSize: 10,
  onChange: (page: number) => {
    rulesPagination.page = page;
  }
});

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

// Sandbox matches preview
const sandboxMatchedParticipants = computed(() => {
  const payload = form.selector_payload;
  if (!payload || !payload.type) return [];
  return participants.value.filter((p) => {
    if (payload.type === "wave_all") return true;
    if (payload.type === "platform_all") {
      return p.identityPlatform.toLowerCase() === (payload.platform || "").toLowerCase();
    }
    if (payload.type === "identity_level") {
      return p.identityPlatform.toLowerCase() === (payload.platform || "").toLowerCase() &&
             p.giftLevel.toLowerCase() === (payload.level || "").toLowerCase();
    }
    if (payload.type === "explicit_override") {
      return (payload.participant_ids || []).includes(p.waveParticipantSnapshotId);
    }
    return false;
  });
});

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
  { title: t("allocation.columns.id"), key: "id", width: 65 },
  { 
    title: t("allocation.columns.product"), 
    key: "product_id", 
    width: 150, 
    render: (row) => {
      const option = productOptions.value.find(o => o.value === row.product_id);
      return option ? option.label : `Product #${row.product_id}`;
    }
  },
  { 
    title: t("allocation.columns.selector"), 
    key: "selector_payload", 
    width: 220, 
    render: (row) => {
      const typeText = selectorTypeText(row.selector_payload.type);
      let details = "";
      if (row.selector_payload.type === "platform_all") {
        details = ` (${row.selector_payload.platform})`;
      } else if (row.selector_payload.type === "identity_level") {
        details = ` (${row.selector_payload.platform}: Lvl ${row.selector_payload.level})`;
      } else if (row.selector_payload.type === "explicit_override") {
        details = ` (${row.selector_payload.participant_ids?.length || 0} selected)`;
      }
      return `${typeText}${details}`;
    } 
  },
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
        { type: row.active ? "success" : "default", size: "small", round: true, bordered: false },
        { default: () => (row.active ? t("allocation.statusOptions.active") : t("allocation.statusOptions.inactive")) },
      ),
  },
  {
    title: t("allocation.columns.actions"),
    key: "actions",
    width: 160,
    render(row) {
      return h(NSpace, { size: "small" }, () => [
        h(NButton, { size: "small", secondary: true, onClick: () => openEditDrawer(row) }, { default: () => t("allocation.editRule") }),
        h(
          NPopconfirm,
          { onPositiveClick: () => handleDelete(row) },
          {
            trigger: () => h(NButton, { size: "small", type: "error", secondary: true }, { default: () => t("common.delete") }),
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
    message.success("Allocation reconciled successfully!");
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
  <div class="membership-allocation-page flex flex-col gap-5">
    <div class="mb-2">
      <div class="app-kicker">{{ t("wave.allocation") }}</div>
      <h2 class="app-title mt-2">{{ t("allocation.title") }}</h2>
      <p class="app-copy mt-2">{{ t("allocation.subtitle") }}</p>
    </div>

    <NAlert v-if="reconcileResult && reconcileResult.failures.length > 0" type="warning">
      {{ t("allocation.replayFailures") }}: {{ reconcileResult.failures.length }}
    </NAlert>

    <!-- Wave Participants Card (Paginated, Searchable) -->
    <NCard class="glow-card" :title="t('allocation.participantContext')">
      <template #header-extra>
        <NSpace align="center" :size="12">
          <NInput 
            v-model:value="searchKeyword" 
            placeholder="Search nickname..." 
            clearable 
            size="small" 
            style="width: 200px" 
          />
          <NSelect
            v-model:value="platformFilter"
            :options="platformOptions"
            placeholder="Filter Platform"
            clearable
            size="small"
            style="width: 150px"
          />
        </NSpace>
      </template>
      <NEmpty v-if="participants.length === 0" :description="t('common.empty')" />
      <NDataTable
        v-else
        :columns="[
          { title: t('allocation.participantColumns.participant'), key: 'displayName' },
          { title: t('allocation.participantColumns.platform'), key: 'identityPlatform', width: 130 },
          { title: t('allocation.participantColumns.type'), key: 'snapshotType', width: 130 },
          { title: t('allocation.participantColumns.giftLevel'), key: 'giftLevel', width: 120 },
          { title: t('allocation.participantColumns.readyLines'), key: 'readyFulfillmentCount', width: 120 },
        ]"
        :data="filteredParticipants"
        :pagination="participantsPagination"
        size="small"
      />
    </NCard>

    <!-- Allocation Rules Card -->
    <NCard class="glow-card" :title="t('allocation.rules')">
      <template #header-extra>
        <NSpace>
          <NButton size="small" secondary @click="openCreateDrawer">{{ t("allocation.addRule") }}</NButton>
          <NButton size="small" secondary @click="openCatalogModal">{{ t("allocation.catalog") }}</NButton>
          <NButton size="small" type="primary" :loading="reconciling" @click="handleReconcile">
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
        :pagination="rulesPagination"
        size="small"
      />
    </NCard>

    <div class="flex justify-between mt-4">
      <NButton @click="router.push(`/waves/${waveId}`)">{{ t("wave.prevStep") }}</NButton>
      <NSpace>
        <NButton secondary @click="router.push(`/waves/${waveId}`)">{{ t("wave.backToOverview") }}</NButton>
        <NButton type="primary" @click="router.push(`/waves/${waveId}/demand-mapping`)">{{ t("wave.nextStep") }}</NButton>
      </NSpace>
    </div>

    <!-- Right Drawer for Creating / Editing Rules -->
    <NDrawer v-model:show="drawerVisible" :width="500" placement="right">
      <NDrawerContent :title="editingRule ? 'Edit Allocation Rule' : 'Create Allocation Rule'" closable>
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
            <NInput v-model:value="form.selector_payload.platform" placeholder="e.g. patreon, fanbox" />
          </NFormItem>
          
          <template v-if="form.selector_payload.type === 'identity_level'">
            <NFormItem :label="t('allocation.allocationPlatform')">
              <NInput v-model:value="form.selector_payload.platform" placeholder="e.g. patreon" />
            </NFormItem>
            <NFormItem :label="t('allocation.allocationLevel')">
              <NInput v-model:value="form.selector_payload.level" placeholder="e.g. Gold Tier" />
            </NFormItem>
          </template>
          
          <NFormItem v-if="form.selector_payload.type === 'explicit_override'" :label="t('allocation.participants')">
            <NSelect
              multiple
              :value="form.selector_payload.participant_ids || []"
              :options="participantOptions"
              @update:value="(value) => form.selector_payload.participant_ids = value as number[]"
              filterable
            />
          </NFormItem>
          
          <NFormItem :label="t('allocation.targetRef')">
            <NInput v-model:value="form.product_target_ref" placeholder="Optional external tag reference" />
          </NFormItem>
          <NFormItem :label="t('allocation.quantity')">
            <NInputNumber v-model:value="form.contribution_quantity" :min="1" class="w-full" />
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

          <!-- Interactive Sandbox Match Preview -->
          <div class="sandbox-preview border-t border-slate-700/10 dark:border-slate-700/30 pt-4 mt-2">
            <div class="flex justify-between items-center mb-3">
              <span class="text-sm font-bold text-slate-500 uppercase tracking-wider">Rule Tester Sandbox</span>
              <NTag size="small" :type="sandboxMatchedParticipants.length ? 'info' : 'default'" :bordered="false">
                {{ sandboxMatchedParticipants.length }} Matches
              </NTag>
            </div>
            
            <NScrollbar style="max-height: 160px; border: 1px solid rgba(148, 163, 184, 0.12); padding: 8px; border-radius: 8px;">
              <div v-if="!form.product_id" class="text-xs text-slate-400 text-center py-4">
                Select a product to activate sandbox testing.
              </div>
              <div v-else-if="sandboxMatchedParticipants.length === 0" class="text-xs text-slate-400 text-center py-4">
                No participants match this selector in the current wave.
              </div>
              <NList v-else size="small" hoverable>
                <NListItem v-for="p in sandboxMatchedParticipants" :key="p.waveParticipantSnapshotId" style="padding: 4px 8px;">
                  <div class="flex justify-between items-center text-xs">
                    <span class="font-medium">{{ p.displayName }}</span>
                    <NSpace :size="4">
                      <NTag size="tiny" :bordered="false">{{ p.identityPlatform }}</NTag>
                      <NTag size="tiny" type="info" :bordered="false" v-if="p.giftLevel">{{ p.giftLevel }}</NTag>
                    </NSpace>
                  </div>
                </NListItem>
              </NList>
            </NScrollbar>
          </div>

          <NButton type="primary" :loading="saving" @click="handleSave" class="w-full mt-4">
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

<style scoped>
.glow-card {
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.12);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.02);
}

:root[data-theme='dark'] .glow-card {
  border: 1px solid rgba(255, 255, 255, 0.05);
  background: #111827;
}

.sandbox-preview {
  background: rgba(148, 163, 184, 0.03);
  padding: 12px;
  border-radius: 8px;
}
</style>
