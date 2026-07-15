<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, useTemplateRef } from 'vue'
import {
  buildPaginationIndicator,
  type PaginationIndicatorModel,
} from '@/shared/lib/table/buildPaginationIndicator'

const props = defineProps<{
  page: number
  pageCount: number
}>()

const root = useTemplateRef('root')
const model = ref<PaginationIndicatorModel>({ left: '', right: '', fontSize: 12 })

function updateModel() {
  const el = root.value
  if (!el) return
  const w = el.clientWidth
  const h = el.clientHeight
  model.value = buildPaginationIndicator(props.page, props.pageCount, w, h)
}

let observer: ResizeObserver | null = null

onMounted(() => {
  if (root.value) {
    observer = new ResizeObserver(() => updateModel())
    observer.observe(root.value)
    updateModel()
  }
})

onUnmounted(() => {
  observer?.disconnect()
})

watch([() => props.page, () => props.pageCount], () => {
  updateModel()
})
</script>

<template>
  <div
    ref="root"
    class="adaptive-pagination-indicator"
    aria-hidden="true"
    :style="{
      fontSize: model.fontSize + 'px',
      lineHeight: '1',
      fontFamily: 'monospace',
      whiteSpace: 'nowrap',
      overflow: 'hidden',
    }"
  >
    <span
      v-if="model.left"
      class="adaptive-pagination-indicator__left"
      :style="{ color: 'rgba(96, 165, 250, 0.1)' }"
      >{{ model.left }}</span
    >
    <span
      v-if="model.right"
      class="adaptive-pagination-indicator__right"
      :style="{ color: 'rgba(251, 191, 36, 0.1)' }"
      >{{ model.right }}</span
    >
  </div>
</template>

<style scoped>
.adaptive-pagination-indicator {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  justify-content: center;
  align-items: center;
  user-select: none;
  pointer-events: none;
}
</style>
