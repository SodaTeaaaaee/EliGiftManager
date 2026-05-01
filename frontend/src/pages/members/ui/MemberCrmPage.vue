<script setup lang="ts">
import { AddOutline, CreateOutline, TrashOutline } from '@vicons/ionicons5'
import { SearchOutline } from '@vicons/ionicons5'
import { computed, h, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import {
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
import { addMemberAddress, deleteMemberAddress, getDashboard, isWailsRuntimeAvailable, listMembers, setDefaultAddress, updateMemberAddress, WAILS_PREVIEW_MESSAGE, type MemberItem } from '@/shared/lib/wails/app'
import type { model } from '../../../../wailsjs/go/models'

const message = useMessage()
const members = ref<MemberItem[]>([])
const keyword = ref('')
const platform = ref('')
const page = ref(1)
const pageSize = ref(12)
const total = ref(0)
const selectedMember = ref<MemberItem | null>(null)
const isLoading = ref(false)
const errorMessage = ref('')
const dashboardStats = ref({ memberCount: 0, addressCount: 0, missingAddresses: 0 })
const showMissingOnly = ref(false)
const tableCardRef = ref<any>(null)
const showAddressModal = ref(false)
const editingAddress = ref<model.MemberAddress | null>(null)
const addressForm = ref({ recipientName: '', phone: '', address: '' })
const isSavingAddress = ref(false)

const platformCatalog = ref<string[]>([])

const platformOptions = computed(() =>
  platformCatalog.value.map((value) => ({ label: value, value })),
)
const filteredMembers = computed(() =>
  showMissingOnly.value ? members.value.filter(m => m.activeAddressCount === 0) : members.value
)

const columns: DataTableColumns<MemberItem> = [
  {
    title: '会员',
    key: 'latestNickname',
    minWidth: 160,
    sorter: true,
    render: (row) =>
      h('div', { class: 'flex items-center gap-3' }, [
        h(
          NAvatar,
          {
            size: 34,
            color: 'var(--accent-surface)',
            style: { color: 'var(--accent)', fontWeight: '700' },
          },
          { default: () => (row.latestNickname || row.platformUid).slice(0, 1).toUpperCase() },
        ),
        h('div', [
          h('div', { class: 'font-semibold' }, row.latestNickname || row.platformUid),
          h('div', { class: 'app-copy' }, `${row.platform} / ${row.platformUid}`),
        ]),
      ]),
  },
  {
    title: '地址状态',
    key: 'activeAddressCount',
    width: 120,
    sorter: true,
    render: (row) =>
      h(
        NTag,
        { type: row.activeAddressCount > 0 ? 'success' : 'error', size: 'small', round: true },
        { default: () => (row.activeAddressCount > 0 ? '已完善' : '缺地址') },
      ),
  },
  { title: '默认收件人', key: 'latestRecipient', minWidth: 100, sorter: true, render: (row) => row.latestRecipient || '-' },
  { title: '手机', key: 'latestPhone', minWidth: 100, sorter: true, render: (row) => row.latestPhone || '-' },
  { title: '地址', key: 'latestAddress', minWidth: 180, sorter: true, ellipsis: { tooltip: true }, render: (row) => row.latestAddress || '-' },
]

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

async function loadMembers() {
  if (!isWailsRuntimeAvailable()) {
    members.value = []
    total.value = 0
    platformCatalog.value = []
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return
  }

  isLoading.value = true
  errorMessage.value = ''

  try {
    const selectedMemberID = selectedMember.value?.id
    const payload = await listMembers(page.value, pageSize.value, keyword.value, platform.value)
    members.value = payload.items
    total.value = payload.total
    platformCatalog.value = payload.platforms
    if (selectedMemberID) {
      selectedMember.value = payload.items.find((member) => member.id === selectedMemberID) ?? null
    }
  } catch (error) {
    console.error(error)
    errorMessage.value = '加载会员数据库失败。'
  } finally {
    isLoading.value = false
  }
}

function searchMembers() {
  page.value = 1
  void loadMembers()
}

function handlePageChange(nextPage: number) {
  if (nextPage === page.value) return
  page.value = nextPage
  void loadMembers()
}

function handlePageSizeChange(nextPageSize: number) {
  if (nextPageSize === pageSize.value) return
  pageSize.value = nextPageSize
  page.value = 1
  void loadMembers()
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
    await loadMembers()
    selectedMember.value = members.value.find((member) => member.id === memberId) ?? null
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
  addressForm.value = { recipientName: addr.recipientName, phone: addr.phone, address: addr.address }
  showAddressModal.value = true
}

async function saveAddress() {
  if (!selectedMember.value) return
  isSavingAddress.value = true
  try {
    if (editingAddress.value) {
      await updateMemberAddress(editingAddress.value.id, addressForm.value.recipientName, addressForm.value.phone, addressForm.value.address)
      message.success('地址已更新')
    } else {
      await addMemberAddress(selectedMember.value.id, addressForm.value.recipientName, addressForm.value.phone, addressForm.value.address)
      message.success('地址已添加')
    }
    showAddressModal.value = false
    await loadMembers()
    selectedMember.value = members.value.find((m) => m.id === selectedMember.value!.id) ?? null
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
    await loadMembers()
    if (selectedMember.value) {
      selectedMember.value = members.value.find((m) => m.id === selectedMember.value!.id) ?? null
    }
  } catch (error) {
    message.error(String(error))
  }
}

const ROW_HEIGHT = 52
let resizeObserver: ResizeObserver | null = null

function recomputePageSize() {
  const el = tableCardRef.value?.$el as HTMLElement | undefined
  if (!el) return
  const headerH = (el.querySelector('.n-card-header') as HTMLElement)?.offsetHeight ?? 44
  const available = el.clientHeight - headerH - 24 - 64
  const newSize = Math.max(5, Math.floor(available / ROW_HEIGHT))
  if (Math.abs(newSize - pageSize.value) > 1) {
    pageSize.value = newSize
  }
}

onMounted(async () => {
  await nextTick()
  await loadDashboardStats()
  await loadMembers()
  const el = tableCardRef.value?.$el
  if (el) {
    resizeObserver = new ResizeObserver(() => recomputePageSize())
    resizeObserver.observe(el)
    recomputePageSize()
  }
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
})
</script>

<template>
  <section class="h-full flex flex-col space-y-5">
    <header class="flex flex-col gap-3 shrink-0 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Member CRM</p>
        <h1 class="app-title mt-2">会员管理</h1>
        <p class="app-copy mt-2">按平台隔离检索会员，维护默认地址与昵称历史。</p>
      </div>
      <NButton :loading="isLoading" secondary strong @click="loadMembers">刷新会员</NButton>
    </header>

    <div class="grid gap-4 shrink-0 md:grid-cols-3">
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
        <p class="mt-1 text-2xl font-semibold text-amber-600">{{ dashboardStats.missingAddresses }}</p>
      </NCard>
    </div>

    <NCard ref="tableCardRef" title="会员列表" size="medium" class="flex-1 min-h-0 overflow-auto">
      <template #header-extra>
        <div class="flex gap-2 items-center">
          <NInput v-model:value="keyword" clearable placeholder="搜索昵称 / UID" style="width: 240px" @keyup.enter="searchMembers">
            <template #prefix>
              <NIcon><SearchOutline /></NIcon>
            </template>
          </NInput>
          <NSelect v-model:value="platform" clearable :options="platformOptions" placeholder="平台" style="width: 150px" @update:value="searchMembers" />
          <NButton @click="searchMembers">搜索</NButton>
          <div class="flex items-center gap-2 ml-2">
            <span class="text-xs text-gray-500" style="white-space: nowrap;">仅显示缺地址</span>
            <NSwitch v-model:value="showMissingOnly" size="small" @update:value="searchMembers" />
          </div>
        </div>
      </template>

      <NEmpty v-if="errorMessage" :description="errorMessage" />
      <div v-else class="space-y-4">
        <NDataTable
          :columns="columns"
          :data="filteredMembers"
          :loading="isLoading"
          :bordered="false"
          :scroll-x="900"
          :pagination="false"
          :row-props="(row) => ({ class: 'cursor-pointer', onClick: () => (selectedMember = row) })"
        />
        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <p class="app-copy">共 {{ showMissingOnly ? filteredMembers.length : total }} 条记录</p>
          <NPagination
            :page="page"
            :page-size="pageSize"
            :item-count="showMissingOnly ? filteredMembers.length : total"
            :page-sizes="[12, 24, 48]"
            show-size-picker
            @update:page="handlePageChange"
            @update:page-size="handlePageSizeChange"
          />
        </div>
      </div>
    </NCard>

    <NDrawer :show="!!selectedMember" :width="460" @update:show="handleDrawerVisibility">
      <NDrawerContent :title="selectedMember?.latestNickname || selectedMember?.platformUid" closable>
        <div class="space-y-5">
          <NCard title="历史地址" size="small">
            <template #header-extra>
              <NButton size="small" type="primary" @click="openAddAddress">
                <template #icon><NIcon><AddOutline /></NIcon></template>
                添加地址
              </NButton>
            </template>
            <div v-if="selectedMember?.addresses?.filter((item) => !item.isDeleted).length" class="space-y-3">
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
                  <NButton v-if="!address.isDefault" size="small" secondary @click="handleDefault(address.id)">
                    设为默认
                  </NButton>
                  <NButton size="small" secondary @click="openEditAddress(address)">
                    <template #icon><NIcon><CreateOutline /></NIcon></template>
                    编辑
                  </NButton>
                  <NButton size="small" secondary type="error" @click="handleDeleteAddress(address.id)">
                    <template #icon><NIcon><TrashOutline /></NIcon></template>
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

    <NModal v-model:show="showAddressModal" preset="card" :title="editingAddress ? '编辑地址' : '添加地址'" style="max-width: 480px">
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
  </section>
</template>
