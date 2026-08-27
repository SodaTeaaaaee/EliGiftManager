<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NDataTable,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NRadio,
  NRadioGroup,
  NSelect,
  NSpace,
  NSpin,
  NSwitch,
  NTag,
} from 'naive-ui'
import { SectionCard, StatCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { StatusBadge } from '@/shared/ui/status'
import { useFeedback } from '@/shared/ui/feedback'
import {
  addException,
  createGrant,
  deleteException,
  deleteQuantitySplitRule,
  deleteRule,
  getProductTotals,
  listCustomers,
  listPlatforms,
  listProducts,
  listQuantitySplitRules,
  listRules,
  upsertQuantitySplitRule,
  upsertRule,
} from '@/shared/api/bridge'
import type {
  CustomerProfile,
  EntitlementRule,
  Platform,
  ProductItem,
  ProductTotal,
  QuantitySplitRule,
  Wave,
} from '@/entities/models'

const props = defineProps<{
  waveId: number
  wave?: Wave | null
}>()

const emit = defineEmits<{
  refresh: []
}>()

const { t } = useI18n()
const router = useRouter()
const feedback = useFeedback()

function errMsg(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

const loading = ref(false)
const actionLoading = ref(false)
const rules = ref<EntitlementRule[]>([])
const totals = ref<ProductTotal[]>([])
const products = ref<ProductItem[]>([])
const platforms = ref<Platform[]>([])
const customers = ref<CustomerProfile[]>([])
const splitRules = ref<QuantitySplitRule[]>([])

const showRuleModal = ref(false)
const ruleForm = ref({
  ID: 0,
  ProductID: null as number | null,
  selectorType: 'platform_level',
  platformId: null as number | null,
  level: '舰长',
  instanceId: null as number | null,
  quantity: 1,
  active: true,
})

const showGrantModal = ref(false)
const grantForm = ref({
  customerId: null as number | null,
  productId: null as number | null,
  quantity: 1,
})

// ── Per-instance entitlement exceptions ──

// The backend exposes AddException/DeleteException only — no list or
// instance-lookup binding yet — so this stays a minimal manual form.
const exceptionForm = ref({
  productId: null as number | null,
  instanceId: null as number | null,
  quantity: 1,
  note: '',
})
const deleteExceptionId = ref<number | null>(null)

// ── Quantity split rules ──

interface SplitComponentDraft {
  productItemId: number | null
  quantity: number
}

const showSplitModal = ref(false)
const splitForm = ref({
  ID: 0,
  platformId: null as number | null,
  externalKey: '',
  components: [] as SplitComponentDraft[],
})

async function loadData() {
  if (!props.waveId) return
  loading.value = true
  try {
    const [ruleRes, totalRes, prodRes, platRes, custRes, splitRes] = await Promise.all([
      listRules(props.waveId),
      getProductTotals(props.waveId),
      listProducts(),
      listPlatforms(),
      listCustomers(),
      listQuantitySplitRules(props.waveId),
    ])
    rules.value = ruleRes
    totals.value = totalRes
    products.value = prodRes
    platforms.value = platRes
    customers.value = custRes
    splitRules.value = splitRes
  } catch (err) {
    console.error('Failed to load wave rules:', err)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.waveId,
  () => {
    void loadData()
  },
)

onMounted(() => {
  void loadData()
})

const productOptions = computed(() =>
  products.value.map((p) => ({
    label: `${p.Name} (${p.FactorySKU})`,
    value: p.ID,
  })),
)

const platformOptions = computed(() =>
  platforms.value.map((p) => ({
    label: `${p.Name} (${p.Key})`,
    value: p.ID,
  })),
)

/** Retail/source platforms — the side a quantity split key matches against. */
const sourcePlatformOptions = computed(() =>
  platforms.value
    .filter((p) => p.Kind === 'source')
    .map((p) => ({
      label: `${p.Name} (${p.Key})`,
      value: p.ID,
    })),
)

const customerOptions = computed(() =>
  customers.value.map((c) => ({
    label: `${c.DisplayName} (ID: ${c.ID})`,
    value: c.ID,
  })),
)

function openCreateRule() {
  ruleForm.value = {
    ID: 0,
    ProductID: products.value[0]?.ID ?? null,
    selectorType: 'platform_level',
    platformId: platforms.value[0]?.ID ?? null,
    level: '舰长',
    instanceId: null,
    quantity: 1,
    active: true,
  }
  showRuleModal.value = true
}

function openEditRule(rule: EntitlementRule) {
  ruleForm.value = {
    ID: rule.ID,
    ProductID: rule.ProductID,
    selectorType: rule.Selector.type,
    platformId: rule.Selector.platform_id ?? platforms.value[0]?.ID ?? null,
    level: rule.Selector.level ?? '',
    instanceId: rule.Selector.instance_id ?? null,
    quantity: rule.Quantity,
    active: rule.Active,
  }
  showRuleModal.value = true
}

const isEditingRule = computed(() => ruleForm.value.ID !== 0)

async function handleSaveRule() {
  if (!ruleForm.value.ProductID) return
  actionLoading.value = true
  try {
    await upsertRule({
      ID: ruleForm.value.ID,
      WaveID: props.waveId,
      ProductID: ruleForm.value.ProductID,
      Selector: {
        type: ruleForm.value.selectorType,
        platform_id: ruleForm.value.platformId ?? undefined,
        level: ruleForm.value.level || undefined,
        instance_id: ruleForm.value.instanceId ?? undefined,
      },
      Quantity: ruleForm.value.quantity,
      Active: ruleForm.value.active,
    })
    showRuleModal.value = false
    await loadData()
    emit('refresh')
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

async function handleDeleteRule(rule: EntitlementRule) {
  if (!window.confirm(t('waveWorkspace.deleteRuleConfirm'))) return
  actionLoading.value = true
  try {
    await deleteRule(rule.ID)
    feedback.success(t('waveWorkspace.ruleDeleteSuccess'))
    await loadData()
    emit('refresh')
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

// ── Per-instance exception flow ──

async function handleAddException() {
  if (!exceptionForm.value.productId || !exceptionForm.value.instanceId) return
  actionLoading.value = true
  try {
    await addException({
      WaveID: props.waveId,
      ProductID: exceptionForm.value.productId,
      InstanceID: exceptionForm.value.instanceId,
      Quantity: exceptionForm.value.quantity,
      Note: exceptionForm.value.note.trim(),
    })
    feedback.success(t('waveWorkspace.exceptionAddSuccess'))
    exceptionForm.value = {
      productId: exceptionForm.value.productId,
      instanceId: null,
      quantity: 1,
      note: '',
    }
    await loadData()
    emit('refresh')
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

async function handleDeleteException() {
  if (!deleteExceptionId.value) return
  actionLoading.value = true
  try {
    await deleteException(deleteExceptionId.value)
    feedback.success(t('waveWorkspace.exceptionDeleteSuccess'))
    deleteExceptionId.value = null
    await loadData()
    emit('refresh')
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

// ── Quantity split flow ──

const isEditingSplit = computed(() => splitForm.value.ID !== 0)

function openCreateSplit() {
  splitForm.value = {
    ID: 0,
    platformId: sourcePlatformOptions.value[0]?.value ?? null,
    externalKey: '',
    components: [{ productItemId: products.value[0]?.ID ?? null, quantity: 1 }],
  }
  showSplitModal.value = true
}

function openEditSplit(rule: QuantitySplitRule) {
  splitForm.value = {
    ID: rule.ID,
    platformId: rule.PlatformID,
    externalKey: rule.ExternalKey,
    components: rule.Components.map((c) => ({
      productItemId: c.ProductItemID,
      quantity: c.Quantity,
    })),
  }
  showSplitModal.value = true
}

function addSplitComponent() {
  splitForm.value.components.push({
    productItemId: products.value[0]?.ID ?? null,
    quantity: 1,
  })
}

function removeSplitComponent(index: number) {
  splitForm.value.components.splice(index, 1)
}

async function handleSaveSplit() {
  const form = splitForm.value
  if (!form.platformId || !form.externalKey.trim() || !form.components.length) return
  if (form.components.some((c) => !c.productItemId || c.quantity <= 0)) {
    feedback.error(t('waveRules.split.invalidComponents'))
    return
  }
  actionLoading.value = true
  try {
    await upsertQuantitySplitRule({
      ID: form.ID,
      WaveID: props.waveId,
      PlatformID: form.platformId,
      ExternalKey: form.externalKey.trim(),
      Components: form.components.map((c) => ({
        // The backend replaces components wholesale and reassigns ids.
        ID: 0,
        RuleID: form.ID,
        ProductItemID: c.productItemId as number,
        Quantity: c.quantity,
      })),
    })
    showSplitModal.value = false
    feedback.success(t('waveRules.split.saveSuccess'))
    await loadData()
    emit('refresh')
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

async function handleDeleteSplit(rule: QuantitySplitRule) {
  if (!window.confirm(t('waveRules.split.deleteConfirm'))) return
  actionLoading.value = true
  try {
    await deleteQuantitySplitRule(rule.ID)
    feedback.success(t('waveRules.split.deleteSuccess'))
    await loadData()
    emit('refresh')
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

function openCreateGrant() {
  grantForm.value = {
    customerId: customers.value[0]?.ID ?? null,
    productId: products.value[0]?.ID ?? null,
    quantity: 1,
  }
  showGrantModal.value = true
}

async function handleSaveGrant() {
  if (!grantForm.value.customerId || !grantForm.value.productId) return
  actionLoading.value = true
  try {
    await createGrant(
      props.waveId,
      grantForm.value.customerId,
      grantForm.value.productId,
      grantForm.value.quantity,
    )
    showGrantModal.value = false
    await loadData()
    emit('refresh')
  } catch (err) {
    feedback.error(t('feedback.error'), errMsg(err))
  } finally {
    actionLoading.value = false
  }
}

function jumpToResultsByProduct(productId: number) {
  void router.push({
    path: `/waves/${props.waveId}/results`,
    query: { product: String(productId) },
  })
}

const ruleColumns = [
  {
    title: t('library.productName'),
    key: 'ProductID',
    render(row: EntitlementRule) {
      const prod = products.value.find((p) => p.ID === row.ProductID)
      return prod ? prod.Name : `Product #${row.ProductID}`
    },
  },
  {
    title: t('waveWorkspace.selector'),
    key: 'Selector',
    render(row: EntitlementRule) {
      const selType = row.Selector.type
      const tag = h(StatusBadge, {
        dimension: 'entitlementSelectorType',
        value: selType,
      })
      let extra = ''
      if (selType === 'platform_level') {
        const plat = platforms.value.find((p) => p.ID === row.Selector.platform_id)
        extra = `${plat ? plat.Name : ''} ${row.Selector.level || ''}`
      } else if (selType === 'instance') {
        extra = `Inst #${row.Selector.instance_id}`
      }
      return h(NSpace, { align: 'center', size: 'small' }, () => [
        tag,
        extra ? h('span', { style: 'font-size: 13px; color: var(--color-text-secondary);' }, extra) : null,
      ])
    },
  },
  {
    title: t('waveWorkspace.quantity'),
    key: 'Quantity',
    width: 100,
    render(row: EntitlementRule) {
      return h(
        NTag,
        { type: row.Quantity >= 0 ? 'success' : 'error', size: 'small' },
        { default: () => (row.Quantity >= 0 ? `+${row.Quantity}` : String(row.Quantity)) },
      )
    },
  },
  {
    title: t('waves.status'),
    key: 'Active',
    width: 100,
    render(row: EntitlementRule) {
      return row.Active
        ? h(NTag, { type: 'success', size: 'small' }, { default: () => t('common.yes') })
        : h(NTag, { type: 'default', size: 'small' }, { default: () => t('common.no') })
    },
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 160,
    render(row: EntitlementRule) {
      return h('span', { class: 'wave-rules-page__action-cell' }, [
        h(
          NButton,
          {
            size: 'tiny',
            secondary: true,
            onClick: () => openEditRule(row),
          },
          { default: () => t('common.edit') },
        ),
        h(
          NButton,
          {
            size: 'tiny',
            type: 'error',
            secondary: true,
            disabled: actionLoading.value,
            onClick: () => void handleDeleteRule(row),
          },
          { default: () => t('waveWorkspace.deleteRule') },
        ),
      ])
    },
  },
]

const splitColumns = [
  {
    title: t('waveRules.split.platform'),
    key: 'PlatformID',
    width: 150,
    render(row: QuantitySplitRule) {
      const plat = platforms.value.find((p) => p.ID === row.PlatformID)
      return plat ? plat.Name : `Platform #${row.PlatformID}`
    },
  },
  {
    title: t('waveRules.split.externalKey'),
    key: 'ExternalKey',
    render(row: QuantitySplitRule) {
      return h('span', { class: 'wave-rules-page__mono' }, row.ExternalKey)
    },
  },
  {
    title: t('waveRules.split.components'),
    key: 'Components',
    render(row: QuantitySplitRule) {
      const parts = row.Components.map((c) => {
        const prod = products.value.find((p) => p.ID === c.ProductItemID)
        const name = prod ? prod.Name : `Product #${c.ProductItemID}`
        return `${name} ×${c.Quantity}`
      })
      return parts.length ? parts.join(' + ') : '—'
    },
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 160,
    render(row: QuantitySplitRule) {
      return h('span', { class: 'wave-rules-page__action-cell' }, [
        h(
          NButton,
          {
            size: 'tiny',
            secondary: true,
            onClick: () => openEditSplit(row),
          },
          { default: () => t('common.edit') },
        ),
        h(
          NButton,
          {
            size: 'tiny',
            type: 'error',
            secondary: true,
            disabled: actionLoading.value,
            onClick: () => void handleDeleteSplit(row),
          },
          { default: () => t('waveRules.split.delete') },
        ),
      ])
    },
  },
]
</script>

<template>
  <div class="wave-rules-page">
    <!-- Product Totals Banner -->
    <SectionCard :title="t('waveWorkspace.productTotals')">
      <div v-if="!totals.length" class="wave-rules-page__empty-totals">
        <EmptyState :title="t('waveWorkspace.emptyRules')" size="sm" />
      </div>
      <div v-else class="wave-rules-page__totals-grid">
        <StatCard
          v-for="total in totals"
          :key="total.ProductID"
          :label="total.Name"
          :value="String(total.Quantity)"
          :clickable="true"
          tone="progress"
          @click="jumpToResultsByProduct(total.ProductID)"
        />
      </div>
    </SectionCard>

    <!-- Rules Table -->
    <SectionCard :title="t('waveWorkspace.productRules')">
      <template #actions>
        <NSpace>
          <NButton size="small" type="primary" @click="openCreateRule">
            {{ t('waveWorkspace.addRule') }}
          </NButton>
          <NButton size="small" type="info" @click="openCreateGrant">
            {{ t('waveWorkspace.createGrant') }}
          </NButton>
          <NButton size="small" :loading="loading" @click="loadData">
            {{ t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>

      <NSpin :show="loading">
        <div v-if="!rules.length" class="wave-rules-page__empty">
          <EmptyState :title="t('waveWorkspace.emptyRules')" size="sm" />
        </div>
        <div v-else class="wave-rules-page__table">
          <NDataTable
            :columns="ruleColumns"
            :data="rules"
            :row-key="(row: EntitlementRule) => row.ID"
            size="small"
          />
        </div>
      </NSpin>
    </SectionCard>

    <!-- Add/Edit Rule Modal -->
    <NModal
      v-model:show="showRuleModal"
      preset="card"
      :title="isEditingRule ? t('waveWorkspace.editRule') : t('waveWorkspace.addRule')"
      style="width: 520px"
    >
      <NForm label-placement="left" label-width="110">
        <NFormItem :label="t('library.productName')">
          <NSelect
            v-model:value="ruleForm.ProductID"
            :options="productOptions"
            :placeholder="t('library.productName')"
          />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.selector')">
          <NRadioGroup v-model:value="ruleForm.selectorType">
            <NSpace vertical>
              <NRadio value="platform_level">
                {{ t('glossary.entitlementSelectorType.platform_level.label') }}
              </NRadio>
              <NRadio value="wave_all">
                {{ t('glossary.entitlementSelectorType.wave_all.label') }}
              </NRadio>
              <NRadio value="instance">
                {{ t('glossary.entitlementSelectorType.instance.label') }}
              </NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>

        <template v-if="ruleForm.selectorType === 'platform_level'">
          <NFormItem :label="t('library.factoryPlatform')">
            <NSelect v-model:value="ruleForm.platformId" :options="platformOptions" />
          </NFormItem>
          <NFormItem :label="t('inbox.kind')">
            <NInput v-model:value="ruleForm.level" />
          </NFormItem>
        </template>

        <template v-else-if="ruleForm.selectorType === 'instance'">
          <NFormItem :label="t('waveWorkspace.instanceID')">
            <NInputNumber v-model:value="ruleForm.instanceId" :min="1" />
          </NFormItem>
        </template>

        <NFormItem :label="t('waveWorkspace.quantity')">
          <NInputNumber v-model:value="ruleForm.quantity" />
        </NFormItem>
        <NFormItem :label="t('waves.status')">
          <NSwitch v-model:value="ruleForm.active" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showRuleModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!ruleForm.ProductID"
            @click="handleSaveRule"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Per-instance Entitlement Exceptions -->
    <SectionCard :title="t('waveWorkspace.exceptions')">
      <p class="wave-rules-page__hint">{{ t('waveWorkspace.exceptionsHint') }}</p>
      <NForm label-placement="left" label-width="120" :show-feedback="false">
        <NFormItem :label="t('library.productName')">
          <NSelect
            v-model:value="exceptionForm.productId"
            :options="productOptions"
            :placeholder="t('common.pleaseSelect')"
          />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.instanceID')">
          <NInputNumber v-model:value="exceptionForm.instanceId" :min="1" class="wave-rules-page__number" />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.quantity')">
          <NInputNumber v-model:value="exceptionForm.quantity" class="wave-rules-page__number" />
        </NFormItem>
        <NFormItem :label="t('library.notes')">
          <NInput v-model:value="exceptionForm.note" />
        </NFormItem>
      </NForm>
      <NSpace>
        <NButton
          size="small"
          type="primary"
          :loading="actionLoading"
          :disabled="!exceptionForm.productId || !exceptionForm.instanceId"
          @click="handleAddException"
        >
          {{ t('waveWorkspace.addException') }}
        </NButton>
      </NSpace>

      <div class="wave-rules-page__exception-delete">
        <NInputNumber
          v-model:value="deleteExceptionId"
          :min="1"
          :placeholder="t('waveWorkspace.exceptionID')"
          class="wave-rules-page__number"
        />
        <NButton
          size="small"
          type="error"
          secondary
          :loading="actionLoading"
          :disabled="!deleteExceptionId"
          @click="handleDeleteException"
        >
          {{ t('waveWorkspace.deleteException') }}
        </NButton>
      </div>
    </SectionCard>

    <!-- Quantity Splits -->
    <SectionCard :title="t('waveRules.split.title')">
      <template #actions>
        <NSpace>
          <NButton size="small" type="primary" @click="openCreateSplit">
            {{ t('waveRules.split.add') }}
          </NButton>
        </NSpace>
      </template>

      <p class="wave-rules-page__hint">{{ t('waveRules.split.hint') }}</p>

      <div v-if="!splitRules.length" class="wave-rules-page__empty">
        <EmptyState :title="t('waveRules.split.empty')" size="sm" />
      </div>
      <div v-else class="wave-rules-page__table">
        <NDataTable
          :columns="splitColumns"
          :data="splitRules"
          :row-key="(row: QuantitySplitRule) => row.ID"
          size="small"
        />
      </div>
    </SectionCard>

    <!-- Create Grant Modal -->
    <NModal
      v-model:show="showGrantModal"
      preset="card"
      :title="t('waveWorkspace.createGrant')"
      style="width: 480px"
    >
      <NForm label-placement="left" label-width="100">
        <NFormItem :label="t('library.displayName')">
          <NSelect v-model:value="grantForm.customerId" :options="customerOptions" />
        </NFormItem>
        <NFormItem :label="t('library.productName')">
          <NSelect v-model:value="grantForm.productId" :options="productOptions" />
        </NFormItem>
        <NFormItem :label="t('waveWorkspace.quantity')">
          <NInputNumber v-model:value="grantForm.quantity" :min="1" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showGrantModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!grantForm.customerId || !grantForm.productId"
            @click="handleSaveGrant"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Add/Edit Quantity Split Modal -->
    <NModal
      v-model:show="showSplitModal"
      preset="card"
      :title="isEditingSplit ? t('waveRules.split.edit') : t('waveRules.split.add')"
      style="width: 560px"
    >
      <NForm label-placement="left" label-width="120">
        <NFormItem :label="t('waveRules.split.platform')">
          <NSelect
            v-model:value="splitForm.platformId"
            :options="sourcePlatformOptions"
            :placeholder="t('common.pleaseSelect')"
          />
        </NFormItem>
        <NFormItem :label="t('waveRules.split.externalKey')">
          <NInput
            v-model:value="splitForm.externalKey"
            :placeholder="t('waveRules.split.externalKeyPlaceholder')"
          />
        </NFormItem>
        <NFormItem :label="t('waveRules.split.components')">
          <div class="wave-rules-page__components">
            <div
              v-for="(component, index) in splitForm.components"
              :key="index"
              class="wave-rules-page__component-row"
            >
              <NSelect
                v-model:value="component.productItemId"
                :options="productOptions"
                size="small"
                :placeholder="t('library.productName')"
              />
              <NInputNumber v-model:value="component.quantity" :min="1" size="small" />
              <NButton
                size="small"
                :disabled="splitForm.components.length <= 1"
                @click="removeSplitComponent(index)"
              >
                {{ t('waveRules.split.removeComponent') }}
              </NButton>
            </div>
            <NButton size="small" dashed @click="addSplitComponent">
              {{ t('waveRules.split.addComponent') }}
            </NButton>
          </div>
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showSplitModal = false">{{ t('common.cancel') }}</NButton>
          <NButton
            type="primary"
            :loading="actionLoading"
            :disabled="!splitForm.platformId || !splitForm.externalKey.trim() || !splitForm.components.length"
            @click="handleSaveSplit"
          >
            {{ t('common.confirm') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.wave-rules-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.wave-rules-page__totals-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--space-3);
}

.wave-rules-page__empty,
.wave-rules-page__empty-totals {
  padding: var(--space-3) 0;
}

.wave-rules-page__table {
  width: 100%;
}

.wave-rules-page__hint {
  margin: 0 0 var(--space-3);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.wave-rules-page__action-cell {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.wave-rules-page__mono {
  font-family: var(--font-mono);
  font-size: var(--font-size-sm);
}

.wave-rules-page__number {
  width: 180px;
}

.wave-rules-page__exception-delete {
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px dashed var(--color-border, var(--card-border-color));
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.wave-rules-page__components {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  width: 100%;
}

.wave-rules-page__component-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.wave-rules-page__component-row .n-select {
  flex: 1;
}
</style>
