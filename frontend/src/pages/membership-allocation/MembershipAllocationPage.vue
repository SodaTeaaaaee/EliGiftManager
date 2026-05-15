<script setup lang="ts">
import { ref, reactive, computed, onMounted, h } from 'vue'
import { useRoute } from 'vue-router'
import { NButton, NPopconfirm, NTag, NModal, NDataTable, NSpace, NCollapse, NCollapseItem, NList, NListItem, useMessage } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import {
  listAllocationPolicyRules,
  createAllocationPolicyRule,
  updateAllocationPolicyRule,
  deleteAllocationPolicyRule,
  reconcileWave,
  generateParticipants,
  listProductsByWave,
  listProductMasters,
  snapshotProductsForWave,
} from '@/shared/lib/wails/app'
import type {
  AllocationPolicyRule,
  CreateAllocationPolicyRuleInput,
  UpdateAllocationPolicyRuleInput,
  SelectorPayload,
  ReconcileResult,
} from '@/entities/allocation-policy'
import { dto } from '@/../wailsjs/go/models'

const route = useRoute()
const message = useMessage()
const waveId = computed(() => Number(route.params.waveId) || 0)

// ── Product options ──

const productOptions = ref<Array<{ label: string; value: number }>>([])

async function loadProducts() {
  if (!waveId.value) return
  try {
    const products = await listProductsByWave(waveId.value)
    productOptions.value = products.map((p: dto.ProductDTO) => ({
      label: `${p.name} (${p.factorySku})`,
      value: p.id,
    }))
  } catch {
    // fallback — products not yet snapshotted
  }
}

// ── Catalog modal (add from product master) ──

const catalogModalVisible = ref(false)
const catalogMasters = ref<any[]>([])
const catalogLoading = ref(false)
const catalogCheckedKeys = ref<Array<string | number>>([])
const catalogSnapshotting = ref(false)

async function openCatalogModal() {
  catalogCheckedKeys.value = []
  catalogModalVisible.value = true
  catalogLoading.value = true
  try {
    catalogMasters.value = await listProductMasters()
  } catch (e: any) {
    message.error(`加载商品目录失败: ${e?.message ?? e}`)
  } finally {
    catalogLoading.value = false
  }
}

async function doAddFromCatalog() {
  if (catalogCheckedKeys.value.length === 0) return
  catalogSnapshotting.value = true
  try {
    const masterIds = catalogCheckedKeys.value.map((k) => Number(k))
    await snapshotProductsForWave({ waveId: waveId.value, masterIds })
    message.success(`已添加 ${masterIds.length} 个商品到波次`)
    catalogModalVisible.value = false
    await loadProducts()
  } catch (e: any) {
    message.error(`添加失败: ${e?.message ?? e}`)
  } finally {
    catalogSnapshotting.value = false
  }
}

const catalogColumns: DataTableColumns<any> = [
  { type: 'selection' as const },
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name' },
  { title: '工厂 SKU', key: 'factorySku', width: 140 },
  { title: '类型', key: 'productKind', width: 100 },
]

// ── List state ──

const rules = ref<AllocationPolicyRule[]>([])
const loading = ref(false)

async function loadRules() {
  if (!waveId.value) return
  loading.value = true
  try {
    rules.value = await listAllocationPolicyRules(waveId.value)
  } catch (e: any) {
    message.error(`加载规则失败: ${e?.message ?? e}`)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadRules()
  loadProducts()
})

// ── Drawer state ──

const drawerVisible = ref(false)
const editingRule = ref<AllocationPolicyRule | null>(null)
const saving = ref(false)

interface RuleForm {
  product_id: number | null
  selector_payload: SelectorPayload
  product_target_ref: string
  contribution_quantity: number
  rule_kind: string
  priority: number
  active: boolean
}

const defaultForm = (): RuleForm => ({
  product_id: null,
  selector_payload: { type: 'wave_all' },
  product_target_ref: '',
  contribution_quantity: 1,
  rule_kind: 'standard',
  priority: 0,
  active: true,
})

const form = reactive<RuleForm>(defaultForm())

function openCreateDrawer() {
  editingRule.value = null
  Object.assign(form, defaultForm())
  drawerVisible.value = true
}

function openEditDrawer(rule: AllocationPolicyRule) {
  editingRule.value = rule
  form.product_id = rule.product_id
  form.selector_payload = { ...rule.selector_payload }
  if (rule.selector_payload.participant_ids) {
    form.selector_payload.participant_ids = [...rule.selector_payload.participant_ids]
  }
  form.product_target_ref = rule.product_target_ref
  form.contribution_quantity = rule.contribution_quantity
  form.rule_kind = rule.rule_kind
  form.priority = rule.priority
  form.active = rule.active
  drawerVisible.value = true
}

