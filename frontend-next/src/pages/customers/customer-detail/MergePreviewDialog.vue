<script setup lang="ts">
/**
 * MergePreviewDialog — pre-merge safety gate (plan §3.6 line 255 "合并安全化").
 * Fixed cross-unit interface (owned by the MERGE unit, imported by the
 * CUSTOMER unit): props { show, sourceProfileId, targetProfileId }, emits
 * `update:show` / `merged`.
 *
 * Flow: opening with both ids set calls `previewMergeProfiles` (read-only,
 * never mutates — see `controller_merge_preview.go`'s own doc comment) and
 * renders both profiles' full identity/address lists side by side, with
 * `MergePreviewConflict` fields (displayName/profileType) highlighted in
 * place plus a dedicated conflicts list, and `duplicateIdentityValues`
 * surfaced as a warning list. Confirming calls `mergeProfiles` (the real,
 * transactional mutation) and swaps the dialog body to an inline receipt.
 * Closing that receipt emits `merged`; an available undo is confirmed in a
 * local modal and emits `undone` with the restored source profile id.
 *
 * Local `NModal` (not `useDialog()`) follows the established codebase
 * convention — no `<NDialogProvider>` is mounted in `App.vue` (see
 * `WaveHistoryDrawer.vue`'s header comment for the same rationale).
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NButton, NSpin } from 'naive-ui'
import { StatusBadge } from '@/shared/ui/status'
import { useFeedback } from '@/shared/ui/feedback'
import { previewMergeProfiles, mergeProfiles, undoCustomerMerge } from '@/shared/api/bridge'
import type { MergeProfilesPreviewResult, MergePreviewProfileSide } from '@/entities/customer'
import type { MergeProfilesResult, UndoCustomerMergeResult } from '@/entities/merge'

const props = defineProps<{
  show: boolean
  sourceProfileId: number | null
  targetProfileId: number | null
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  merged: [MergeProfilesResult]
  undone: [UndoCustomerMergeResult]
}>()

const { t } = useI18n({ useScope: 'global' })
const feedback = useFeedback()

const loading = ref(false)
const loadError = ref<string | null>(null)
const preview = ref<MergeProfilesPreviewResult | null>(null)
const executing = ref(false)
const receipt = ref<MergeProfilesResult | null>(null)
const showUndoConfirm = ref(false)
const undoing = ref(false)

async function loadPreview(sourceId: number, targetId: number): Promise<void> {
  loading.value = true
  loadError.value = null
  preview.value = null
  try {
    preview.value = await previewMergeProfiles(sourceId, targetId)
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

// Dialog stays mounted (no v-if at the call site) — (re)load whenever it
// opens with both ids resolved. Reading ids from the watch tuple (not
// `props.*` inside the async body) avoids a stale-id race if the caller
// swaps ids while the dialog happens to still be open.
watch(
  () => [props.show, props.sourceProfileId, props.targetProfileId] as const,
  ([visible, sourceId, targetId]) => {
    if (visible && sourceId != null && targetId != null) {
      receipt.value = null
      showUndoConfirm.value = false
      void loadPreview(sourceId, targetId)
    }
  },
)

interface SideEntry {
  key: 'source' | 'target'
  labelKey: string
  side: MergePreviewProfileSide
}

const sides = computed<SideEntry[]>(() => {
  if (!preview.value) return []
  return [
    { key: 'source', labelKey: 'merge.sourceSide', side: preview.value.source },
    { key: 'target', labelKey: 'merge.targetSide', side: preview.value.target },
  ]
})

// Conflict field set is closed today (displayName|profileType per
// `profile_merge_preview_usecase.go`) but read generically off the DTO so a
// future backend-added field highlights automatically without a frontend
// change.
const conflictFields = computed<Set<string>>(() => new Set((preview.value?.conflicts ?? []).map((c) => c.field)))

function conflictFieldLabel(field: string): string {
  return field === 'displayName' || field === 'profileType' ? t(`merge.conflictField.${field}`) : field
}

function close(): void {
  if (undoing.value) return
  if (receipt.value) {
    const result = receipt.value
    receipt.value = null
    emit('update:show', false)
    emit('merged', result)
    return
  }
  emit('update:show', false)
}

function handleVisibility(value: boolean): void {
  if (value) {
    emit('update:show', true)
  } else {
    close()
  }
}

function retry(): void {
  if (props.sourceProfileId == null || props.targetProfileId == null) return
  void loadPreview(props.sourceProfileId, props.targetProfileId)
}

async function handleConfirm(): Promise<void> {
  if (props.sourceProfileId == null || props.targetProfileId == null || executing.value) return
  executing.value = true
  try {
    const result = await mergeProfiles({ sourceProfileId: props.sourceProfileId, targetProfileId: props.targetProfileId })
    receipt.value = result
  } catch (err) {
    feedback.error(t('feedback.error'), err instanceof Error ? err.message : String(err))
  } finally {
    executing.value = false
  }
}

async function handleUndo(): Promise<void> {
  if (!receipt.value?.undoAvailable || undoing.value) return
  undoing.value = true
  try {
    const result = await undoCustomerMerge({ mergeId: receipt.value.mergeId })
    feedback.success(t('merge.undoSuccess'))
    showUndoConfirm.value = false
    receipt.value = null
    emit('update:show', false)
    emit('undone', result)
  } catch (err) {
    feedback.error(t('merge.undoError'), err instanceof Error ? err.message : String(err))
  } finally {
    undoing.value = false
  }
}
</script>

<template>
  <NModal
    :show="show"
    preset="card"
    :title="receipt ? t('merge.receiptTitle') : t('merge.previewTitle')"
    :style="{ width: 'min(960px, 96vw)' }"
    :mask-closable="!executing && !undoing && !showUndoConfirm"
    :close-on-esc="!executing && !undoing && !showUndoConfirm"
    @update:show="handleVisibility"
  >
    <div v-if="receipt" class="merge-preview__receipt">
      <dl class="merge-preview__receipt-grid">
        <div class="merge-preview__receipt-row">
          <dt>{{ t('merge.receipt.migratedIdentityCount') }}</dt>
          <dd class="tabular-nums">{{ receipt.migratedIdentityCount }}</dd>
        </div>
        <div class="merge-preview__receipt-row">
          <dt>{{ t('merge.receipt.migratedAddressCount') }}</dt>
          <dd class="tabular-nums">{{ receipt.migratedAddressCount }}</dd>
        </div>
        <div class="merge-preview__receipt-row">
          <dt>{{ t('merge.receipt.updatedDemandDocs') }}</dt>
          <dd class="tabular-nums">{{ receipt.updatedDemandDocs }}</dd>
        </div>
      </dl>
      <p v-if="!receipt.undoAvailable" class="merge-preview__undo-unavailable">
        {{ t('merge.undoUnavailable') }}
      </p>
    </div>

    <div v-else-if="loading" class="merge-preview__loading">
      <NSpin size="large" />
    </div>

    <div v-else-if="loadError" class="merge-preview__error">
      <p>{{ loadError }}</p>
      <NButton size="small" @click="retry">{{ t('common.retry') }}</NButton>
    </div>

    <div v-else-if="preview" class="merge-preview">
      <p class="merge-preview__subtitle">{{ t('merge.previewSubtitle') }}</p>
      <p class="merge-preview__summary">
        {{ t('merge.movedSummary', { identityCount: preview.movedIdentityCount, addressCount: preview.movedAddressCount }) }}
      </p>

      <div class="merge-preview__sides">
        <section v-for="entry in sides" :key="entry.key" class="merge-preview__side">
          <h3 class="merge-preview__side-title">{{ t(entry.labelKey) }}</h3>
          <div class="merge-preview__side-card">
            <div class="merge-preview__field" :class="{ 'merge-preview__field--conflict': conflictFields.has('displayName') }">
              <span class="merge-preview__field-label">{{ t('customerDetail.profile.displayNameLabel') }}</span>
              <span class="merge-preview__field-value">{{ entry.side.displayName }}</span>
            </div>
            <div class="merge-preview__field" :class="{ 'merge-preview__field--conflict': conflictFields.has('profileType') }">
              <span class="merge-preview__field-label">{{ t('customerDetail.profile.profileTypeLabel') }}</span>
              <StatusBadge dimension="profileType" :value="entry.side.profileType" size="sm" />
            </div>

            <div class="merge-preview__subsection">
              <h4 class="merge-preview__subsection-title">{{ t('customerDetail.sections.identities') }}</h4>
              <ul v-if="entry.side.identities.length > 0" class="merge-preview__identity-list">
                <li v-for="identity in entry.side.identities" :key="identity.id" class="merge-preview__identity-item">
                  <span class="merge-preview__identity-main">{{ identity.identityPlatform }} · {{ identity.identityValue }}</span>
                  <StatusBadge dimension="identityType" :value="identity.identityType" size="sm" />
                  <span v-if="identity.isPrimary" class="merge-preview__primary-tag">{{ t('customerDetail.identities.isPrimaryLabel') }}</span>
                </li>
              </ul>
              <p v-else class="merge-preview__muted">{{ t('customerDetail.identities.empty') }}</p>
            </div>

            <div class="merge-preview__subsection">
              <h4 class="merge-preview__subsection-title">{{ t('customerDetail.sections.addresses') }}</h4>
              <ul v-if="entry.side.addresses.length > 0" class="merge-preview__address-list">
                <li v-for="address in entry.side.addresses" :key="address.id" class="merge-preview__address-item">
                  <span>{{ address.recipientName }} · {{ address.phone }}</span>
                  <span>{{ address.province }}{{ address.city }}{{ address.district }} {{ address.addressLine1 }}</span>
                  <span v-if="address.isDefault" class="merge-preview__default-tag">{{ t('customerDetail.addresses.defaultBadge') }}</span>
                </li>
              </ul>
              <p v-else class="merge-preview__muted">{{ t('customerDetail.addresses.empty') }}</p>
            </div>
          </div>
        </section>
      </div>

      <section class="merge-preview__conflicts">
        <h3 class="merge-preview__section-title">{{ t('merge.conflictsTitle') }}</h3>
        <ul v-if="preview.conflicts.length > 0" class="merge-preview__conflict-list">
          <li v-for="conflict in preview.conflicts" :key="conflict.field" class="merge-preview__conflict-item">
            <span class="merge-preview__conflict-field">{{ conflictFieldLabel(conflict.field) }}</span>
            <span class="merge-preview__conflict-values">
              <span class="merge-preview__conflict-value merge-preview__conflict-value--source">{{ conflict.sourceValue }}</span>
              <span class="merge-preview__conflict-arrow" aria-hidden="true">→</span>
              <span class="merge-preview__conflict-value merge-preview__conflict-value--target">{{ conflict.targetValue }}</span>
            </span>
          </li>
        </ul>
        <p v-else class="merge-preview__muted">{{ t('merge.noConflicts') }}</p>
      </section>

      <section class="merge-preview__duplicates">
        <h3 class="merge-preview__section-title">{{ t('merge.duplicateIdentitiesTitle') }}</h3>
        <template v-if="preview.duplicateIdentityValues.length > 0">
          <p class="merge-preview__hint">{{ t('merge.duplicateIdentitiesHint') }}</p>
          <ul class="merge-preview__duplicate-list">
            <li v-for="value in preview.duplicateIdentityValues" :key="value" class="merge-preview__duplicate-item">{{ value }}</li>
          </ul>
        </template>
        <p v-else class="merge-preview__muted">{{ t('merge.noDuplicates') }}</p>
      </section>
    </div>

    <template #footer>
      <div class="merge-preview__footer">
        <template v-if="receipt">
          <NButton
            v-if="receipt.undoAvailable"
            type="warning"
            :disabled="undoing"
            @click="showUndoConfirm = true"
          >
            {{ t('merge.undoAction') }}
          </NButton>
          <NButton type="primary" :disabled="undoing" @click="close">{{ t('common.close') }}</NButton>
        </template>
        <template v-else>
          <NButton :disabled="executing" @click="close">{{ t('merge.cancelAction') }}</NButton>
          <NButton type="primary" :loading="executing" :disabled="!preview || executing" @click="handleConfirm">
            {{ executing ? t('merge.executing') : t('merge.confirmAction') }}
          </NButton>
        </template>
      </div>
    </template>
  </NModal>

  <NModal
    :show="showUndoConfirm"
    preset="dialog"
    type="warning"
    :title="t('merge.undoConfirmTitle')"
    :content="t('merge.undoConfirmBody')"
    :positive-text="undoing ? t('merge.undoing') : t('merge.undoAction')"
    :negative-text="t('common.cancel')"
    :loading="undoing"
    :mask-closable="!undoing"
    :close-on-esc="!undoing"
    @positive-click="handleUndo"
    @negative-click="showUndoConfirm = false"
    @update:show="(value: boolean) => { if (!value && !undoing) showUndoConfirm = false }"
  />
</template>

<style scoped>
.merge-preview {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.merge-preview__subtitle {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.merge-preview__summary {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.merge-preview__loading,
.merge-preview__error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-8) 0;
  text-align: center;
}

.merge-preview__error p {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--status-error-fg);
}

.merge-preview__sides {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: var(--space-3);
}

.merge-preview__side {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-width: 0;
}

.merge-preview__side-title {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  text-transform: uppercase;
  letter-spacing: 0.02em;
  color: var(--color-text-muted);
}

.merge-preview__side-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.merge-preview__field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
}

.merge-preview__field--conflict {
  background: var(--status-warning-bg);
  border-color: var(--status-warning-border);
}

.merge-preview__field-label {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.merge-preview__field-value {
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
  text-align: right;
  word-break: break-word;
}

.merge-preview__subsection {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.merge-preview__subsection-title {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-secondary);
}

.merge-preview__identity-list,
.merge-preview__address-list,
.merge-preview__conflict-list,
.merge-preview__duplicate-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin: 0;
  padding: 0;
  list-style: none;
}

.merge-preview__identity-item,
.merge-preview__address-item {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--color-surface-raised, var(--color-surface));
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.merge-preview__identity-main {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.merge-preview__primary-tag,
.merge-preview__default-tag {
  display: inline-flex;
  align-items: center;
  padding: 1px var(--space-2);
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--status-info-fg);
  background: var(--status-info-bg);
  border: 1px solid var(--status-info-border);
}

.merge-preview__muted {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.merge-preview__section-title {
  margin: 0 0 var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.merge-preview__conflict-item {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--status-warning-bg);
  border: 1px solid var(--status-warning-border);
}

.merge-preview__conflict-field {
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-semibold);
  color: var(--status-warning-fg);
}

.merge-preview__conflict-values {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.merge-preview__conflict-arrow {
  color: var(--color-text-muted);
}

.merge-preview__hint {
  margin: 0 0 var(--space-2);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--status-warning-fg);
}

.merge-preview__duplicate-item {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--status-warning-bg);
  border: 1px solid var(--status-warning-border);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  color: var(--status-warning-fg);
}

.merge-preview__receipt-grid {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
}

.merge-preview__receipt-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-surface);
}

.merge-preview__receipt-row dt {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.merge-preview__receipt-row dd {
  margin: 0;
  font-family: var(--font-body);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.merge-preview__undo-unavailable {
  margin: var(--space-3) 0 0;
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.merge-preview__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
