<script setup lang="ts">
/**
 * IntakeWizard — configure one file kind on an already-added platform
 * (builtin shortcut or custom create: same path). Steps: documentType →
 * sample → confirm. Does not create platforms.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { WizardFrame, type WizardStep } from '@/shared/ui/wizard'
import { useFeedback } from '@/shared/ui/feedback'
import { useIntakeWizardState, type IntakeWizardStepKey } from './useIntakeWizardState'
import type { IntegrationProfile } from '@/entities/profile'
import StepDocumentType from './StepDocumentType.vue'
import StepSampleUploadAndMapping from './StepSampleUploadAndMapping.vue'
import StepConfirm from './StepConfirm.vue'

const props = defineProps<{
  existingProfile: IntegrationProfile
  initialDocumentType?: string
}>()

const emit = defineEmits<{
  done: [profile: IntegrationProfile]
  cancel: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const state = useIntakeWizardState({
  existingProfile: props.existingProfile,
  initialDocumentType: props.initialDocumentType,
})

const STEP_COMPONENTS: Record<IntakeWizardStepKey, unknown> = {
  documentType: StepDocumentType,
  sampleUpload: StepSampleUploadAndMapping,
  confirm: StepConfirm,
}

const wizardSteps = computed<WizardStep[]>(() =>
  state.steps.value.map((key) => ({ key, title: t(`intakeWizard.steps.${key}.title`) })),
)

const currentStepComponent = computed(() => STEP_COMPONENTS[state.current.value])

function handleNext(): void {
  state.goNext()
}

function handleBack(): void {
  state.goBack()
}

function handleCancel(): void {
  emit('cancel')
}

async function handleFinish(): Promise<void> {
  try {
    const profile = await state.finish()
    if (state.bindWarning.value) {
      feedback.info(state.bindWarning.value)
    } else {
      feedback.success(t('feedback.success'))
    }
    emit('done', profile)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  }
}
</script>

<template>
  <div class="intake-wizard">
    <WizardFrame
      :steps="wizardSteps"
      :current="state.current.value"
      :can-next="state.canProceedFromCurrentStep.value && !state.persisting.value"
      :can-back="!state.persisting.value"
      :next-label="t('intakeWizard.nav.next')"
      :back-label="t('intakeWizard.nav.back')"
      :finish-label="t('intakeWizard.nav.finish')"
      :cancel-label="t('intakeWizard.nav.cancel')"
      @next="handleNext"
      @back="handleBack"
      @cancel="handleCancel"
      @finish="handleFinish"
    >
      <component :is="currentStepComponent" :state="state" />
    </WizardFrame>
  </div>
</template>

<style scoped>
.intake-wizard {
  display: flex;
  flex-direction: column;
}
</style>
