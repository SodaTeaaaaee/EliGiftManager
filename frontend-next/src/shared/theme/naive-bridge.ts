/**
 * Naive UI bridge — the ONLY file allowed to know that Naive UI's theming
 * API exists. Maps OUR resolved design tokens onto naive-ui's `theme` /
 * `themeOverrides` shape so Naive-rendered internals (form controls, modal
 * chrome, NDataTable internals, etc.) visually match the token system
 * instead of drifting to Naive's own default palette.
 *
 * Token values are read live via `getComputedStyle(document.documentElement)`
 * rather than duplicated as literals here — tokens.css (Layer 1/2) stays the
 * single source of truth, and a skin override "just works" through this
 * bridge without touching this file.
 *
 * `useNaiveTheme()` recomputes whenever the theme store's resolved theme,
 * density, or skin changes (all three can move CSS custom property values).
 */
import type { GlobalThemeOverrides } from "naive-ui";
import { darkTheme } from "naive-ui";
import { computed, ref, watch, type ComputedRef } from "vue";
import { storeToRefs } from "pinia";
import { useThemeStore } from "./theme";

/** Static fallback used before the DOM/tokens exist (SSR-less first paint, or
 * a test environment without a real document). Mirrors the light-theme
 * values baked into tokens.css so there is no visible flash of mismatch. */
const FALLBACK_TOKENS = {
  "--color-accent": "#b64925",
  "--color-accent-hover": "#9c3a1c",
  "--status-info-fg": "#284158",
  "--status-success-fg": "#455926",
  "--status-warning-fg": "#6e3d12",
  "--status-error-fg": "#66201a",
  "--color-text-primary": "#2f231b",
  "--color-text-secondary": "#574433",
  "--color-text-muted": "#8f755b",
  "--color-border": "#f1eadf",
  "--color-surface": "#fefdfb",
  "--color-surface-raised": "#ffffff",
  "--control-radius": "7px",
  "--radius-lg": "10px",
  "--font-body":
    '"Schibsted Grotesk", "Segoe UI", "Microsoft YaHei UI", "PingFang SC", "Noto Sans SC", sans-serif',
} as const;

type TokenName = keyof typeof FALLBACK_TOKENS;

function readToken(name: TokenName): string {
  if (typeof document === "undefined" || typeof window === "undefined") {
    return FALLBACK_TOKENS[name];
  }
  const value = getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim();
  return value || FALLBACK_TOKENS[name];
}

function buildThemeOverrides(): GlobalThemeOverrides {
  const accent = readToken("--color-accent");
  const accentHover = readToken("--color-accent-hover");
  const infoFg = readToken("--status-info-fg");
  const successFg = readToken("--status-success-fg");
  const warningFg = readToken("--status-warning-fg");
  const errorFg = readToken("--status-error-fg");
  const textPrimary = readToken("--color-text-primary");
  const textSecondary = readToken("--color-text-secondary");
  const textMuted = readToken("--color-text-muted");
  const border = readToken("--color-border");
  const surface = readToken("--color-surface");
  const surfaceRaised = readToken("--color-surface-raised");
  const controlRadius = readToken("--control-radius");
  const cardRadius = readToken("--radius-lg");
  const fontBody = readToken("--font-body");

  const common: GlobalThemeOverrides["common"] = {
    primaryColor: accent,
    primaryColorHover: accentHover,
    primaryColorPressed: accentHover,
    primaryColorSuppl: accent,
    infoColor: infoFg,
    successColor: successFg,
    warningColor: warningFg,
    errorColor: errorFg,
    textColorBase: textPrimary,
    textColor1: textPrimary,
    textColor2: textSecondary,
    textColor3: textMuted,
    borderColor: border,
    popoverColor: surfaceRaised,
    cardColor: surface,
    modalColor: surfaceRaised,
    fontFamily: fontBody,
    fontWeightStrong: "600",
    borderRadius: controlRadius,
    borderRadiusSmall: controlRadius,
  };

  return {
    common,
    Card: { borderRadius: cardRadius },
    Button: { borderRadiusMedium: controlRadius, borderRadiusSmall: controlRadius },
    Input: { borderRadius: controlRadius },
    Select: { peers: { InternalSelection: { borderRadius: controlRadius } } },
    Modal: { borderRadius: cardRadius },
    Drawer: { borderRadius: "0" },
  };
}

/**
 * `useNaiveTheme()` — call once per app (e.g. in App.vue) and feed the result
 * straight to `<NConfigProvider :theme :theme-overrides>`.
 */
export function useNaiveTheme(): {
  theme: ComputedRef<typeof darkTheme | null>;
  themeOverrides: ComputedRef<GlobalThemeOverrides>;
} {
  const store = useThemeStore();
  const { resolvedTheme, density, skinId } = storeToRefs(store);

  // Bumped whenever a change could have altered the resolved CSS custom
  // property values (theme flip, density flip, skin swap). getComputedStyle
  // reads happen lazily inside the computed below, after the DOM attribute
  // mutation + skin <style> swap have had a chance to apply.
  const recomputeTick = ref(0);
  watch([resolvedTheme, density, skinId], () => {
    // Read on next microtask so the store's own watchers (which set the
    // data-theme/data-density attributes and swap the skin <style> tag)
    // run first.
    queueMicrotask(() => {
      recomputeTick.value += 1;
    });
  });

  const theme = computed(() => (resolvedTheme.value === "dark" ? darkTheme : null));

  const themeOverrides = computed<GlobalThemeOverrides>(() => {
    void recomputeTick.value; // establish reactive dependency, value itself unused
    return buildThemeOverrides();
  });

  return { theme, themeOverrides };
}
