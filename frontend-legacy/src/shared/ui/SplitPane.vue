<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  initialSplit: {
    type: Number,
    default: 50 // Percentage (0-100)
  },
  minLeft: {
    type: Number,
    default: 20 // Percentage
  },
  minRight: {
    type: Number,
    default: 20 // Percentage
  }
})

const containerRef = ref<HTMLElement | null>(null)
const leftWidth = ref(props.initialSplit)
const isDragging = ref(false)

function onMouseDown(e: MouseEvent) {
  isDragging.value = true
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
  // Prevent text selection while dragging
  document.body.style.userSelect = 'none'
}

function onMouseMove(e: MouseEvent) {
  if (!isDragging.value || !containerRef.value) return

  const containerRect = containerRef.value.getBoundingClientRect()
  let newLeftWidth = ((e.clientX - containerRect.left) / containerRect.width) * 100

  // Apply constraints
  if (newLeftWidth < props.minLeft) newLeftWidth = props.minLeft
  if (newLeftWidth > 100 - props.minRight) newLeftWidth = 100 - props.minRight

  leftWidth.value = newLeftWidth
}

function onMouseUp() {
  isDragging.value = false
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
  document.body.style.userSelect = ''
}

onUnmounted(() => {
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
})
</script>

<template>
  <div class="split-pane-container" ref="containerRef" :class="{ 'is-dragging': isDragging }">
    <div class="split-pane-left" :style="{ width: `${leftWidth}%` }">
      <slot name="left"></slot>
    </div>
    
    <div class="split-pane-divider" @mousedown="onMouseDown">
      <div class="divider-handle"></div>
    </div>
    
    <div class="split-pane-right" :style="{ width: `${100 - leftWidth}%` }">
      <slot name="right"></slot>
    </div>
  </div>
</template>

<style scoped>
.split-pane-container {
  display: flex;
  width: 100%;
  height: 100%;
  position: relative;
  overflow: hidden;
}

.split-pane-left, .split-pane-right {
  height: 100%;
  overflow: auto;
}

.split-pane-divider {
  width: 8px;
  height: 100%;
  background-color: transparent;
  cursor: col-resize;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 10;
  margin: 0 -4px; /* overlap to increase grab area without affecting layout width */
}

.split-pane-divider::after {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 3px;
  width: 1px;
  background-color: var(--border-color, rgba(148, 163, 184, 0.2));
  transition: background-color 0.2s;
}

.split-pane-divider:hover::after,
.is-dragging .split-pane-divider::after {
  background-color: var(--accent);
  width: 2px;
  left: 3px;
}

.divider-handle {
  width: 4px;
  height: 24px;
  border-radius: 4px;
  background-color: var(--border-color, rgba(148, 163, 184, 0.4));
  transition: background-color 0.2s;
}

.split-pane-divider:hover .divider-handle,
.is-dragging .divider-handle {
  background-color: var(--accent);
}

.is-dragging .split-pane-left,
.is-dragging .split-pane-right {
  pointer-events: none; /* Prevent iframes or other elements from capturing mouse events during drag */
}
</style>
