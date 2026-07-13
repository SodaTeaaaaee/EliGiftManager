<script setup lang="ts">
/**
 * SavedViews — named saved filter-sets for a `FilterBar`/`useUrlFilters`
 * instance. Two groups: caller-supplied, non-deletable preset views (plan
 * 3.3.2's 阻塞项/可提交 pattern) and user-saved views persisted to
 * `localStorage['eligiftmanager:saved-views:<scopeId>']`.
 *
 * Pair with the same `filters` object passed to `FilterBar`:
 * `<SavedViews :filters="filters" scope-id="fulfillment-grid" :presets="presets" />`.
 */
import { ref, watch } from 'vue'
import { NInput, NPopconfirm, NPopover } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import type { FilterBarFilters, FilterSnapshot, FilterViewPreset, SavedFilterView } from './types'

const props = withDefaults(
  defineProps<{
    filters: FilterBarFilters
    /** localStorage scope id -> persisted under `eligiftmanager:saved-views:<scopeId>`. */
    scopeId: string
    /** Built-in, non-deletable views (already-i18n'd `label`). */
    presets?: FilterViewPreset[]
  }>(),
  {
    presets: () => [],
  },
)

const emit = defineEmits<{
  /** Fires after a preset or saved view is applied (preset or user view). */
  apply: []
}>()

const { t } = useI18n()

const STORAGE_PREFIX = 'eligiftmanager:saved-views:'

function storageKey(scopeId: string): string {
  return `${STORAGE_PREFIX}${scopeId}`
}

function readStorage(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    // localStorage can throw in locked-down / privacy-mode contexts —
    // fall back to an empty list rather than crash the bar.
    return null
  }
}

function writeStorage(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    // best-effort persistence only
  }
}

function isSavedFilterView(value: unknown): value is SavedFilterView {
  if (typeof value !== 'object' || value === null) return false
  const candidate = value as Partial<SavedFilterView>
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.name === 'string' &&
    typeof candidate.createdAt === 'number' &&
    typeof candidate.snapshot === 'object' &&
    candidate.snapshot !== null
  )
}

function loadSavedViews(scopeId: string): SavedFilterView[] {
  const raw = readStorage(storageKey(scopeId))
  if (!raw) return []
  try {
    const parsed: unknown = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed.filter(isSavedFilterView) : []
  } catch {
    return []
  }
}

