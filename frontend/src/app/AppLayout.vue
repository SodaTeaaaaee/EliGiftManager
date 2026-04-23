<script setup lang="ts">
import { NLayout, NLayoutContent, NLayoutSider, NScrollbar } from 'naive-ui'
import { RouterView } from 'vue-router'
import Sidebar from '@/shared/ui/Sidebar.vue'
</script>

<template>
  <NLayout has-sider class="app-viewport">
    <NLayoutSider
      bordered
      :width="220"
      content-style="padding: 16px 12px;"
      style="background: var(--surface-strong)"
    >
      <Sidebar />
    </NLayoutSider>

    <NLayoutContent style="background: transparent">
      <NScrollbar style="height: 100%">
        <div class="px-5 py-5 lg:px-6 lg:py-6">
          <RouterView />
        </div>
      </NScrollbar>
    </NLayoutContent>
  </NLayout>
</template>
