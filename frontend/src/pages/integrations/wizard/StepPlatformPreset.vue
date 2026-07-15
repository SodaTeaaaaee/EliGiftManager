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
import { SelectableCard } from '@/shared/ui/cards'
import { PLATFORM_PRESETS } from '@/shared/lib/demand-intake/platform-presets'
import type { UseIntakeWizardStateApi } from './useIntakeWizardState'

const props = defineProps<{ state: UseIntakeWizardStateApi }>()

const { t } = useI18n({ useScope: 'global' })

const isCustom = computed(() => props.state.presetKey.value === 'custom')
const isFactory = computed(() => props.state.isFactorySurface.value)

function selectPreset(key: string): void {
  props.state.applyPreset(key)
}
</script>

<template>
  <div class="step-platform-preset">
    <div class="step-platform-preset__grid">
      <SelectableCard
        v-for="preset in PLATFORM_PRESETS"
        :key="preset.key"
        :label="t(preset.labelKey)"
        :description="t(preset.descKey)"
        :selected="state.presetKey.value === preset.key"
        @select="selectPreset(preset.key)"
      />
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
        <NFormItem v-if="!isFactory" :label="t('intakeWizard.profileFields.sourceSurfaceLabel')" :show-feedback="false">
          <NInput
            v-model:value="state.sourceSurface.value"
            :placeholder="t('intakeWizard.profileFields.sourceSurfacePlaceholder')"
          />
        </NFormItem>
      </template>

      <NFormItem
        :label="t('intakeWizard.profileFields.factorySupplierPlatformLabel')"
        :show-feedback="false"
      >
        <NInput
          v-model:value="state.factorySupplierPlatform.value"
          :placeholder="t('intakeWizard.profileFields.factorySupplierPlatformPlaceholder')"
        />
      </NFormItem>
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

.step-platform-preset__fields {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  max-width: 420px;
}
</style>
