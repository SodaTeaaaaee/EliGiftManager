<script setup lang="ts">
/**
 * Custom platform create — name + source vs factory only.
 * After create, the parent opens the same platform detail used for every
 * added platform (builtin shortcut or custom).
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NFormItem, NInput, NModal } from 'naive-ui'
import { SelectableCard } from '@/shared/ui/cards'
import { ErrorBanner, useFeedback } from '@/shared/ui/feedback'
import { createProfile } from '@/shared/api/bridge'
import type { IntegrationProfile } from '@/entities/profile'
import type { BusinessSurfaceChoice } from './wizard/deriveProfileDefaults'
import {
  buildCustomCreateProfileInput,
  suggestProfileKey,
} from './wizard/deriveProfileDefaults'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  created: [profile: IntegrationProfile]
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const displayName = ref('')
const profileKey = ref('')
const profileKeyTouched = ref(false)
const surface = ref<BusinessSurfaceChoice>('source')
const factorySupplierPlatform = ref('')
const submitting = ref(false)
const submitError = ref('')

const canSubmit = computed(() => {
  if (!displayName.value.trim() || !profileKey.value.trim() || submitting.value) return false
  if (surface.value === 'factory') {
    return (factorySupplierPlatform.value.trim() || displayName.value.trim()).length > 0
  }
  return true
})

function reset(): void {
  displayName.value = ''
  profileKey.value = ''
  profileKeyTouched.value = false
  surface.value = 'source'
  factorySupplierPlatform.value = ''
  submitting.value = false
  submitError.value = ''
}

watch(
  () => props.show,
  (visible) => {
    if (visible) reset()
  },
)

function handleDisplayName(value: string): void {
  displayName.value = value
  if (!profileKeyTouched.value) {
    profileKey.value = suggestProfileKey(value)
  }
}

function handleProfileKey(value: string): void {
  profileKeyTouched.value = true
  profileKey.value = value
}

function close(): void {
  if (submitting.value) return
  emit('update:show', false)
}

async function submit(): Promise<void> {
  if (!canSubmit.value) return
  submitError.value = ''
  submitting.value = true
  try {
    const profile = await createProfile(buildCustomCreateProfileInput({
      displayName: displayName.value,
      profileKey: profileKey.value,
      surface: surface.value,
      factorySupplierPlatform: factorySupplierPlatform.value,
    }))
    feedback.success(t('integrations.customCreate.created'))
    emit('created', profile)
    emit('update:show', false)
  } catch (err) {
    submitError.value = err instanceof Error ? err.message : String(err)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('integrations.customCreate.title')"
    :style="{ width: 'min(520px, 94vw)' }"
    :mask-closable="false"
    @update:show="(value: boolean) => { if (!value) close() }"
  >
    <div class="custom-create">
      <NFormItem :label="t('integrations.customCreate.displayName')" :show-feedback="false">
        <NInput
          :value="displayName"
          :placeholder="t('integrations.customCreate.displayNamePlaceholder')"
          @update:value="handleDisplayName"
        />
      </NFormItem>
      <NFormItem :label="t('integrations.customCreate.profileKey')" :show-feedback="false">
        <NInput
          :value="profileKey"
          :placeholder="t('integrations.customCreate.profileKeyPlaceholder')"
          @update:value="handleProfileKey"
        />
      </NFormItem>

      <p class="custom-create__label">{{ t('integrations.customCreate.surface') }}</p>
      <div class="custom-create__surfaces">
        <SelectableCard
          :label="t('intakeWizard.platformKind.source.label')"
          :description="t('intakeWizard.platformKind.source.description')"
          :selected="surface === 'source'"
          @select="surface = 'source'"
        />
        <SelectableCard
          :label="t('intakeWizard.platformKind.factory.label')"
          :description="t('intakeWizard.platformKind.factory.description')"
          :selected="surface === 'factory'"
          @select="surface = 'factory'"
        />
      </div>

      <NFormItem
        v-if="surface === 'factory'"
        :label="t('integrations.customCreate.factoryLabel')"
        :show-feedback="false"
      >
        <NInput
          v-model:value="factorySupplierPlatform"
          :placeholder="t('integrations.customCreate.factoryLabelPlaceholder')"
        />
      </NFormItem>

      <ErrorBanner
        v-if="submitError"
        :message="t('integrations.customCreate.failed')"
        :detail="submitError"
      />

      <div class="custom-create__actions">
        <NButton :disabled="submitting" @click="close">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :disabled="!canSubmit" :loading="submitting" @click="submit">
          {{ t('integrations.customCreate.submit') }}
        </NButton>
      </div>
    </div>
  </NModal>
</template>

<style scoped>
.custom-create {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.custom-create__label {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.custom-create__surfaces {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--space-3);
}

.custom-create__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
