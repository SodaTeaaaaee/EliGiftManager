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
import { SectionCard } from '@/shared/ui/cards'
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
// instance-lookup binding yet — so each product card carries a minimal
// manual add form, and deletion happens by exception id below.
interface ExceptionDraft {
  instanceId: number | null
  quantity: number
  note: string
}

const exceptionForms = ref<Record<number, ExceptionDraft>>({})
const deleteExceptionId = ref<number | null>(null)

/** Seed one draft per known product, keeping any partially-filled entries. */
function resetExceptionForms() {
  const next: Record<number, ExceptionDraft> = {}
  for (const p of products.value) {
    next[p.ID] = exceptionForms.value[p.ID] ?? { instanceId: null, quantity: 1, note: '' }
  }
  exceptionForms.value = next
}

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
    resetExceptionForms()
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

/** Retail/source platforms — the side a rule selector or split key matches against. */
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

// ── Product-centered cards ──

/**
 * One card per product that carries rules and/or a positive total (grants
 * and exceptions surface through totals). Split rules stay in their own
 * section; cards only mention how many split components reference them.
 */
interface ProductCardData {
  product: ProductItem
  rules: EntitlementRule[]
  total: ProductTotal | null
  splitMentionCount: number
  form: ExceptionDraft | undefined
}

const productCards = computed<ProductCardData[]>(() => {
  const byId = new Map<number, ProductCardData>()
  const ensure = (p: ProductItem): ProductCardData => {
    let card = byId.get(p.ID)
    if (!card) {
      card = {
        product: p,
        rules: [],
        total: null,
        splitMentionCount: 0,
        form: exceptionForms.value[p.ID],
      }
      byId.set(p.ID, card)
    }
    return card
  }
  for (const rule of rules.value) {
    const p = products.value.find((x) => x.ID === rule.ProductID)
    if (p) ensure(p).rules.push(rule)
  }
  for (const total of totals.value) {
    if (total.Quantity <= 0) continue
    const p = products.value.find((x) => x.ID === total.ProductID)
    if (p) ensure(p).total = total
  }
  for (const split of splitRules.value) {
    const seen = new Set<number>()
    for (const c of split.Components) {
      if (seen.has(c.ProductItemID)) continue
      seen.add(c.ProductItemID)
      const card = byId.get(c.ProductItemID)
      if (card) card.splitMentionCount++
    }
  }
  const cards = [...byId.values()]
  cards.sort((a, b) => a.product.Name.localeCompare(b.product.Name, undefined, { numeric: true }))
  return cards
})

/** Rules whose product no longer exists land in the fallback zone below. */
const unlinkedRules = computed(() =>
  rules.value.filter((r) => !products.value.some((p) => p.ID === r.ProductID)),
)

function selectorExtra(rule: EntitlementRule): string {
  const sel = rule.Selector
  if (sel.type === 'platform_level') {
    const plat = platforms.value.find((p) => p.ID === sel.platform_id)
    return `${plat ? plat.Name : ''} ${sel.level || ''}`.trim()
  }
  if (sel.type === 'instance') return `Inst #${sel.instance_id}`
  return ''
}

function openCreateRule() {
  ruleForm.value = {
    ID: 0,
    ProductID: products.value[0]?.ID ?? null,
    selectorType: 'platform_level',
    platformId: sourcePlatformOptions.value[0]?.value ?? null,
    level: '舰长',
    instanceId: null,
    quantity: 1,
    active: true,
  }
  showRuleModal.value = true
}

