<script setup lang="ts">
/**
 * CreateWaveDialog — modal form to create a new wave (plan 3.2). Presentational
 * and self-contained: owns its own form state, resets on every open, and
 * only talks to the backend through the bridge wrapper.
 */
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { NModal, NForm, NFormItem, NInput, NSelect, NButton } from "naive-ui";
import type { SelectOption } from "naive-ui";
import { useFeedback } from "@/shared/ui/feedback";
import { createWave } from "@/shared/api/bridge";
import type { dto } from "../../../../wailsjs/go/models";

const props = defineProps<{
  show: boolean;
}>();

const emit = defineEmits<{
  "update:show": [value: boolean];
  created: [wave: dto.WaveDTO];
}>();

const { t } = useI18n({ useScope: "global" });
const feedback = useFeedback();

const name = ref("");
const waveType = ref("mixed");
const notes = ref("");
const submitting = ref(false);

const waveTypeOptions = computed<SelectOption[]>(() => [
  { label: t("wavesList.waveType.membership"), value: "membership" },
  { label: t("wavesList.waveType.retail"), value: "retail" },
  { label: t("wavesList.waveType.mixed"), value: "mixed" },
]);

function resetForm(): void {
  name.value = "";
  waveType.value = "mixed";
  notes.value = "";
}

// Re-mounted-in-place (no v-if on this dialog in the parent) — reset the
// form every time it is (re)opened rather than only once at mount.
watch(
  () => props.show,
  (visible) => {
    if (visible) resetForm();
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
    const wave = await createWave({
      name: name.value.trim(),
      waveType: waveType.value,
      notes: notes.value,
    });
    emit("created", wave);
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
    :title="t('wavesList.createDialog.title')"
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
      <p class="create-wave-dialog__hint">{{ t("wavesList.createDialog.namePrefillHint") }}</p>
      <NFormItem :label="t('wavesList.createDialog.typeLabel')">
        <NSelect v-model:value="waveType" :options="waveTypeOptions" :disabled="submitting" />
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
      <div class="create-wave-dialog__footer">
        <NButton :disabled="submitting" @click="close">{{ t("wavesList.createDialog.cancel") }}</NButton>
        <NButton type="primary" :loading="submitting" :disabled="!canSubmit" @click="handleSubmit">
          {{ t("wavesList.createDialog.submit") }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<style scoped>
.create-wave-dialog__hint {
  margin: calc(var(--space-2) * -1) 0 var(--space-3);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.create-wave-dialog__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
