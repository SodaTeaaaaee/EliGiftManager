<script setup lang="ts">
import { NInput, NRadio, NRadioGroup } from 'naive-ui'
import type { DisplayNameEditState, DisplayNameMode } from '@/shared/lib/customer-resolution'

defineProps<{
  state: DisplayNameEditState
  labels: {
    auto: string
    pinned: string
    autoPreview: string
    nameInput: string
  }
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:mode': [DisplayNameMode]
  'update:name': [string]
}>()
</script>

<template>
  <div class="display-name-mode">
    <NRadioGroup
      :value="state.mode"
      :disabled="disabled"
      @update:value="(value: DisplayNameMode) => emit('update:mode', value)"
    >
      <NRadio value="auto">{{ labels.auto }}</NRadio>
      <NRadio value="pinned">{{ labels.pinned }}</NRadio>
    </NRadioGroup>

    <p v-if="state.mode === 'auto'" class="display-name-mode__preview">
      {{ labels.autoPreview }}: <strong>{{ state.autoName }}</strong>
    </p>
    <label v-else class="display-name-mode__input">
      <span>{{ labels.nameInput }}</span>
      <NInput
        :value="state.draftName"
        :disabled="disabled"
        @update:value="(value: string) => emit('update:name', value)"
      />
    </label>
  </div>
</template>

<style scoped>
.display-name-mode,
.display-name-mode__input {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.display-name-mode__preview {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--font-size-sm);
}

.display-name-mode__input > span {
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
}
</style>
