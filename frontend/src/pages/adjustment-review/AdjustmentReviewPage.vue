<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const waveId = ref(Number(route.params.waveId) || 0)
</script>

<template>
  <div class="p-4">
    <n-card title="履约调整审核">
      <n-empty description="履约调整审核页面（开发中）">
        <template #extra>
          <n-text depth="3">波次 ID: {{ waveId }}</n-text>
        </template>
      </n-empty>
    </n-card>
  </div>
</template>
