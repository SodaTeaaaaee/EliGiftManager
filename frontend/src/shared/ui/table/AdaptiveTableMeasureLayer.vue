<script setup lang="ts">
import { NDataTable } from 'naive-ui'
import { ref, watch } from 'vue'

const props = defineProps<{
  data: any[]
  columns: any[]
  rowKey?: (row: any) => any
  size?: 'small' | 'medium' | 'large'
  tableLayout?: 'auto' | 'fixed'
  width?: number
}>()

const root = ref<HTMLElement | null>(null)
const tableRef = ref<any>(null)
const width = ref(props.width ?? 0)

watch(() => props.width, (w) => {
  if (w != null) width.value = w
})

function setWidth(w: number) {
  width.value = w
}

function measure(): { headerHeight: number; rowHeights: number[] } | null {
  const el = tableRef.value?.$el as HTMLElement | undefined
  if (!el) {
    // Fallback: try to find thead in the measure layer
    const thead = root.value?.querySelector('.n-data-table-thead')
    const headerHeight = thead instanceof HTMLElement ? thead.offsetHeight : 0
    const trs = root.value?.querySelectorAll('tbody tr')
    const rowHeights = trs ? Array.from(trs).map((tr) => (tr as HTMLElement).offsetHeight) : []
    return { headerHeight, rowHeights }
  }
  const thead = el.querySelector('.n-data-table-thead')
  const headerHeight = thead instanceof HTMLElement ? thead.offsetHeight : 0
  const trs = el.querySelectorAll('tbody tr')
  const rowHeights = Array.from(trs).map((tr) => (tr as HTMLElement).offsetHeight)
  return { headerHeight, rowHeights }
}

defineExpose({ setWidth, measure })
</script>

<template>
  <div
    ref="root"
    class="measure-layer"
    :style="{ width: width + 'px' }"
  >
    <NDataTable
      ref="tableRef"
      :data="data"
      :columns="columns"
      :row-key="rowKey"
      :size="size ?? 'small'"
      :table-layout="tableLayout ?? 'auto'"
      :bordered="false"
      :pagination="false"
    />
  </div>
</template>

<style scoped>
.measure-layer {
  position: absolute;
  left: -99999px;
  top: 0;
  visibility: hidden;
  pointer-events: none;
  overflow: visible;
}
</style>
