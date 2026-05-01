<script setup lang="ts">
import { DownloadOutline } from '@vicons/ionicons5'
import { computed, h, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NCard, NAlert, NButton, NDataTable, NIcon, NInput, NInputNumber, NModal, NSelect, NTag, useMessage, type DataTableColumns } from 'naive-ui'
import { addDispatchToMember, addMemberAddress, isWailsRuntimeAvailable, listDispatchRecords, listProductsWithTags, listTemplates, listWaveMembers, previewExport, removeDispatchFromMember, setDispatchAddress, updateDispatchQuantity, WAILS_PREVIEW_MESSAGE, type DispatchRecordItem, type TemplateItem } from '@/shared/lib/wails/app'

const message = useMessage()
const route = useRoute()
const router = useRouter()

const waveId = computed(() => Number(route.params.waveId) || 0)

const templates = ref<TemplateItem[]>([])
const records = ref<DispatchRecordItem[]>([])
const quantityEdits = ref<Record<number, number>>({})
const previewExportResult = ref<{ totalRecords: number; missingAddressCount: number } | null>(null)
const isPreviewLoading = ref(false)
const errorMessage = ref('')

// ---- member popup state ----
const showMemberPopup = ref(false)
const selectedMember = ref<{ memberId: number; nickname: string; platformUid: string; platform: string } | null>(null)
const memberRecords = ref<DispatchRecordItem[]>([])
const showAddAddressForm = ref(false)
const newAddressForm = ref({ recipientName: '', phone: '', address: '' })
const isSavingAddress = ref(false)

// ---- add gift modal state ----
const showAddGiftModal = ref(false)
const availableProducts = ref<{ id: number; name: string; factorySku: string }[]>([])
const addGiftQuantity = ref(1)
const addGiftProductId = ref<number | null>(null)

// ---- address & cover map state ----
const memberAddresses = ref<{ id: number; recipientName: string; phone: string; address: string }[]>([])
const selectedAddressId = ref<number | null>(null)
const productCoverMap = ref<Record<number, string>>({})

// ---- computed ----
const exportTemplates = computed(() => templates.value.filter(t => t.type === 'export_order').map(toOption))

function toOption(template: TemplateItem) {
  return { label: `${template.platform || '通用'} / ${template.name}`, value: template.id }
}

const platformTemplateSelections = ref<Record<string, number | null>>({})

const exportPlatforms = computed(() => {
  const platforms = [...new Set(records.value.map(r => r.platform))]
  for (const platform of platforms) {
    if (!(platform in platformTemplateSelections.value)) {
      const candidates = templates.value.filter(t => t.type === 'export_order' && t.platform === platform)
      platformTemplateSelections.value[platform] = candidates.length === 1 ? candidates[0].id : null
    }
  }
  return platforms.map(platform => {
    const candidates = templates.value.filter(t => t.type === 'export_order' && t.platform === platform)
    return {
      platform,
      templateId: platformTemplateSelections.value[platform] ?? null,
      options: candidates.map(t => ({ label: t.name, value: t.id })),
    }
  })
})

const memberGroups = computed(() => {
  const map = new Map<number, { memberId: number; nickname: string; platformUid: string; platform: string; records: DispatchRecordItem[]; addressStatus: string }>()
  for (const r of records.value) {
    if (!map.has(r.memberId)) {
      map.set(r.memberId, {
        memberId: r.memberId,
        nickname: r.memberNickname || r.platformUid,
        platformUid: r.platformUid,
        platform: r.platform,
        records: [],
        addressStatus: r.hasAddress ? '已绑定' : '待补全',
      })
    }
    map.get(r.memberId)!.records.push(r)
  }
  return [...map.values()]
})

const memberGroupColumns: DataTableColumns = [
  { title: '会员', key: 'nickname', minWidth: 140 },
  { title: '平台', key: 'platform', width: 100 },
  { title: 'UID', key: 'platformUid', minWidth: 140 },
  { title: '礼物数', key: 'records', width: 80, render: (row: any) => String(row.records.length) },
  { title: '地址', key: 'addressStatus', width: 80, render: (row: any) => h(NTag, { type: row.addressStatus === '已绑定' ? 'success' : 'warning', size: 'small', round: true }, { default: () => row.addressStatus }) },
]

const giftColumns: DataTableColumns = [
  { title: '', key: 'productImage', width: 56, render: (row: any) => {
      const cover = productCoverMap.value[row.productId]
      return cover ? h('img', { src: '/local-images/' + cover, class: 'w-10 h-10 rounded object-cover' }) : h('div', { class: 'w-10 h-10 rounded bg-gray-100' })
    }
  },
  { title: '礼物', key: 'productName', minWidth: 140 },
  { title: 'SKU', key: 'factorySku', width: 100 },
  {
    title: '数量', key: 'quantity', width: 130, render: (row: any) =>
      h(NInputNumber, { value: row.quantity, size: 'small', min: 1, onUpdateValue: (v: number | null) => { if (v && v !== row.quantity) handleUpdateQuantity(row.id, v) } }),
  },
  {
    title: '', key: 'actions', width: 60, render: (row: any) =>
      h(NButton, { size: 'tiny', type: 'error', secondary: true, onClick: () => handleRemoveGift(row.id) }, { default: () => '删除' }),
  },
]

