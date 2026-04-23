<script setup lang="ts">
import { CloudUploadOutline, PeopleCircleOutline } from '@vicons/ionicons5'
import { computed, h, ref } from 'vue'
import {
  NAvatar,
  NButton,
  NCard,
  NDataTable,
  NIcon,
  NInput,
  NTag,
  type DataTableColumns,
} from 'naive-ui'

interface MemberRecord {
  name: string
  level: string
  city: string
  address: string
  orders: number
}

const members: MemberRecord[] = [
  { name: 'Mika', level: 'VIP', city: 'Tokyo', address: '已完整', orders: 12 },
  { name: 'Haruka', level: 'Gold', city: 'Osaka', address: '待校验', orders: 8 },
  { name: 'Nana', level: 'Silver', city: 'Nagoya', address: '缺少门牌', orders: 5 },
  { name: 'Yui', level: 'New', city: 'Fukuoka', address: '已完整', orders: 2 },
]

const searchKeyword = ref('')

function getAddressClass(status: string) {
  if (status === '已完整') {
    return 'success'
  }

  if (status === '待校验') {
    return 'warning'
  }

  return 'error'
}

function getLevelClass(level: string) {
  if (level === 'VIP') {
    return 'error'
  }

  if (level === 'Gold') {
    return 'warning'
  }

  if (level === 'Silver') {
    return 'info'
  }

  return 'default'
}

const filteredMembers = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase()

  if (keyword.length === 0) {
    return members
  }

  return members.filter((member) =>
    [member.name, member.level, member.city, member.address].some((value) =>
      value.toLowerCase().includes(keyword),
    ),
  )
})

const columns: DataTableColumns<MemberRecord> = [
  {
    title: '粉丝',
    key: 'name',
    render: (row) =>
      h('div', {
        style: {
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
        },
      }, [
        h(
          NAvatar,
          {
            size: 34,
            color: 'var(--accent-surface)',
            style: {
              color: 'var(--accent)',
              fontWeight: '600',
            },
          },
          {
            default: () => row.name.slice(0, 1),
          },
        ),
        h('span', {
          style: {
            fontWeight: '600',
          },
        }, row.name),
      ]),
  },
  {
    title: '等级',
    key: 'level',
    render: (row) =>
      h(
        NTag,
        {
          type: getLevelClass(row.level),
          size: 'small',
          round: true,
        },
        {
          default: () => row.level,
        },
      ),
  },
  {
    title: '城市',
    key: 'city',
  },
  {
    title: '地址状态',
    key: 'address',
    render: (row) =>
      h(
        NTag,
        {
          type: getAddressClass(row.address),
          size: 'small',
          round: true,
        },
        {
          default: () => row.address,
        },
      ),
  },
  {
    title: '派发次数',
    key: 'orders',
  },
]
</script>

<template>
  <section class="space-y-5">
    <header>
      <p class="app-kicker">Member CRM</p>
      <h1 class="app-title mt-2">会员地址库</h1>
      <p class="app-copy mt-2">统一管理粉丝档案、地址完整度和历史派发记录。</p>
    </header>

    <NCard size="medium">
      <template #header>
        <div class="flex items-center gap-2">
          <NIcon :size="18">
            <PeopleCircleOutline />
          </NIcon>
          <span class="app-heading-sm">会员档案</span>
        </div>
      </template>
      <template #header-extra>
        <NTag size="small" round>{{ filteredMembers.length }} profiles</NTag>
      </template>

      <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <NInput
          v-model:value="searchKeyword"
          class="md:max-w-md"
          placeholder="搜索昵称、手机或收件城市"
        />
        <NButton secondary strong>
          <template #icon>
            <NIcon :size="18">
              <CloudUploadOutline />
            </NIcon>
          </template>
          批量导入
        </NButton>
      </div>

      <NDataTable
        class="mt-4"
        :columns="columns"
        :data="filteredMembers"
        :pagination="false"
        :bordered="false"
        :single-line="false"
      />
    </NCard>
  </section>
</template>
