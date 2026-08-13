<script setup lang="ts">
/**
 * IntegrationsPage — added platforms (source / factory) plus an add-time
 * strip of uninstalled builtins. Custom「新建」and builtin「添加」only differ
 * at install; afterwards both open the same detail drawer.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NSpin } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { ErrorBanner, useFeedback } from '@/shared/ui/feedback'
import { listProfiles, seedBuiltinPlatform } from '@/shared/api/bridge'
import { useWindowFocusRefresh } from '@/shared/lib/useWindowFocusRefresh'
import type { IntegrationProfile } from '@/entities/profile'
import {
  installableBuiltins,
  partitionProfilesForList,
  type BuiltinPlatformDef,
} from './profileAvailability'
import IntegrationCard from './IntegrationCard.vue'
import IntegrationDetailDrawer from './IntegrationDetailDrawer.vue'
import CustomCreateModal from './CustomCreateModal.vue'

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

const groupedProfiles = computed(() => partitionProfilesForList(profiles.value))
const sourceProfiles = computed(() => groupedProfiles.value.source)
const factoryProfiles = computed(() => groupedProfiles.value.factory)
const hasAdded = computed(() => profiles.value.length > 0)
const availableBuiltins = computed(() => installableBuiltins(profiles.value))

const installingKey = ref<string | null>(null)
const showCreate = ref(false)

function openCreate(): void {
  showCreate.value = true
}

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

function handleCreated(profile: IntegrationProfile): void {
  void loadProfiles().then(() => openDetail(profile))
}

async function installBuiltin(item: BuiltinPlatformDef): Promise<void> {
  installingKey.value = item.installKey
  try {
    const profile = await seedBuiltinPlatform(item.installKey)
    await loadProfiles()
    feedback.success(t('integrations.builtins.installed'))
    openDetail(profile)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    installingKey.value = null
  }
}
</script>

<template>
  <div class="integrations-page">
    <PageHeader :title="t('integrations.title')" :description="t('integrations.subtitle')">
      <template #actions>
        <NButton type="primary" @click="openCreate">{{ t('integrations.newIntegration') }}</NButton>
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

    <NSpin v-else :show="loading">
      <div class="integrations-page__groups">
        <SectionCard v-if="hasAdded" :title="t('integrations.added.title')">
          <div v-if="sourceProfiles.length" class="integrations-page__subgroup">
            <h3 class="integrations-page__subgroup-title">{{ t('integrations.groups.source') }}</h3>
            <div class="integrations-page__grid">
              <IntegrationCard
                v-for="profile in sourceProfiles"
                :key="profile.id"
                :profile="profile"
                @click="openDetail(profile)"
              />
            </div>
          </div>
          <div v-if="factoryProfiles.length" class="integrations-page__subgroup">
            <h3 class="integrations-page__subgroup-title">{{ t('integrations.groups.factory') }}</h3>
            <div class="integrations-page__grid">
              <IntegrationCard
                v-for="profile in factoryProfiles"
                :key="profile.id"
                :profile="profile"
                @click="openDetail(profile)"
              />
            </div>
          </div>
        </SectionCard>

        <SectionCard v-if="availableBuiltins.length" :title="t('integrations.builtins.title')" :description="t('integrations.builtins.description')">
          <div class="integrations-page__grid">
            <div v-for="item in availableBuiltins" :key="item.installKey" class="integrations-page__builtin">
              <div class="integrations-page__builtin-copy">
                <h3 class="integrations-page__builtin-title">{{ t(`integrations.builtins.${item.i18nKey}.name`) }}</h3>
                <p class="integrations-page__builtin-desc">{{ t(`integrations.builtins.${item.i18nKey}.description`) }}</p>
              </div>
              <NButton
                type="primary"
                size="small"
                :loading="installingKey === item.installKey"
                :disabled="installingKey != null && installingKey !== item.installKey"
                @click="installBuiltin(item)"
              >
                {{ t('integrations.builtins.install') }}
              </NButton>
            </div>
          </div>
        </SectionCard>

        <SectionCard v-if="!hasAdded && !availableBuiltins.length" :title="t('integrations.empty.title')" :description="t('integrations.empty.description')">
          <NButton type="primary" @click="openCreate">{{ t('integrations.newIntegration') }}</NButton>
        </SectionCard>
      </div>
    </NSpin>

    <CustomCreateModal :show="showCreate" @update:show="(value) => (showCreate = value)" @created="handleCreated" />

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

.integrations-page__subgroup {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.integrations-page__subgroup + .integrations-page__subgroup {
  margin-top: var(--space-4);
}

.integrations-page__subgroup-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-secondary);
}

.integrations-page__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--space-3);
}

.integrations-page__builtin {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--card-padding);
  border: 1px dashed var(--card-border-color);
  border-radius: var(--card-radius);
  background: var(--card-bg);
}

.integrations-page__builtin-copy {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.integrations-page__builtin-title {
  margin: 0;
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.integrations-page__builtin-desc {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
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
