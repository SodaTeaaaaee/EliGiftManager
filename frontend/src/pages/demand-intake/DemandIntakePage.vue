<template>
  <div class="demand-intake-page">
    <h1 class="text-xl font-medium mb-4">需求导入</h1>

    <n-space vertical size="large">
      <n-card title="导入需求">
        <n-space vertical :size="16">
          <n-space>
            <n-select
              v-model:value="form.integrationProfileId"
              :options="profileOptions"
              :loading="profilesLoading"
              placeholder="集成配置 (必选)"
              style="width: 280px"
              filterable
            />
            <n-select
              v-model:value="form.kind"
              :options="kindOptions"
              placeholder="Kind"
              style="width: 200px"
              :disabled="profileSelected"
            />
            <n-select
              v-model:value="form.captureMode"
              :options="captureModeOptions"
              placeholder="Capture Mode"
              style="width: 200px"
            />
            <n-input
              v-model:value="form.sourceChannel"
              placeholder="Source Channel"
              style="width: 200px"
              :disabled="profileSelected"
            />
            <n-input
              v-model:value="form.sourceSurface"
              placeholder="Source Surface"
              style="width: 200px"
              :disabled="profileSelected"
            />
            <n-input
              v-model:value="form.sourceDocumentNo"
              placeholder="Source Document No"
              style="width: 200px"
            />
          </n-space>

          <n-space>
            <n-input
              v-model:value="form.sourceCustomerRef"
              placeholder="客户引用标识 (Source Customer Ref)"
              style="width: 280px"
            />
            <n-input-number
              v-model:value="form.customerProfileId"
              placeholder="客户档案ID (可选)"
              :min="1"
              clearable
              style="width: 220px"
            />
          </n-space>

          <n-card
            v-for="(line, idx) in form.lines"
            :key="idx"
            :title="`Line ${idx + 1}`"
            size="small"
          >
            <template #header-extra>
              <n-button
                text
                type="error"
                size="small"
                @click="removeLine(idx)"
                :disabled="form.lines.length <= 1"
              >
                删除
              </n-button>
            </template>
            <n-space vertical :size="8">
              <n-space>
                <n-select
                  v-model:value="line.lineType"
                  :options="lineTypeOptions"
                  placeholder="Line Type"
                  style="width: 180px"
                />
                <n-select
                  v-model:value="line.routingDisposition"
                  :options="routingDispositionOptions"
                  placeholder="Routing Disposition"
                  style="width: 180px"
                />
                <n-select
                  v-model:value="line.recipientInputState"
                  :options="recipientInputStateOptions"
                  placeholder="输入状态"
                  style="width: 180px"
                />
                <n-select
                  v-model:value="line.obligationTriggerKind"
                  :options="obligationTriggerKindOptions"
                  placeholder="触发类型"
                  style="width: 200px"
                />
                <n-select
                  v-model:value="line.entitlementAuthority"
                  :options="entitlementAuthorityOptions"
                  placeholder="权益判定"
                  style="width: 180px"
                />
              </n-space>
              <n-space>
                <n-input
                  v-model:value="line.externalTitle"
                  placeholder="External Title"
                  style="width: 180px"
                />
                <n-input-number
                  v-model:value="line.requestedQuantity"
                  placeholder="Qty"
                  :min="1"
                  style="width: 100px"
                />
                <n-input-number
                  v-model:value="line.productMasterId"
                  placeholder="商品主档ID"
                  :min="1"
                  clearable
                  style="width: 140px"
                />
                <n-input
                  v-model:value="line.entitlementCode"
                  placeholder="权益编码"
                  style="width: 140px"
                />
                <n-input
                  v-model:value="line.giftLevelSnapshot"
                  placeholder="等级快照"
                  style="width: 140px"
                />
              </n-space>
              <n-space>
                <n-input
                  v-model:value="line.routingReasonCode"
                  placeholder="路由原因码"
                  style="width: 160px"
                />
                <n-input
                  v-model:value="line.eligibilityContextRef"
                  placeholder="资格上下文引用"
                  style="width: 200px"
                />
                <n-input
                  v-model:value="line.recipientInputPayload"
                  placeholder="输入附件(JSON)"
                  style="width: 200px"
                />
              </n-space>
            </n-space>
          </n-card>

          <n-button dashed size="small" @click="addLine"> 添加一行 </n-button>

          <n-button type="primary" @click="importDemand" :loading="loading" :disabled="!formValid">
            导入需求
          </n-button>
        </n-space>
      </n-card>

      <n-alert v-if="error" type="error" :title="error" />

      <n-card v-if="result" title="导入成功">
        <n-space vertical>
          <p>需求单 ID: {{ result.id }}</p>
          <p>Kind: {{ result.kind }}</p>
          <p>SourceDocumentNo: {{ result.sourceDocumentNo }}</p>

          <n-divider />

          <n-space align="center">
            <n-select
              v-model:value="selectedWaveId"
              :options="waveOptions"
              placeholder="选择波次"
              style="width: 200px"
              :loading="wavesLoading"
            />
            <n-button
              type="primary"
              @click="assignToWave"
              :loading="assigning"
              :disabled="!selectedWaveId"
            >
              接手到此波次
            </n-button>
          </n-space>

          <n-alert v-if="assignMsg" type="success" :title="assignMsg" />
          <n-alert v-if="assignErr" type="error" :title="assignErr" />
        </n-space>
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { NCard, NButton, NInput, NInputNumber, NSelect, NSpace, NAlert, NDivider } from 'naive-ui'
import { importDemandDocument, listWaves, assignDemandToWave, listProfiles } from '@/shared/lib/wails/app'
import { dto } from '@/../wailsjs/go/models'

