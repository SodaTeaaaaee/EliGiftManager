<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import WaveStepWizard from '@/shared/ui/WaveStepWizard.vue'
import { useUndoRedo } from '@/shared/composables/useUndoRedo'

const route = useRoute()
const message = useMessage()

const waveId = computed(() => {
  const id = Number(route.params.waveId)
  return Number.isFinite(id) ? id : null
})

useUndoRedo({
  scopeType: 'wave',
  scopeKey: () => waveId.value,
  onSuccess: (summary, action) => {
    const label = action === 'undo' ? '撤销' : '重做'
    message.success(`${label}：${summary}`)
  },
  onError: (err) => {
    message.warning(err)
  },
  onNotReady: () => {
    message.info('撤销/重做：后端未连接')
  },
})
</script>

<template>
  <div class="wave-workspace p-4">
    <WaveStepWizard />
    <router-view />
  </div>
</template>
