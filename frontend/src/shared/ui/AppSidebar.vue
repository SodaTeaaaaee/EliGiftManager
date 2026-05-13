<!-- TODO(V2): 完整实现 — 随着后续页面添加更多导航项 -->
<template>
  <div class="app-sidebar">
    <div class="sidebar-header">
      <span class="text-sm font-semibold tracking-tight">EliGiftManager V2</span>
    </div>
    <n-menu
      :value="activeKey"
      :options="menuOptions"
      @update:value="onMenuSelect"
    />
    <div class="sidebar-footer">
      <n-menu
        :value="null"
        :options="footerOptions"
        @update:value="onFooterSelect"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h } from "vue";
import { NMenu, NIcon, type MenuOption } from "naive-ui";
import { useRouter, useRoute } from "vue-router";

const router = useRouter();
const route = useRoute();

const menuOptions: MenuOption[] = [
  {
    label: "需求导入",
    key: "demand-intake",
    icon: () => h(NIcon, null, { default: () => "📥" }),
  },
  {
    label: "波次总览",
    key: "wave-overview",
    icon: () => h(NIcon, null, { default: () => "📊" }),
  },
];

const footerOptions: MenuOption[] = [
  {
    label: "设置",
    key: "settings",
    icon: () => h(NIcon, null, { default: () => "⚙️" }),
  },
];

const activeKey = computed(() => {
  const name = route.name;
  if (typeof name === "string") return name;
  return null;
});

function onMenuSelect(key: string) {
  router.push({ name: key });
}

function onFooterSelect(key: string) {
  // TODO(V2): 完整实现 — 路由到设置页面
  void key;
}
</script>

<style scoped>
.app-sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.sidebar-header {
  padding: 16px 20px 12px;
  color: var(--text);
}

.sidebar-footer {
  margin-top: auto;
}
</style>