function generateId(): string {
  try {
    return crypto.randomUUID()
  } catch {
    return `view-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  }
}

const savedViews = ref<SavedFilterView[]>(loadSavedViews(props.scopeId))

watch(
  () => props.scopeId,
  (scopeId) => {
    savedViews.value = loadSavedViews(scopeId)
  },
)

watch(
  savedViews,
  (views) => writeStorage(storageKey(props.scopeId), JSON.stringify(views)),
  { deep: true },
)

const showSaveInput = ref(false)
const draftName = ref('')

function openSaveDraft(): void {
  draftName.value = ''
  showSaveInput.value = true
}

function cancelSave(): void {
  showSaveInput.value = false
  draftName.value = ''
}

function commitSave(): void {
  const name = draftName.value.trim()
  if (!name) return
  savedViews.value = [
    ...savedViews.value,
    { id: generateId(), name, snapshot: props.filters.getSnapshot(), createdAt: Date.now() },
  ]
  showSaveInput.value = false
  draftName.value = ''
}

function removeView(id: string): void {
  savedViews.value = savedViews.value.filter((view) => view.id !== id)
}

function applyView(snapshot: FilterSnapshot): void {
  props.filters.applySnapshot(snapshot)
  emit('apply')
}
</script>

<template>
  <NPopover trigger="click" placement="bottom-start" :show-arrow="false" class="saved-views__popover">
    <template #trigger>
      <button type="button" class="saved-views__trigger">
        <svg class="saved-views__trigger-icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
          <path d="M3 3h10v10l-5-3-5 3z" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round" />
        </svg>
        <span>{{ t('filterBar.savedViews.title') }}</span>
      </button>
    </template>

    <div class="saved-views__menu">
      <section v-if="presets.length > 0" class="saved-views__group">
        <p class="saved-views__group-title">{{ t('filterBar.savedViews.presetsGroup') }}</p>
        <button
          v-for="preset in presets"
          :key="preset.id"
          type="button"
          class="saved-views__item"
          @click="applyView(preset.snapshot)"
        >
          <span class="saved-views__item-label">{{ preset.label }}</span>
        </button>
      </section>

      <section class="saved-views__group">
        <p class="saved-views__group-title">{{ t('filterBar.savedViews.savedGroup') }}</p>
        <p v-if="savedViews.length === 0" class="saved-views__empty">{{ t('filterBar.savedViews.empty') }}</p>
        <div v-for="view in savedViews" :key="view.id" class="saved-views__row">
          <button type="button" class="saved-views__item saved-views__item--saved" @click="applyView(view.snapshot)">
            <span class="saved-views__item-label">{{ view.name }}</span>
          </button>
          <NPopconfirm
            :positive-text="t('common.confirm')"
            :negative-text="t('common.cancel')"
            @positive-click="removeView(view.id)"
          >
            <template #trigger>
              <button type="button" class="saved-views__delete" :aria-label="t('filterBar.savedViews.delete')">
                <svg viewBox="0 0 14 14" width="12" height="12" aria-hidden="true">
                  <path
                    d="M2 4h10M5.5 4V2.5h3V4M3.5 4l.6 8h5.8l.6-8"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </button>
            </template>
            {{ t('filterBar.savedViews.deleteConfirmContent') }}
          </NPopconfirm>
        </div>
      </section>

      <section class="saved-views__group saved-views__group--save">
        <button v-if="!showSaveInput" type="button" class="saved-views__save-trigger" @click="openSaveDraft">
          + {{ t('filterBar.savedViews.saveCurrent') }}
        </button>
        <form v-else class="saved-views__save-form" @submit.prevent="commitSave">
          <NInput v-model:value="draftName" size="small" :placeholder="t('filterBar.savedViews.namePlaceholder')" autofocus />
          <div class="saved-views__save-actions">
            <button type="submit" class="saved-views__save-confirm" :disabled="!draftName.trim()">
              {{ t('filterBar.savedViews.save') }}
            </button>
            <button type="button" class="saved-views__save-cancel" @click="cancelSave">
              {{ t('filterBar.savedViews.cancel') }}
            </button>
          </div>
        </form>
      </section>
    </div>
  </NPopover>
</template>

<style scoped>
.saved-views__trigger {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  height: var(--control-height);
  padding: 0 var(--control-padding-x);
  border-radius: var(--control-radius);
  border: 1px solid var(--control-border-color);
  background: var(--control-bg);
  color: var(--color-text-secondary);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition:
    border-color var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}

.saved-views__trigger:hover {
  border-color: var(--color-border-strong);
  color: var(--color-text-primary);
}

.saved-views__trigger:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.saved-views__menu {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-width: 220px;
  max-width: 280px;
}

.saved-views__group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.saved-views__group + .saved-views__group {
  padding-top: var(--space-2);
  border-top: 1px solid var(--color-border);
}

.saved-views__group-title {
  margin: 0 0 2px;
  padding: 0 var(--space-2);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
}

.saved-views__item {
  display: flex;
  align-items: center;
  width: 100%;
  padding: var(--space-1) var(--space-2);
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-primary);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  text-align: left;
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}

.saved-views__item:hover {
  background: var(--color-inset);
}

.saved-views__item-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.saved-views__row {
  display: flex;
  align-items: center;
  gap: 2px;
}

.saved-views__row .saved-views__item {
  flex: 1;
  min-width: 0;
}

.saved-views__delete {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex-shrink: 0;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
}

.saved-views__delete:hover {
  background: var(--status-error-bg);
  color: var(--status-error-fg);
}

.saved-views__empty {
  margin: 0;
  padding: var(--space-1) var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.saved-views__save-trigger {
  padding: var(--space-1) var(--space-2);
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-accent);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  text-align: left;
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}

.saved-views__save-trigger:hover {
  background: var(--color-accent-subtle);
}

.saved-views__save-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: 0 var(--space-1);
}

.saved-views__save-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
}

.saved-views__save-confirm {
  padding: 4px var(--space-3);
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-accent);
  color: var(--color-on-accent);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-out),
    opacity var(--duration-fast) var(--ease-out);
}

.saved-views__save-confirm:hover:not(:disabled) {
  background: var(--color-accent-hover);
}

.saved-views__save-confirm:disabled {
  opacity: 0.5;
  cursor: default;
}

.saved-views__save-cancel {
  padding: 4px var(--space-3);
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-muted);
  font-family: var(--font-body);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}

.saved-views__save-cancel:hover {
  background: var(--color-inset);
}
</style>
