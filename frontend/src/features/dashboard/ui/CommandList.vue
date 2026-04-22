<script setup lang="ts">
import type { DashboardCommand } from '@/features/dashboard/model/commands'

defineProps<{
  commands: DashboardCommand[]
}>()
</script>

<template>
  <section
    class="command-list"
    aria-label="Common commands"
  >
    <div
      v-for="item in commands"
      :key="item.command"
      class="command-list__item"
    >
      <span class="command-list__label">{{ item.label }}</span>
      <code>{{ item.command }}</code>
    </div>
  </section>
</template>
