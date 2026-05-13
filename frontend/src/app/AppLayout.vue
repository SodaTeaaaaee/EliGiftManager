<template>
  <n-layout has-sider class="app-viewport">
    <n-layout-sider bordered :width="220">
      <AppSidebar />
    </n-layout-sider>
    <n-layout-content>
      <div class="p-4">
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
