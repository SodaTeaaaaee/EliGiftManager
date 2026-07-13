<script setup lang="ts">
/**
 * FilterBar — renders a schema-driven combination filter bar (plan 3.3.2):
 * one dropdown per enum-multi dimension (options/labels/tones from
 * `useGlossary()`), one debounced keyword input per keyword field, an
 * active-filter chip row with per-chip clear + clear-all, and a slot for the
 * caller's own result count. Pair with `useUrlFilters(schema)` — pass its
 * return value straight through as the `filters` prop.
 *
 * Deliberately not a generic Vue SFC — see the `FilterBarFilters` doc comment
 * in `./types.ts` for why the plain, erased-key surface is both sufficient
 * and safer here.
 */
import { computed, onBeforeMount, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { NInput, NPopover } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { glossaryTables, useGlossary, type GlossaryDimension, type StatusTone } from '@/shared/i18n/glossary'
import { registerFilterFocusTarget } from '@/shared/lib/view-hotkeys'
import type { ActiveFilterChip, FilterBarFilters } from './types'

const props = withDefaults(
  defineProps<{
    filters: FilterBarFilters
    /** Debounce (ms) between a keystroke and committing the keyword field to the URL. Default 300. */
    keywordDebounceMs?: number
  }>(),
  {
    keywordDebounceMs: 300,
  },
)

const emit = defineEmits<{
  /**
   * Fires once per committed change (enum toggle, debounced keyword commit,
   * chip clear, clear-all). `filters.state` is already reactive, so this is
   * only useful to callers who prefer an imperative hook (e.g. to trigger a
   * server-side refetch) over watching state directly.
   */
  update: []
}>()

const { t } = useI18n()
const { label, tone } = useGlossary()

interface EnumOptionView {
  value: string
  label: string
  tone: StatusTone
}

/** Every field's currently-active values, read loosely (mirrors useUrlFilters' own internal casting style for the same reason: schema-generic code can't keep literal value types through a runtime loop). */
const rawState = computed(() => props.filters.state as Record<string, string[] | string>)

// `filters.isActive`/`filters.activeChips` are raw ComputedRefs on a plain
// (non-`reactive()`) object — Vue only auto-unwraps refs that are TOP-LEVEL
// template bindings, not nested property access like `filters.isActive` in a
// child's template. Re-expose them as fresh, locally top-level computeds so
// the template can read them directly.
const isActive = computed(() => props.filters.isActive.value)
const activeChips = computed(() => props.filters.activeChips.value)

const optionsByKey = computed<Record<string, EnumOptionView[]>>(() => {
  const map: Record<string, EnumOptionView[]> = {}
  for (const field of props.filters.schema) {
    if (field.type !== 'enum-multi') continue
    const values = field.values ?? (Object.keys(glossaryTables[field.dimension]) as string[])
    map[field.key] = values.map((value) => ({
      value,
      label: label(field.dimension, value),
      tone: tone(field.dimension, value),
    }))
  }
  return map
})

function dimensionLabel(dimension: GlossaryDimension): string {
  return t(`statusKit.dimensionNames.${dimension}`)
}

function selectedValues(key: string): string[] {
  const value = rawState.value[key]
  return Array.isArray(value) ? value : []
}

function isChecked(key: string, value: string): boolean {
  return selectedValues(key).includes(value)
}

function toggleValue(key: string, value: string): void {
  props.filters.toggleEnumValue(key, value)
  emit('update')
}

// --- keyword fields: a local draft per field, committed to the URL only after keywordDebounceMs ---
const keywordDrafts = reactive<Record<string, string>>({})
const pendingTimers: Record<string, ReturnType<typeof setTimeout>> = {}

for (const field of props.filters.schema) {
  if (field.type === 'keyword') {
    const initial = rawState.value[field.key]
    keywordDrafts[field.key] = typeof initial === 'string' ? initial : ''
  }
}

// --- Ctrl+F hotkey: register this bar's first keyword input so the global
// handler (shared/lib/view-hotkeys.ts) can focus it. Only registers when
// the schema actually has a keyword field.
const keywordInputRefs = ref<Record<string, { focus?: () => void } | null>>({})

function setKeywordInputRef(key: string, el: unknown): void {
  keywordInputRefs.value[key] = el as { focus?: () => void } | null
}

const firstKeywordFieldKey = computed(() => props.filters.schema.find((field) => field.type === 'keyword')?.key)

function focusKeyword(): void {
  const key = firstKeywordFieldKey.value
  if (!key) return
  keywordInputRefs.value[key]?.focus?.()
}

let unregisterFilterFocus: (() => void) | undefined
onBeforeMount(() => {
  if (firstKeywordFieldKey.value) unregisterFilterFocus = registerFilterFocusTarget(focusKeyword)
})

// Keep drafts in sync with state changes that didn't originate from this
// component's own debounce commit (browser back/forward, a saved view being
// applied, another FilterBar instance on the same schema, ...). Skips fields
// the user is actively mid-typing so we never clobber their keystrokes.
watch(
  () => props.filters.schema.map((field) => (field.type === 'keyword' ? rawState.value[field.key] : undefined)),
  () => {
    for (const field of props.filters.schema) {
      if (field.type !== 'keyword') continue
      if (pendingTimers[field.key] !== undefined) continue
      const committed = rawState.value[field.key]
      if (typeof committed === 'string' && keywordDrafts[field.key] !== committed) {
        keywordDrafts[field.key] = committed
      }
    }
  },
)

function onKeywordInput(key: string, value: string): void {
  keywordDrafts[key] = value
  if (pendingTimers[key] !== undefined) clearTimeout(pendingTimers[key])
  pendingTimers[key] = setTimeout(() => {
    delete pendingTimers[key]
    props.filters.setKeyword(key, value)
    emit('update')
  }, props.keywordDebounceMs)
}

function chipToneClass(chip: ActiveFilterChip): string {
  if (chip.type !== 'enum-multi' || !chip.dimension) return 'filter-bar__chip--neutral'
  return `filter-bar__chip--${tone(chip.dimension, chip.value)}`
}

function chipLabel(chip: ActiveFilterChip): string {
  if (chip.type === 'enum-multi' && chip.dimension) {
    return `${dimensionLabel(chip.dimension)}: ${label(chip.dimension, chip.value)}`
  }
  return t('filterBar.keywordChipLabel', { value: chip.value })
}

function clearChip(chip: ActiveFilterChip): void {
  if (chip.type === 'enum-multi') {
    props.filters.toggleEnumValue(chip.fieldKey, chip.value)
  } else {
    props.filters.clearField(chip.fieldKey)
    keywordDrafts[chip.fieldKey] = ''
  }
  emit('update')
}

function handleClearAll(): void {
  props.filters.clearAll()
  for (const key of Object.keys(keywordDrafts)) keywordDrafts[key] = ''
  emit('update')
}

onBeforeUnmount(() => {
  unregisterFilterFocus?.()
  for (const timer of Object.values(pendingTimers)) clearTimeout(timer)
})
</script>

<template>
  <div class="filter-bar">
    <div class="filter-bar__controls">
      <template v-for="field in filters.schema" :key="field.key">
        <NPopover v-if="field.type === 'enum-multi'" trigger="click" placement="bottom-start" :show-arrow="false" class="filter-bar__popover">
          <template #trigger>
            <button
              type="button"
              class="filter-bar__trigger"
              :class="{ 'filter-bar__trigger--active': selectedValues(field.key).length > 0 }"
            >
              <span class="filter-bar__trigger-label">{{ dimensionLabel(field.dimension) }}</span>
              <span v-if="selectedValues(field.key).length > 0" class="filter-bar__trigger-count tabular-nums">
                {{ selectedValues(field.key).length }}
              </span>
              <svg class="filter-bar__trigger-caret" viewBox="0 0 10 6" width="10" height="6" aria-hidden="true">
                <path d="M1 1 L5 5 L9 1" stroke="currentColor" stroke-width="1.5" fill="none" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </button>
          </template>
          <div
            class="filter-bar__menu"
            role="group"
            :aria-label="t('filterBar.optionsMenuLabel', { dimension: dimensionLabel(field.dimension) })"
          >
            <label v-for="opt in optionsByKey[field.key]" :key="opt.value" class="filter-bar__option">
              <input
                type="checkbox"
                class="filter-bar__option-input"
                :checked="isChecked(field.key, opt.value)"
                @change="toggleValue(field.key, opt.value)"
              />
              <span class="filter-bar__option-dot" :style="{ background: `var(--status-${opt.tone}-fg)` }" aria-hidden="true" />
              <span class="filter-bar__option-label">{{ opt.label }}</span>
            </label>
          </div>
        </NPopover>

        <NInput
          v-else
          :ref="(el: unknown) => setKeywordInputRef(field.key, el)"
          class="filter-bar__keyword"
          :value="keywordDrafts[field.key]"
          clearable
          size="medium"
          :placeholder="t('filterBar.keywordPlaceholder')"
          @update:value="(value: string) => onKeywordInput(field.key, value)"
        >
          <template #prefix>
            <svg viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
              <circle cx="7" cy="7" r="5" fill="none" stroke="currentColor" stroke-width="1.4" />
              <path d="M11 11 L14.5 14.5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
            </svg>
          </template>
        </NInput>
      </template>

      <button type="button" class="filter-bar__clear-all" :disabled="!isActive" @click="handleClearAll">
        {{ t('filterBar.clearAll') }}
      </button>

      <div v-if="$slots['result-count']" class="filter-bar__result-count">
        <slot name="result-count" />
      </div>
    </div>

    <div v-if="activeChips.length > 0" class="filter-bar__chips" role="list" :aria-label="t('filterBar.activeFilters')">
      <button
        v-for="chip in activeChips"
        :key="chip.id"
        type="button"
        class="filter-bar__chip"
        :class="chipToneClass(chip)"
        role="listitem"
        :aria-label="t('filterBar.removeFilter', { label: chipLabel(chip) })"
        @click="clearChip(chip)"
      >
        <span class="filter-bar__chip-label">{{ chipLabel(chip) }}</span>
        <svg class="filter-bar__chip-remove" viewBox="0 0 12 12" width="10" height="10" aria-hidden="true">
          <path d="M2 2 L10 10 M10 2 L2 10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
        </svg>
      </button>
    </div>
    <p v-else class="filter-bar__empty">{{ t('filterBar.noActiveFilters') }}</p>
  </div>
</template>

<style scoped>
.filter-bar {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  font-family: var(--font-body);
}

.filter-bar__controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}

