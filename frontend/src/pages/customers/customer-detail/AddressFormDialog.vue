<script setup lang="ts">
/**
 * AddressFormDialog — create/edit a customer address (plan §3.6 line 256).
 * Backend `CustomerAddress.province/city/district` are plain unconstrained
 * strings (see contract `backendContract` (e)) — no region-code dataset
 * exists in the backend, so the cascade below is a pure frontend concern.
 *
 * DECISION (contract `uiPrimitives` — no cascade dataset found anywhere in
 * the old tree, which just used free-text `NInput` for all three fields):
 * ship a lightweight two-level cascade — province (closed NSelect, the 34
 * PRC provincial divisions) narrows city (dependent NSelect, a compact list
 * of representative prefecture-level cities per province, NOT exhaustive).
 * District stays a free-text `NInput` — a full province→city→district
 * dataset would be hundreds of entries and the plan explicitly says "do not
 * over-engineer"; acceptance only requires "selecting province narrows city
 * options" (one cascading level), which this satisfies.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NForm, NFormItem, NInput, NModal, NSelect, NSwitch } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { createAddress, updateAddress } from '@/shared/api/bridge'
import type { CustomerAddressDTO } from '@/entities/address'

const props = defineProps<{
  show: boolean
  customerProfileId: number
  writesEnabled: boolean
  /** `null` = create mode; otherwise the address being edited. */
  address: CustomerAddressDTO | null
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  saved: [CustomerAddressDTO]
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

// Compact province → representative-city map. Deliberately NOT exhaustive
// (see header comment) — enough to demonstrate real cascading narrowing.
const REGION_DATA: Record<string, string[]> = {
  北京市: ['北京市'],
  天津市: ['天津市'],
  河北省: ['石家庄市', '唐山市', '保定市', '邯郸市', '秦皇岛市'],
  山西省: ['太原市', '大同市', '长治市', '临汾市'],
  内蒙古自治区: ['呼和浩特市', '包头市', '鄂尔多斯市', '赤峰市'],
  辽宁省: ['沈阳市', '大连市', '鞍山市', '锦州市'],
  吉林省: ['长春市', '吉林市', '延边朝鲜族自治州'],
  黑龙江省: ['哈尔滨市', '齐齐哈尔市', '大庆市'],
  上海市: ['上海市'],
  江苏省: ['南京市', '苏州市', '无锡市', '常州市', '徐州市'],
  浙江省: ['杭州市', '宁波市', '温州市', '嘉兴市', '绍兴市'],
  安徽省: ['合肥市', '芜湖市', '蚌埠市', '安庆市'],
  福建省: ['福州市', '厦门市', '泉州市', '漳州市'],
  江西省: ['南昌市', '赣州市', '九江市'],
  山东省: ['济南市', '青岛市', '烟台市', '潍坊市', '临沂市'],
  河南省: ['郑州市', '洛阳市', '开封市', '南阳市'],
  湖北省: ['武汉市', '宜昌市', '襄阳市'],
  湖南省: ['长沙市', '株洲市', '衡阳市'],
  广东省: ['广州市', '深圳市', '东莞市', '佛山市', '珠海市'],
  广西壮族自治区: ['南宁市', '柳州市', '桂林市'],
  海南省: ['海口市', '三亚市'],
  重庆市: ['重庆市'],
  四川省: ['成都市', '绵阳市', '德阳市', '宜宾市'],
  贵州省: ['贵阳市', '遵义市'],
  云南省: ['昆明市', '大理白族自治州', '曲靖市'],
  西藏自治区: ['拉萨市', '日喀则市'],
  陕西省: ['西安市', '咸阳市', '宝鸡市'],
  甘肃省: ['兰州市', '天水市'],
  青海省: ['西宁市'],
  宁夏回族自治区: ['银川市', '石嘴山市'],
  新疆维吾尔自治区: ['乌鲁木齐市', '喀什地区', '伊犁哈萨克自治州'],
  香港特别行政区: ['香港特别行政区'],
  澳门特别行政区: ['澳门特别行政区'],
  台湾省: ['台北市', '高雄市', '台中市'],
}

const provinceOptions = computed<SelectOption[]>(() =>
  Object.keys(REGION_DATA).map((name) => ({ label: name, value: name })),
)

const label = ref('')
const recipientName = ref('')
const phone = ref('')
const province = ref<string | null>(null)
const city = ref<string | null>(null)
const district = ref('')
const addressLine1 = ref('')
const addressLine2 = ref('')
const postalCode = ref('')
const isDefault = ref(false)
const submitting = ref(false)

const cityOptions = computed<SelectOption[]>(() => {
  if (!province.value) return []
  return (REGION_DATA[province.value] ?? []).map((name) => ({ label: name, value: name }))
})

// Narrow city whenever province changes to something that no longer offers
// the currently-selected city (the actual "cascading constraint" behavior).
watch(province, (next, prev) => {
  if (next !== prev && city.value && !(REGION_DATA[next ?? '']?.includes(city.value))) {
    city.value = null
  }
})