const addGiftOptions = computed(() =>
  availableProducts.value.map(p => ({ label: `${p.name} (${p.factorySku})`, value: p.id })),
)

// ---- member popup & gift actions ----
async function openMemberPopup(group: typeof memberGroups.value[0]) {
  selectedMember.value = { memberId: group.memberId, nickname: group.nickname, platformUid: group.platformUid, platform: group.platform }
  memberRecords.value = group.records
  showMemberPopup.value = true

  // Load member addresses
  try {
    const allMembers = await listWaveMembers(waveId.value)
    const member = allMembers.find(m => m.id === group.memberId)
    if (member?.addresses) {
      memberAddresses.value = member.addresses
        .filter(a => !(a as any).isDeleted)
        .map(a => ({ id: a.id, recipientName: a.recipientName, phone: a.phone, address: a.address }))
    } else {
      memberAddresses.value = []
    }
    // Find current address from dispatch records
    const currentRecord = group.records.find(r => r.memberAddressId)
    selectedAddressId.value = currentRecord?.memberAddressId ?? null
  } catch { memberAddresses.value = [] }

  // Load product cover images
  try {
    const result = await listProductsWithTags(waveId.value, '', 1, 500)
    const map: Record<number, string> = {}
    for (const item of result.items) {
      map[item.id] = (item as any).coverImage || ''
    }
    productCoverMap.value = map
  } catch { productCoverMap.value = {} }
}

async function handleUpdateQuantity(recordId: number, qty: number) {
  if (qty < 1) return
  try {
    await updateDispatchQuantity(recordId, qty)
    records.value = await listDispatchRecords(waveId.value)
    const group = memberGroups.value.find(g => g.memberId === selectedMember.value?.memberId)
    if (group) memberRecords.value = group.records
  } catch (e) { message.error(String(e)) }
}

async function handleRemoveGift(recordId: number) {
  try {
    await removeDispatchFromMember(recordId)
    records.value = await listDispatchRecords(waveId.value)
    const group = memberGroups.value.find(g => g.memberId === selectedMember.value?.memberId)
    if (group) memberRecords.value = group.records
    else showMemberPopup.value = false
  } catch (e) { message.error(String(e)) }
}

async function handleAddGift() {
  if (!addGiftProductId.value || !selectedMember.value) return
  try {
    await addDispatchToMember(waveId.value, selectedMember.value.memberId, addGiftProductId.value, addGiftQuantity.value)
    showAddGiftModal.value = false
    addGiftProductId.value = null
    addGiftQuantity.value = 1
    records.value = await listDispatchRecords(waveId.value)
    const group = memberGroups.value.find(g => g.memberId === selectedMember.value!.memberId)
    if (group) memberRecords.value = group.records
  } catch (e) { message.error(String(e)) }
}

async function handleAddAddress() {
  if (!selectedMember.value) return
  isSavingAddress.value = true
  try {
    await addMemberAddress(selectedMember.value.memberId, newAddressForm.value.recipientName, newAddressForm.value.phone, newAddressForm.value.address)
    message.success('地址已添加')
    showAddAddressForm.value = false
    newAddressForm.value = { recipientName: '', phone: '', address: '' }
    records.value = await listDispatchRecords(waveId.value)
    // Re-open popup to refresh addresses
    const group = memberGroups.value.find(g => g.memberId === selectedMember.value?.memberId)
    if (group) memberRecords.value = group.records
  } catch (e) { message.error(String(e)) }
  finally { isSavingAddress.value = false }
}

async function handleSetAddress(addressId: number) {
  if (!selectedMember.value || !waveId.value) return
  try {
    await setDispatchAddress(waveId.value, selectedMember.value.memberId, addressId)
    message.success('地址已更新')
    records.value = await listDispatchRecords(waveId.value)
    const group = memberGroups.value.find(g => g.memberId === selectedMember.value.memberId)
    if (group) memberRecords.value = group.records
  } catch (e) { message.error(String(e)) }
}

async function openAddGiftModal() {
  try {
    const result = await listProductsWithTags(waveId.value, '', 1, 500)
    availableProducts.value = result.items
    showAddGiftModal.value = true
  } catch (e) { message.error(String(e)) }
}

// ---- existing logic ----
async function guardRuntime() {
  if (!isWailsRuntimeAvailable()) { errorMessage.value = WAILS_PREVIEW_MESSAGE; return false }
  return true
}

async function loadTemplates() {
  if (!(await guardRuntime())) return
  try { templates.value = await listTemplates() }
  catch (e) { console.error('加载模板失败', e) }
}

