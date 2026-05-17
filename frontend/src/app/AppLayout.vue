<template>
  <n-layout has-sider class="app-viewport">
    <n-layout-sider bordered :width="248" content-style="background: var(--surface-strong);">
      <AppSidebar />
    </n-layout-sider>
    <n-layout-content content-style="background: transparent;">
      <div class="app-shell-content">
        <RouterView />
      </div>
    </n-layout-content>
  </n-layout>
</template>

<script setup lang="ts">
import { NLayout, NLayoutContent, NLayoutSider } from "naive-ui";
import { RouterView } from "vue-router";
import AppSidebar from "@/shared/ui/AppSidebar.vue";
</script>

<style scoped>
.app-shell-content {
  height: 100%;
  overflow: auto;
  padding: 24px;
}
</style>
