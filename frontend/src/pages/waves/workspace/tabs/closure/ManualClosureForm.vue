<script setup lang="ts">
/**
 * ManualClosureForm — the per-fulfillment-line manual closure decision form
 * (`WaveClosureTab.vue`, shown when `PlanChannelClosureResult.decision` is
 * `'manual_closure'` or `'unsupported'`). Ported from
 * `frontend/src/pages/wave-workspace/WaveChannelSyncStep.vue`'s
 * `manualForms`/`decisionKindOptions` pattern (lines 66-75, 240-251), fixing
 * the two things the old tree got wrong: (1) no hardcoded English
 * placeholder VALUES seeded into `reasonCode` (only the placeholder
 * ATTRIBUTE, via i18n) — forms start blank; (2) `mark_sync_completed_manually`
 * is filtered out of the options entirely (not just disabled) when the
 * profile's `allowsManualClosure` is false, mirroring the backend's hard
 * check (`closure_action_usecase.go:79-81`) so a rejected submit can only
 * happen via devtools tampering, never a normal click.
 *
 * `reasonCode` is confirmed free text server-side (`internal/domain/enums.go:156`)
 * — rendered as a plain `NInput`, never a glossary/`StatusBadge` field.
 * `decisionKind` IS a fixed 3-value enum — its `<NSelect>` options render
 * through `useGlossary().label('closureDecisionKind', ...)`, and once a
 * decision is picked the corresponding `StatusBadge` previews it inline.
 *
 * `operatorId` is collected ONCE per submission (not per line) — the DTO's
 * per-entry `operatorId` field is filled with the same value for every
 * entry. This is a deliberate simplification over the old tree's per-line
 * operator field: one person is realistically submitting one batch of
 * closure decisions at a time, and repeating the same operator input once
 * per line added friction without adding information.
 */
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NForm, NFormItem, NSelect, NInput, NAutoComplete, NButton } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { SectionCard } from '@/shared/ui/cards'
import { StatusBadge } from '@/shared/ui/status'
import { CalloutBar } from '@/shared/ui/guidance'
import { useFeedback } from '@/shared/ui/feedback'
import { useGlossary, type ClosureDecisionKindValue } from '@/shared/i18n/glossary'
import { recordChannelClosureDecision } from '@/shared/api/bridge'
import { useOperatorRosterStore } from '@/shared/model/operator-roster'
import type { dto } from '@/../wailsjs/go/models'

const props = defineProps<{
  waveId: number
  profile: dto.IntegrationProfileSummaryDTO | null
  items: dto.ChannelSyncItemDTO[]
}>()

const emit = defineEmits<{ submitted: [] }>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const { label: glossaryLabel } = useGlossary()
const operatorRoster = useOperatorRosterStore()
const operatorIdOptions = computed(() => operatorRoster.list())

const ALL_DECISION_KINDS: ClosureDecisionKindValue[] = [
  'mark_sync_unsupported',
  'mark_sync_skipped',
  'mark_sync_completed_manually',
]

interface ManualFormEntry {
  decisionKind: ClosureDecisionKindValue | ''
  reasonCode: string
  note: string
  evidenceRef: string
}

const forms = reactive<Record<number, ManualFormEntry>>({})
const operatorId = ref('')
const submitting = ref(false)

function blankEntry(): ManualFormEntry {
  return { decisionKind: '', reasonCode: '', note: '', evidenceRef: '' }
}

// Re-seed (blank, never a hardcoded placeholder value) whenever a fresh plan
// result hands us a new candidate-item set.
watch(
  () => props.items,
  (items) => {
    for (const key of Object.keys(forms)) delete forms[Number(key)]
    for (const item of items) forms[item.fulfillmentLineId] = blankEntry()
  },
  { immediate: true },
)

const allowsManualClosure = computed(() => props.profile?.allowsManualClosure === true)

const decisionKindOptions = computed<SelectOption[]>(() =>
  ALL_DECISION_KINDS.filter((kind) => kind !== 'mark_sync_completed_manually' || allowsManualClosure.value).map(
    (value) => ({ label: glossaryLabel('closureDecisionKind', value), value }),
  ),
)

