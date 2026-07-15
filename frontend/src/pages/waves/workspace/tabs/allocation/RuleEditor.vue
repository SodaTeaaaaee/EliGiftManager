<script setup lang="ts">
/**
 * RuleEditor — create/edit form for one allocation policy rule
 * (`WaveAllocationTab.vue`'s rules section, P4 plan §3.3). Builds a
 * `SelectorPayload` discriminated union from a glossary-driven selector-type
 * dropdown + the matching conditional field(s) (`platform` / `level` /
 * comma-separated `participant_ids`), matching `internal/domain/models.go`'s
 * `SelectorPayload` shape exactly.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NForm, NFormItem, NSelect, NInput, NInputNumber, NSwitch, NButton } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { useFeedback } from '@/shared/ui/feedback'
import { useGlossary, type AllocationSelectorTypeValue } from '@/shared/i18n/glossary'
import { createAllocationPolicyRule, updateAllocationPolicyRule } from '@/shared/api/bridge'
import type { AllocationPolicyRule, SelectorPayload } from '@/entities/allocation-policy'
import type { Product } from '@/entities/product'

const props = defineProps<{
  show: boolean
  waveId: number
  products: Product[]
  /** `null` = create mode; otherwise the rule being edited. */
  rule: AllocationPolicyRule | null
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  saved: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const { label: glossaryLabel } = useGlossary()

const SELECTOR_TYPES: AllocationSelectorTypeValue[] = ['wave_all', 'platform_all', 'identity_level', 'explicit_override']

const isEdit = computed(() => props.rule !== null)

const selectorType = ref<AllocationSelectorTypeValue>('wave_all')
const platform = ref('')
const level = ref('')
const participantIdsText = ref('')
const productId = ref<number | null>(null)
const ruleKind = ref('')
const priority = ref<number | null>(1)
const contributionQuantity = ref<number | null>(1)
const active = ref(true)
const submitting = ref(false)

function resetForm(): void {
  const rule = props.rule
  if (!rule) {
    selectorType.value = 'wave_all'
    platform.value = ''
    level.value = ''
    participantIdsText.value = ''
    productId.value = props.products[0]?.id ?? null
    ruleKind.value = ''
    priority.value = 1
    contributionQuantity.value = 1
    active.value = true
    return
  }
  selectorType.value = (rule.selectorPayload?.type ?? 'wave_all') as AllocationSelectorTypeValue
  platform.value = rule.selectorPayload?.platform ?? ''
  level.value = rule.selectorPayload?.level ?? ''
  participantIdsText.value = (rule.selectorPayload?.participant_ids ?? []).join(', ')
  productId.value = rule.productId
  ruleKind.value = rule.ruleKind
  priority.value = rule.priority
  contributionQuantity.value = rule.contributionQuantity
  active.value = rule.active
}

// This dialog stays mounted (no `v-if` at the call site) — reset on every open.
watch(
  () => props.show,
  (visible) => {
    if (visible) resetForm()
  },
)

const selectorTypeOptions = computed<SelectOption[]>(() =>
  SELECTOR_TYPES.map((value) => ({ label: glossaryLabel('allocationSelectorType', value), value })),
)

const productOptions = computed<SelectOption[]>(() =>
  props.products.map((product) => ({ label: `${product.name} (${product.factorySku})`, value: product.id })),
)

const showPlatform = computed(() => selectorType.value === 'platform_all')
const showLevel = computed(() => selectorType.value === 'identity_level')
const showParticipantIds = computed(() => selectorType.value === 'explicit_override')

function parseParticipantIds(text: string): number[] {
  return text
    .split(',')
    .map((part) => part.trim())
    .filter((part) => part.length > 0)
    .map((part) => Number(part))
    .filter((n) => Number.isInteger(n) && n > 0)
}

function buildSelectorPayload(): SelectorPayload {
  switch (selectorType.value) {
    case 'platform_all':
      return { type: 'platform_all', platform: platform.value.trim() }
    case 'identity_level':
      return { type: 'identity_level', level: level.value.trim() }
    case 'explicit_override':
      return { type: 'explicit_override', participant_ids: parseParticipantIds(participantIdsText.value) }
    case 'wave_all':
    default:
      return { type: 'wave_all' }
  }
}

const canSubmit = computed(
  () =>
    !submitting.value &&
    productId.value != null &&
    priority.value != null &&
    contributionQuantity.value != null &&
    (!showPlatform.value || platform.value.trim().length > 0) &&
    (!showLevel.value || level.value.trim().length > 0) &&
    (!showParticipantIds.value || parseParticipantIds(participantIdsText.value).length > 0),
)

function close(): void {
  emit('update:show', false)
}

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value || productId.value == null || priority.value == null || contributionQuantity.value == null) return
  submitting.value = true
  try {
    const selectorPayload = buildSelectorPayload()
    if (isEdit.value && props.rule) {
      await updateAllocationPolicyRule({
        id: props.rule.id,
        productId: productId.value,
        selectorPayload,
        ruleKind: ruleKind.value.trim(),
        priority: priority.value,
        contributionQuantity: contributionQuantity.value,
        active: active.value,
      })
    } else {
      await createAllocationPolicyRule({
        waveId: props.waveId,
        productId: productId.value,
        selectorPayload,
        productTargetRef: '',
        ruleKind: ruleKind.value.trim(),
        priority: priority.value,
        contributionQuantity: contributionQuantity.value,
        active: active.value,
      })
    }
    emit('saved')
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
    :title="t(isEdit ? 'allocation.rules.editRule' : 'allocation.rules.addRule')"
    :style="{ width: 'min(520px, 92vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <NForm label-placement="top">
      <NFormItem :label="t('allocation.rules.selectorType')">
        <NSelect v-model:value="selectorType" :options="selectorTypeOptions" :disabled="submitting" />
      </NFormItem>

      <NFormItem v-if="showPlatform" :label="t('allocation.rules.platform')">
        <NInput v-model:value="platform" :disabled="submitting" />
      </NFormItem>

      <NFormItem v-if="showLevel" :label="t('allocation.rules.level')">
        <NInput v-model:value="level" :disabled="submitting" />
      </NFormItem>

      <NFormItem v-if="showParticipantIds" :label="t('allocation.rules.participantIds')">
        <NInput v-model:value="participantIdsText" :disabled="submitting" placeholder="1, 2, 3" />
      </NFormItem>

      <NFormItem :label="t('allocation.rules.product')">
        <NSelect
          v-model:value="productId"
          :options="productOptions"
          filterable
          :disabled="submitting"
          :placeholder="products.length === 0 ? t('allocation.rules.noProducts') : undefined"
        />
      </NFormItem>

      <NFormItem :label="t('allocation.rules.ruleName')">
        <NInput v-model:value="ruleKind" :disabled="submitting" />
      </NFormItem>

      <NFormItem :label="t('allocation.rules.priority')">
        <NInputNumber v-model:value="priority" :min="1" :precision="0" :disabled="submitting" style="width: 100%" />
      </NFormItem>

      <NFormItem :label="t('allocation.rules.quantity')">
        <NInputNumber v-model:value="contributionQuantity" :min="0" :precision="0" :disabled="submitting" style="width: 100%" />
      </NFormItem>

      <NFormItem :label="t('allocation.rules.active')">
        <NSwitch v-model:value="active" :disabled="submitting" />
      </NFormItem>
    </NForm>

    <template #footer>
      <div class="rule-editor__footer">
        <NButton :disabled="submitting" @click="close">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('common.save') }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.rule-editor__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