async function handleSave() {
  if (!form.product_id) {
    message.warning('请填写商品 ID')
    return
  }
  saving.value = true
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
      }
      await updateAllocationPolicyRule(input)
      message.success('规则已更新')
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
      }
      await createAllocationPolicyRule(input)
      message.success('规则已创建')
    }
    drawerVisible.value = false
    await loadRules()
  } catch (e: any) {
    message.error(`保存失败: ${e?.message ?? e}`)
  } finally {
    saving.value = false
  }
}

// ── Delete ──

async function handleDelete(rule: AllocationPolicyRule) {
  try {
    await deleteAllocationPolicyRule(rule.id)
    message.success('规则已删除')
    await loadRules()
  } catch (e: any) {
    message.error(`删除失败: ${e?.message ?? e}`)
  }
}

// ── Reconcile ──

const reconciling = ref(false)
const reconcileResult = ref<ReconcileResult | null>(null)

async function handleReconcile() {
  reconciling.value = true
  reconcileResult.value = null
  try {
    await generateParticipants(waveId.value)
    reconcileResult.value = await reconcileWave(waveId.value)
    message.success('分配执行完成')
    await loadRules()
  } catch (e: any) {
    message.error(`执行分配失败: ${e?.message ?? e}`)
  } finally {
    reconciling.value = false
  }
}

// ── Selector type helpers ──

const selectorTypeOptions = [
  { label: '全波次', value: 'wave_all' },
  { label: '按平台', value: 'platform_all' },
  { label: '身份等级', value: 'identity_level' },
  { label: '指定参与者', value: 'explicit_override' },
]

const ruleKindOptions = [
  { label: '标准', value: 'standard' },
  { label: '补发', value: 'supplement' },
  { label: '替换', value: 'replacement' },
]

function selectorTypeLabel(type: string): string {
  return selectorTypeOptions.find((o) => o.value === type)?.label ?? type
}

function handleSelectorTypeChange(type: string) {
  form.selector_payload = { type: type as SelectorPayload['type'] }
}

// ── Participant IDs input ──

const participantIdsText = computed({
  get() {
    return (form.selector_payload.participant_ids ?? []).join(', ')
  },
  set(val: string) {
    const ids = val
      .split(/[,，\s]+/)
      .map((s) => Number(s.trim()))
      .filter((n) => !isNaN(n) && n > 0)
    form.selector_payload.participant_ids = ids
  },
})

// ── Table columns ──

