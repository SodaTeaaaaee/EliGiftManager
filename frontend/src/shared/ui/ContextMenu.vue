<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useContextMenu, type ContextMenuItem } from '@/shared/composables/useContextMenu'

const { state, hide } = useContextMenu()

const adjustedStyle = computed(() => {
  const x = state.x
  const y = state.y
  const vw = typeof window !== 'undefined' ? window.innerWidth : 1920
  const vh = typeof window !== 'undefined' ? window.innerHeight : 1080
  const menuW = 180
  const menuH = Math.max(state.items.length * 36 + 8, 44)

  let left = x
  let top = y
  if (x + menuW > vw) left = x - menuW
  if (y + menuH > vh) top = y - menuH
  if (left < 0) left = 4
  if (top < 0) top = 4

  return { left: left + 'px', top: top + 'px' }
})

function onItemClick(item: ContextMenuItem) {
  hide()
  item.action()
}

function onOverlayClick() {
  hide()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') hide()
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <!-- backdrop to capture outside clicks -->
  <div
    v-if="state.visible"
    class="fixed inset-0 z-[9998]"
    @click="onOverlayClick"
    @contextmenu.prevent="onOverlayClick"
  />
  <!-- menu -->
  <div
    v-if="state.visible"
    class="fixed z-[9999] py-1 min-w-[160px] rounded-lg shadow-lg border border-gray-200 dark:border-gray-700"
    :style="{
      left: adjustedStyle.left,
      top: adjustedStyle.top,
      background: 'var(--surface-strong, #fff)',
    }"
  >
    <div v-for="item in state.items" :key="item.key">
      <div v-if="item.divider" class="my-1 border-t border-gray-100 dark:border-gray-700" />
      <div
        class="px-3 py-2 text-sm cursor-pointer transition-colors hover:bg-black/5 dark:hover:bg-white/5"
        :style="{ color: 'var(--text, #111)' }"
        @click.stop="onItemClick(item)"
      >
        {{ item.label }}
      </div>
    </div>
  </div>
</template>
