<script setup lang="ts">
/**
 * Design-lab showcase for the feedback kit (shared/ui/feedback/**) — the
 * app's ONLY feedback path. Wraps the actual demo (`FeedbackSectionBody`)
 * in its own local `<FeedbackProvider>` so this section works standalone
 * regardless of whether the app root has wired one up yet (see `App.vue`'s
 * `[P0-INTEGRATION]` marker) — the toast viewport / receipt tray rendered
 * here are scoped to this demo only, not shared with the rest of the app.
 */
import FeedbackProvider from '@/shared/ui/feedback/FeedbackProvider.vue'
import FeedbackSectionBody from './FeedbackSectionBody.vue'
</script>

<template>
  <FeedbackProvider>
    <FeedbackSectionBody />
  </FeedbackProvider>
</template>
