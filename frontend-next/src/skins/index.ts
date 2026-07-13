/**
 * Skin registry — the typed catalogue of installable skin packages.
 *
 * A skin is a static asset bundle that overrides Layer 1/2 tokens (see
 * `shared/theme/tokens.css`) and optionally supplies decorative assets for
 * component-reserved slots (EmptyState illustration, task-center hero strip,
 * side-nav corner mascot). A skin NEVER ships code — only CSS custom
 * property overrides + images — so mounting one is just "load this CSS,
 * point these slots at these asset URLs", no app rebuild required in
 * principle (today the registry is a static TS module bundled at build
 * time; nothing stops a future version from fetching `manifest.json` files
 * from a user-writable skins directory instead).
 *
 * Adding a new skin:
 *   1. `src/skins/<id>/manifest.json` — metadata (see `SkinManifest`).
 *   2. `src/skins/<id>/tokens.css` — CSS custom property overrides, scoped
 *      under `:root[data-theme="light"]` / `:root[data-theme="dark"]` (or
 *      bare `:root` for theme-independent overrides). See
 *      `src/skins/default/tokens.css` for the override contract by example.
 *   3. Optionally add `src/skins/<id>/assets/*` and populate `assets` below.
 *   4. Register the entry in `SKIN_REGISTRY`.
 */

/** Decorative asset slots a skin may fill. Components fall back to a
 * neutral built-in placeholder (geometric shape / no-op) when a slot is
 * absent — a skin is never required to supply all of them. */
export interface SkinAssetSlots {
  /** EmptyState illustration, keyed by empty-state scene id (e.g.
   * "inbox", "search-no-results", "wave-empty"). */
  emptyState?: Record<string, string>;
  /** Task-center / action-center page header decorative strip. */
  hero?: string;
  /** Side-nav footer corner mascot. */
  mascot?: string;
}

export interface SkinManifest {
  id: string;
  name: string;
  author: string;
  /** Which resolved themes this skin has been authored/verified for. A skin
   * that only supports light must not be selectable while dark is active
   * (settings UI concern — the registry just publishes the fact). */
  supports: {
    light: boolean;
    dark: boolean;
  };
}

export interface SkinDefinition extends SkinManifest {
  /** URL to the skin's tokens.css, resolved at build time via `import.meta.url`
   * style static imports (`?url`) so Vite fingerprints/bundles it correctly. */
  tokensUrl: string;
  /** Decorative asset slots; omitted entirely for skins with no illustrations
   * (the default skin deliberately ships none — see its manifest). */
  assets?: SkinAssetSlots;
}

// `?url` gives us the final built asset URL as a string instead of inlining
// the CSS — skin-loader.ts fetches/injects it as a <link>/<style> tag rather
// than importing it as a side-effecting stylesheet (so it can be swapped at
// runtime without a page reload).
import defaultTokensUrl from "./default/tokens.css?url";
import defaultManifest from "./default/manifest.json";
import duskTokensUrl from "./dusk/tokens.css?url";
import duskManifest from "./dusk/manifest.json";

export const SKIN_REGISTRY: readonly SkinDefinition[] = [
  {
    ...(defaultManifest as SkinManifest),
    tokensUrl: defaultTokensUrl,
    // No `assets` — the default skin is the neutral baseline described in
    // the plan (plain geometric placeholders / whitespace only). A future
    // illustrated skin populates this.
  },
  {
    ...(duskManifest as SkinManifest),
    tokensUrl: duskTokensUrl,
    // No `assets` — Dusk is a color/token-only sample reskin; illustrator
    // assets are explicitly out of scope for this demonstration skin, so
    // every decorative slot falls back to its built-in neutral placeholder,
    // same as the default skin above.
  },
];

export const DEFAULT_SKIN_ID = "default";

export function getSkinById(id: string): SkinDefinition | undefined {
  return SKIN_REGISTRY.find((skin) => skin.id === id);
}

export function listSkins(): readonly SkinDefinition[] {
  return SKIN_REGISTRY;
}