function resetForm(): void {
  const existing = props.address
  label.value = existing?.label ?? ''
  recipientName.value = existing?.recipientName ?? ''
  phone.value = existing?.phone ?? ''
  province.value = existing?.province || null
  city.value = existing?.city || null
  district.value = existing?.district ?? ''
  addressLine1.value = existing?.addressLine1 ?? ''
  addressLine2.value = existing?.addressLine2 ?? ''
  postalCode.value = existing?.postalCode ?? ''
  isDefault.value = existing?.isDefault ?? false
}

watch(
  () => props.show,
  (visible) => {
    if (visible) resetForm()
  },
)

const isEditMode = computed(() => props.address != null)

// Light validation only (contract: "required recipient/phone/province" —
// do not over-engineer beyond that).
const phonePattern = /^1[3-9]\d{9}$/
const canSubmit = computed(
  () =>
    !submitting.value &&
    props.writesEnabled &&
    recipientName.value.trim().length > 0 &&
    phonePattern.test(phone.value.trim()) &&
    !!province.value,
)

function close(): void {
  emit('update:show', false)
}

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value || !props.writesEnabled) return
  submitting.value = true
  try {
    const payload = {
      customerProfileId: props.customerProfileId,
      label: label.value.trim(),
      recipientName: recipientName.value.trim(),
      phone: phone.value.trim(),
      country: 'CN',
      province: province.value ?? '',
      city: city.value ?? '',
      district: district.value.trim(),
      addressLine1: addressLine1.value.trim(),
      addressLine2: addressLine2.value.trim(),
      postalCode: postalCode.value.trim(),
      isDefault: isDefault.value,
      isTest: props.address?.isTest ?? false,
      validationStatus: props.address?.validationStatus ?? '',
      validationDetail: props.address?.validationDetail ?? '',
      extraData: props.address?.extraData ?? '',
    }
    const saved = props.address
      ? await updateAddress({ id: props.address.id, ...payload })
      : await createAddress(payload)
    feedback.success(t('customerDetail.feedback.addressSaved'))
    emit('saved', saved)
    close()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="isEditMode ? t('customerDetail.addresses.editAction') : t('customerDetail.addresses.createAction')"
    :style="{ width: 'min(520px, 92vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(v: boolean) => emit('update:show', v)"
  >
    <NForm label-placement="top">
      <p v-if="!writesEnabled" class="address-form-dialog__disabled">{{ t('customerDetail.writesDisabledReason') }}</p>
      <NFormItem :label="t('address.fields.label')">
        <NInput v-model:value="label" :disabled="submitting || !writesEnabled" />
      </NFormItem>
      <NFormItem :label="t('address.fields.recipientName')">
        <NInput v-model:value="recipientName" :disabled="submitting || !writesEnabled" />
      </NFormItem>
      <NFormItem :label="t('address.fields.phone')">
        <NInput v-model:value="phone" :disabled="submitting || !writesEnabled" />
      </NFormItem>
      <p class="address-form-dialog__hint">{{ t('address.cascadeHint') }}</p>
      <div class="address-form-dialog__region-row">
        <NFormItem :label="t('address.fields.province')">
          <NSelect
            v-model:value="province"
            :options="provinceOptions"
            :placeholder="t('address.cascadePlaceholder.province')"
            :disabled="submitting || !writesEnabled"
            filterable
          />
        </NFormItem>
        <NFormItem :label="t('address.fields.city')">
          <NSelect
            v-model:value="city"
            :options="cityOptions"
            :placeholder="t('address.cascadePlaceholder.city')"
            :disabled="submitting || !province || !writesEnabled"
            filterable
          />
        </NFormItem>
        <NFormItem :label="t('address.fields.district')">
          <NInput v-model:value="district" :placeholder="t('address.cascadePlaceholder.district')" :disabled="submitting || !writesEnabled" />
        </NFormItem>
      </div>
      <NFormItem :label="t('address.fields.addressLine1')">
        <NInput v-model:value="addressLine1" :disabled="submitting || !writesEnabled" />
      </NFormItem>
      <NFormItem :label="t('address.fields.addressLine2')">
        <NInput v-model:value="addressLine2" :disabled="submitting || !writesEnabled" />
      </NFormItem>
      <NFormItem :label="t('address.fields.postalCode')">
        <NInput v-model:value="postalCode" :disabled="submitting || !writesEnabled" />
      </NFormItem>
      <NFormItem :label="t('address.fields.isDefault')">
        <NSwitch v-model:value="isDefault" :disabled="submitting || !writesEnabled" />
      </NFormItem>
    </NForm>
    <template #footer>
      <div class="address-form-dialog__footer">
        <NButton :disabled="submitting" @click="close">{{ t('address.cancelAction') }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('address.saveAction') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.address-form-dialog__hint {
  margin: calc(var(--space-2) * -1) 0 var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.address-form-dialog__disabled {
  margin: 0 0 var(--space-2);
  color: var(--status-warning-fg);
  font-size: var(--font-size-xs);
}

.address-form-dialog__region-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-2);
}

.address-form-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
