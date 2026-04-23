<script setup lang="ts">
import { RouterView } from 'vue-router'

import Sidebar from '@/shared/ui/Sidebar.vue'
</script>

<template>
  <div class="app-viewport flex min-h-0 gap-5 overflow-hidden bg-neutral-950 p-5 text-neutral-50">
    <Sidebar />

    <main class="min-h-0 min-w-0 flex-1 overflow-hidden rounded-[30px] bg-neutral-800 shadow-2xl shadow-black/30">
      <div class="h-full overflow-y-auto p-6 lg:p-8">
        <RouterView />
      </div>
    </main>
  </div>
</template>
