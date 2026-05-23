<script setup lang="ts">
import { ref, reactive, onMounted, computed } from "vue";
import { NCard, NSelect, NSpace, NSwitch, NCheckbox, useMessage } from "naive-ui";
import { storeToRefs } from "pinia";
import { useI18n } from "@/shared/i18n";
import { useThemeStore, themePreferenceOptions } from "@/shared/model/theme";
import { localeOptions, useLocaleStore } from "@/shared/model/locale";
import { getSettings, saveSettings } from "@/shared/lib/wails/app";

const { t } = useI18n();
const themeStore = useThemeStore();
const localeStore = useLocaleStore();
const { preference } = storeToRefs(themeStore);
const { locale } = storeToRefs(localeStore);
const message = useMessage();

const mergeSettings = reactive({
  autoMergeCrossPlatform: false,
  autoMergeByEmail: false,
  autoMergeByPhone: false,
});

async function loadMergeSettings() {
  try {
    const res = await getSettings();
    mergeSettings.autoMergeCrossPlatform = res.autoMergeCrossPlatform;
    mergeSettings.autoMergeByEmail = res.autoMergeByEmail;
    mergeSettings.autoMergeByPhone = res.autoMergeByPhone;
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

async function handleSaveSettings() {
  try {
    await saveSettings({
      autoMergeCrossPlatform: mergeSettings.autoMergeCrossPlatform,
      autoMergeByEmail: mergeSettings.autoMergeByEmail,
      autoMergeByPhone: mergeSettings.autoMergeByPhone,
    });
    message.success(t("settings.saveSettingsSuccess"));
  } catch (e: any) {
    message.error(e?.message ?? String(e));
  }
}

onMounted(() => {
  loadMergeSettings();
});

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

      <NCard>
        <NSpace vertical :size="16">
          <div class="app-heading-sm">{{ t("settings.autoMergeTitle") }}</div>
          <NSpace vertical :size="12">
            <NSpace align="center" justify="space-between" style="width: 100%">
              <span class="text-sm">{{ t("settings.autoMergeCrossPlatform") }}</span>
              <NSwitch v-model:value="mergeSettings.autoMergeCrossPlatform" @update:value="handleSaveSettings" />
            </NSpace>
            
            <div v-if="mergeSettings.autoMergeCrossPlatform" style="padding-left: 24px; border-left: 2px solid var(--accent-surface); margin-top: 4px;">
              <NSpace vertical :size="12">
                <NCheckbox v-model:checked="mergeSettings.autoMergeByEmail" @update:checked="handleSaveSettings">
                  {{ t("settings.autoMergeByEmail") }}
                </NCheckbox>
                <NCheckbox v-model:checked="mergeSettings.autoMergeByPhone" @update:checked="handleSaveSettings">
                  {{ t("settings.autoMergeByPhone") }}
                </NCheckbox>
              </NSpace>
            </div>
          </NSpace>
        </NSpace>
      </NCard>
    </NSpace>
  </div>
</template>

