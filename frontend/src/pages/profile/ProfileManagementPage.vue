<template>
  <div class="profile-hub-page">
    <div class="profile-hub-header">
      <div>
        <div class="app-kicker">{{ t("nav.profiles") }}</div>
        <h1 class="app-title mt-2">{{ t("profile.hubTitle") }}</h1>
        <p class="app-copy mt-3">{{ t("profile.hubSubtitle") }}</p>
      </div>
    </div>

    <n-tabs v-model:value="activeTab" type="line" animated>
      <!-- Tab 1: Integration Profiles -->
      <n-tab-pane name="profiles" :tab="t('profile.tabProfiles')">
        <ProfileListPanel />
      </n-tab-pane>

      <!-- Tab 2: Document Templates -->
      <n-tab-pane name="templates" :tab="t('profile.tabTemplates')">
        <TemplateManagementPanel />
      </n-tab-pane>

      <!-- Tab 3: Profile-Template Bindings -->
      <n-tab-pane name="bindings" :tab="t('profile.tabBindings')">
        <ProfileTemplateBindingPanel />
      </n-tab-pane>

      <!-- Tab 5: CSV Import -->
      <n-tab-pane name="csv-import" :tab="t('profile.tabCsvImport')">
        <TemplateCsvImportPanel />
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { NTabs, NTabPane } from "naive-ui";
import { useI18n } from "@/shared/i18n";

// Lazy-imported sub-panels — each is a standalone page component reused as a panel
import ProfileListPanel from "./ProfileListPanel.vue";
import TemplateManagementPanel from "@/pages/template/TemplateManagementPage.vue";
import ProfileTemplateBindingPanel from "@/pages/template/ProfileTemplateBindingPage.vue";
import TemplateCsvImportPanel from "@/pages/template/TemplateCsvImportPage.vue";

const { t } = useI18n();
const activeTab = ref("profiles");
</script>

<style scoped>
.profile-hub-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.profile-hub-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}
</style>
