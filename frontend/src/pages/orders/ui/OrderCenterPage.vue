<script setup lang="ts">
import {
  AlbumsOutline,
  GiftOutline,
  RibbonOutline,
  SearchOutline,
  TicketOutline,
} from '@vicons/ionicons5'
import { computed, h, ref, type Component } from 'vue'
import {
  NButton,
  NCard,
  NDataTable,
  NIcon,
  NInput,
  NList,
  NListItem,
  NProgress,
  NSelect,
  NTag,
  NThing,
  type DataTableColumns,
} from 'naive-ui'

interface GiftBatch {
  name: string
  batch: string
  progress: number
  icon: Component
}

interface OrderRecord {
  id: string
  member: string
  gift: string
  status: string
}

const gifts: GiftBatch[] = [
  { name: '限定徽章套装', batch: 'Batch 1', progress: 82, icon: RibbonOutline },
  { name: '签名明信片', batch: 'Batch 2', progress: 68, icon: TicketOutline },
  { name: '应援手幅', batch: 'Batch 3', progress: 54, icon: AlbumsOutline },
  { name: '周年纪念盒', batch: 'Batch 4', progress: 40, icon: GiftOutline },
]

const orders: OrderRecord[] = [
  { id: 'EGM-2401', member: 'Mika', gift: '徽章套装', status: '待确认' },
  { id: 'EGM-2402', member: 'Haruka', gift: '明信片', status: '可派发' },
  { id: 'EGM-2403', member: 'Nana', gift: '手幅', status: '地址异常' },
  { id: 'EGM-2404', member: 'Yui', gift: '纪念盒', status: '已锁定' },
]

const searchKeyword = ref('')
const selectedStatus = ref('all')
const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '待确认', value: '待确认' },
  { label: '可派发', value: '可派发' },
  { label: '地址异常', value: '地址异常' },
  { label: '已锁定', value: '已锁定' },
]

function getStatusClass(status: string) {
  if (status === '可派发') {
    return 'success'
  }

  if (status === '地址异常') {
    return 'error'
  }

  if (status === '已锁定') {
    return 'default'
  }

  return 'warning'
}

const filteredOrders = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase()

  return orders.filter((order) => {
    const matchesKeyword =
      keyword.length === 0 ||
      [order.id, order.member, order.gift].some((value) => value.toLowerCase().includes(keyword))
    const matchesStatus = selectedStatus.value === 'all' || order.status === selectedStatus.value

    return matchesKeyword && matchesStatus
  })
})

const columns: DataTableColumns<OrderRecord> = [
  {
    title: '订单号',
    key: 'id',
    render: (row) =>
      h('span', {
        style: {
          fontWeight: '600',
        },
      }, row.id),
  },
  {
    title: '粉丝',
    key: 'member',
  },
  {
    title: '礼物',
    key: 'gift',
  },
  {
    title: '状态',
    key: 'status',
    render: (row) =>
      h(
        NTag,
        {
          type: getStatusClass(row.status),
          size: 'small',
          round: true,
        },
        {
          default: () => row.status,
        },
      ),
  },
]
</script>

<template>
  <section class="space-y-5">
    <header>
      <p class="app-kicker">Order Center</p>
      <h1 class="app-title mt-2">派发中心</h1>
      <p class="app-copy mt-2">左侧选择礼物批次，右侧筛选并处理待派发订单。</p>
    </header>

    <div class="grid min-h-[620px] gap-4 xl:grid-cols-[300px_1fr]">
      <NCard size="medium">
        <template #header>
          <span class="app-heading-sm">礼物列表</span>
        </template>
        <template #header-extra>
          <NTag size="small" round>{{ gifts.length }} batches</NTag>
        </template>

        <NList hoverable clickable>
          <NListItem v-for="gift in gifts" :key="gift.name">
            <NThing>
              <template #avatar>
                <NIcon :size="18">
                  <component :is="gift.icon" />
                </NIcon>
              </template>
              <template #header>{{ gift.name }}</template>
              <template #description>{{ gift.batch }}</template>
              <template #header-extra>
                <NTag size="small" round>{{ gift.progress }}%</NTag>
              </template>
            </NThing>
            <NProgress class="mt-3" type="line" :percentage="gift.progress" :show-indicator="false" />
          </NListItem>
        </NList>
      </NCard>

      <NCard size="medium">
        <template #header>
          <span class="app-heading-sm">待派发订单</span>
        </template>
        <template #header-extra>
          <NTag size="small" round>{{ filteredOrders.length }} items</NTag>
        </template>

        <div class="grid gap-3 md:grid-cols-[1fr_180px_120px]">
          <NInput
            v-model:value="searchKeyword"
            placeholder="搜索订单号 / 粉丝名"
          />
          <NSelect v-model:value="selectedStatus" :options="statusOptions" />
          <NButton type="primary">
            <template #icon>
              <NIcon :size="18">
                <SearchOutline />
              </NIcon>
            </template>
            筛选
          </NButton>
        </div>

        <NDataTable
          class="mt-4"
          :columns="columns"
          :data="filteredOrders"
          :pagination="false"
          :bordered="false"
          :single-line="false"
        />
      </NCard>
    </div>
  </section>
</template>
