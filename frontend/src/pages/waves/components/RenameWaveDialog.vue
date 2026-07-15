<script setup lang="ts">
/**
 * RenameWaveDialog — modal form to rename a wave / edit its notes (plan
 * 3.2). The parent (`WavesPage.vue`) mounts this with `v-if` keyed on the
 * target wave, so a fresh instance (and fresh form state) is created every
 * time a different wave is renamed.
 */
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { NModal, NForm, NFormItem, NInput, NButton } from "naive-ui";
import { useFeedback } from "@/shared/ui/feedback";
import { updateWave } from "@/shared/api/bridge";
import type { dto } from "../../../../wailsjs/go/models";

const props = defineProps<{
  show: boolean;
  wave: dto.WaveDTO;
}>();

const emit = defineEmits<{
  "update:show": [value: boolean];
  renamed: [wave: dto.WaveDTO];
}>();

const { t } = useI18n({ useScope: "global" });
const feedback = useFeedback();

const name = ref(props.wave.name);
const notes = ref(props.wave.notes);
const submitting = ref(false);

// Defensive: if the parent ever reuses this instance across different
// targets (rather than the current v-if-per-target pattern), keep the form
// in sync with whichever wave is now the target.
watch(
  () => props.wave.id,
  () => {
    name.value = props.wave.name;
    notes.value = props.wave.notes;
  },
);

const canSubmit = computed(() => !submitting.value && name.value.trim().length > 0);

function close(): void {
  emit("update:show", false);
}

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value) return;
  submitting.value = true;
  try {
    const updated = await updateWave({
      waveId: props.wave.id,
      name: name.value.trim(),
      notes: notes.value,
      levelTags: props.wave.levelTags,
    });
    emit("renamed", updated);
    close();
  } catch (err) {
    feedback.error(t("feedback.error"), err instanceof Error ? err.message : String(err));
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="t('wavesList.renameDialog.title')"
    :style="{ width: 'min(480px, 92vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <NForm label-placement="top">
      <NFormItem :label="t('wavesList.createDialog.nameLabel')">
        <NInput
          v-model:value="name"
          :placeholder="t('wavesList.createDialog.namePlaceholder')"
          :disabled="submitting"
          @keydown.enter.prevent="handleSubmit"
        />
      </NFormItem>
      <NFormItem :label="t('wavesList.createDialog.notesLabel')">
        <NInput
          v-model:value="notes"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 5 }"
          :disabled="submitting"
        />
      </NFormItem>
    </NForm>
    <template #footer>
      <div class="rename-wave-dialog__footer">
        <NButton :disabled="submitting" @click="close">{{ t("wavesList.createDialog.cancel") }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t("wavesList.renameDialog.submit") }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.rename-wave-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
