<script setup lang="ts">
import { useMessage } from 'naive-ui'
import WaveStepWizard from '@/shared/ui/WaveStepWizard.vue'
import { useUndoRedo } from '@/shared/composables/useUndoRedo'

const message = useMessage()

useUndoRedo({
  onUndo: () => {
    message.info('Undo is not yet available for wave workspace')
  },
  onRedo: () => {
    message.info('Redo is not yet available for wave workspace')
  },
})
</script>

<template>
  <div class="wave-workspace p-4">
    <WaveStepWizard />
    <router-view />
  </div>
</template>
