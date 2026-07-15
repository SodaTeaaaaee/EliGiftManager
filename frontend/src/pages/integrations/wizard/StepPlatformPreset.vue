<script setup lang="ts">
/**
 * StepPlatformPreset — wizard step 1 (create mode only; never mounted in
 * remap mode since `REMAP_STEPS` excludes it). Presets as a card-radio grid
 * (`PLATFORM_PRESETS`), plus the profile-identity fields (`profileKey`
 * always; `sourceChannel`/`sourceSurface` only editable once `custom` is
 * selected — named presets fix these from the preset itself).
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NInput, NFormItem } from 'naive-ui'
import { PLATFORM_PRESETS } from '@/shared/lib/demand-intake/platform-presets'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t } = useI18n({ useScope: 'global' })

const isCustom = computed(() => props.state.presetKey.value === 'custom')

function selectPreset(key: string): void {
  props.state.applyPreset(key)
}
</script>

<template>
  <div class="step-platform-preset">
    <div class="step-platform-preset__grid">
      <button
        v-for="preset in PLATFORM_PRESETS"
        :key="preset.key"
        type="button"
        class="step-platform-preset__card"
        :class="{ 'step-platform-preset__card--active': state.presetKey.value === preset.key }"
        @click="selectPreset(preset.key)"
      >
        <span class="step-platform-preset__card-label">{{ t(preset.labelKey) }}</span>
        <span class="step-platform-preset__card-desc">{{ t(preset.descKey) }}</span>
      </button>
    </div>

    <div class="step-platform-preset__fields">
      <NFormItem :label="t('intakeWizard.profileFields.profileKeyLabel')" :show-feedback="false">
        <NInput
          :value="state.profileKey.value"
          :placeholder="t('intakeWizard.profileFields.profileKeyPlaceholder')"
          @update:value="state.setProfileKey"
        />
      </NFormItem>

      <template v-if="isCustom">
        <NFormItem :label="t('intakeWizard.profileFields.sourceChannelLabel')" :show-feedback="false">
          <NInput
            v-model:value="state.sourceChannel.value"
            :placeholder="t('intakeWizard.profileFields.sourceChannelPlaceholder')"
          />
        </NFormItem>
        <NFormItem :label="t('intakeWizard.profileFields.sourceSurfaceLabel')" :show-feedback="false">
          <NInput
            v-model:value="state.sourceSurface.value"
            :placeholder="t('intakeWizard.profileFields.sourceSurfacePlaceholder')"
          />
        </NFormItem>
      </template>
    </div>
  </div>
</template>

<style scoped>
.step-platform-preset {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.step-platform-preset__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--space-3);
}

.step-platform-preset__card {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  text-align: left;
  padding: var(--space-3);
  border: 1px solid var(--card-border-color);
  border-radius: var(--card-radius);
  background: var(--color-surface);
  cursor: pointer;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}

.step-platform-preset__card:hover {
  border-color: var(--color-accent);
}

.step-platform-preset__card--active {
  border-color: var(--color-accent);
  background: var(--color-accent-subtle);
}

.step-platform-preset__card-label {
  font-family: var(--font-display);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.step-platform-preset__card-desc {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  line-height: var(--line-height-relaxed);
  color: var(--color-text-muted);
}

.step-platform-preset__fields {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  max-width: 420px;
}
</style>
