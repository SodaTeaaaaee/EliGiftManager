<script setup lang="ts">
/**
 * Design Lab — the living reference for the token/typography/motion system
 * plus a full tour of every shared/ui kit built for frontend. The
 * header's four controls (theme/density/locale/skin) are wired to the real
 * app-wide stores (`useThemeStore`, `useAppLocale`), not local demo state —
 * flipping them here changes the live app, exactly like a real settings
 * page would, which is the point: this page doubles as a smoke test for the
 * whole theme system.
 *
 * Sections reveal with a single staggered fade/rise on page load (motion
 * tokens `--duration-slower` / `--ease-spring`, capped stagger delay so a
 * long page doesn't take forever to finish revealing) and respect
 * `prefers-reduced-motion` via the media query in the scoped styles below.
 */
import { computed, markRaw } from "vue";
import { useI18n } from "vue-i18n";
import { NSelect } from "naive-ui";
import { useThemeStore, type Density, type ThemePreference } from "@/shared/theme/theme";
import { useAppLocale, type SupportedLocale } from "@/shared/i18n";
import { listSkins } from "@/skins";
import ShellSection from "./sections/ShellSection.vue";
import StatusSection from "./sections/StatusSection.vue";
import DataGridSection from "./sections/DataGridSection.vue";
import FilterBarSection from "./sections/FilterBarSection.vue";
import CardsMiscSection from "./sections/CardsMiscSection.vue";
import FeedbackSection from "./sections/FeedbackSection.vue";

const { t } = useI18n({ useScope: "global" });
const themeStore = useThemeStore();
const { locale, localeOptions, setLocale } = useAppLocale();

const themeOptions = computed(() => [
  { value: "system", label: t("designLab.controls.themeOptions.system") },
  { value: "light", label: t("designLab.controls.themeOptions.light") },
  { value: "dark", label: t("designLab.controls.themeOptions.dark") },
]);

const densityOptions = computed(() => [
  { value: "comfortable", label: t("designLab.controls.densityOptions.comfortable") },
  { value: "compact", label: t("designLab.controls.densityOptions.compact") },
]);

const skinOptions = computed(() => listSkins().map((skin) => ({ value: skin.id, label: skin.name })));

function handleThemeChange(value: ThemePreference): void {
  themeStore.setPreference(value);
}

function handleDensityChange(value: Density): void {
  themeStore.setDensity(value);
}

function handleSkinChange(value: string): void {
  themeStore.setSkinId(value);
}

function handleLocaleChange(value: SupportedLocale): void {
  setLocale(value);
}

/** Section registry — drives both the TOC pills and the staggered reveal list. */
const sections = [
  { key: "shell", titleKey: "designLab.toc.shell", component: markRaw(ShellSection) },
  { key: "status", titleKey: "designLab.toc.status", component: markRaw(StatusSection) },
  { key: "dataGrid", titleKey: "designLab.toc.dataGrid", component: markRaw(DataGridSection) },
  { key: "filterBar", titleKey: "designLab.toc.filterBar", component: markRaw(FilterBarSection) },
  { key: "cards", titleKey: "designLab.toc.cards", component: markRaw(CardsMiscSection) },
  { key: "feedback", titleKey: "designLab.toc.feedback", component: markRaw(FeedbackSection) },
] as const;

// Capped so a page with many sections still finishes its reveal quickly —
// each section's CSS animation-delay is `index * step`, clamped at 5 steps.
const STAGGER_STEP_MS = 60;
function revealDelay(index: number): string {
  return `${Math.min(index, 5) * STAGGER_STEP_MS}ms`;
}
</script>

<template>
  <div class="design-lab-page">
    <header class="design-lab-page__header">
      <p class="design-lab-page__kicker">{{ t("designLab.title") }}</p>
      <h1 class="design-lab-page__title">{{ t("designLab.title") }}</h1>
      <p class="design-lab-page__subtitle">{{ t("designLab.subtitle") }}</p>

      <div class="design-lab-page__controls">
        <label class="design-lab-page__control">
          <span class="design-lab-page__control-label">{{ t("designLab.controls.theme") }}</span>
          <NSelect
            :value="themeStore.preference"
            :options="themeOptions"
            :consistent-menu-width="false"
            @update:value="handleThemeChange"
          />
        </label>
        <label class="design-lab-page__control">
          <span class="design-lab-page__control-label">{{ t("designLab.controls.density") }}</span>
          <NSelect
            :value="themeStore.density"
            :options="densityOptions"
            :consistent-menu-width="false"
            @update:value="handleDensityChange"
          />
        </label>
        <label class="design-lab-page__control">
          <span class="design-lab-page__control-label">{{ t("designLab.controls.locale") }}</span>
          <NSelect
            :value="locale"
            :options="localeOptions"
            :consistent-menu-width="false"
            @update:value="handleLocaleChange"
          />
        </label>
        <label class="design-lab-page__control">
          <span class="design-lab-page__control-label">{{ t("designLab.controls.skin") }}</span>
          <NSelect
            :value="themeStore.skinId"
            :options="skinOptions"
            :consistent-menu-width="false"
            @update:value="handleSkinChange"
          />
        </label>
      </div>

      <nav class="design-lab-page__toc" :aria-label="t('designLab.toc.title')">
        <a v-for="section in sections" :key="section.key" class="design-lab-page__toc-link" :href="`#design-lab-${section.key}`">
          {{ t(section.titleKey) }}
        </a>
      </nav>
    </header>

    <div class="design-lab-page__sections">
      <section
        v-for="(section, index) in sections"
        :id="`design-lab-${section.key}`"
        :key="section.key"
        class="design-lab-page__reveal"
        :style="{ animationDelay: revealDelay(index) }"
      >
        <component :is="section.component" />
      </section>
    </div>
  </div>
</template>

<style scoped>
.design-lab-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
  padding-bottom: var(--space-16);
}

.design-lab-page__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--color-border);
}

.design-lab-page__kicker {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-accent);
}

.design-lab-page__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-3xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.design-lab-page__subtitle {
  margin: 0;
  max-width: 48em;
  font-family: var(--font-body);
  font-size: var(--font-size-base);
  line-height: var(--line-height-relaxed);
  color: var(--color-text-secondary);
}

.design-lab-page__controls {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-4);
  margin-top: var(--space-3);
}

.design-lab-page__control {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 160px;
}

.design-lab-page__control-label {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
}

.design-lab-page__toc {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin-top: var(--space-4);
}

.design-lab-page__toc-link {
  display: inline-flex;
  align-items: center;
  height: var(--control-height-sm);
  padding: 0 var(--space-3);
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-text-secondary);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  transition:
    border-color var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}

.design-lab-page__toc-link:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.design-lab-page__toc-link:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.design-lab-page__sections {
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
}

.design-lab-page__reveal {
  scroll-margin-top: var(--space-6);
  animation: design-lab-reveal var(--duration-slower) var(--ease-spring) both;
}

@keyframes design-lab-reveal {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .design-lab-page__reveal {
    animation: none;
  }
}
</style>
