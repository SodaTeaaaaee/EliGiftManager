<script setup lang="ts">
import { computed } from 'vue'
import { getCurrentSkinAssets } from '@/shared/theme/skin-loader'

/**
 * EmptyState — icon-or-illustration, title/description, and a primary
 * action slot. Used for "no waves yet", "no results", inbox-zero, etc.
 *
 * ILLUSTRATION HOOK (for future skins): pass a `scene` id (e.g.
 * "wave-empty", "search-no-results", "inbox-zero") and this component looks
 * it up in the active skin's `assets.emptyState` map (see
 * `src/skins/index.ts` → `SkinAssetSlots.emptyState`, populated at runtime
 * via `getCurrentSkinAssets()` from `shared/theme/skin-loader.ts`). The
 * default skin ships no illustrations, so today every scene falls back to
 * the built-in neutral geometric mark below — a soft token-colored ring +
 * dot drawn in inline SVG, no external asset dependency. A future
 * illustrated skin (e.g. a 二次元 skin) supplies real artwork per scene id
 * and this component swaps to an `<img>` automatically, no code changes
 * required in call sites. If neither a scene image nor the `#icon` slot is
 * given, the geometric mark is what renders.
 */
const props = withDefaults(
  defineProps<{
    title: string
    description?: string
    /** Empty-state scene id, used to look up a skin-supplied illustration. */
    scene?: string
    /** Visual size — `sm` for nested/inline empty states (e.g. inside a drawer), `md` for full-section placement. */
    size?: 'sm' | 'md'
  }>(),
  {
    description: undefined,
    scene: undefined,
    size: 'md',
  },
)

const skinIllustrationUrl = computed(() => {
  if (!props.scene) return undefined
  return getCurrentSkinAssets()?.emptyState?.[props.scene]
})
</script>

<template>
  <div class="empty-state" :class="`empty-state--${size}`">
    <div class="empty-state__art">
      <slot name="icon">
        <img v-if="skinIllustrationUrl" :src="skinIllustrationUrl" alt="" class="empty-state__illustration" />
        <svg v-else class="empty-state__mark" viewBox="0 0 64 64" aria-hidden="true" focusable="false">
          <circle cx="32" cy="32" r="26" fill="none" stroke="var(--color-border-strong)" stroke-width="2" stroke-dasharray="4 6" />
          <circle cx="32" cy="32" r="7" fill="var(--color-accent-subtle)" stroke="var(--color-accent)" stroke-width="1.5" />
        </svg>
      </slot>
    </div>
    <h3 class="empty-state__title">{{ title }}</h3>
    <p v-if="description" class="empty-state__description">{{ description }}</p>
    <div v-if="$slots.default" class="empty-state__action">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: var(--space-2);
  padding: var(--space-8) var(--space-6);
}

.empty-state--sm {
  padding: var(--space-5) var(--space-4);
}

.empty-state__art {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: var(--space-2);
}

.empty-state--sm .empty-state__art {
  margin-bottom: var(--space-1);
}

.empty-state__mark {
  width: 64px;
  height: 64px;
}

.empty-state--sm .empty-state__mark {
  width: 40px;
  height: 40px;
}

.empty-state__illustration {
  max-width: 160px;
  max-height: 120px;
}

.empty-state__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.empty-state__description {
  margin: 0;
  max-width: 32em;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  line-height: var(--line-height-relaxed);
  color: var(--color-text-muted);
}

.empty-state__action {
  margin-top: var(--space-3);
}
</style>
