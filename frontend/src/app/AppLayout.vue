<script setup lang="ts">
import { NLayout, NLayoutContent, NLayoutSider } from 'naive-ui'
import { RouterView } from 'vue-router'
import Sidebar from '@/shared/ui/Sidebar.vue'
</script>

<template>
  <NLayout has-sider class="app-viewport">
    <NLayoutSider bordered :width="220" content-style="padding: 16px 12px;" style="background: var(--surface-strong)">
      <Sidebar />
    </NLayoutSider>

    <NLayoutContent style="background: transparent">
      <div class="px-5 py-5 lg:px-6 lg:py-6 h-full" style="display: flex; flex-direction: column;">
        <RouterView class="flex-1 min-h-0" />
      </div>
    </NLayoutContent>
  </NLayout>
</template>
