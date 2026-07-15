<script setup lang="ts">
/**
 * IntegrationsPage — the接入管理 top-level page (plan P4). Lists every
 * `IntegrationProfile` (bridge `listProfiles()`) grouped by `demandKind`
 * (`integrations.groups.membership`/`.retail`), each profile rendered as an
 * `IntegrationCard`. "New Integration" opens `IntakeWizard` (create mode)
 * in a modal; clicking a card opens `IntegrationDetailDrawer` for that
 * profile's id (the drawer owns its own fetch/mutations).
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { listProfiles } from '@/shared/api/bridge'
import { useWindowFocusRefresh } from '@/shared/lib/useWindowFocusRefresh'
import type { IntegrationProfile } from '@/entities/profile'
import IntegrationCard from './IntegrationCard.vue'
import IntegrationDetailDrawer from './IntegrationDetailDrawer.vue'
import IntakeWizard from './wizard/IntakeWizard.vue'

const { t } = useI18n({ useScope: 'global' })

const profiles = ref<IntegrationProfile[]>([])
const loading = ref(true)
const hasLoadedOnce = ref(false)

async function loadProfiles(): Promise<void> {
  loading.value = true
  try {
    profiles.value = await listProfiles()
  } finally {
    loading.value = false
    hasLoadedOnce.value = true
  }
}

onMounted(loadProfiles)
useWindowFocusRefresh(loadProfiles)

const membershipProfiles = computed(() => profiles.value.filter((p) => p.demandKind === 'membership_entitlement'))
const retailProfiles = computed(() => profiles.value.filter((p) => p.demandKind === 'retail_order'))
const showEmpty = computed(() => hasLoadedOnce.value && !loading.value && profiles.value.length === 0)

// ── Create wizard ──

const showWizard = ref(false)

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
        <NButton type="primary" @click="openWizard">{{ t('integrations.newIntegration') }}</NButton>
      </template>
    </PageHeader>

    <EmptyState v-if="showEmpty" :title="t('integrations.empty.title')" :description="t('integrations.empty.description')">
      <NButton type="primary" @click="openWizard">{{ t('integrations.empty.action') }}</NButton>
    </EmptyState>

    <template v-else>
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
    </template>

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

.integrations-page__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--space-3);
}
</style>
