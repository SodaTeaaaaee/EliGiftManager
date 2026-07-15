<template>
  <div class="page-header">
    <div class="page-header-content">
      <div v-if="kicker" class="app-kicker mb-2">{{ kicker }}</div>
      <h1 class="app-title">{{ title }}</h1>
      <p v-if="description" class="app-copy mt-2">{{ description }}</p>
    </div>
    <div class="page-header-actions" v-if="$slots.actions">
      <slot name="actions"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  description?: string;
  kicker?: string;
}>();
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
}

.page-header-content {
  flex: 1;
}

.page-header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}
</style>