// ── Options ──

const profileOptions = ref<Array<{ label: string; value: number }>>([])
const profilesLoading = ref(false)
const profilesList = ref<dto.IntegrationProfileDTO[]>([])

async function loadProfiles() {
  profilesLoading.value = true
  try {
    const list = await listProfiles()
    profilesList.value = list
    profileOptions.value = list.map((p) => ({
      label: `${p.profileKey} (${p.sourceChannel})`,
      value: p.id,
    }))
  } catch {
    // offline — handled by guard
  } finally {
    profilesLoading.value = false
  }
}

onMounted(() => {
  loadProfiles()
})

const kindOptions = [
  { label: 'Membership Entitlement', value: 'membership_entitlement' },
  { label: 'Retail Order', value: 'retail_order' },
]

const captureModeOptions = [
  { label: 'Manual Entry', value: 'manual_entry' },
  { label: 'Document Import', value: 'document_import' },
  { label: 'API Ingest', value: 'api_ingest' },
]

const lineTypeOptions = [
  { label: 'Entitlement Rule', value: 'entitlement_rule' },
  { label: 'SKU Order', value: 'sku_order' },
  { label: 'Manual Entry', value: 'manual_entry' },
]

const routingDispositionOptions = [
  { label: 'Accepted', value: 'accepted' },
  { label: 'Pending Intake', value: 'pending_intake' },
  { label: 'Deferred', value: 'deferred' },
  { label: 'Excluded Manual', value: 'excluded_manual' },
  { label: 'Excluded Duplicate', value: 'excluded_duplicate' },
  { label: 'Excluded Revoked', value: 'excluded_revoked' },
]

const recipientInputStateOptions = [
  { label: 'Not Required', value: 'not_required' },
  { label: 'Waiting', value: 'waiting_for_input' },
  { label: 'Partial', value: 'partially_collected' },
  { label: 'Ready', value: 'ready' },
  { label: 'Waived', value: 'waived' },
  { label: 'Expired', value: 'expired' },
]

const obligationTriggerKindOptions = [
  { label: 'Periodic Membership', value: 'periodic_membership' },
  { label: 'Loyalty Membership', value: 'loyalty_membership' },
  { label: 'Supporter Purchase', value: 'supporter_only_purchase' },
  { label: 'Member Discount', value: 'member_only_discount_purchase' },
  { label: 'Campaign Reward', value: 'campaign_reward' },
  { label: 'Manual Compensation', value: 'manual_compensation' },
]

const entitlementAuthorityOptions = [
  { label: 'Local Policy', value: 'local_policy' },
  { label: 'Upstream Platform', value: 'upstream_platform' },
  { label: 'Manual Grant', value: 'manual_grant' },
]

// ── Form state ──

