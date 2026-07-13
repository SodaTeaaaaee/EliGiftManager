/**
 * Theme store — owns the three orthogonal display axes (theme preference,
 * density, active skin) and is the single writer of the `data-theme` /
 * `data-density` attributes on <html> that tokens.css keys off of.
 *
 * Deliberately NOT i18n-aware: this store only holds machine values
 * ("system" | "light" | "dark", "comfortable" | "compact", a skin id string).
 * Human-readable labels for a settings UI are the caller's concern.
 */
import { defineStore } from "pinia";
import type { App } from "vue";
import { computed, ref, watch } from "vue";
import { DEFAULT_SKIN_ID } from "@/skins";
import { applySkin } from "./skin-loader";

export type ThemePreference = "system" | "light" | "dark";
export type ResolvedTheme = "light" | "dark";
export type Density = "comfortable" | "compact";

const STORAGE_PREFIX = "eligiftmanager:";
const STORAGE_KEY_PREFERENCE = `${STORAGE_PREFIX}theme-preference`;
const STORAGE_KEY_DENSITY = `${STORAGE_PREFIX}density`;
const STORAGE_KEY_SKIN = `${STORAGE_PREFIX}skin-id`;

function readStorage(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    // localStorage can throw in locked-down / privacy-mode contexts —
    // fall back to in-memory defaults rather than crash the app shell.
    return null;
  }
}

function writeStorage(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    // best-effort persistence only
  }
}

function isThemePreference(value: string | null): value is ThemePreference {
  return value === "system" || value === "light" || value === "dark";
}

function isDensity(value: string | null): value is Density {
  return value === "comfortable" || value === "compact";
}

function resolveInitialPreference(): ThemePreference {
  const stored = readStorage(STORAGE_KEY_PREFERENCE);
  return isThemePreference(stored) ? stored : "system";
}

function resolveInitialDensity(): Density {
  const stored = readStorage(STORAGE_KEY_DENSITY);
  return isDensity(stored) ? stored : "comfortable";
}

function prefersDarkMediaQuery(): MediaQueryList | null {
  if (typeof window === "undefined" || !window.matchMedia) return null;
  return window.matchMedia("(prefers-color-scheme: dark)");
}

export const useThemeStore = defineStore("theme", () => {
  const preference = ref<ThemePreference>(resolveInitialPreference());
  const density = ref<Density>(resolveInitialDensity());
  const skinId = ref<string>(readStorage(STORAGE_KEY_SKIN) ?? DEFAULT_SKIN_ID);

  // System-scheme snapshot, refreshed by the media-query listener below.
  const systemPrefersDark = ref(prefersDarkMediaQuery()?.matches ?? false);

  const resolvedTheme = computed<ResolvedTheme>(() => {
    if (preference.value === "system") {
      return systemPrefersDark.value ? "dark" : "light";
    }
    return preference.value;
  });

  function setPreference(next: ThemePreference): void {
    preference.value = next;
  }

  function setDensity(next: Density): void {
    density.value = next;
  }

  function setSkinId(next: string): void {
    skinId.value = next;
  }

  function applyToDocument(): void {
    if (typeof document === "undefined") return;
    const root = document.documentElement;
    root.setAttribute("data-theme", resolvedTheme.value);
    root.setAttribute("data-density", density.value);
  }

  watch(preference, (value) => {
    writeStorage(STORAGE_KEY_PREFERENCE, value);
    applyToDocument();
  });

  watch(density, (value) => {
    writeStorage(STORAGE_KEY_DENSITY, value);
    applyToDocument();
  });

  watch(
    skinId,
    (value) => {
      writeStorage(STORAGE_KEY_SKIN, value);
      void applySkin(value);
    },
    { immediate: true },
  );

  watch(resolvedTheme, applyToDocument);

  let mediaQuery: MediaQueryList | null = null;
  function handleSystemChange(event: MediaQueryListEvent): void {
    systemPrefersDark.value = event.matches;
  }

  /** Wires the prefers-color-scheme listener; call once, e.g. from initTheme(). */
  function startSystemListener(): void {
    if (mediaQuery) return;
    mediaQuery = prefersDarkMediaQuery();
    if (!mediaQuery) return;
    systemPrefersDark.value = mediaQuery.matches;
    mediaQuery.addEventListener("change", handleSystemChange);
  }

  function stopSystemListener(): void {
    mediaQuery?.removeEventListener("change", handleSystemChange);
    mediaQuery = null;
  }

  return {
    preference,
    density,
    skinId,
    resolvedTheme,
    setPreference,
    setDensity,
    setSkinId,
    applyToDocument,
    startSystemListener,
    stopSystemListener,
  };
});

/**
 * Install helper — call once during app bootstrap (after `app.use(pinia)`),
 * e.g. from main.ts:
 *
 *   const pinia = createPinia();
 *   app.use(pinia);
 *   initTheme(app);
 *
 * `app` is optional: if omitted, the currently-active Pinia instance is used
 * (works for calls made after `app.use(pinia)` within the same setup script).
 */
export function initTheme(_app?: App): void {
  const store = useThemeStore();
  store.startSystemListener();
  store.applyToDocument();
}
