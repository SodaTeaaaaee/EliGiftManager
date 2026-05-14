<script setup lang="ts">
import WaveStepWizard from '@/shared/ui/WaveStepWizard.vue'
</script>

<template>
  <div class="wave-workspace p-4">
    <WaveStepWizard />
    <router-view />
  </div>
</template>
