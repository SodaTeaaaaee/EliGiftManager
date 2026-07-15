<script setup lang="ts">
/**
 * IntakeWizard — composes `WizardFrame` over the 5 named steps (create
 * mode) or the 3 shared steps (remap mode — `existingProfile` passed in,
 * see `useIntakeWizardState.ts`'s `isRemapMode`). Owns nothing but the
 * composable instance + which step component to render; the actual persist
 * sequence lives in `state.finish()`.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { WizardFrame, type WizardStep } from '@/shared/ui/wizard'
import { useFeedback } from '@/shared/ui/feedback'
import { useIntakeWizardState, type IntakeWizardStepKey } from './useIntakeWizardState'
import type { IntegrationProfile } from '@/entities/profile'
import StepPlatformPreset from './StepPlatformPreset.vue'
import StepBusinessSurface from './StepBusinessSurface.vue'
import StepSampleUploadAndMapping from './StepSampleUploadAndMapping.vue'
import StepCapabilityToggles from './StepCapabilityToggles.vue'
import StepConfirm from './StepConfirm.vue'

const props = defineProps<{
  /** Pass an existing profile to run the wizard in "remap" mode (only regenerates the
   *  column-mapping template + rebinds it as default — profile fields are untouched). */
  existingProfile?: IntegrationProfile | null
}>()

const emit = defineEmits<{
  done: [profile: IntegrationProfile]
  cancel: []
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const state = useIntakeWizardState({ existingProfile: props.existingProfile ?? null })

const STEP_COMPONENTS: Record<IntakeWizardStepKey, unknown> = {
  platformPreset: StepPlatformPreset,
  businessSurface: StepBusinessSurface,
  sampleUpload: StepSampleUploadAndMapping,
  capabilities: StepCapabilityToggles,
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
    feedback.success(t('feedback.success'))
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