interface LineForm {
  lineType: string
  obligationTriggerKind: string
  entitlementAuthority: string
  recipientInputState: string
  routingDisposition: string
  routingReasonCode: string
  eligibilityContextRef: string
  entitlementCode: string
  giftLevelSnapshot: string
  productMasterId: number | null
  recipientInputPayload: string
  externalTitle: string
  requestedQuantity: number
}

const makeLine = (): LineForm => ({
  lineType: 'entitlement_rule',
  obligationTriggerKind: 'periodic_membership',
  entitlementAuthority: 'local_policy',
  recipientInputState: 'not_required',
  routingDisposition: 'accepted',
  routingReasonCode: '',
  eligibilityContextRef: '',
  entitlementCode: '',
  giftLevelSnapshot: '',
  productMasterId: null,
  recipientInputPayload: '',
  externalTitle: '',
  requestedQuantity: 1,
})

const form = reactive({
  kind: 'membership_entitlement',
  captureMode: 'manual_entry',
  sourceChannel: '',
  sourceSurface: '',
  sourceDocumentNo: '',
  sourceCustomerRef: '',
  customerProfileId: null as number | null,
  integrationProfileId: null as number | null,
  lines: [makeLine()] as LineForm[],
})

// Profile-driven auto-fill
const profileSelected = computed(() => form.integrationProfileId != null)

watch(
  () => form.integrationProfileId,
  (newId) => {
    if (newId != null) {
      const profile = profilesList.value.find((p) => p.id === newId)
      if (profile) {
        form.kind = profile.demandKind || form.kind
        form.sourceChannel = profile.sourceChannel || form.sourceChannel
        form.sourceSurface = profile.sourceSurface || ''
      }
    }
  },
)

const formValid = computed(() => {
  if (!form.kind || !form.captureMode) return false
  if (!form.integrationProfileId) return false
  for (const line of form.lines) {
    if (!line.lineType || !line.routingDisposition || line.requestedQuantity < 1) return false
  }
  return true
})

function addLine() {
  form.lines.push(makeLine())
}

function removeLine(idx: number) {
  if (form.lines.length > 1) {
    form.lines.splice(idx, 1)
  }
}

// ── Import ──

const loading = ref(false)
const result = ref<dto.DemandDocumentDTO | null>(null)
const error = ref<string | null>(null)

async function importDemand() {
  loading.value = true
  error.value = null
  result.value = null

  try {
    const input = {
      kind: form.kind,
      captureMode: form.captureMode,
      sourceChannel: form.sourceChannel || 'manual',
      sourceSurface: form.sourceSurface || undefined,
      sourceDocumentNo: form.sourceDocumentNo || `IMPORT-${Date.now()}`,
      sourceCustomerRef: form.sourceCustomerRef,
      customerProfileId: form.customerProfileId || undefined,
      integrationProfileId: form.integrationProfileId || undefined,
      lines: form.lines.map((l) => ({ ...l })),
    }
    result.value = await importDemandDocument(input)
  } catch (e: any) {
    error.value = e?.message ?? String(e)
  } finally {
    loading.value = false
  }
}

// ── Assign to wave ──

const waves = ref<dto.WaveDTO[]>([])
const wavesLoading = ref(false)
const selectedWaveId = ref<number | null>(null)
const assigning = ref(false)
const assignMsg = ref('')
const assignErr = ref('')

const waveOptions = computed(() =>
  waves.value.map((w) => ({
    label: `${w.waveNo} — ${w.name}`,
    value: w.id,
  })),
)

// Load waves when result appears
watch(result, (val) => {
  if (val) {
    loadWaves()
  }
})

async function loadWaves() {
  wavesLoading.value = true
  try {
    waves.value = await listWaves()
  } catch {
    // offline — handled by guard
  } finally {
    wavesLoading.value = false
  }
}

async function assignToWave() {
  if (!result.value || !selectedWaveId.value) return
  assigning.value = true
  assignMsg.value = ''
  assignErr.value = ''
  try {
    await assignDemandToWave(selectedWaveId.value, result.value.id)
    assignMsg.value = `需求单 ${result.value.id} 已接手到波次 ${selectedWaveId.value}`
  } catch (e: any) {
    const msg: string = e?.message ?? String(e)
    if (/unique|duplicate|already/i.test(msg)) {
      assignErr.value = '该需求已接手到此波次'
    } else {
      assignErr.value = msg
    }
  } finally {
    assigning.value = false
  }
}
</script>
