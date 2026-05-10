<script setup lang="ts">
import { AddOutline, CreateOutline, TrashOutline } from '@vicons/ionicons5'
import { SearchOutline } from '@vicons/ionicons5'
import { computed, h, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  NAlert,
  NAvatar,
  NButton,
  NCard,
  NDataTable,
  NDrawer,
  NDrawerContent,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NModal,
  NPagination,
  NSelect,
  NSwitch,
  NTag,
  NTimeline,
  NTimelineItem,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { useTableMode } from '@/shared/model/settings'
import { useAdaptiveTable } from '@/shared/composables/useAdaptiveTable'
import AdaptivePaginationIndicator from '@/shared/ui/table/AdaptivePaginationIndicator.vue'
import AdaptiveTableMeasureLayer from '@/shared/ui/table/AdaptiveTableMeasureLayer.vue'
import {
  useTableSort,
  nextSortOrderAscFirst,
  type SortDescriptor,
} from '@/shared/composables/useTableSort'
import { collectAllPages } from '@/shared/lib/table/collectAllPages'
import {
  addMemberAddress,
  deleteMemberAddress,
  getDashboard,
  isWailsRuntimeAvailable,
  listMembers,
  setDefaultAddress,
  updateMemberAddress,
  WAILS_PREVIEW_MESSAGE,
  type MemberItem,
} from '@/shared/lib/wails/app'
import type { model } from '../../../../wailsjs/go/models'

const message = useMessage()
const allMembers = ref<MemberItem[]>([])
const keyword = ref('')
const platform = ref('')
const selectedMember = ref<MemberItem | null>(null)
const isLoading = ref(false)
const errorMessage = ref('')
const dashboardStats = ref({ memberCount: 0, addressCount: 0, missingAddresses: 0 })
const showMissingOnly = ref(false)
const showAddressModal = ref(false)
const editingAddress = ref<model.MemberAddress | null>(null)
const addressForm = ref({ recipientName: '', phone: '', address: '' })
const isSavingAddress = ref(false)

const tableMode = useTableMode()
const platformCatalog = ref<string[]>([])

const platformOptions = computed(() =>
  platformCatalog.value.map((value) => ({ label: value, value })),
)

// Sort descriptors
const memberSortDescriptors: SortDescriptor<MemberItem>[] = [
  { key: 'platform', getValue: (m) => m.platform },
  { key: 'platformUid', getValue: (m) => m.platformUid },
  { key: 'latestNickname', getValue: (m) => m.latestNickname || '' },
  { key: 'activeAddressCount', getValue: (m) => m.activeAddressCount },
  { key: 'latestRecipient', getValue: (m) => m.latestRecipient || '' },
  { key: 'latestPhone', getValue: (m) => m.latestPhone || '' },
  { key: 'latestAddress', getValue: (m) => m.latestAddress || '' },
]

// Filter first, then sort
const filteredMembers = computed(() => {
  if (showMissingOnly.value) {
    return allMembers.value.filter((m) => m.activeAddressCount === 0)
  }
  return allMembers.value
})

const {
  sortedItems: displayMembers,
  sortState: memberSortState,
  applyNaiveSorterEvent,
} = useTableSort(filteredMembers, memberSortDescriptors)

const columns = computed<DataTableColumns<MemberItem>>(() => [
  {
    title: '会员',
    key: '__summary',
    width: 48,
    render: (row: any) =>
      h(
        NAvatar,
        {
          size: 34,
          color: 'var(--accent-surface)',
          style: { color: 'var(--accent)', fontWeight: '700' },
        },
        { default: () => (row.latestNickname || row.platformUid).slice(0, 1).toUpperCase() },
      ),
  },
  {
    title: '昵称',
    key: 'latestNickname',
    minWidth: 100,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder:
      memberSortState.value.columnKey === 'latestNickname' ? memberSortState.value.order : false,
    render: (row) => row.latestNickname || '-',
  },
  {
    title: '平台',
    key: 'platform',
    width: 90,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder: memberSortState.value.columnKey === 'platform' ? memberSortState.value.order : false,
  },
  {
    title: 'UID',
    key: 'platformUid',
    minWidth: 100,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder:
      memberSortState.value.columnKey === 'platformUid' ? memberSortState.value.order : false,
  },
  {
    title: '地址状态',
    key: 'activeAddressCount',
    width: 120,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder:
      memberSortState.value.columnKey === 'activeAddressCount'
        ? memberSortState.value.order
        : false,
    render: (row) =>
      h(
        NTag,
        { type: row.activeAddressCount > 0 ? 'success' : 'error', size: 'small', round: true },
        { default: () => (row.activeAddressCount > 0 ? '已完善' : '缺地址') },
      ),
  },
  {
    title: '默认收件人',
    key: 'latestRecipient',
    minWidth: 100,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder:
      memberSortState.value.columnKey === 'latestRecipient' ? memberSortState.value.order : false,
    render: (row) => row.latestRecipient || '-',
  },
  {
    title: '手机',
    key: 'latestPhone',
    minWidth: 100,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder:
      memberSortState.value.columnKey === 'latestPhone' ? memberSortState.value.order : false,
    render: (row) => row.latestPhone || '-',
  },
  {
    title: '地址',
    key: 'latestAddress',
    minWidth: 180,
    sorter: 'default' as const,
    customNextSortOrder: nextSortOrderAscFirst,
    sortOrder:
      memberSortState.value.columnKey === 'latestAddress' ? memberSortState.value.order : false,
    ellipsis: { tooltip: true },
    render: (row) => row.latestAddress || '-',
  },
])

// Adaptive table refs
const memberLayoutRef = ref<HTMLElement | null>(null)
const memberTableRef = ref<HTMLElement | null>(null)
const memberFooterRef = ref<HTMLElement | null>(null)
const memberMeasureLayer = ref<InstanceType<typeof AdaptiveTableMeasureLayer> | null>(null)

const {
  currentPage,
  pageCount: totalPages,
  renderItems: renderMembers,
  tableBodyMaxHeight,
  viewportWidth,
  measurementInvalidationVersion: memberMeasurementVersion,
  measurementRequestId: memberMeasurementRequestId,
  requestRemeasure: requestMemberRemeasure,
  applyMeasuredRows: applyMemberMeasuredRows,
  handlePageChange,
  refreshLayout,
  teardown,
  init,
} = useAdaptiveTable(displayMembers, tableMode, {
  layoutRef: memberLayoutRef,
  tableRef: memberTableRef,
  paginationRef: memberFooterRef,
  rowHeightHint: 56,
  contentSignature: () => displayMembers.value.map((m) => m.id).join(','),
})

// Measure columns (mirrors real first column for accurate row-height measurement)
const memberMeasureColumns = computed<DataTableColumns<MemberItem>>(() => [
  {
    title: '会员',
    key: '__summary',
    width: 48,
    render: (row: any) =>
      h(
        NAvatar,
        {
          size: 34,
          color: 'var(--accent-surface)',
          style: { color: 'var(--accent)', fontWeight: '700' },
        },
        { default: () => (row.latestNickname || row.platformUid).slice(0, 1).toUpperCase() },
      ),
  },
  {
    title: '昵称',
    key: 'latestNickname',
    minWidth: 100,
    render: (row: any) => row.latestNickname || '-',
  },
  { title: '平台', key: 'platform', width: 90 },
  { title: 'UID', key: 'platformUid', minWidth: 100 },
  {
    title: '地址状态',
    key: 'activeAddressCount',
    width: 120,
    render: (row: any) =>
      h(
        NTag,
        { type: row.activeAddressCount > 0 ? 'success' : 'error', size: 'small', round: true },
        { default: () => (row.activeAddressCount > 0 ? '已完善' : '缺地址') },
      ),
  },
  {
    title: '默认收件人',
    key: 'latestRecipient',
    minWidth: 100,
    render: (row: any) => row.latestRecipient || '-',
  },
  {
    title: '手机',
    key: 'latestPhone',
    minWidth: 100,
    render: (row: any) => row.latestPhone || '-',
  },
  {
    title: '地址',
    key: 'latestAddress',
    minWidth: 180,
    ellipsis: { tooltip: true },
    render: (row: any) => row.latestAddress || '-',
  },
])

let memberMeasureRunning = false
let memberMeasurePending = false

async function runMemberRemeasure() {
  if (memberMeasureRunning) {
    memberMeasurePending = true
    return
  }
  memberMeasureRunning = true
  const requestId = memberMeasurementRequestId.value
  try {
    await nextTick()
    await new Promise((r) => requestAnimationFrame(r))
    await new Promise((r) => requestAnimationFrame(r))
    refreshLayout()
    await nextTick()
    const result = memberMeasureLayer.value?.measure()
    if (!result) return
    memberMeasureLayer.value?.setWidth(viewportWidth.value)
    applyMemberMeasuredRows(result.rowHeights, result.headerHeight, requestId)
  } finally {
    memberMeasureRunning = false
    if (memberMeasurePending) {
      memberMeasurePending = false
      await runMemberRemeasure()
    }
  }
}

watch(
  [() => tableMode.value, () => memberMeasurementVersion.value],
  async () => {
    if (tableMode.value !== 'paginated') return
    await runMemberRemeasure()
  },
  { flush: 'post' },
)

async function loadDashboardStats() {
  if (!isWailsRuntimeAvailable()) return
  try {
    const d = await getDashboard()
    dashboardStats.value = {
      memberCount: d.memberCount,
      addressCount: d.addressCount,
      missingAddresses: d.missingAddresses,
    }
  } catch (error) {
    console.error(error)
  }
}

async function loadAllMembers() {
  if (!isWailsRuntimeAvailable()) {
    allMembers.value = []
    platformCatalog.value = []
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return
  }
  isLoading.value = true
  errorMessage.value = ''
  try {
    const selectedMemberID = selectedMember.value?.id
    const result = await collectAllPages<MemberItem>(async (p, ps) => {
      const payload = await listMembers(p, ps, keyword.value, platform.value)
      return { items: payload.items, total: payload.total, platforms: payload.platforms }
    }, 200)
    allMembers.value = result.items
    platformCatalog.value = result.platforms
    if (selectedMemberID) {
      selectedMember.value = result.items.find((m) => m.id === selectedMemberID) ?? null
    }
  } catch (error) {
    console.error(error)
    errorMessage.value = '加载会员数据库失败。'
  } finally {
    isLoading.value = false
  }
}

async function searchMembers() {
  await loadAllMembers()
}

async function refreshMembers() {
  await loadDashboardStats()
  await loadAllMembers()
  await nextTick()
  requestMemberRemeasure()
}

function closeMemberDetail() {
  selectedMember.value = null
}

function handleDrawerVisibility(show: boolean) {
  if (!show) {
    closeMemberDetail()
  }
}

async function handleDefault(addressId: number) {
  if (!selectedMember.value) return

  try {
    const memberId = selectedMember.value.id
    await setDefaultAddress(memberId, addressId)
    message.success('默认地址已更新')
    await loadAllMembers()
    await loadDashboardStats()
    selectedMember.value = allMembers.value.find((member) => member.id === memberId) ?? null
    await nextTick()
    requestMemberRemeasure()
  } catch (error) {
    message.error(String(error))
  }
}

function openAddAddress() {
  editingAddress.value = null
  addressForm.value = { recipientName: '', phone: '', address: '' }
  showAddressModal.value = true
}

function openEditAddress(addr: model.MemberAddress) {
  editingAddress.value = addr
  addressForm.value = {
    recipientName: addr.recipientName,
    phone: addr.phone,
    address: addr.address,
  }
  showAddressModal.value = true
}

async function saveAddress() {
  if (!selectedMember.value) return
  isSavingAddress.value = true
  try {
    if (editingAddress.value) {
      await updateMemberAddress(
        editingAddress.value.id,
        addressForm.value.recipientName,
        addressForm.value.phone,
        addressForm.value.address,
      )
      message.success('地址已更新')
    } else {
      await addMemberAddress(
        selectedMember.value.id,
        addressForm.value.recipientName,
        addressForm.value.phone,
        addressForm.value.address,
      )
      message.success('地址已添加')
    }
    showAddressModal.value = false
    await loadAllMembers()
    await loadDashboardStats()
    selectedMember.value = allMembers.value.find((m) => m.id === selectedMember.value!.id) ?? null
    await nextTick()
    requestMemberRemeasure()
  } catch (error) {
    message.error(String(error))
  } finally {
    isSavingAddress.value = false
  }
}

async function handleDeleteAddress(addressId: number) {
  try {
    await deleteMemberAddress(addressId)
    message.success('地址已删除')
    await loadAllMembers()
    await loadDashboardStats()
    if (selectedMember.value) {
      selectedMember.value = allMembers.value.find((m) => m.id === selectedMember.value!.id) ?? null
    }
    await nextTick()
    requestMemberRemeasure()
  } catch (error) {
    message.error(String(error))
  }
}

onMounted(async () => {
  await nextTick()
  await loadDashboardStats()
  await loadAllMembers()
  await init()
  await nextTick()
  requestMemberRemeasure()
})

onBeforeUnmount(() => {
  teardown()
})
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Title -->
    <header class="shrink-0 px-1 py-2">
      <p class="app-kicker">Members</p>
      <h1 class="app-title mt-2">会员管理中心</h1>
    </header>

    <!-- Stats cards -->
    <div class="shrink-0 px-1 grid gap-3 md:grid-cols-3 mb-3">
      <NCard>
        <p class="app-copy">会员总数</p>
        <p class="mt-1 text-2xl font-semibold">{{ dashboardStats.memberCount }}</p>
      </NCard>
      <NCard>
        <p class="app-copy">地址总数</p>
        <p class="mt-1 text-2xl font-semibold">{{ dashboardStats.addressCount }}</p>
      </NCard>
      <NCard>
        <p class="app-copy">缺失地址会员</p>
        <p class="mt-1 text-2xl font-semibold text-amber-600">
          {{ dashboardStats.missingAddresses }}
        </p>
      </NCard>
    </div>

    <!-- Search bar -->
    <div class="shrink-0 px-1 flex items-center gap-2 mb-3">
      <NInput
        v-model:value="keyword"
        placeholder="搜索会员名或UID..."
        clearable
        @keyup.enter="searchMembers"
        style="flex: 1; max-width: 300px"
      >
        <template #prefix
          ><NIcon><SearchOutline /></NIcon
        ></template>
      </NInput>
      <NSelect
        v-model:value="platform"
        :options="platformOptions"
        placeholder="全部平台"
        clearable
        style="width: 140px"
        @update:value="searchMembers"
      />
      <div class="flex items-center gap-1 ml-2">
        <span class="text-xs text-gray-500">仅缺地址</span>
        <NSwitch v-model:value="showMissingOnly" />
      </div>
      <NButton size="small" type="primary" @click="searchMembers" class="ml-2">
        <template #icon
          ><NIcon><SearchOutline /></NIcon
        ></template>
        搜索
      </NButton>
      <NButton size="small" @click="refreshMembers" class="ml-1">刷新</NButton>
    </div>

    <!-- Error / empty state -->
    <NAlert v-if="errorMessage" type="warning" :show-icon="false" class="mx-1 mb-3">
      {{ errorMessage }}
    </NAlert>
    <NAlert
      v-else-if="!isLoading && displayMembers.length === 0 && allMembers.length === 0"
      type="info"
      :show-icon="false"
      class="mx-1 mb-3"
    >
      暂无会员数据
    </NAlert>

    <!-- Table viewport area -->
    <div ref="memberLayoutRef" class="flex-1 min-h-0 flex flex-col overflow-hidden px-1">
      <div
        v-if="tableMode === 'scroll'"
        ref="memberTableRef"
        class="flex-1 min-h-0 overflow-hidden"
      >
        <NDataTable
          :columns="columns"
          :data="renderMembers"
          :loading="isLoading"
          :bordered="false"
          :remote="true"
          :pagination="false"
          :max-height="tableBodyMaxHeight"
          :row-props="
            (row: any) => ({
              class: 'cursor-pointer',
              onClick: () => {
                selectedMember = row
              },
            })
          "
          size="small"
          @update:sorter="(s: any) => applyNaiveSorterEvent(s)"
        />
      </div>
      <template v-if="tableMode === 'paginated'">
        <div ref="memberTableRef" class="shrink-0">
          <NDataTable
            :columns="columns"
            :data="renderMembers"
            :loading="isLoading"
            :bordered="false"
            :remote="true"
            :pagination="false"
            :row-props="
              (row: any) => ({
                class: 'cursor-pointer',
                onClick: () => {
                  selectedMember = row
                },
              })
            "
            size="small"
            @update:sorter="(s: any) => applyNaiveSorterEvent(s)"
          />
        </div>
        <AdaptivePaginationIndicator :page="currentPage" :page-count="totalPages" />
      </template>
    </div>

    <!-- Footer -->
    <div
      v-if="tableMode === 'paginated'"
      ref="memberFooterRef"
      class="flex justify-center shrink-0"
      style="padding: 8px 0 12px 0"
    >
      <div style="transform: scale(1.3); transform-origin: top center; display: inline-flex">
        <NPagination
          :page="currentPage"
          :page-count="totalPages"
          size="small"
          @update:page="handlePageChange"
        />
      </div>
    </div>

    <!-- Measure layer -->
    <AdaptiveTableMeasureLayer
      v-if="tableMode === 'paginated' && displayMembers.length"
      ref="memberMeasureLayer"
      :data="displayMembers"
      :columns="memberMeasureColumns"
      :width="viewportWidth"
      size="small"
    />

    <NDrawer :show="!!selectedMember" :width="460" @update:show="handleDrawerVisibility">
      <NDrawerContent
        :title="selectedMember?.latestNickname || selectedMember?.platformUid"
        closable
      >
        <div class="space-y-5">
          <NCard title="历史地址" size="small">
            <template #header-extra>
              <NButton size="small" type="primary" @click="openAddAddress">
                <template #icon
                  ><NIcon><AddOutline /></NIcon
                ></template>
                添加地址
              </NButton>
            </template>
            <div
              v-if="selectedMember?.addresses?.filter((item) => !item.isDeleted).length"
              class="space-y-3"
            >
              <div
                v-for="address in selectedMember.addresses.filter((item) => !item.isDeleted)"
                :key="address.id"
                class="rounded-xl border border-slate-200 p-3 dark:border-slate-700"
              >
                <div class="flex items-center justify-between">
                  <strong>{{ address.recipientName }}</strong>
                  <NTag v-if="address.isDefault" type="success" size="small" round>默认</NTag>
                </div>
                <p class="app-copy mt-1">{{ address.phone }}</p>
                <p class="app-copy mt-1">{{ address.address }}</p>
                <div class="mt-2 flex gap-2">
                  <NButton
                    v-if="!address.isDefault"
                    size="small"
                    secondary
                    @click="handleDefault(address.id)"
                  >
                    设为默认
                  </NButton>
                  <NButton size="small" secondary @click="openEditAddress(address)">
                    <template #icon
                      ><NIcon><CreateOutline /></NIcon
                    ></template>
                    编辑
                  </NButton>
                  <NButton
                    size="small"
                    secondary
                    type="error"
                    @click="handleDeleteAddress(address.id)"
                  >
                    <template #icon
                      ><NIcon><TrashOutline /></NIcon
                    ></template>
                    删除
                  </NButton>
                </div>
              </div>
            </div>
            <NEmpty v-else description="暂无地址" />
          </NCard>

          <NCard title="昵称历史" size="small">
            <NTimeline>
              <NTimelineItem
                v-for="nickname in selectedMember?.nicknames ?? []"
                :key="nickname.id"
                type="info"
                :title="nickname.nickname"
                :content="new Date(nickname.createdAt).toLocaleString()"
              />
            </NTimeline>
          </NCard>
        </div>
      </NDrawerContent>
    </NDrawer>

    <NModal
      v-model:show="showAddressModal"
      preset="card"
      :title="editingAddress ? '编辑地址' : '添加地址'"
      style="max-width: 480px"
    >
      <NForm label-placement="top">
        <NFormItem label="收件人" required>
          <NInput v-model:value="addressForm.recipientName" placeholder="收件人姓名" />
        </NFormItem>
        <NFormItem label="手机号" required>
          <NInput v-model:value="addressForm.phone" placeholder="手机号码" />
        </NFormItem>
        <NFormItem label="地址" required>
          <NInput v-model:value="addressForm.address" type="textarea" placeholder="详细地址" />
        </NFormItem>
        <NButton type="primary" block :loading="isSavingAddress" @click="saveAddress">
          {{ editingAddress ? '保存修改' : '添加地址' }}
        </NButton>
      </NForm>
    </NModal>
  </div>
</template>