const columns = computed<DataTableColumns<AllocationPolicyRule>>(() => [
  {
    title: 'ID',
    key: 'id',
    width: 60,
  },
  {
    title: '商品 ID',
    key: 'product_id',
    width: 80,
  },
  {
    title: '选择器类型',
    key: 'selector_payload',
    width: 120,
    render(row) {
      return selectorTypeLabel(row.selector_payload.type)
    },
  },
  {
    title: '目标引用',
    key: 'product_target_ref',
    ellipsis: { tooltip: true },
  },
  {
    title: '数量',
    key: 'contribution_quantity',
    width: 70,
  },
  {
    title: '规则类型',
    key: 'rule_kind',
    width: 90,
  },
  {
    title: '优先级',
    key: 'priority',
    width: 70,
  },
  {
    title: '状态',
    key: 'active',
    width: 70,
    render(row) {
      return h(
        NTag,
        { type: row.active ? 'success' : 'default', size: 'small' },
        { default: () => (row.active ? '启用' : '停用') },
      )
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 140,
    render(row) {
      return h('div', { style: 'display:flex;gap:8px' }, [
        h(
          NButton,
          { size: 'small', onClick: () => openEditDrawer(row) },
          { default: () => '编辑' },
        ),
        h(
          NPopconfirm,
          { onPositiveClick: () => handleDelete(row) },
          {
            trigger: () =>
              h(
                NButton,
                { size: 'small', type: 'error' },
                { default: () => '删除' },
              ),
            default: () => '确认删除此规则？',
          },
        ),
      ])
    },
  },
])
</script>

<template>
  <div class="p-4">
    <n-space vertical size="large">
      <n-space justify="space-between" align="center">
        <h2 class="text-lg font-medium">会员分配规则</h2>
        <n-space>
          <n-button @click="openCreateDrawer">添加规则</n-button>
          <n-button @click="openCatalogModal">从商品目录添加</n-button>
          <n-button
            type="primary"
            :loading="reconciling"
            @click="handleReconcile"
          >
            执行分配
          </n-button>
        </n-space>
      </n-space>

      <!-- Reconcile result -->
      <n-alert
        v-if="reconcileResult && reconcileResult.failures.length === 0"
        type="success"
        title="分配执行成功"
        closable
        @close="reconcileResult = null"
      >
        创建 {{ reconcileResult.created }} 条，删除 {{ reconcileResult.deleted }} 条，重放 {{ reconcileResult.replayed_count }} 条。
      </n-alert>

      <n-alert
        v-if="reconcileResult && reconcileResult.failures.length > 0"
        type="warning"
        title="分配执行完成（部分失败）"
        closable
        @close="reconcileResult = null"
      >
        <p>
          创建 {{ reconcileResult.created }} 条，删除 {{ reconcileResult.deleted }} 条，重放 {{ reconcileResult.replayed_count }} 条。
        </p>
        <n-collapse style="margin-top: 8px">
          <n-collapse-item title="失败详情" name="failures">
            <n-list bordered>
              <n-list-item
                v-for="f in reconcileResult.failures"
                :key="f.adjustment_id"
              >
                Adjustment #{{ f.adjustment_id }}: {{ f.reason }}
              </n-list-item>
            </n-list>
          </n-collapse-item>
        </n-collapse>
      </n-alert>

      <!-- Rules table -->
      <n-data-table
        :columns="columns"
        :data="rules"
        :loading="loading"
        :bordered="true"
        :single-line="false"
        size="small"
      />

      <!-- Create/Edit drawer -->
      <n-drawer v-model:show="drawerVisible" :width="480" placement="right">
        <n-drawer-content :title="editingRule ? '编辑规则' : '添加规则'">
          <n-space vertical size="large">
            <n-form-item label="商品">
              <n-select
                v-model:value="form.product_id"
                :options="productOptions"
                placeholder="选择商品"
                filterable
                style="width: 100%"
              />
            </n-form-item>

            <n-form-item label="选择器类型">
              <n-select
                :value="form.selector_payload.type"
                :options="selectorTypeOptions"
                @update:value="handleSelectorTypeChange"
              />
            </n-form-item>

            <!-- platform_all: platform -->
            <n-form-item
              v-if="form.selector_payload.type === 'platform_all'"
              label="平台"
            >
              <n-input
                v-model:value="form.selector_payload.platform"
                placeholder="平台标识"
              />
            </n-form-item>

            <!-- identity_level: platform + level -->
            <template v-if="form.selector_payload.type === 'identity_level'">
              <n-form-item label="平台">
                <n-input
                  v-model:value="form.selector_payload.platform"
                  placeholder="平台标识"
                />
              </n-form-item>
              <n-form-item label="等级">
                <n-input
                  v-model:value="form.selector_payload.level"
                  placeholder="身份等级"
                />
              </n-form-item>
            </template>

            <!-- explicit_override: participant_ids -->
            <n-form-item
              v-if="form.selector_payload.type === 'explicit_override'"
              label="参与者 ID 列表"
            >
              <n-input
                :value="participantIdsText"
                @update:value="(v: string) => (participantIdsText = v)"
                type="textarea"
                placeholder="逗号分隔的参与者 ID，如: 1, 2, 3"
                :autosize="{ minRows: 2, maxRows: 4 }"
              />
            </n-form-item>

            <n-form-item label="目标引用">
              <n-input
                v-model:value="form.product_target_ref"
                placeholder="product_target_ref"
              />
            </n-form-item>

            <n-form-item label="分配数量">
              <n-input-number
                v-model:value="form.contribution_quantity"
                style="width: 100%"
                placeholder="正数为加赠，负数为规则层抵扣"
              />
            </n-form-item>

            <n-form-item label="规则类型">
              <n-select
                v-model:value="form.rule_kind"
                :options="ruleKindOptions"
              />
            </n-form-item>

            <n-form-item label="优先级">
              <n-input-number
                v-model:value="form.priority"
                :min="0"
                style="width: 100%"
              />
            </n-form-item>

            <n-form-item label="启用">
              <n-switch v-model:value="form.active" />
            </n-form-item>
          </n-space>

          <template #footer>
            <n-space justify="end">
              <n-button @click="drawerVisible = false">取消</n-button>
              <n-button type="primary" :loading="saving" @click="handleSave">
                保存
              </n-button>
            </n-space>
          </template>
        </n-drawer-content>
      </n-drawer>

      <!-- Catalog modal: add products from master catalog -->
      <n-modal v-model:show="catalogModalVisible" preset="card" title="从商品目录添加到波次" style="width: 640px">
        <n-data-table
          :columns="catalogColumns"
          :data="catalogMasters"
          :loading="catalogLoading"
          :row-key="(row: any) => row.id"
          v-model:checked-row-keys="catalogCheckedKeys"
          size="small"
          :max-height="400"
        />
        <template #footer>
          <n-space justify="end" style="margin-top: 12px">
            <n-button @click="catalogModalVisible = false">取消</n-button>
            <n-button
              type="primary"
              :loading="catalogSnapshotting"
              :disabled="catalogCheckedKeys.length === 0"
              @click="doAddFromCatalog"
            >
              添加选中 ({{ catalogCheckedKeys.length }})
            </n-button>
          </n-space>
        </template>
      </n-modal>
    </n-space>
  </div>
</template>