const allLinesDecided = computed(() =>
  props.items.every((item) => (forms[item.fulfillmentLineId]?.decisionKind ?? '') !== ''),
)

const canSubmit = computed(
  () => !submitting.value && props.profile != null && props.items.length > 0 && allLinesDecided.value && operatorId.value.trim().length > 0,
)

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value || !props.profile) return
  submitting.value = true
  try {
    const entries = props.items.map((item) => {
      const form = forms[item.fulfillmentLineId] ?? blankEntry()
      return {
        fulfillmentLineId: item.fulfillmentLineId,
        decisionKind: form.decisionKind as string,
        reasonCode: form.reasonCode.trim(),
        note: form.note.trim(),
        evidenceRef: form.evidenceRef.trim(),
        operatorId: operatorId.value.trim(),
      }
    })
    await recordChannelClosureDecision({
      waveId: props.waveId,
      integrationProfileId: props.profile.id,
      entries,
    })
    operatorRoster.add(operatorId.value.trim())
    feedback.success(t('waveWorkspace.closure.manualForm.success'))
    emit('submitted')
  } catch (err) {
    feedback.error(t('waveWorkspace.closure.manualForm.error'), err instanceof Error ? err.message : String(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <SectionCard :title="t('waveWorkspace.closure.manualForm.title')">
    <div class="manual-closure-form">
      <CalloutBar
        v-if="!allowsManualClosure"
        tone="info"
        :message="t('waveWorkspace.closure.manualForm.manualClosureDisabledHint')"
      />

      <NForm label-placement="top">
        <NFormItem :label="t('waveWorkspace.closure.manualForm.operatorId')">
          <NAutoComplete
            v-model:value="operatorId"
            :options="operatorIdOptions"
            :disabled="submitting"
            style="max-width: 320px"
          />
        </NFormItem>
      </NForm>

      <div v-for="item in items" :key="item.fulfillmentLineId" class="manual-closure-form__row">
        <div class="manual-closure-form__row-header">
          <span>{{ t('waveWorkspace.closure.manualForm.line', { id: item.fulfillmentLineId }) }}</span>
          <StatusBadge
            v-if="forms[item.fulfillmentLineId]?.decisionKind"
            dimension="closureDecisionKind"
            :value="forms[item.fulfillmentLineId]!.decisionKind"
            size="sm"
          />
        </div>
        <NForm label-placement="top">
          <NFormItem :label="t('waveWorkspace.closure.manualForm.decisionKind')">
            <NSelect
              v-model:value="forms[item.fulfillmentLineId]!.decisionKind"
              :options="decisionKindOptions"
              :placeholder="t('waveWorkspace.closure.manualForm.decisionKindPlaceholder')"
              :disabled="submitting"
            />
          </NFormItem>
          <NFormItem :label="t('waveWorkspace.closure.manualForm.reasonCode')">
            <NInput
              v-model:value="forms[item.fulfillmentLineId]!.reasonCode"
              :placeholder="t('waveWorkspace.closure.manualForm.reasonCodePlaceholder')"
              :disabled="submitting"
            />
          </NFormItem>
          <NFormItem :label="t('waveWorkspace.closure.manualForm.note')">
            <NInput v-model:value="forms[item.fulfillmentLineId]!.note" :disabled="submitting" />
          </NFormItem>
          <NFormItem :label="t('waveWorkspace.closure.manualForm.evidenceRef')">
            <NInput v-model:value="forms[item.fulfillmentLineId]!.evidenceRef" :disabled="submitting" />
          </NFormItem>
        </NForm>
      </div>

      <div class="manual-closure-form__footer">
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t('waveWorkspace.closure.manualForm.submit') }}
        </NButton>
      </div>
    </div>
  </SectionCard>
</template>

<style scoped>
.manual-closure-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.manual-closure-form__row {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--card-border-color);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.manual-closure-form__row-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.manual-closure-form__footer {
  display: flex;
  justify-content: flex-end;
}
</style>
