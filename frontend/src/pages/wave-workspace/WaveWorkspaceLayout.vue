<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import WaveStepWizard from '@/shared/ui/WaveStepWizard.vue'
import { useUndoRedo } from '@/shared/composables/useUndoRedo'
import { undoWaveAction, redoWaveAction } from '@/shared/lib/wails/app'

const route = useRoute()
const message = useMessage()

function getWaveId(): number {
  return Number(route.params.waveId) || 0
}

useUndoRedo({
  onUndo: async () => {
    const waveId = getWaveId()
    if (!waveId) return
    try {
      const summary = await undoWaveAction(waveId)
      message.success(`Undo: ${summary}`)
    } catch (e: any) {
      const msg = e?.message || String(e)
      if (msg.includes('nothing to undo') || msg.includes('no history')) {
        message.warning('Nothing to undo')
      } else {
        message.error(`Undo failed: ${msg}`)
      }
    }
  },
  onRedo: async () => {
    const waveId = getWaveId()
    if (!waveId) return
    try {
      const summary = await redoWaveAction(waveId)
      message.success(`Redo: ${summary}`)
    } catch (e: any) {
      const msg = e?.message || String(e)
      if (msg.includes('nothing to redo') || msg.includes('no history')) {
        message.warning('Nothing to redo')
      } else {
        message.error(`Redo failed: ${msg}`)
      }
    }
  },
})
</script>

<template>
  <div class="wave-workspace p-4">
    <WaveStepWizard />
    <router-view />
  </div>
</template>
