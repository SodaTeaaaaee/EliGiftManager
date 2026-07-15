<script setup lang="ts">
import { computed } from 'vue'
import { NSteps, NStep, NButton, NButtonGroup } from 'naive-ui'
import type { WizardStep } from './types'

/**
 * WizardFrame — the house wrapper around NSteps (plan P4's multi-step
 * intake wizard, reusable later for e.g. a P5 shipment-CSV mapping
 * wizard). Mirrors DataGrid's posture toward NDataTable: a narrow,
 * opinionated API on top of naive-ui's loose one — pages must NOT import
 * NSteps directly.
 *
 * Presentational + controlled: the parent page owns `current` (and all
 * step-body state) and reacts to `next`/`back`/`finish`/`cancel` to advance
 * it. WizardFrame never mutates `current` itself. Nav button visibility is
 * derived from `current`'s position in `steps`: Back hides on the first
 * step, Next swaps to Finish on the last step. `canNext`/`canBack` only
 * control the *disabled* state (e.g. gating Next until the current step's
 * required fields validate) — they do not affect visibility.
 *
 * Copy-agnostic like DataGrid: all button labels are supplied by the
 * consumer as already-resolved strings (`t(...)` happens in the page, not
 * here). `cancelLabel` is optional — omit it to hide the Cancel button
 * entirely (e.g. a wizard with no cancel affordance).
 */
const props = withDefaults(
  defineProps<{
    steps: WizardStep[]
    current: string
    /** Disables the Next/Finish button (e.g. current step hasn't validated yet). Default `true`. */
    canNext?: boolean
    /** Disables the Back button. Default `true`. */
    canBack?: boolean
    nextLabel: string
    backLabel: string
    finishLabel: string
    /** Omit to hide the Cancel button entirely. */
    cancelLabel?: string
  }>(),
  {
    canNext: true,
    canBack: true,
    cancelLabel: undefined,
  },
)

const emit = defineEmits<{
  next: []
  back: []
  finish: []
  cancel: []
}>()

const currentIndex = computed(() => {
  const index = props.steps.findIndex((step) => step.key === props.current)
  return index === -1 ? 0 : index
})

const isFirstStep = computed(() => currentIndex.value === 0)
const isLastStep = computed(() => currentIndex.value === props.steps.length - 1)

/** NSteps' own `current` prop is 1-based. */
const naiveStepsCurrent = computed(() => currentIndex.value + 1)
</script>

<template>
  <div class="wizard-frame">
    <NSteps class="wizard-frame__steps" :current="naiveStepsCurrent">
      <NStep v-for="step in steps" :key="step.key" :title="step.title" />
    </NSteps>

    <div class="wizard-frame__body">
      <slot />
    </div>

    <div class="wizard-frame__nav">
      <slot name="footer">
        <NButton v-if="cancelLabel" quaternary @click="emit('cancel')">
          {{ cancelLabel }}
        </NButton>
        <div class="wizard-frame__nav-spacer" />
        <NButtonGroup>
          <NButton v-if="!isFirstStep" :disabled="!canBack" @click="emit('back')">
            {{ backLabel }}
          </NButton>
          <NButton v-if="!isLastStep" type="primary" :disabled="!canNext" @click="emit('next')">
            {{ nextLabel }}
          </NButton>
          <NButton v-else type="primary" :disabled="!canNext" @click="emit('finish')">
            {{ finishLabel }}
          </NButton>
        </NButtonGroup>
      </slot>
    </div>
  </div>
</template>

<style scoped>
.wizard-frame {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.wizard-frame__steps {
  padding: 0 var(--space-2);
}

.wizard-frame__body {
  min-height: 0;
}

.wizard-frame__nav {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding-top: var(--space-4);
  border-top: 1px solid var(--card-border-color);
}

.wizard-frame__nav-spacer {
  flex: 1;
}
</style>
