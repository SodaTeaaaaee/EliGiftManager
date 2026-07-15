import { computed } from "vue";
import { storeToRefs } from "pinia";
import { messages, type MessageSchema, type SupportedLocale } from "./messages";
import { useLocaleStore } from "@/shared/model/locale";

function getByPath(obj: Record<string, any>, path: string): string {
  const value = path.split(".").reduce<any>((acc, key) => acc?.[key], obj);
  return typeof value === "string" ? value : path;
}

export function resolveBrowserLocale(): SupportedLocale {
  if (typeof navigator === "undefined") return "zh-CN";
  const langs = navigator.languages?.length ? navigator.languages : [navigator.language];
  for (const lang of langs) {
    if (!lang) continue;
    if (lang.toLowerCase().startsWith("zh")) return "zh-CN";
    if (lang.toLowerCase().startsWith("en")) return "en-US";
  }
  return "zh-CN";
}

export function useI18n() {
  const localeStore = useLocaleStore();
  const { locale } = storeToRefs(localeStore);

  const dict = computed<MessageSchema>(() => messages[locale.value]);

  function t(path: string): string {
    return getByPath(dict.value as unknown as Record<string, any>, path);
  }

  return {
    locale,
    t,
  };
}
