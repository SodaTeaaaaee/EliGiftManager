<script setup lang="ts">
/**
 * SuggestedMergesList — system-detected merge-suggestion list (plan §3.6
 * line 255, semantics ported from the old tree's
 * `CustomerManagementPage.vue` tab 2, NOT its merge-confirmation UX — every
 * "accept" here routes through `MergePreviewDialog` instead of calling
 * `mergeProfiles` directly).
 *
 * Fixed cross-unit interface (owned by the MERGE unit, imported by the
 * CUSTOMER unit): no props (self-fetches via `getMergeSuggestions`), emits
 * `preview` so the parent opens `MergePreviewDialog` with the suggested
 * pair. Dismiss is handled entirely locally via `dismissMergeSuggestion` +
 * refetch.
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NModal, NSpin } from 'naive-ui'
import { EmptyState } from '@/shared/ui/empty-state'
import { useFeedback } from '@/shared/ui/feedback'
import { getMergeSuggestions, dismissMergeSuggestion } from '@/shared/api/bridge'
import type { MergeSuggestionDTO } from '@/entities/customer'

const emit = defineEmits<{
  preview: [{ sourceProfileId: number; targetProfileId: number }]
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const suggestions = ref<MergeSuggestionDTO[]>([])
const loading = ref(false)
const pendingDismissId = ref<number | null>(null)
const dismissing = ref(false)

async function refetch(): Promise<void> {
  loading.value = true
  try {
    suggestions.value = await getMergeSuggestions()
  } finally {
    loading.value = false
  }
}

onMounted(refetch)

function requestPreview(suggestion: MergeSuggestionDTO): void {
  emit('preview', { sourceProfileId: suggestion.sourceProfileId, targetProfileId: suggestion.targetProfileId })
}

function requestDismiss(id: number): void {
  pendingDismissId.value = id
}

function cancelDismiss(): void {
  pendingDismissId.value = null
}

async function confirmDismiss(): Promise<void> {
  if (pendingDismissId.value == null) return
  dismissing.value = true
  try {
    await dismissMergeSuggestion(pendingDismissId.value)
    pendingDismissId.value = null
    await refetch()
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    dismissing.value = false
  }
}

defineExpose({ refetch })
</script>

<template>
  <div class="suggested-merges">
    <header class="suggested-merges__header">
      <h3 class="suggested-merges__title">{{ t('suggestedMerges.title') }}</h3>
      <p class="suggested-merges__subtitle">{{ t('suggestedMerges.subtitle') }}</p>
    </header>

    <div v-if="loading" class="suggested-merges__loading">
      <NSpin size="small" />
    </div>

    <EmptyState v-else-if="suggestions.length === 0" :title="t('suggestedMerges.empty')" size="sm" />

    <ul v-else class="suggested-merges__list">
      <li v-for="suggestion in suggestions" :key="suggestion.id" class="suggested-merges__item">
        <div class="suggested-merges__profiles">
          <span class="suggested-merges__profile-name">{{ suggestion.sourceProfile.displayName }}</span>
          <span class="suggested-merges__arrow" aria-hidden="true">→</span>
          <span class="suggested-merges__profile-name">{{ suggestion.targetProfile.displayName }}</span>
        </div>
        <p class="suggested-merges__reason">{{ t('suggestedMerges.reasonLabel') }}: {{ suggestion.reason }}</p>
        <div class="suggested-merges__actions">
          <NButton size="small" type="primary" @click="requestPreview(suggestion)">
            {{ t('suggestedMerges.previewAction') }}
          </NButton>
          <NButton size="small" quaternary @click="requestDismiss(suggestion.id)">
            {{ t('suggestedMerges.dismissAction') }}
          </NButton>
        </div>
      </li>
    </ul>

    <NModal
      :show="pendingDismissId != null"
      preset="dialog"
      type="warning"
      :title="t('suggestedMerges.dismissAction')"
      :content="t('suggestedMerges.dismissConfirm')"
      :positive-text="t('common.confirm')"
      :negative-text="t('common.cancel')"
      :loading="dismissing"
      @positive-click="confirmDismiss"
      @negative-click="cancelDismiss"
      @update:show="(value: boolean) => { if (!value) cancelDismiss() }"
    />
  </div>
</template>

<style scoped>
.suggested-merges {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.suggested-merges__header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.suggested-merges__title {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.suggested-merges__subtitle {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.suggested-merges__loading {
  display: flex;
  justify-content: center;
  padding: var(--space-4) 0;
}

.suggested-merges__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.suggested-merges__item {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
}

.suggested-merges__profiles {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.suggested-merges__arrow {
  color: var(--color-text-muted);
}

.suggested-merges__reason {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.suggested-merges__actions {
  display: flex;
  gap: var(--space-2);
}
</style>
