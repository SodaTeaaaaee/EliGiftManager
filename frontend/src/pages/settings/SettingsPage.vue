<script setup lang="ts">
/**
 * SettingsPage — the app's single settings surface (plan §3.7). Five
 * sections, each a thin UI wrapper over an already-complete store/bridge:
 * - Appearance: theme preference / density / skin, all driven by
 *   `useThemeStore()` (frontend/src/shared/theme/theme.ts) — every
 *   change applies live (the store's own watchers stamp
 *   `data-theme`/`data-density` on `<html>` and swap the skin stylesheet).
 * - Language: `useAppLocale()` (frontend/src/shared/i18n/index.ts) —
 *   switching re-renders the whole app instantly (vue-i18n reactive locale).
 * - Operator roster: full CRUD over `useOperatorRosterStore()`
 *   (frontend/src/shared/model/operator-roster.ts), the same
 *   localStorage-backed list `BatchAdjustDialog.vue`, `RowDetailDrawer.vue`,
 *   and `ManualClosureForm.vue` consume for their operatorId pickers.
 * - Data directory: resolved via the `getDataDir()` bridge wrapper (hard
 *   -fail — a missing/unresolvable data dir is a real error state), opened
 *   via the already-existing `revealInFolder()` wrapper.
 * - Duplicate-customer governance: versioned MergePolicy controls candidate
 *   detection only. Every save uses revision CAS and explicitly runs a scan;
 *   execution remains read-only `suggest_only` in this UI.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NInput, NPopconfirm, NSelect, NSwitch } from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { EmptyState } from '@/shared/ui/empty-state'
import { ErrorBanner, useFeedback } from '@/shared/ui/feedback'
import { isMergePolicyRevisionConflict } from '@/shared/lib/customer-resolution'
import { useThemeStore, type Density, type ThemePreference } from '@/shared/theme/theme'
import { useAppLocale, type SupportedLocale } from '@/shared/i18n'
import { listSkins } from '@/skins'
import { useOperatorRosterStore } from '@/shared/model/operator-roster'
import { getDataDir, getMergePolicy, revealInFolder, scanMergeCandidates, updateMergePolicy } from '@/shared/api/bridge'
import type { MergePolicyDTO, MergeScanRunDTO } from '@/entities/merge'
import AdvancedFeaturePolicySettings from './AdvancedFeaturePolicySettings.vue'
import ImportEvidenceSettings from './ImportEvidenceSettings.vue'
import { useCustomerResolutionFeaturePolicy } from '@/shared/composables/useCustomerResolutionFeaturePolicy'

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()
const featurePolicy = useCustomerResolutionFeaturePolicy()
const candidateWritesEnabled = computed(() => featurePolicy.isEnabled('candidateScanEnabled'))

// ── Appearance ──

const themeStore = useThemeStore()

const themeOptions = computed<SelectOption[]>(() => [
  { label: t('settings.appearance.themeOptions.system'), value: 'system' },
  { label: t('settings.appearance.themeOptions.light'), value: 'light' },
  { label: t('settings.appearance.themeOptions.dark'), value: 'dark' },
])

const densityOptions = computed<SelectOption[]>(() => [
  { label: t('settings.appearance.densityOptions.comfortable'), value: 'comfortable' },
  { label: t('settings.appearance.densityOptions.compact'), value: 'compact' },
])

const skinOptions = computed<SelectOption[]>(() =>
  listSkins().map((skin) => ({
    label: skin.name,
    value: skin.id,
    disabled: !skin.supports[themeStore.resolvedTheme],
  })),
)

const selectedSkinUnsupported = computed(() => {
  const skin = listSkins().find((entry) => entry.id === themeStore.skinId)
  return skin != null && !skin.supports[themeStore.resolvedTheme]
})

function handleThemeChange(value: ThemePreference): void {
  themeStore.setPreference(value)
}

function handleDensityChange(value: Density): void {
  themeStore.setDensity(value)
}

function handleSkinChange(value: string): void {
  themeStore.setSkinId(value)
}

// ── Language ──

const { locale, localeOptions, setLocale } = useAppLocale()

async function handleLocaleChange(value: SupportedLocale): Promise<void> {
  await setLocale(value)
}

// ── Operator roster ──

const operatorRoster = useOperatorRosterStore()
const newOperatorId = ref('')

function handleAddOperator(): void {
  const trimmed = newOperatorId.value.trim()
  if (!trimmed) return
  operatorRoster.add(trimmed)
  newOperatorId.value = ''
}

function handleRemoveOperator(id: string): void {
  operatorRoster.remove(id)
}

// ── Data directory ──

const dataDirPath = ref<string | null>(null)
const dataDirLoading = ref(true)
const dataDirFailed = ref(false)
const openingDataDir = ref(false)

async function loadDataDir(): Promise<void> {
  dataDirLoading.value = true
  dataDirFailed.value = false
  try {
    dataDirPath.value = await getDataDir()
  } catch (err) {
    dataDirPath.value = null
    dataDirFailed.value = true
    feedback.error(t('settings.dataDir.loadFailed'), err instanceof Error ? err.message : String(err))
  } finally {
    dataDirLoading.value = false
  }
}

async function handleOpenDataDir(): Promise<void> {
  if (!dataDirPath.value) return
  openingDataDir.value = true
  try {
    await revealInFolder(dataDirPath.value)
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    openingDataDir.value = false
  }
}

// ── Duplicate-customer governance ──

const mergePolicy = ref<MergePolicyDTO | null>(null)
const candidateDetectionEnabled = ref(false)
const emailEvidenceMode = ref('off')
const phoneEvidenceMode = ref('off')
const mergePolicyLoading = ref(false)
const mergePolicySaving = ref(false)
const mergeScanRunning = ref(false)
const mergeScanRun = ref<MergeScanRunDTO | null>(null)
const mergePolicyError = ref<string | null>(null)
const mergePolicyErrorKind = ref<'load' | 'save' | null>(null)

const emailEvidenceEnabled = computed(() => emailEvidenceMode.value !== 'off')
const phoneEvidenceEnabled = computed(() => phoneEvidenceMode.value !== 'off')
const emailEvidenceDormant = computed(() => emailEvidenceEnabled.value && !candidateDetectionEnabled.value)
const phoneEvidenceDormant = computed(() => phoneEvidenceEnabled.value && !candidateDetectionEnabled.value)

async function loadMergePolicy(): Promise<void> {
  mergePolicyLoading.value = true
  mergePolicyError.value = null
  mergePolicyErrorKind.value = null
  try {
    const policy = await getMergePolicy()
    mergePolicy.value = policy
    candidateDetectionEnabled.value = policy.rules.candidateDetectionEnabled
    emailEvidenceMode.value = policy.rules.emailEvidenceMode
    phoneEvidenceMode.value = policy.rules.phoneEvidenceMode
  } catch (err) {
    mergePolicyError.value = err instanceof Error ? err.message : String(err)
    mergePolicyErrorKind.value = 'load'
  } finally {
    mergePolicyLoading.value = false
  }
}

async function runMergeScan(): Promise<void> {
  if (!candidateWritesEnabled.value) return
  mergeScanRunning.value = true
  try {
    mergeScanRun.value = await scanMergeCandidates()
    await loadMergePolicy()
  } catch (err) {
    feedback.error(t('settings.mergePolicy.scanFailed'), err instanceof Error ? err.message : String(err))
  } finally {
    mergeScanRunning.value = false
  }
}

async function persistMergePolicy(): Promise<void> {
  if (!mergePolicy.value) return
  mergePolicySaving.value = true
  mergePolicyError.value = null
  mergePolicyErrorKind.value = null
  try {
    mergePolicy.value = await updateMergePolicy({
      expectedRevision: mergePolicy.value.revision,
      rules: {
        candidateDetectionEnabled: candidateDetectionEnabled.value,
        emailEvidenceMode: emailEvidenceMode.value,
        phoneEvidenceMode: phoneEvidenceMode.value,
        executionMode: 'suggest_only',
      },
      actorRef: 'local_user',
    })
    feedback.success(t('settings.feedback.settingsSaved'))
    if (candidateWritesEnabled.value) await runMergeScan()
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err)
    if (isMergePolicyRevisionConflict(err)) {
      feedback.error(t('settings.mergePolicy.revisionConflict'), message)
      await loadMergePolicy()
    } else {
      mergePolicyError.value = message
      mergePolicyErrorKind.value = 'save'
      feedback.error(t('settings.mergePolicy.saveFailed'), message)
    }
  } finally {
    mergePolicySaving.value = false
  }
}

function handleCandidateDetectionChange(value: boolean): void {
  candidateDetectionEnabled.value = value
  void persistMergePolicy()
}

function handleEmailEvidenceChange(value: boolean): void {
  emailEvidenceMode.value = value
    ? (mergePolicy.value?.rules.emailEvidenceMode === 'legacy_raw_exact' ? 'legacy_raw_exact' : 'normalized_verified')
    : 'off'
  void persistMergePolicy()
}

function handlePhoneEvidenceChange(value: boolean): void {
  phoneEvidenceMode.value = value
    ? (mergePolicy.value?.rules.phoneEvidenceMode === 'legacy_raw_exact' ? 'legacy_raw_exact' : 'normalized')
    : 'off'
  void persistMergePolicy()
}

onMounted(() => {
  void featurePolicy.load()
  void loadDataDir()
  void loadMergePolicy()
})
</script>

<template>
  <div class="settings-page">
    <PageHeader :title="t('settings.title')" :description="t('settings.subtitle')" />

    <SectionCard :title="t('settings.sections.appearance')">
      <div class="settings-page__form-grid">
        <div class="settings-page__field">
          <label class="settings-page__field-label">{{ t('settings.appearance.themeLabel') }}</label>
          <NSelect :value="themeStore.preference" :options="themeOptions" @update:value="handleThemeChange" />
        </div>
        <div class="settings-page__field">
          <label class="settings-page__field-label">{{ t('settings.appearance.densityLabel') }}</label>
          <NSelect :value="themeStore.density" :options="densityOptions" @update:value="handleDensityChange" />
        </div>
        <div class="settings-page__field">
          <label class="settings-page__field-label">{{ t('settings.appearance.skinLabel') }}</label>
          <NSelect :value="themeStore.skinId" :options="skinOptions" @update:value="handleSkinChange" />
          <p v-if="selectedSkinUnsupported" class="settings-page__hint">
            {{ t('settings.appearance.skinUnsupportedHint') }}
          </p>
        </div>
      </div>
    </SectionCard>

    <SectionCard :title="t('settings.sections.language')">
      <div class="settings-page__form-grid">
        <div class="settings-page__field">
          <label class="settings-page__field-label">{{ t('settings.language.label') }}</label>
          <NSelect :value="locale" :options="localeOptions" @update:value="handleLocaleChange" />
        </div>
      </div>
    </SectionCard>

    <SectionCard :title="t('settings.sections.operatorRoster')" :description="t('settings.operatorRoster.description')">
      <div class="settings-page__operator-add">
        <NInput
          v-model:value="newOperatorId"
          :placeholder="t('settings.operatorRoster.addPlaceholder')"
          @keydown.enter="handleAddOperator"
        />
        <NButton type="primary" :disabled="!newOperatorId.trim()" @click="handleAddOperator">
          {{ t('settings.operatorRoster.addAction') }}
        </NButton>
      </div>

      <EmptyState v-if="operatorRoster.ids.length === 0" size="sm" :title="t('settings.operatorRoster.empty')" />
      <ul v-else class="settings-page__roster-list">
        <li v-for="id in operatorRoster.ids" :key="id" class="settings-page__roster-item">
          <span class="settings-page__roster-item-label">{{ id }}</span>
          <NPopconfirm
            :positive-text="t('common.confirm')"
            :negative-text="t('common.cancel')"
            @positive-click="handleRemoveOperator(id)"
          >
            <template #trigger>
              <NButton size="small" quaternary>{{ t('settings.operatorRoster.removeAction') }}</NButton>
            </template>
            {{ t('settings.operatorRoster.removeConfirm', { name: id }) }}
          </NPopconfirm>
        </li>
      </ul>
    </SectionCard>

    <SectionCard :title="t('settings.sections.dataDir')" :description="t('settings.dataDir.description')">
      <div class="settings-page__data-dir">
        <div class="settings-page__field">
          <label class="settings-page__field-label">{{ t('settings.dataDir.pathLabel') }}</label>
          <code class="settings-page__data-dir-path">
            {{ dataDirLoading ? t('common.loading') : (dataDirPath ?? t('settings.dataDir.loadFailed')) }}
          </code>
        </div>
        <NButton :disabled="!dataDirPath || openingDataDir" :loading="openingDataDir" @click="handleOpenDataDir">
          {{ t('settings.dataDir.openAction') }}
        </NButton>
      </div>
    </SectionCard>

    <SectionCard :title="t('settings.sections.mergePolicy')" :description="t('settings.mergePolicy.description')">
      <ErrorBanner
        v-if="mergePolicyError"
        :message="mergePolicyErrorKind === 'save' ? t('settings.mergePolicy.saveFailed') : t('settings.mergePolicy.loadFailed')"
        :detail="mergePolicyError"
        @retry="mergePolicyErrorKind === 'save' ? persistMergePolicy() : loadMergePolicy()"
      />
      <div class="settings-page__toggle-list">
        <div class="settings-page__toggle-row">
          <div class="settings-page__toggle-copy">
            <span class="settings-page__toggle-label">{{ t('settings.mergePolicy.detectionLabel') }}</span>
            <span class="settings-page__toggle-desc">{{ t('settings.mergePolicy.detectionDesc') }}</span>
          </div>
          <NSwitch
            :value="candidateDetectionEnabled"
            :disabled="mergePolicyLoading || mergePolicySaving || mergeScanRunning || !mergePolicy"
            @update:value="handleCandidateDetectionChange"
          />
        </div>
        <div class="settings-page__toggle-row">
          <div class="settings-page__toggle-copy">
            <span class="settings-page__toggle-label">{{ t('settings.mergePolicy.emailLabel') }}</span>
            <span class="settings-page__toggle-desc">{{ t('settings.mergePolicy.emailDesc') }}</span>
            <span v-if="emailEvidenceDormant" class="settings-page__toggle-desc">{{ t('settings.mergePolicy.dormant') }}</span>
          </div>
          <NSwitch
            :value="emailEvidenceEnabled"
            :disabled="!candidateDetectionEnabled || mergePolicySaving || mergeScanRunning"
            @update:value="handleEmailEvidenceChange"
          />
        </div>
        <div class="settings-page__toggle-row">
          <div class="settings-page__toggle-copy">
            <span class="settings-page__toggle-label">{{ t('settings.mergePolicy.phoneLabel') }}</span>
            <span class="settings-page__toggle-desc">{{ t('settings.mergePolicy.phoneDesc') }}</span>
            <span v-if="phoneEvidenceDormant" class="settings-page__toggle-desc">{{ t('settings.mergePolicy.dormant') }}</span>
          </div>
          <NSwitch
            :value="phoneEvidenceEnabled"
            :disabled="!candidateDetectionEnabled || mergePolicySaving || mergeScanRunning"
            @update:value="handlePhoneEvidenceChange"
          />
        </div>
        <div class="settings-page__toggle-row">
          <div class="settings-page__toggle-copy">
            <span class="settings-page__toggle-label">{{ t('settings.mergePolicy.executionLabel') }}</span>
            <span class="settings-page__toggle-desc">{{ t('settings.mergePolicy.executionDesc') }}</span>
          </div>
          <code>{{ t('settings.mergePolicy.executionValue') }}</code>
        </div>
        <div class="settings-page__scan-summary">
          <NButton :loading="mergeScanRunning" :disabled="mergePolicySaving || !mergePolicy || !candidateWritesEnabled" @click="runMergeScan">
            {{ t('settings.mergePolicy.scanAction') }}
          </NButton>
          <span v-if="mergeScanRun && mergeScanRun.status === 'completed'">
            {{ t('settings.mergePolicy.scanCompleted', {
              profiles: mergeScanRun.profilesScanned,
              pairs: mergeScanRun.pairsEvaluated,
              created: mergeScanRun.candidatesCreated,
              updated: mergeScanRun.candidatesUpdated,
              blocked: mergeScanRun.candidatesBlocked,
            }) }}
          </span>
          <span v-else-if="mergeScanRun?.errorMessage" class="settings-page__scan-error">
            {{ mergeScanRun.errorMessage }}
          </span>
          <span v-else-if="mergePolicy?.needsScan">{{ t('settings.mergePolicy.scanNeeded') }}</span>
        </div>
      </div>
    </SectionCard>

    <SectionCard :title="t('settings.sections.featurePolicy')" :description="t('settings.featurePolicy.description')">
      <AdvancedFeaturePolicySettings />
    </SectionCard>

    <SectionCard :title="t('settings.sections.importEvidence')" :description="t('settings.importEvidence.description')">
      <ImportEvidenceSettings />
    </SectionCard>
  </div>
</template>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.settings-page__form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--space-4);
}

.settings-page__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.settings-page__field-label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-secondary);
}

.settings-page__hint {
  margin: var(--space-1) 0 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.settings-page__operator-add {
  display: flex;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
  max-width: 480px;
}

.settings-page__roster-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.settings-page__roster-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--card-bg);
}

.settings-page__roster-item-label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.settings-page__data-dir {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.settings-page__data-dir-path {
  display: inline-block;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  font-family: var(--font-mono, monospace);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  word-break: break-all;
}

.settings-page__toggle-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.settings-page__toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--color-border);
}

.settings-page__toggle-row:last-of-type {
  border-bottom: none;
}

.settings-page__toggle-copy {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}

.settings-page__scan-summary {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-2);
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}

.settings-page__scan-error {
  color: var(--status-error-fg);
}

.settings-page__toggle-label {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.settings-page__toggle-desc {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}
</style>
