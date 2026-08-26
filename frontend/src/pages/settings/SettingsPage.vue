<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NForm,
  NFormItem,
  NInputNumber,
  NRadio,
  NRadioGroup,
  NSpace,
  NSpin,
} from 'naive-ui'
import { PageHeader } from '@/shared/ui/shell'
import { SectionCard } from '@/shared/ui/cards'
import { useFeedback } from '@/shared/ui/feedback'
import { useNaiveTheme } from '@/shared/theme/naive-bridge'
import { getDataDir, getSettings, revealInFolder, saveSettings } from '@/shared/api/bridge'
import type { AppSettings } from '@/entities/models'

const { t, locale } = useI18n()
const feedback = useFeedback()
const { densityMode, themeMode } = useNaiveTheme()

const loading = ref(false)
const saving = ref(false)
const dataDir = ref('')

const form = ref<AppSettings>({
  Locale: 'zh-CN',
  Theme: 'system',
  Density: 'comfortable',
  DuplicateRecordMinutes: 10,
  DuplicateAskDays: 10,
})

async function loadData() {
  loading.value = true
  try {
    const [settings, dir] = await Promise.all([
      getSettings(),
      getDataDir().catch(() => ''),
    ])
    form.value = {
      Locale: settings.Locale || 'zh-CN',
      Theme: settings.Theme || 'system',
      Density: settings.Density || 'comfortable',
      DuplicateRecordMinutes: settings.DuplicateRecordMinutes || 10,
      DuplicateAskDays: settings.DuplicateAskDays || 10,
    }
    dataDir.value = dir
  } catch (err) {
    console.error('Failed to load settings:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadData()
})

async function handleSave() {
  saving.value = true
  try {
    await saveSettings({
      Locale: form.value.Locale,
      Theme: form.value.Theme,
      Density: form.value.Density,
      DuplicateRecordMinutes: form.value.DuplicateRecordMinutes,
      DuplicateAskDays: form.value.DuplicateAskDays,
    })
    locale.value = form.value.Locale
    themeMode.value = form.value.Theme as 'system' | 'light' | 'dark'
    densityMode.value = form.value.Density as 'comfortable' | 'compact'
    feedback.success(t('settings.saveSuccess'))
  } catch (err) {
    console.error('Failed to save settings:', err)
    feedback.error(t('feedback.error'))
  } finally {
    saving.value = false
  }
}

async function handleRevealDataDir() {
  if (!dataDir.value) return
  try {
    await revealInFolder(dataDir.value)
  } catch (err) {
    console.error('Failed to reveal data dir:', err)
  }
}
</script>

<template>
  <div class="settings-page">
    <PageHeader :title="t('settings.title')" :description="t('settings.subtitle')">
      <template #actions>
        <NButton type="primary" size="small" :loading="saving" @click="handleSave">
          {{ t('common.save') }}
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <div class="settings-page__content">
        <!-- Appearance & Language Section -->
        <SectionCard :title="t('settings.appearance')">
          <NForm label-placement="left" label-width="140">
            <NFormItem :label="t('settings.theme')">
              <NRadioGroup v-model:value="form.Theme">
                <NSpace>
                  <NRadio value="system">
                    {{ t('designLab.controls.themeOptions.system') }}
                  </NRadio>
                  <NRadio value="light">
                    {{ t('designLab.controls.themeOptions.light') }}
                  </NRadio>
                  <NRadio value="dark">
                    {{ t('designLab.controls.themeOptions.dark') }}
                  </NRadio>
                </NSpace>
              </NRadioGroup>
            </NFormItem>

            <NFormItem :label="t('settings.density')">
              <NRadioGroup v-model:value="form.Density">
                <NSpace>
                  <NRadio value="comfortable">
                    {{ t('designLab.controls.densityOptions.comfortable') }}
                  </NRadio>
                  <NRadio value="compact">
                    {{ t('designLab.controls.densityOptions.compact') }}
                  </NRadio>
                </NSpace>
              </NRadioGroup>
            </NFormItem>

            <NFormItem :label="t('settings.locale')">
              <NRadioGroup v-model:value="form.Locale">
                <NSpace>
                  <NRadio value="zh-CN">{{ t('common.locales.zhCN') }}</NRadio>
                  <NRadio value="en-US">{{ t('common.locales.enUS') }}</NRadio>
                </NSpace>
              </NRadioGroup>
            </NFormItem>
          </NForm>
        </SectionCard>

        <!-- Deduplication Windows Section -->
        <SectionCard :title="t('settings.deduplication')">
          <NForm label-placement="left" label-width="220">
            <NFormItem :label="t('settings.duplicateRecordMinutes')">
              <NSpace vertical>
                <NInputNumber v-model:value="form.DuplicateRecordMinutes" :min="1" />
                <span class="settings-page__hint">
                  {{ t('settings.duplicateRecordMinutesDesc') }}
                </span>
              </NSpace>
            </NFormItem>

            <NFormItem :label="t('settings.duplicateAskDays')">
              <NSpace vertical>
                <NInputNumber v-model:value="form.DuplicateAskDays" :min="1" />
                <span class="settings-page__hint">
                  {{ t('settings.duplicateAskDaysDesc') }}
                </span>
              </NSpace>
            </NFormItem>
          </NForm>
        </SectionCard>

        <!-- Data Directory Section -->
        <SectionCard v-if="dataDir" :title="t('settings.dataDirectory')">
          <div class="settings-page__data-dir">
            <code class="settings-page__path">{{ dataDir }}</code>
            <NButton size="small" @click="handleRevealDataDir">
              {{ t('common.details') }}
            </NButton>
          </div>
        </SectionCard>
      </div>
    </NSpin>
  </div>
</template>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.settings-page__content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.settings-page__hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  line-height: var(--line-height-normal);
}

.settings-page__data-dir {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.settings-page__path {
  font-family: var(--font-mono);
  font-size: var(--font-size-sm);
  background: var(--color-surface-hover);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
}
</style>
