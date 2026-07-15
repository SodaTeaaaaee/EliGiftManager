<script setup lang="ts">
/**
 * ReceiptTray — the undo/action receipt log (plan 3.3.3 / docs decision #25).
 * A small fixed tray button (bottom-right, distinct from the top-right toast
 * viewport) that opens a read-only popover listing the session-scoped log,
 * newest first, capped at `MAX_RECEIPT_ENTRIES` by the provider.
 */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ReceiptEntry } from './types'
import { useRelativeTime } from './useRelativeTime'

const props = defineProps<{
  receipts: ReceiptEntry[]
}>()

const { t } = useI18n()
const { format } = useRelativeTime()

const open = ref(false)
const rootEl = ref<HTMLElement | null>(null)

function toggle(): void {
  open.value = !open.value
}

function close(): void {
  open.value = false
}

function onDocumentPointerDown(event: PointerEvent): void {
  if (!open.value) return
  if (rootEl.value && !rootEl.value.contains(event.target as Node)) close()
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') close()
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocumentPointerDown)
  document.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  document.removeEventListener('keydown', onKeydown)
})

function kindLabel(kind: ReceiptEntry['kind']): string {
  return t(`feedback.receiptTray.kinds.${kind}`)
}
</script>

<template>
  <div ref="rootEl" class="receipt-tray">
    <button
      type="button"
      class="receipt-tray__trigger"
      :class="{ 'receipt-tray__trigger--open': open }"
      :aria-expanded="open"
      :aria-label="t('feedback.receiptTray.buttonLabel')"
      @click="toggle"
    >
      <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <path d="M8 4H6a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2v-2" />
        <path d="M9 3.5h6a1 1 0 0 1 1 1V6a1 1 0 0 1-1 1H9a1 1 0 0 1-1-1V4.5a1 1 0 0 1 1-1Z" />
        <path d="M7 11h6M7 15h9" />
      </svg>
      <span v-if="props.receipts.length" class="receipt-tray__count tabular-nums">{{ props.receipts.length }}</span>
    </button>

    <Transition name="receipt-tray-pop">
      <div
        v-if="open"
        class="receipt-tray__panel"
        role="dialog"
        :aria-label="t('feedback.receiptTray.title')"
      >
        <header class="receipt-tray__header">
          <h3 class="receipt-tray__title">{{ t('feedback.receiptTray.title') }}</h3>
        </header>

        <p v-if="!props.receipts.length" class="receipt-tray__empty">
          {{ t('feedback.receiptTray.empty') }}
        </p>

        <ul v-else class="receipt-tray__list">
          <li v-for="receiptEntry in props.receipts" :key="receiptEntry.id" class="receipt-tray__item">
            <span class="receipt-tray__kind" :class="`receipt-tray__kind--${receiptEntry.kind}`">
              {{ kindLabel(receiptEntry.kind) }}
            </span>
            <span class="receipt-tray__summary">{{ receiptEntry.summary }}</span>
            <span class="receipt-tray__time tabular-nums">{{ format(receiptEntry.at) }}</span>
          </li>
        </ul>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.receipt-tray {
  position: fixed;
  right: var(--space-5);
  bottom: var(--space-5);
  z-index: var(--z-popover);
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.receipt-tray__trigger {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border);
  background: var(--color-surface-raised);
  color: var(--color-text-secondary);
  box-shadow: var(--shadow-md);
  cursor: pointer;
  transition:
    color var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out),
    transform var(--duration-fast) var(--ease-out);
}

.receipt-tray__trigger:hover {
  color: var(--color-accent);
  border-color: var(--color-accent);
}

.receipt-tray__trigger--open {
  color: var(--color-accent);
  border-color: var(--color-accent);
  transform: scale(0.96);
}

.receipt-tray__trigger:focus-visible {
  outline: var(--focus-ring-width) solid var(--focus-ring-color);
  outline-offset: var(--focus-ring-offset);
}

.receipt-tray__count {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border-radius: var(--radius-full);
  background: var(--color-accent);
  color: var(--color-on-accent);
  font-size: 10px;
  font-weight: var(--font-weight-semibold);
  line-height: 18px;
  text-align: center;
}

.receipt-tray__panel {
  position: absolute;
  right: 0;
  bottom: calc(44px + var(--space-3));
  width: 340px;
  max-height: 380px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border-radius: var(--card-radius);
  border: 1px solid var(--color-border);
  background: var(--color-surface-raised);
  box-shadow: var(--shadow-lg);
  font-family: var(--font-body);
}

.receipt-tray__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.receipt-tray__title {
  margin: 0;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.receipt-tray__empty {
  margin: var(--space-2) 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.receipt-tray__list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.receipt-tray__item {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: baseline;
  gap: var(--space-2);
  padding: var(--space-2);
  border-radius: var(--radius-sm);
  border-bottom: 1px solid var(--color-border);
}

.receipt-tray__item:last-child {
  border-bottom: none;
}

.receipt-tray__kind {
  flex-shrink: 0;
  padding: 1px var(--space-2);
  border-radius: var(--radius-full);
  font-size: 10px;
  font-weight: var(--font-weight-medium);
  white-space: nowrap;
}

.receipt-tray__kind--undo {
  color: var(--status-info-fg);
  background: var(--status-info-bg);
}
.receipt-tray__kind--redo {
  color: var(--status-progress-fg);
  background: var(--status-progress-bg);
}
.receipt-tray__kind--action {
  color: var(--status-neutral-fg);
  background: var(--status-neutral-bg);
}

.receipt-tray__summary {
  min-width: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-primary);
  line-height: var(--line-height-normal);
  word-break: break-word;
}

.receipt-tray__time {
  flex-shrink: 0;
  font-size: 10px;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.receipt-tray-pop-enter-active,
.receipt-tray-pop-leave-active {
  transition:
    opacity var(--duration-base) var(--ease-out),
    transform var(--duration-base) var(--ease-out);
}
.receipt-tray-pop-enter-from,
.receipt-tray-pop-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.98);
}
</style>
