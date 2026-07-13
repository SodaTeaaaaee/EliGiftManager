<script setup lang="ts">
/**
 * Generic placeholder for a plan-2.1 top-level nav destination whose real
 * page has not been rebuilt yet (Waves / Inbox / Customers / Products /
 * Integrations / Settings). Reads `route.meta.navTitleKey` (an i18n message
 * key, see `app/router/index.ts`) so one component serves every pending
 * route — swapping in the real page later is just a router `component:`
 * change, this file is never imported directly by name.
 */
import { computed } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { PageHeader } from "@/shared/ui/shell";
import { EmptyState } from "@/shared/ui/empty-state";

const { t } = useI18n({ useScope: "global" });
const route = useRoute();

const titleKey = computed(() => route.meta.navTitleKey ?? "nav.home");
const title = computed(() => t(titleKey.value));
</script>

<template>
  <div class="placeholder-page">
    <PageHeader :title="title" />
    <EmptyState :title="title" :description="t('nav.placeholderDescription')" />
  </div>
</template>

<style scoped>
.placeholder-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
</style>
