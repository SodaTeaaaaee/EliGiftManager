import { defineStore } from "pinia";
import { ref } from "vue";
import type { SupportedLocale } from "@/shared/i18n/messages";
import { resolveBrowserLocale } from "@/shared/i18n";

const LOCALE_STORAGE_KEY = "eligiftmanager:locale";

function normalizeLocale(value: string | null): SupportedLocale {
  if (value === "zh-CN" || value === "en-US") {
    return value;
  }
  return resolveBrowserLocale();
}

export const localeOptions = [
  { label: "简体中文", value: "zh-CN" as const },
  { label: "English", value: "en-US" as const },
];

export const useLocaleStore = defineStore("locale", () => {
  const locale = ref<SupportedLocale>("zh-CN");
  const hasHydrated = ref(false);

  function hydrate() {
    if (hasHydrated.value) return;
    if (typeof window !== "undefined") {
      locale.value = normalizeLocale(window.localStorage.getItem(LOCALE_STORAGE_KEY));
    } else {
      locale.value = resolveBrowserLocale();
    }
    hasHydrated.value = true;
  }

  function setLocale(value: SupportedLocale) {
    locale.value = value;
    if (typeof window !== "undefined") {
      window.localStorage.setItem(LOCALE_STORAGE_KEY, value);
    }
  }

  return {
    locale,
    hasHydrated,
    hydrate,
    setLocale,
  };
});
