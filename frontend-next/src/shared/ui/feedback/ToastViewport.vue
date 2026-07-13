<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ToastRecord } from './types'
import Toast from './Toast.vue'

defineProps<{
  toasts: ToastRecord[]
}>()

const emit = defineEmits<{
  dismiss: [id: string]
  'toggle-detail': [id: string]
  'pointer-enter': [id: string]
  'pointer-leave': [id: string]
}>()

const { t } = useI18n()
</script>

<template>
  <Teleport to="body">
    <div class="toast-viewport" :aria-label="t('feedback.toast.politeRegionLabel')">
      <TransitionGroup name="toast-move" tag="div" class="toast-viewport__stack">
        <Toast
          v-for="toast in toasts"
          :key="toast.id"
          :toast="toast"
          @dismiss="emit('dismiss', $event)"
          @toggle-detail="emit('toggle-detail', $event)"
          @pointer-enter="emit('pointer-enter', $event)"
          @pointer-leave="emit('pointer-leave', $event)"
        />
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-viewport {
  position: fixed;
  top: var(--space-5);
  right: var(--space-5);
  z-index: var(--z-toast);
  display: flex;
  pointer-events: none;
}

.toast-viewport__stack {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.toast-move-enter-active {
  transition:
    opacity var(--duration-slow) var(--ease-out),
    transform var(--duration-slow) var(--ease-out);
}
.toast-move-leave-active {
  transition:
    opacity var(--duration-fast) var(--ease-in),
    transform var(--duration-fast) var(--ease-in);
  position: absolute;
  width: 100%;
}
.toast-move-enter-from {
  opacity: 0;
  transform: translateX(24px) scale(0.98);
}
.toast-move-leave-to {
  opacity: 0;
  transform: translateX(24px) scale(0.98);
}
.toast-move-move {
  transition: transform var(--duration-slow) var(--ease-out);
}
</style>
