<script setup lang="ts">
import { NEmpty } from 'naive-ui'
</script>
<template>
  <NEmpty description="选择一个发货任务开始工作，或创建一个新任务">
    <template #extra>
      <p class="app-copy mt-2 text-center">点击上方「发货任务列表」选择已有任务，或「新建发货任务」创建新的发货批次。</p>
    </template>
  </NEmpty>
</template>
