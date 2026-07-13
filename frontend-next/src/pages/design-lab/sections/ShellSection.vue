<script setup lang="ts">
/**
 * Design-lab showcase for the shell/navigation kit (shared/ui/shell/**).
 * Demos AppShell + SideNav (top-level nav, collapse + badges), WorkspaceNav
 * (wave-workspace skeleton with status dots + counts), PageHeader, NavBadge,
 * and ContentErrorBoundary's live catch/retry/copy flow — all with
 * realistic fulfillment-domain sample data (CJK + Latin names mixed).
 *
 * Nav items deliberately all point back at `/design-lab` with a distinct
 * query param per item (rather than real app routes, which don't exist yet)
 * so clicking around this demo never navigates away from the design lab —
 * the query change is still enough to demo `RouterLink`'s live active-state
 * detection inside `SideNav`/`WorkspaceNav`.
 */
import { h, defineComponent, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  GridOutline,
  TicketOutline,
  DownloadOutline,
  PeopleOutline,
  CubeOutline,
  GitNetworkOutline,
  SettingsOutline,
  CloudUploadOutline,
  ListOutline,
  CheckmarkCircleOutline,
  ShieldCheckmarkOutline,
  AirplaneOutline,
  SyncOutline,
} from '@vicons/ionicons5'
import AppShell from '@/shared/ui/shell/AppShell.vue'
import WorkspaceNav from '@/shared/ui/shell/WorkspaceNav.vue'
import PageHeader from '@/shared/ui/shell/PageHeader.vue'
import NavBadge from '@/shared/ui/shell/NavBadge.vue'
import ContentErrorBoundary from '@/shared/ui/shell/ContentErrorBoundary.vue'
import type { NavGroupSpec, NavItemSpec, WorkspaceNavGroupSpec } from '@/shared/ui/shell/types'
import type { StatusTone } from '@/shared/i18n/glossary'

const { t } = useI18n()

const navGroups: NavGroupSpec[] = [
  {
    key: 'action-center',
    labelKey: 'shellKit.demo.nav.sectionActionCenter',
    items: [{ key: 'home', labelKey: 'shellKit.demo.nav.home', icon: GridOutline, to: { path: '/design-lab' } }],
  },
  {
    key: 'fulfillment',
    labelKey: 'shellKit.demo.nav.sectionFulfillment',
    items: [
      {
        key: 'waves',
        labelKey: 'shellKit.demo.nav.waves',
        icon: TicketOutline,
        to: { path: '/design-lab', query: { shellNavDemo: 'waves' } },
        badge: { count: 3, tone: 'progress' },
      },
      {
        key: 'inbox',
        labelKey: 'shellKit.demo.nav.inbox',
        icon: DownloadOutline,
        to: { path: '/design-lab', query: { shellNavDemo: 'inbox' } },
        badge: { count: 12, tone: 'warning' },
      },
    ],
  },
  {
    key: 'master-data',
    labelKey: 'shellKit.demo.nav.sectionMasterData',
    items: [
      { key: 'customers', labelKey: 'shellKit.demo.nav.customers', icon: PeopleOutline, to: { path: '/design-lab', query: { shellNavDemo: 'customers' } } },
      { key: 'products', labelKey: 'shellKit.demo.nav.products', icon: CubeOutline, to: { path: '/design-lab', query: { shellNavDemo: 'products' } } },
      {
        key: 'integrations',
        labelKey: 'shellKit.demo.nav.integrations',
        icon: GitNetworkOutline,
        to: { path: '/design-lab', query: { shellNavDemo: 'integrations' } },
        badge: { count: 128, tone: 'error' },
        disabled: true,
      },
    ],
  },
]

const settingsItem: NavItemSpec = {
  key: 'settings',
  labelKey: 'shellKit.demo.nav.settings',
  icon: SettingsOutline,
  to: { path: '/design-lab', query: { shellNavDemo: 'settings' } },
}

