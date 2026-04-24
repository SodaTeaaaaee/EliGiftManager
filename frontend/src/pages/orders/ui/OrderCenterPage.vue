<script setup lang="ts">
import { SearchOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { NButton, NCard, NDataTable, NEmpty, NIcon, NInput, NTag, type DataTableColumns } from 'naive-ui'
import { WAILS_PREVIEW_MESSAGE, isWailsRuntimeAvailable, listDispatchRecords, validateBatch, type DispatchRecordItem } from '@/shared/lib/wails/app'

const records = ref<DispatchRecordItem[]>([])
const keyword = ref('')
const isLoading = ref(false)
const validatingBatch = ref('')
const errorMessage = ref('')
const validationMessage = ref('')

const batches = computed(() => Array.from(new Set(records.value.map((record) => record.batchName))).filter(Boolean))
const pendingAddressCount = computed(() => records.value.filter((record) => !record.hasAddress || record.status === 'pending_address').length)
const totalQuantity = computed(() => records.value.reduce((sum, record) => sum + record.quantity, 0))

const filteredRecords = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return records.value
  return records.value.filter((record) =>
    [record.batchName, record.memberNickname, record.platformUid, record.productName, record.factorySku, record.recipientName, record.phone, record.address]
      .some((value) => value.toLowerCase().includes(text)),
  )
})

const columns: DataTableColumns<DispatchRecordItem> = [
  { title: '批次', key: 'batchName', minWidth: 150 },
  { title: '会员', key: 'memberNickname', minWidth: 140, render: (row) => row.memberNickname || row.platformUid },
  { title: '商品', key: 'productName', minWidth: 180 },
  { title: 'SKU', key: 'factorySku', minWidth: 130 },
  { title: '数量', key: 'quantity', width: 80 },
  {
    title: '状态',
    key: 'status',
    width: 130,
    render: (row) => h(NTag, { type: row.hasAddress ? 'success' : 'warning', size: 'small', round: true }, { default: () => (row.hasAddress ? '可导出' : '待补地址') }),
  },
  { title: '收件信息', key: 'address', minWidth: 260, ellipsis: { tooltip: true }, render: (row) => row.hasAddress ? `${row.recipientName} / ${row.phone} / ${row.address}` : '-' },
]

async function loadRecords() {
  if (!isWailsRuntimeAvailable()) {
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return
  }

  isLoading.value = true
  errorMessage.value = ''
  validationMessage.value = ''
  try {
    records.value = await listDispatchRecords()
  } catch (error) {
    console.error('加载派发记录失败', error)
    errorMessage.value = '加载派发记录失败，请查看后端日志。'
  } finally {
    isLoading.value = false
  }
}

async function handleValidate(batchName: string) {
  validatingBatch.value = batchName
  validationMessage.value = ''
  try {
    const result = await validateBatch(batchName)
    validationMessage.value = `${result.batchName}: 共 ${result.totalRecords} 条，${result.pendingAddressRecords} 条待补地址。`
  } catch (error) {
    console.error('批次校验失败', error)
    validationMessage.value = '批次校验失败，请确认该批次存在。'
  } finally {
    validatingBatch.value = ''
  }
}

onMounted(loadRecords)
</script>

<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Dispatch Center</p>
        <h1 class="app-title mt-2">派发中心</h1>
        <p class="app-copy mt-2">读取 dispatch_records 并联动会员、商品、地址，支持按批次预校验。</p>
      </div>
      <NButton :loading="isLoading" secondary strong @click="loadRecords">刷新派发记录</NButton>
    </header>

    <div class="grid gap-4 md:grid-cols-3">
      <NCard><p class="app-copy">记录数</p><p class="mt-1 text-2xl font-semibold">{{ records.length }}</p></NCard>
      <NCard><p class="app-copy">礼物件数</p><p class="mt-1 text-2xl font-semibold">{{ totalQuantity }}</p></NCard>
      <NCard><p class="app-copy">待补地址</p><p class="mt-1 text-2xl font-semibold text-amber-600">{{ pendingAddressCount }}</p></NCard>
    </div>

    <NCard size="medium">
      <template #header>批次校验</template>
      <div v-if="batches.length === 0" class="app-copy">暂无批次可校验</div>
      <div v-else class="flex flex-wrap gap-2">
        <NButton v-for="batch in batches" :key="batch" size="small" secondary :loading="validatingBatch === batch" @click="handleValidate(batch)">
          {{ batch }}
        </NButton>
      </div>
      <NTag v-if="validationMessage" class="mt-3" type="info" round>{{ validationMessage }}</NTag>
    </NCard>

    <NCard size="medium">
      <template #header>派发明细</template>
      <template #header-extra>
        <NInput v-model:value="keyword" clearable placeholder="搜索批次、会员、商品或地址" style="width: 300px">
          <template #prefix><NIcon><SearchOutline /></NIcon></template>
        </NInput>
      </template>
      <NEmpty v-if="errorMessage" :description="errorMessage" />
      <NDataTable v-else :columns="columns" :data="filteredRecords" :loading="isLoading" :bordered="false" :scroll-x="1000" :pagination="{ pageSize: 12 }" />
    </NCard>
  </section>
</template>
