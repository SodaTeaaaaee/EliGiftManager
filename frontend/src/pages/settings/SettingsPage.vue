<script setup lang="ts">
import { computed } from "vue";
import { NCard, NSelect, NSpace } from "naive-ui";
import { storeToRefs } from "pinia";
import { useI18n } from "@/shared/i18n";
import { useThemeStore, themePreferenceOptions } from "@/shared/model/theme";
import { localeOptions, useLocaleStore } from "@/shared/model/locale";

const { t } = useI18n();
const themeStore = useThemeStore();
const localeStore = useLocaleStore();
const { preference } = storeToRefs(themeStore);
const { locale } = storeToRefs(localeStore);

const localizedThemeOptions = computed(() =>
  themePreferenceOptions.map((item) => ({
    value: item.value,
    label:
      item.value === "system"
        ? t("settings.system")
        : item.value === "light"
          ? t("settings.light")
          : t("settings.dark"),
  })),
);

const localizedLocaleOptions = computed(() =>
  localeOptions.map((item) => ({
    value: item.value,
    label: item.value === "zh-CN" ? t("settings.chinese") : t("settings.english"),
  })),
);
</script>

<template>
  <div class="settings-page">
    <div class="mb-6">
      <div class="app-kicker">{{ t("nav.settings") }}</div>
      <h1 class="app-title mt-2">{{ t("settings.title") }}</h1>
      <p class="app-copy mt-2">{{ t("settings.subtitle") }}</p>
    </div>

    <NSpace vertical :size="20">
      <NCard>
        <NSpace vertical :size="16">
          <div class="app-heading-sm">{{ t("settings.theme") }}</div>
          <NSelect
            :value="preference"
            :options="localizedThemeOptions"
            @update:value="(value) => themeStore.setPreference(value)"
          />
        </NSpace>
      </NCard>

      <NCard>
        <NSpace vertical :size="16">
          <div class="app-heading-sm">{{ t("settings.locale") }}</div>
          <NSelect
            :value="locale"
            :options="localizedLocaleOptions"
            @update:value="(value) => localeStore.setLocale(value)"
          />
        </NSpace>
      </NCard>
    </NSpace>
  </div>
</template>
