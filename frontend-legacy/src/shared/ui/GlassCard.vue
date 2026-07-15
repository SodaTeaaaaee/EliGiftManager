<template>
  <div class="glass-card" :class="{ 'hover-effect': hoverable, 'no-padding': !padding }">
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
defineProps({
  hoverable: {
    type: Boolean,
    default: false,
  },
  padding: {
    type: Boolean,
    default: true,
  }
});
</script>

<style scoped>
.glass-card {
  transition: all 0.3s ease;
}
.glass-card.no-padding {
  padding: 0;
}
.glass-card:not(.no-padding) {
  padding: 24px;
}
.glass-card.hover-effect:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1);
}

:root[data-theme='dark'] .glass-card.hover-effect:hover {
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5), 0 8px 10px -6px rgba(0, 0, 0, 0.5);
}
</style>
