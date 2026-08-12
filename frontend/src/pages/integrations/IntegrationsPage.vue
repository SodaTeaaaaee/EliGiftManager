<script setup lang="ts">
/**
 * IntegrationsPage — the接入管理 top-level page (plan P4). Lists every
 * `IntegrationProfile` (bridge `listProfiles()`) grouped by surface /
 * demandKind, each profile rendered as an `IntegrationCard`. "New Integration"
 * opens `IntakeWizard` (create mode) in a modal; clicking a card opens
 * `IntegrationDetailDrawer`.
 *
 * Loading and load-error states are user-visible (no silent failure).
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NSpin } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { ErrorBanner, useFeedback } from '@/shared/ui/feedback'
import { listProfiles, seedDefaultProfiles } from '@/shared/api/bridge'
import { useWindowFocusRefresh } from '@/shared/lib/useWindowFocusRefresh'
import type { IntegrationProfile } from '@/entities/profile'
import IntegrationCard from './IntegrationCard.vue'
import IntegrationDetailDrawer from './IntegrationDetailDrawer.vue'
import IntakeWizard from './wizard/IntakeWizard.vue'

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const profiles = ref<IntegrationProfile[]>([])
const loading = ref(true)
const hasLoadedOnce = ref(false)
const loadError = ref('')

async function loadProfiles(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    profiles.value = await listProfiles()
  } catch (err) {
    profiles.value = []
    loadError.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
    hasLoadedOnce.value = true
  }
}

onMounted(loadProfiles)
useWindowFocusRefresh(loadProfiles)

const membershipProfiles = computed(() =>
  profiles.value.filter(
    (p) => p.sourceSurface !== 'factory' && p.demandKind === 'membership_entitlement',
  ),
)
const retailProfiles = computed(() =>
  profiles.value.filter(
    (p) => p.sourceSurface !== 'factory' && p.demandKind === 'retail_order',
  ),
)
const factoryProfiles = computed(() =>
  profiles.value.filter((p) => p.sourceSurface === 'factory'),
)
const showEmpty = computed(
  () => hasLoadedOnce.value && !loading.value && !loadError.value && profiles.value.length === 0,
)

// ── Create wizard ──

const showWizard = ref(false)
const seedingDefaults = ref(false)

async function installDefaultProfiles(): Promise<void> {
  seedingDefaults.value = true
  try {
    await seedDefaultProfiles()
    await loadProfiles()
    feedback.success(t('integrations.defaultsInstalled'))
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    seedingDefaults.value = false
  }
}

function openWizard(): void {
  showWizard.value = true
}

function handleWizardDone(): void {
  showWizard.value = false
  void loadProfiles()
}

function handleWizardCancel(): void {
  showWizard.value = false
}

// ── Detail drawer ──

const detailProfileId = ref<number | null>(null)
const showDetail = ref(false)

function openDetail(profile: IntegrationProfile): void {
  detailProfileId.value = profile.id
  showDetail.value = true
}

function handleDetailVisibility(visible: boolean): void {
  showDetail.value = visible
}

function handleDetailChanged(): void {
  void loadProfiles()
}
</script>

<template>
  <div class="integrations-page">
    <PageHeader :title="t('integrations.title')" :description="t('integrations.subtitle')">
      <template #actions>
        <NButton :loading="seedingDefaults" @click="installDefaultProfiles">
          {{ t('integrations.installDefaults') }}
        </NButton>
        <NButton type="primary" @click="openWizard">{{ t('integrations.newIntegration') }}</NButton>
      </template>
    </PageHeader>

    <ErrorBanner
      v-if="loadError"
      :message="t('integrations.loadError')"
      :detail="loadError"
      @retry="loadProfiles"
    />

    <div v-if="loading && !hasLoadedOnce" class="integrations-page__loading">
      <NSpin size="medium" />
      <span class="integrations-page__loading-label">{{ t('integrations.loading') }}</span>
    </div>

    <EmptyState v-else-if="showEmpty" :title="t('integrations.empty.title')" :description="t('integrations.empty.description')">
      <NButton type="primary" @click="openWizard">{{ t('integrations.empty.action') }}</NButton>
    </EmptyState>

    <NSpin v-else :show="loading">
      <div class="integrations-page__groups">
        <SectionCard v-if="membershipProfiles.length" :title="t('integrations.groups.membership')">
          <div class="integrations-page__grid">
            <IntegrationCard
              v-for="profile in membershipProfiles"
              :key="profile.id"
              :profile="profile"
              @click="openDetail(profile)"
            />
          </div>
        </SectionCard>

        <SectionCard v-if="retailProfiles.length" :title="t('integrations.groups.retail')">
          <div class="integrations-page__grid">
            <IntegrationCard
              v-for="profile in retailProfiles"
              :key="profile.id"
              :profile="profile"
              @click="openDetail(profile)"
            />
          </div>
        </SectionCard>

        <SectionCard v-if="factoryProfiles.length" :title="t('integrations.groups.factory')">
          <div class="integrations-page__grid">
            <IntegrationCard
              v-for="profile in factoryProfiles"
              :key="profile.id"
              :profile="profile"
              @click="openDetail(profile)"
            />
          </div>
        </SectionCard>
      </div>
    </NSpin>

    <NModal
      :show="showWizard"
      preset="card"
      :title="t('intakeWizard.title')"
      :style="{ width: 'min(760px, 94vw)' }"
      :mask-closable="false"
      @update:show="(value: boolean) => (showWizard = value)"
    >
      <IntakeWizard v-if="showWizard" @done="handleWizardDone" @cancel="handleWizardCancel" />
    </NModal>

    <IntegrationDetailDrawer
      :profile-id="detailProfileId"
      :show="showDetail"
      @update:show="handleDetailVisibility"
      @changed="handleDetailChanged"
    />
  </div>
</template>

<style scoped>
.integrations-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.integrations-page__groups {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.integrations-page__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--space-3);
}

.integrations-page__loading {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-8) 0;
  justify-content: center;
}

.integrations-page__loading-label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>
