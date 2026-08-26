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
import {
  createGrant,
  getProductTotals,
  listCustomers,
  listPlatforms,
  listProducts,
  listRules,
  upsertRule,
} from '@/shared/api/bridge'
import type {
  CustomerProfile,
  EntitlementRule,
  Platform,
  ProductItem,
  ProductTotal,
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

const loading = ref(false)
const actionLoading = ref(false)
const rules = ref<EntitlementRule[]>([])
const totals = ref<ProductTotal[]>([])
const products = ref<ProductItem[]>([])
const platforms = ref<Platform[]>([])
const customers = ref<CustomerProfile[]>([])

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

async function loadData() {
  if (!props.waveId) return
  loading.value = true
  try {
    const [ruleRes, totalRes, prodRes, platRes, custRes] = await Promise.all([
      listRules(props.waveId),
      getProductTotals(props.waveId),
      listProducts(),
      listPlatforms(),
      listCustomers(),
    ])
    rules.value = ruleRes
    totals.value = totalRes
    products.value = prodRes
    platforms.value = platRes
    customers.value = custRes
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
    console.error('Failed to save rule:', err)
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
    console.error('Failed to create grant:', err)
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
        : h(NTag, { type: 'neutral', size: 'small' }, { default: () => t('common.no') })
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
      :title="t('waveWorkspace.addRule')"
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
</style>
