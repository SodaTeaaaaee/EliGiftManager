<script setup lang="ts">
/**
 * TopProgressBar — thin fixed bar at the top of the viewport, driven by
 * `useRouteProgressStore()` (plan §4.5). Self-built per the §4.3 component
 * boundary (page/shell code must not reach for `NLoadingBar`) — styled
 * entirely off theme tokens so it tracks both the light/dark theme and any
 * active skin automatically. Mount exactly once, from `App.vue`.
 */
import { useI18n } from 'vue-i18n'
import { useRouteProgressStore } from '@/shared/model/route-progress'

const { t } = useI18n()
const routeProgress = useRouteProgressStore()
</script>

<template>
  <Teleport to="body">
    <Transition name="top-progress-bar-fade">
      <div
        v-if="routeProgress.visible"
        class="top-progress-bar"
        role="progressbar"
        :aria-label="t('feedback.topProgressBar.loadingLabel')"
        :aria-valuenow="routeProgress.progress"
        aria-valuemin="0"
        aria-valuemax="100"
        :style="{ width: routeProgress.progress + '%' }"
      />
    </Transition>
  </Teleport>
</template>

<style scoped>
.top-progress-bar {
  position: fixed;
  top: 0;
  left: 0;
  height: 2px;
  z-index: var(--z-fixed);
  background: var(--color-accent);
  box-shadow: 0 0 8px 0 var(--color-accent);
  transition:
    width var(--duration-slower) var(--ease-out),
    opacity var(--duration-base) var(--ease-out);
}

.top-progress-bar-fade-enter-active {
  transition: none;
}
.top-progress-bar-fade-leave-active {
  transition: opacity var(--duration-base) var(--ease-out);
}
.top-progress-bar-fade-enter-from {
  opacity: 1;
}
.top-progress-bar-fade-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .top-progress-bar {
    transition: opacity var(--duration-fast) var(--ease-out);
  }
  .top-progress-bar-fade-leave-active {
    transition: opacity var(--duration-fast) var(--ease-out);
  }
}
</style>
