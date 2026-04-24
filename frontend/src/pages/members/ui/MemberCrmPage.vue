<script setup lang="ts">
import { SearchOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { NAvatar, NButton, NCard, NDataTable, NEmpty, NIcon, NInput, NTag, type DataTableColumns } from 'naive-ui'
import { WAILS_PREVIEW_MESSAGE, isWailsRuntimeAvailable, listMembers, type MemberItem } from '@/shared/lib/wails/app'

const members = ref<MemberItem[]>([])
const keyword = ref('')
const isLoading = ref(false)
const errorMessage = ref('')

const filteredMembers = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return members.value
  return members.value.filter((member) =>
    [member.latestNickname, member.platform, member.platformUid, member.latestRecipient, member.latestPhone, member.latestAddress]
      .some((value) => value.toLowerCase().includes(text)),
  )
})

const completedCount = computed(() => members.value.filter((member) => member.activeAddressCount > 0).length)
const missingCount = computed(() => members.value.length - completedCount.value)

const columns: DataTableColumns<MemberItem> = [
  {
    title: '会员',
    key: 'latestNickname',
    minWidth: 180,
    render: (row) => h('div', { class: 'flex items-center gap-3' }, [
      h(NAvatar, { size: 34, color: 'var(--accent-surface)', style: { color: 'var(--accent)', fontWeight: '700' } }, { default: () => (row.latestNickname || row.platformUid).slice(0, 1).toUpperCase() }),
      h('div', [h('div', { class: 'font-semibold' }, row.latestNickname || row.platformUid), h('div', { class: 'app-copy' }, `${row.platform} / ${row.platformUid}`)]),
    ]),
  },
  {
    title: '地址状态',
    key: 'activeAddressCount',
    width: 120,
    render: (row) => h(NTag, { type: row.activeAddressCount > 0 ? 'success' : 'error', size: 'small', round: true }, { default: () => (row.activeAddressCount > 0 ? '已完善' : '缺地址') }),
  },
  { title: '收件人', key: 'latestRecipient', minWidth: 120, render: (row) => row.latestRecipient || '-' },
  { title: '手机', key: 'latestPhone', minWidth: 130, render: (row) => row.latestPhone || '-' },
  { title: '地址', key: 'latestAddress', minWidth: 260, ellipsis: { tooltip: true }, render: (row) => row.latestAddress || '-' },
  { title: '派发次数', key: 'dispatchCount', width: 100 },
]

async function loadMembers() {
  if (!isWailsRuntimeAvailable()) {
    errorMessage.value = WAILS_PREVIEW_MESSAGE
    return
  }

  isLoading.value = true
  errorMessage.value = ''
  try {
    members.value = await listMembers()
  } catch (error) {
    console.error('加载会员失败', error)
    errorMessage.value = '加载会员数据库失败，请查看后端日志。'
  } finally {
    isLoading.value = false
  }
}

onMounted(loadMembers)
</script>

<template>
  <section class="space-y-5">
    <header class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="app-kicker">Member CRM</p>
        <h1 class="app-title mt-2">会员与地址库</h1>
        <p class="app-copy mt-2">读取 members、member_nicknames 和 member_addresses，定位缺地址会员。</p>
      </div>
      <NButton :loading="isLoading" secondary strong @click="loadMembers">刷新会员</NButton>
    </header>

    <NCard size="medium">
      <div class="grid gap-4 md:grid-cols-3">
        <div><p class="app-copy">会员总数</p><p class="mt-1 text-2xl font-semibold">{{ members.length }}</p></div>
        <div><p class="app-copy">地址已完善</p><p class="mt-1 text-2xl font-semibold text-green-600">{{ completedCount }}</p></div>
        <div><p class="app-copy">缺少地址</p><p class="mt-1 text-2xl font-semibold text-amber-600">{{ missingCount }}</p></div>
      </div>
    </NCard>

    <NCard size="medium">
      <template #header>会员列表</template>
      <template #header-extra>
        <NInput v-model:value="keyword" clearable placeholder="搜索昵称、平台 UID、手机或地址" style="width: 280px">
          <template #prefix><NIcon><SearchOutline /></NIcon></template>
        </NInput>
      </template>
      <NEmpty v-if="errorMessage" :description="errorMessage" />
      <NDataTable v-else :columns="columns" :data="filteredMembers" :loading="isLoading" :bordered="false" :scroll-x="900" :pagination="{ pageSize: 12 }" />
    </NCard>
  </section>
</template>
