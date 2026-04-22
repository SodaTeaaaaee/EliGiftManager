<script setup lang="ts">
defineProps<{
  title: string
  value: string
  detail: string
}>()
</script>

<template>
  <article class="surface-card">
    <h2 class="surface-card__title">
      {{ title }}
    </h2>
    <p class="surface-card__value">
      {{ value }}
    </p>
    <p class="surface-card__detail">
      {{ detail }}
    </p>
  </article>
</template>
