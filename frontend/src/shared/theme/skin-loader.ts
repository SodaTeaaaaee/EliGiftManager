/**
 * Skin loader — applies a skin's tokens.css by fetching it and injecting its
 * text content into a single `<style id="skin-tokens">` element placed at
 * the end of `<head>`, so it always wins the cascade over the base
 * `tokens.css`/`typography.css` imports (later source order, same
 * specificity: `:root[...]` rules).
 *
 * A `<link>` element would work too, but injecting the fetched text instead
 * lets `applySkin` be awaited (callers — e.g. the theme store — know the
 * override is actually in the DOM before doing anything that depends on
 * resolved token values, like the naive-ui bridge's getComputedStyle reads)
 * and avoids a flash-of-unstyled-override on skin switch.
 */
import { DEFAULT_SKIN_ID, getSkinById, type SkinAssetSlots } from "@/skins";

const STYLE_ELEMENT_ID = "skin-tokens";

let currentSkinId = DEFAULT_SKIN_ID;
let currentAssets: SkinAssetSlots | undefined;

function getOrCreateStyleElement(): HTMLStyleElement | null {
  if (typeof document === "undefined") return null;
  let el = document.getElementById(STYLE_ELEMENT_ID) as HTMLStyleElement | null;
  if (!el) {
    el = document.createElement("style");
    el.id = STYLE_ELEMENT_ID;
    document.head.appendChild(el);
  }
  return el;
}

/**
 * Loads and injects the given skin's tokens.css, replacing whatever skin
 * override was previously applied. Falls back to an empty override (i.e.
 * the base tokens.css alone) if the skin id is unknown or the fetch fails —
 * a bad skin id must never leave the app unstyled.
 */
export async function applySkin(id: string): Promise<void> {
  const skin = getSkinById(id) ?? getSkinById(DEFAULT_SKIN_ID);
  const styleEl = getOrCreateStyleElement();

  if (!skin || !styleEl) {
    currentSkinId = DEFAULT_SKIN_ID;
    currentAssets = undefined;
    return;
  }

  try {
    const response = await fetch(skin.tokensUrl);
    if (!response.ok) throw new Error(`skin fetch failed: ${response.status}`);
    const css = await response.text();
    styleEl.textContent = css;
    currentSkinId = skin.id;
    currentAssets = skin.assets;
  } catch {
    // Network/build hiccup — leave the previous override in place rather
    // than silently reverting, but make sure asset slots don't reference a
    // half-applied skin's images.
    currentAssets = undefined;
  }
}

/** Current skin's decorative asset slots, for components like EmptyState.
 * Returns undefined fields for any slot the active skin doesn't supply —
 * callers must render their built-in neutral placeholder in that case. */
export function getCurrentSkinAssets(): SkinAssetSlots | undefined {
  return currentAssets;
}

export function getCurrentSkinId(): string {
  return currentSkinId;
}