.filter-bar__trigger {
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
    color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}

.filter-bar__trigger:hover {
  border-color: var(--color-border-strong);
  color: var(--color-text-primary);
}

.filter-bar__trigger:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.filter-bar__trigger--active {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: var(--color-accent-subtle);
}

.filter-bar__trigger-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.25em;
  padding: 0 4px;
  border-radius: var(--radius-full);
  background: var(--color-accent);
  color: var(--color-on-accent);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  line-height: 1.25em;
}

.filter-bar__trigger-caret {
  flex-shrink: 0;
  opacity: 0.7;
}

.filter-bar__menu {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 200px;
  max-height: 320px;
  overflow-y: auto;
}

.filter-bar__option {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}

.filter-bar__option:hover {
  background: var(--color-inset);
}

.filter-bar__option-input {
  accent-color: var(--color-accent);
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  cursor: pointer;
}

.filter-bar__option-dot {
  width: var(--statusbadge-dot-size);
  height: var(--statusbadge-dot-size);
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.filter-bar__option-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  white-space: nowrap;
}

.filter-bar__keyword {
  width: 220px;
}

.filter-bar__clear-all {
  height: var(--control-height);
  padding: 0 var(--space-2);
  border: none;
  background: transparent;
  color: var(--color-text-muted);
  font-family: var(--font-body);
  font-size: var(--font-size-sm);
  cursor: pointer;
  border-radius: var(--control-radius);
  transition:
    color var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out);
}