function openCreateRuleFor(product: ProductItem) {
  ruleForm.value = {
    ID: 0,
    ProductID: product.ID,
    selectorType: 'platform_level',
    platformId: sourcePlatformOptions.value[0]?.value ?? null,
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
    platformId: rule.Selector.platform_id ?? sourcePlatformOptions.value[0]?.value ?? null,
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

async function handleAddExceptionFor(productId: number) {
  const form = exceptionForms.value[productId]
  if (!form || !form.instanceId) return
  actionLoading.value = true
  try {
    await addException({
      WaveID: props.waveId,
      ProductID: productId,
      InstanceID: form.instanceId,
      Quantity: form.quantity,
      Note: form.note.trim(),
    })
    feedback.success(t('waveWorkspace.exceptionAddSuccess'))
    form.instanceId = null
    form.quantity = 1
    form.note = ''
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
      return row.ExternalKey
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
    <!-- Product-centered rule cards -->
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
        <div v-if="!productCards.length && !unlinkedRules.length" class="wave-rules-page__empty">
          <EmptyState :title="t('waveRules.emptyCards')" size="sm" />
        </div>
        <div v-else class="wave-rules-page__cards">
          <div
            v-for="card in productCards"
            :key="card.product.ID"
            class="wave-rules-page__card"
          >
            <div class="wave-rules-page__card-header">
              <div class="wave-rules-page__card-id">
                <span class="wave-rules-page__card-name">{{ card.product.Name }}</span>
                <span class="wave-rules-page__card-sku">{{ card.product.FactorySKU }}</span>
              </div>
              <NSpace size="small" align="center">
                <NButton
                  size="tiny"
                  type="primary"
                  quaternary
                  @click="jumpToResultsByProduct(card.product.ID)"
                >
                  {{ t('waveWorkspace.productTotals') }} × {{ card.total?.Quantity ?? 0 }}
                </NButton>
                <NButton size="tiny" secondary @click="openCreateRuleFor(card.product)">
                  {{ t('waveWorkspace.addRule') }}
                </NButton>
              </NSpace>
            </div>

            <p v-if="card.splitMentionCount > 0" class="wave-rules-page__split-mention">
              {{ t('waveRules.splitMention', { n: card.splitMentionCount }) }}
            </p>

            <div v-if="card.rules.length" class="wave-rules-page__rules">
              <div v-for="rule in card.rules" :key="rule.ID" class="wave-rules-page__rule-row">
                <StatusBadge dimension="entitlementSelectorType" :value="rule.Selector.type" />
                <span
                  v-if="selectorExtra(rule)"
                  class="wave-rules-page__rule-extra"
                >{{ selectorExtra(rule) }}</span>
                <NTag :type="rule.Quantity >= 0 ? 'success' : 'error'" size="small">
                  {{ rule.Quantity >= 0 ? `+${rule.Quantity}` : rule.Quantity }}
                </NTag>
                <NTag v-if="rule.Active" type="success" size="small">
                  {{ t('common.yes') }}
                </NTag>
                <NTag v-else type="default" size="small">{{ t('common.no') }}</NTag>
                <span class="wave-rules-page__rule-actions">
                  <NButton size="tiny" secondary @click="openEditRule(rule)">
                    {{ t('common.edit') }}
                  </NButton>
                  <NButton
                    size="tiny"
                    type="error"
                    secondary
                    :disabled="actionLoading"
                    @click="handleDeleteRule(rule)"
                  >
                    {{ t('waveWorkspace.deleteRule') }}
                  </NButton>
                </span>
              </div>
            </div>
            <p v-else class="wave-rules-page__no-rules">{{ t('waveRules.noRulesInProduct') }}</p>

            <div class="wave-rules-page__card-exception">
              <span class="wave-rules-page__section-label">{{ t('waveWorkspace.exceptions') }}</span>
              <div v-if="card.form" class="wave-rules-page__exception-form">
                <NInputNumber
                  v-model:value="card.form.instanceId"
                  :min="1"
                  size="small"
                  :placeholder="t('waveWorkspace.instanceID')"
                />
                <NInputNumber
                  v-model:value="card.form.quantity"
                  size="small"
                  :placeholder="t('waveWorkspace.quantity')"
                />
                <NInput
                  v-model:value="card.form.note"
                  size="small"
                  :placeholder="t('library.notes')"
                />
                <NButton
                  size="tiny"
                  type="primary"
                  :loading="actionLoading"
                  :disabled="!card.form.instanceId"
                  @click="handleAddExceptionFor(card.product.ID)"
                >
                  {{ t('waveWorkspace.addException') }}
                </NButton>
              </div>
            </div>
          </div>
        </div>
      </NSpin>
    </SectionCard>

    <!-- Fallback: rules whose product no longer exists -->
    <SectionCard v-if="unlinkedRules.length" :title="t('waveRules.unlinkedTitle')">
      <p class="wave-rules-page__hint">{{ t('waveRules.unlinkedHint') }}</p>
      <div class="wave-rules-page__cards wave-rules-page__cards--single">
        <div class="wave-rules-page__card">
          <div v-for="rule in unlinkedRules" :key="rule.ID" class="wave-rules-page__rule-row">
            <StatusBadge dimension="entitlementSelectorType" :value="rule.Selector.type" />
            <span class="wave-rules-page__rule-extra">
              {{ t('waveRules.unlinkedProductRef', { id: rule.ProductID }) }}
            </span>
            <span
              v-if="selectorExtra(rule)"
              class="wave-rules-page__rule-extra"
            >{{ selectorExtra(rule) }}</span>
            <NTag :type="rule.Quantity >= 0 ? 'success' : 'error'" size="small">
              {{ rule.Quantity >= 0 ? `+${rule.Quantity}` : rule.Quantity }}
            </NTag>
            <NButton
              size="tiny"
              type="error"
              secondary
              :disabled="actionLoading"
              @click="handleDeleteRule(rule)"
            >
              {{ t('waveWorkspace.deleteRule') }}
            </NButton>
          </div>
        </div>
      </div>
    </SectionCard>

    <!-- Exception maintenance (manual delete by id) -->
    <SectionCard :title="t('waveRules.exceptionTools')">
      <p class="wave-rules-page__hint">{{ t('waveRules.exceptionToolsHint') }}</p>
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
          <NFormItem :label="t('waveRules.selectPlatform')">
            <NSelect v-model:value="ruleForm.platformId" :options="sourcePlatformOptions" />
          </NFormItem>
          <NFormItem :label="t('waveRules.membershipLevel')">
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

.wave-rules-page__cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(420px, 1fr));
  gap: var(--space-3);
}

.wave-rules-page__cards--single {
  grid-template-columns: 1fr;
}

.wave-rules-page__card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.wave-rules-page__card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.wave-rules-page__card-id {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  min-width: 0;
}

.wave-rules-page__card-name {
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.wave-rules-page__card-sku {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.wave-rules-page__split-mention,
.wave-rules-page__no-rules {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.wave-rules-page__rules {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.wave-rules-page__rule-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.wave-rules-page__rule-extra {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.wave-rules-page__rule-actions {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
}

.wave-rules-page__card-exception {
  border-top: 1px dashed var(--color-border, var(--card-border-color));
  padding-top: var(--space-2);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.wave-rules-page__section-label {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-secondary);
}

.wave-rules-page__exception-form {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.wave-rules-page__exception-form .n-input {
  width: 150px;
}

.wave-rules-page__empty {
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

.wave-rules-page__number {
  width: 180px;
}

.wave-rules-page__exception-delete {
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