const workspaceGroups: WorkspaceNavGroupSpec[] = [
  {
    key: 'overview',
    items: [{ key: 'overview', labelKey: 'shellKit.demo.workspace.overview', icon: GridOutline, to: { path: '/design-lab', query: { workspaceNavDemo: 'overview' } } }],
  },
  {
    key: 'prep',
    labelKey: 'shellKit.demo.workspace.sectionPrep',
    items: [
      {
        key: 'intake',
        labelKey: 'shellKit.demo.workspace.intake',
        icon: CloudUploadOutline,
        to: { path: '/design-lab', query: { workspaceNavDemo: 'intake' } },
        tone: 'success',
      },
      {
        key: 'allocation',
        labelKey: 'shellKit.demo.workspace.allocation',
        icon: ListOutline,
        to: { path: '/design-lab', query: { workspaceNavDemo: 'allocation' } },
        tone: 'progress',
        count: 46,
      },
    ],
  },
  {
    key: 'review',
    labelKey: 'shellKit.demo.workspace.sectionReview',
    items: [
      {
        key: 'lines',
        labelKey: 'shellKit.demo.workspace.lines',
        icon: CheckmarkCircleOutline,
        to: { path: '/design-lab', query: { workspaceNavDemo: 'lines' } },
        tone: 'warning',
        count: 12,
      },
      {
        key: 'readiness',
        labelKey: 'shellKit.demo.workspace.readiness',
        icon: ShieldCheckmarkOutline,
        to: { path: '/design-lab', query: { workspaceNavDemo: 'readiness' } },
        tone: 'neutral',
      },
    ],
  },
  {
    key: 'execution',
    labelKey: 'shellKit.demo.workspace.sectionExecution',
    items: [
      {
        key: 'factory',
        labelKey: 'shellKit.demo.workspace.factory',
        icon: CloudUploadOutline,
        to: { path: '/design-lab', query: { workspaceNavDemo: 'factory' } },
        disabled: false,
      },
      { key: 'shipments', labelKey: 'shellKit.demo.workspace.shipments', icon: AirplaneOutline, to: { path: '/design-lab', query: { workspaceNavDemo: 'shipments' } } },
      {
        key: 'closure',
        labelKey: 'shellKit.demo.workspace.closure',
        icon: SyncOutline,
        to: { path: '/design-lab', query: { workspaceNavDemo: 'closure' } },
        tone: 'error',
        count: 3,
      },
    ],
  },
]

const navBadgeSamples: { count: number; tone: StatusTone }[] = [
  { count: 0, tone: 'neutral' },
  { count: 3, tone: 'progress' },
  { count: 12, tone: 'warning' },
  { count: 46, tone: 'info' },
  { count: 128, tone: 'error' },
]

// ── ContentErrorBoundary live demo ──────────────────────────────────────
const shouldThrow = ref(false)

/** Reads `shouldThrow` reactively INSIDE its render fn, so flipping the ref
 * re-renders (and re-throws) this component without needing a remount — the
 * boundary's own retry flow is what performs the remount afterwards. */
const ThrowingDemoWidget = defineComponent({
  name: 'ShellSectionThrowingDemoWidget',
  setup() {
    return () => {
      if (shouldThrow.value) {
        throw new Error('Demo crash: wave W-2026-07-Aki drift projection threw during render (simulated for design-lab).')
      }
      return h('p', { class: 'shell-section__throw-ok' }, t('shellKit.demo.contentOk'))
    }
  },
})

function triggerError(): void {
  shouldThrow.value = true
}

function handleBoundaryRetry(): void {
  shouldThrow.value = false
}
</script>

