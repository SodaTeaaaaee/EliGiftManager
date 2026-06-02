<template>
  <n-layout has-sider class="app-viewport">
    <n-layout-sider
      bordered
      collapse-mode="width"
      :width="248"
      :collapsed-width="64"
      :collapsed="sidebar.appCollapsed"
      show-trigger="arrow-circle"
      content-style="background: transparent;"
      style="background: transparent;"
      @update:collapsed="sidebar.setAppCollapsed"
    >
      <AppSidebar :collapsed="sidebar.appCollapsed" />
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
import { useSidebarStore } from "@/shared/model/sidebar";

const sidebar = useSidebarStore();
</script>

<style scoped>
.app-shell-content {
  height: 100%;
  overflow: auto;
  padding: 24px;
}
</style>
