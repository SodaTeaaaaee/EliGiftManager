<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NSteps, NStep } from 'naive-ui'

const route = useRoute()
const router = useRouter()

const waveId = computed(() => route.params.waveId as string)

const steps = [
  { title: '总览', routeName: 'wave-overview-step', path: '' },
  { title: '需求映射', routeName: 'wave-demand-mapping', path: 'demand-mapping' },
  { title: '分配规则', routeName: 'wave-allocation', path: 'allocation' },
  { title: '履约调整', routeName: 'wave-adjustment-review', path: 'adjustment-review' },
]

const currentStep = computed(() => {
  const name = route.name as string
  const byName = steps.findIndex(s => s.routeName === name)
  if (byName >= 0) return byName + 1
  // Fallback: path-based matching — iterate in reverse so more-specific paths win
  const path = route.path
  for (let i = steps.length - 1; i >= 0; i--) {
    const s = steps[i]
    if (s.path === '' && /\/waves\/[^/]+$/.test(path)) return i + 1
    if (s.path && path.endsWith('/' + s.path)) return i + 1
  }
  return 1
})

function handleStepClick(step: number) {
  const target = steps[step - 1]
  if (!target || !waveId.value) return
  const base = `/waves/${waveId.value}`
  const path = target.path ? `${base}/${target.path}` : base
  router.push(path)
}
</script>

<template>
  <div class="wave-step-wizard mb-4">
    <n-steps :current="currentStep" size="small" @update:current="handleStepClick">
      <n-step v-for="step in steps" :key="step.routeName" :title="step.title" />
    </n-steps>
  </div>
</template>
