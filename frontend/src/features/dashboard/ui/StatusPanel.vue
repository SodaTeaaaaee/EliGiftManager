<script setup lang="ts">
import type { BootstrapPayload } from '@/shared/types/app'

defineProps<{
  payload: BootstrapPayload
  status: string
}>()
</script>

<template>
  <aside class="hero__status">
    <div>
      <p class="hero__status-label">
        Bridge status
      </p>
      <p class="hero__status-value">
        {{ status }}
      </p>
    </div>

    <div class="hero__status-meta">
      <div>
        <span class="hero__meta-label">Version</span>
        <span class="hero__meta-value">{{ payload.version }}</span>
      </div>
      <div>
        <span class="hero__meta-label">Module</span>
        <span class="hero__meta-value">{{ payload.module }}</span>
      </div>
    </div>
  </aside>
</template>