<template>
  <section class="shell-section">
    <header class="shell-section__header">
      <h2 class="shell-section__title">{{ t('shellKit.demo.title') }}</h2>
      <p class="shell-section__subtitle">{{ t('shellKit.demo.subtitle') }}</p>
    </header>

    <!-- AppShell + SideNav -->
    <article class="shell-card">
      <h3 class="shell-card__title">{{ t('shellKit.demo.appShellGroupTitle') }}</h3>
      <p class="shell-card__hint">{{ t('shellKit.demo.appShellHint') }}</p>
      <div class="shell-section__preview-box">
        <AppShell :groups="navGroups" :settings-item="settingsItem">
          <template #brand>
            <span class="shell-section__brand-mark" aria-hidden="true">EG</span>
            <span class="shell-section__brand-text">{{ t('shellKit.sideNav.brandName') }}</span>
          </template>
          <PageHeader :kicker="t('shellKit.demo.nav.home')" :title="t('shellKit.demo.contentTitle')" :description="t('shellKit.demo.contentBody')" />
        </AppShell>
      </div>
    </article>

    <!-- WorkspaceNav -->
    <article class="shell-card">
      <h3 class="shell-card__title">{{ t('shellKit.demo.workspaceNavGroupTitle') }}</h3>
      <p class="shell-card__hint">{{ t('shellKit.demo.workspaceNavHint') }}</p>
      <div class="shell-section__preview-box shell-section__preview-box--short">
        <div class="shell-section__workspace-shell">
          <WorkspaceNav :groups="workspaceGroups">
            <template #header>
              <p class="shell-section__workspace-kicker">{{ t('shellKit.demo.workspace.waveKicker') }}</p>
              <p class="shell-section__workspace-name">{{ t('shellKit.demo.workspace.waveName') }}</p>
              <p class="shell-section__workspace-meta tabular-nums">{{ t('shellKit.demo.workspace.waveMeta') }}</p>
            </template>
          </WorkspaceNav>
          <div class="shell-section__workspace-content">
            <PageHeader :kicker="t('shellKit.demo.pageHeader.kicker')" :title="t('shellKit.demo.pageHeader.title')" :description="t('shellKit.demo.pageHeader.description')">
              <template #actions>
                <button type="button" class="shell-section__btn">{{ t('shellKit.demo.pageHeader.actionExport') }}</button>
                <button type="button" class="shell-section__btn shell-section__btn--primary">{{ t('shellKit.demo.pageHeader.actionAdjust') }}</button>
              </template>
            </PageHeader>
          </div>
        </div>
      </div>
    </article>

    <!-- PageHeader standalone -->
    <article class="shell-card">
      <h3 class="shell-card__title">{{ t('shellKit.demo.pageHeaderGroupTitle') }}</h3>
      <PageHeader :kicker="t('shellKit.demo.pageHeader.kicker')" :title="t('shellKit.demo.pageHeader.title')" :description="t('shellKit.demo.pageHeader.description')">
        <template #actions>
          <button type="button" class="shell-section__btn">{{ t('shellKit.demo.pageHeader.actionExport') }}</button>
          <button type="button" class="shell-section__btn shell-section__btn--primary">{{ t('shellKit.demo.pageHeader.actionAdjust') }}</button>
        </template>
      </PageHeader>
    </article>

    <!-- NavBadge -->
    <article class="shell-card">
      <h3 class="shell-card__title">{{ t('shellKit.demo.navBadgeGroupTitle') }}</h3>
      <p class="shell-card__hint">{{ t('shellKit.demo.navBadgeHint') }}</p>
      <div class="shell-section__badge-row">
        <NavBadge v-for="sample in navBadgeSamples" :key="sample.tone" :count="sample.count" :tone="sample.tone" />
        <NavBadge :count="240" tone="error" />
      </div>
    </article>

    <!-- ContentErrorBoundary -->
    <article class="shell-card">
      <h3 class="shell-card__title">{{ t('shellKit.demo.errorBoundaryGroupTitle') }}</h3>
      <p class="shell-card__hint">{{ t('shellKit.demo.errorBoundaryHint') }}</p>
      <button type="button" class="shell-section__btn shell-section__btn--primary" style="margin-bottom: var(--space-3)" @click="triggerError">
        {{ t('shellKit.demo.triggerError') }}
      </button>
      <div class="shell-section__boundary-box">
        <ContentErrorBoundary @retry="handleBoundaryRetry">
          <ThrowingDemoWidget />
        </ContentErrorBoundary>
      </div>
    </article>
  </section>
</template>

<style scoped>
.shell-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.shell-section__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.shell-section__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.shell-section__subtitle {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.shell-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  background: var(--card-bg);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  padding: var(--card-padding);
  box-shadow: var(--card-shadow);
}

.shell-card__title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.shell-card__hint {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.shell-section__preview-box {
  height: 480px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.shell-section__preview-box--short {
  height: 360px;
}

.shell-section__brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: var(--radius-md);
  background: var(--color-accent);
  color: var(--color-on-accent);
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-bold);
}

.shell-section__brand-text {
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.shell-section__workspace-shell {
  display: flex;
  align-items: stretch;
  height: 100%;
  background: var(--color-bg-app);
}

.shell-section__workspace-content {
  flex: 1 1 auto;
  min-width: 0;
  overflow-y: auto;
  padding: var(--space-5);
}

.shell-section__workspace-kicker {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--color-accent);
}

.shell-section__workspace-name {
  margin: var(--space-1) 0 0;
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.shell-section__workspace-meta {
  margin: 2px 0 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.shell-section__badge-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.shell-section__btn {
  display: inline-flex;
  align-items: center;
  height: var(--control-height);
  padding: 0 var(--space-4);
  border-radius: var(--control-radius);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-text-primary);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}

.shell-section__btn:hover {
  background: var(--color-inset);
}

.shell-section__btn:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.shell-section__btn--primary {
  border-color: var(--color-accent);
  background: var(--color-accent);
  color: var(--color-on-accent);
}

.shell-section__btn--primary:hover {
  background: var(--color-accent-hover);
}

.shell-section__boundary-box {
  min-height: 160px;
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
}

.shell-section__throw-ok {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}
</style>