.filter-bar__clear-all:hover:not(:disabled) {
  color: var(--color-text-primary);
  background: var(--color-inset);
}

.filter-bar__clear-all:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.filter-bar__clear-all:disabled {
  opacity: 0.5;
  cursor: default;
}

.filter-bar__result-count {
  margin-left: auto;
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  white-space: nowrap;
}

.filter-bar__chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.filter-bar__chip {
  display: inline-flex;
  align-items: center;
  gap: var(--statusbadge-gap);
  padding: var(--statusbadge-padding-y) var(--statusbadge-padding-x);
  border-radius: var(--statusbadge-radius);
  border: 1px solid transparent;
  font-family: var(--font-body);
  font-size: var(--statusbadge-font-size);
  font-weight: var(--statusbadge-font-weight);
  cursor: pointer;
  transition:
    filter var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out);
}

.filter-bar__chip:hover {
  filter: brightness(0.97);
}

.filter-bar__chip:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.filter-bar__chip-remove {
  flex-shrink: 0;
  opacity: 0.75;
}

.filter-bar__chip--success {
  color: var(--status-success-fg);
  background: var(--status-success-bg);
  border-color: var(--status-success-border);
}
.filter-bar__chip--warning {
  color: var(--status-warning-fg);
  background: var(--status-warning-bg);
  border-color: var(--status-warning-border);
}
.filter-bar__chip--error {
  color: var(--status-error-fg);
  background: var(--status-error-bg);
  border-color: var(--status-error-border);
}
.filter-bar__chip--info {
  color: var(--status-info-fg);
  background: var(--status-info-bg);
  border-color: var(--status-info-border);
}
.filter-bar__chip--progress {
  color: var(--status-progress-fg);
  background: var(--status-progress-bg);
  border-color: var(--status-progress-border);
}
.filter-bar__chip--neutral {
  color: var(--status-neutral-fg);
  background: var(--status-neutral-bg);
  border-color: var(--status-neutral-border);
}

.filter-bar__empty {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>
