import { defineStore } from "pinia";
import { ref } from "vue";

export type ThemePreference = "system" | "light" | "dark";

const THEME_STORAGE_KEY = "eligiftmanager:theme-preference";

function normalizeThemePreference(value: string | null): ThemePreference {
  if (value === "light" || value === "dark") {
    return value;
  }

  return "system";
}

export const themePreferenceOptions = [
  { label: "跟随系统", value: "system" as const },
  { label: "浅色", value: "light" as const },
  { label: "深色", value: "dark" as const },
];

export const useThemeStore = defineStore("theme", () => {
  const preference = ref<ThemePreference>("system");
  const hasHydrated = ref(false);

  function hydrate() {
    if (hasHydrated.value) {
      return;
    }

    if (typeof window !== "undefined") {
      preference.value = normalizeThemePreference(
        window.localStorage.getItem(THEME_STORAGE_KEY),
      );
    }

    hasHydrated.value = true;
  }

  function setPreference(value: ThemePreference) {
    preference.value = value;

    if (typeof window !== "undefined") {
      window.localStorage.setItem(THEME_STORAGE_KEY, value);
    }
  }

  return {
    preference,
    hasHydrated,
    hydrate,
    setPreference,
  };
});