async function loadRecords() {
  if (!waveId.value) return
  try { records.value = await listDispatchRecords(waveId.value) }
  catch (e) { console.error('加载发货记录失败', e) }
}

function goPrev() {
  router.push({ name: 'waves-step-tags', params: { waveId: String(waveId.value) } })
}
function goNext() {
  router.push({ name: 'waves-step-export', params: { waveId: String(waveId.value) } })
}

onMounted(async () => {
  await loadTemplates()
  await loadRecords()
  // Auto-preview on mount
  if (waveId.value) {
    isPreviewLoading.value = true
    try {
      previewExportResult.value = await previewExport(waveId.value)
      await loadRecords()
    } catch (e) { console.error('自动预览失败', e) }
    finally { isPreviewLoading.value = false }
  }
})
</script>
<template>
  <NCard size="small" class="h-full overflow-auto">
    <template #header>
      <span class="flex items-center gap-2">
        <NIcon><DownloadOutline /></NIcon>步骤三：导出预览与编辑
      </span>
    </template>
    <div class="space-y-3">
      <div v-if="exportPlatforms.length" class="space-y-2 mb-3">
        <div class="text-xs text-gray-500">导出模板（已自动匹配）</div>
        <div v-for="ep in exportPlatforms" :key="ep.platform" class="flex items-center gap-2">
          <NTag size="small" round>{{ ep.platform }}</NTag>
          <NSelect :value="ep.templateId" :options="ep.options" size="small" style="width:220px" placeholder="选择导出模板"
            @update:value="(v: number) => { platformTemplateSelections[ep.platform] = v }" />
        </div>
      </div>
      <NAlert v-if="previewExportResult" type="info" :show-icon="false">
        共 {{ previewExportResult.totalRecords }} 条记录
        <span v-if="previewExportResult.missingAddressCount > 0">，{{ previewExportResult.missingAddressCount }} 条缺失地址</span>
      </NAlert>
      <p class="text-xs text-gray-400 mb-2">点击会员行查看该会员的礼物明细</p>
      <NDataTable
        :columns="memberGroupColumns"
        :data="memberGroups"
        :bordered="false"
        :pagination="{ pageSize: 10 }"
        :row-props="(row: any) => ({ class: 'cursor-pointer', onClick: () => openMemberPopup(row) })"
      />
    </div>
    <div class="flex justify-between mt-6 pt-4 border-t border-gray-100 dark:border-gray-700">
      <NButton @click="goPrev">上一步</NButton>
      <NButton type="primary" @click="goNext">下一步</NButton>
    </div>

    <!-- Member Gift Detail Popup -->
    <NModal v-model:show="showMemberPopup" :mask-closable="false" style="width: 85vw; max-width: 1100px;">
      <NCard :title="`${selectedMember?.nickname} · 礼物明细`" size="medium" closable @close="showMemberPopup = false">
        <!-- Address Management -->
        <div class="mb-4 p-3 border rounded-lg">
          <div class="text-xs text-gray-500 mb-1">收件地址</div>
          <div class="flex items-center gap-2 mb-2">
            <NSelect :value="selectedAddressId" :options="memberAddresses.map(a => ({ label: `${a.recipientName} ${a.phone} ${a.address}`, value: a.id }))"
              placeholder="选择地址" size="small" style="flex:1; min-width: 200px" clearable
              @update:value="(v: number) => { selectedAddressId = v; if (v) handleSetAddress(v) }" />
            <NButton size="tiny" secondary @click="showAddAddressForm = !showAddAddressForm">{{ showAddAddressForm ? '取消' : '添加地址' }}</NButton>
          </div>
          <div v-if="showAddAddressForm" class="mt-2 space-y-2">
            <NInput v-model:value="newAddressForm.recipientName" size="small" placeholder="收件人" />
            <NInput v-model:value="newAddressForm.phone" size="small" placeholder="手机号" />
            <NInput v-model:value="newAddressForm.address" size="small" placeholder="地址" />
            <NButton size="small" type="primary" :loading="isSavingAddress" @click="handleAddAddress">保存地址</NButton>
          </div>
        </div>

        <!-- Gift List -->
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-medium">礼物清单（{{ memberRecords.length }} 件）</span>
          <NButton size="small" type="primary" @click="openAddGiftModal">添加礼物</NButton>
        </div>
        <NDataTable :columns="giftColumns" :data="memberRecords" :bordered="false" :pagination="{ pageSize: 10 }" size="small" />
      </NCard>
    </NModal>

    <!-- Add Gift Modal -->
    <NModal v-model:show="showAddGiftModal" preset="card" title="添加礼物" style="max-width: 420px">
      <div class="space-y-3">
        <NSelect v-model:value="addGiftProductId" :options="addGiftOptions" placeholder="选择商品" />
        <NInputNumber v-model:value="addGiftQuantity" :min="1" />
        <NButton type="primary" block @click="handleAddGift">确认添加</NButton>
      </div>
    </NModal>
  </NCard>
</template>
