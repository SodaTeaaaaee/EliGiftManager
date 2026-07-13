<script setup lang="ts">
/**
 * CloseWaveDialog — modal confirmation to close a wave, with an optional
 * `force` override for residual open items (plan 3.2). The parent
 * (`WavesPage.vue`) mounts this with `v-if` keyed on the target wave, so a
 * fresh instance is created every time a different wave is closed.
 */
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { NModal, NForm, NFormItem, NInput, NCheckbox, NButton } from "naive-ui";
import { useFeedback } from "@/shared/ui/feedback";
import { closeWave } from "@/shared/api/bridge";
import type { dto } from "../../../../wailsjs/go/models";

const props = defineProps<{
  show: boolean;
  wave: dto.WaveDTO;
}>();

const emit = defineEmits<{
  "update:show": [value: boolean];
  closed: [result: dto.CloseWaveResult];
}>();

const { t } = useI18n({ useScope: "global" });
const feedback = useFeedback();

const force = ref(false);
const note = ref("");
const submitting = ref(false);

// Defensive: keep the form in sync if the parent ever reuses this instance
// across different targets instead of the current v-if-per-target pattern.
watch(
  () => props.wave.id,
  () => {
    force.value = false;
    note.value = "";
  },
);

const noteMissing = computed(() => force.value && note.value.trim().length === 0);
const canSubmit = computed(() => !submitting.value && !noteMissing.value);

function close(): void {
  emit("update:show", false);
}

async function handleSubmit(): Promise<void> {
  if (!canSubmit.value) return;
  submitting.value = true;
  try {
    const result = await closeWave({
      waveId: props.wave.id,
      note: note.value.trim(),
      force: force.value,
    });
    if (result.forced && result.residualItemCount > 0) {
      feedback.info(t("wavesList.closeDialog.residualWarning", { count: result.residualItemCount }));
    }
    emit("closed", result);
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
    :title="t('wavesList.closeDialog.title')"
    :style="{ width: 'min(480px, 92vw)' }"
    :mask-closable="!submitting"
    :close-on-esc="!submitting"
    @update:show="(value: boolean) => emit('update:show', value)"
  >
    <p class="close-wave-dialog__confirm">{{ t("wavesList.closeDialog.confirmText") }}</p>
    <NForm label-placement="top">
      <NFormItem>
        <NCheckbox v-model:checked="force" :disabled="submitting">
          {{ t("wavesList.closeDialog.forceLabel") }}
        </NCheckbox>
      </NFormItem>
      <NFormItem :label="t('wavesList.closeDialog.noteLabel')">
        <NInput
          v-model:value="note"
          type="textarea"
          :placeholder="t('wavesList.closeDialog.notePlaceholder')"
          :autosize="{ minRows: 2, maxRows: 5 }"
          :disabled="submitting"
        />
      </NFormItem>
      <p v-if="noteMissing" class="close-wave-dialog__note-required">{{ t("wavesList.closeDialog.noteRequired") }}</p>
    </NForm>
    <template #footer>
      <div class="close-wave-dialog__footer">
        <NButton :disabled="submitting" @click="close">{{ t("wavesList.createDialog.cancel") }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t("wavesList.closeDialog.submit") }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.close-wave-dialog__confirm {
  margin: 0 0 var(--space-3);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.close-wave-dialog__note-required {
  margin: calc(var(--space-2) * -1) 0 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--status-warning-fg);
}

.close-wave-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
