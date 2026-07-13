<script setup lang="ts">
/**
 * FeedbackProvider — mounts the app's ONLY feedback path (plan 4.3): the
 * toast viewport and the receipt-tray host. Provides `FeedbackApi` (see
 * `context.ts` / `useFeedback()`) to the whole subtree via `<slot />`.
 *
 * Mount once, near the app root (inside the router view is fine — it just
 * needs to wrap everything that might call `useFeedback()`), e.g.:
 *   <FeedbackProvider>
 *     <RouterView />
 *   </FeedbackProvider>
 */
import { onBeforeUnmount, provide, reactive } from 'vue'
import { FEEDBACK_INJECTION_KEY } from './context'
import ReceiptTray from './ReceiptTray.vue'
import ToastViewport from './ToastViewport.vue'
import {
  DEFAULT_TOAST_DURATION_MS,
  MAX_RECEIPT_ENTRIES,
  MAX_VISIBLE_TOASTS,
  type FeedbackApi,
  type ReceiptEntry,
  type ReceiptInput,
  type ToastKind,
  type ToastRecord,
} from './types'

let idCounter = 0
function makeId(prefix: string): string {
  idCounter += 1
  return `${prefix}-${Date.now().toString(36)}-${idCounter}`
}

const toasts = reactive<ToastRecord[]>([])
const receipts = reactive<ReceiptEntry[]>([])

// Per-toast auto-dismiss bookkeeping. `timers` holds the live setTimeout
// handle; `startedAt`/`remaining` support pause-on-hover by letting us
// compute and re-schedule the leftover duration instead of just resetting
// the full window on every hover-out.
const timers = new Map<string, ReturnType<typeof setTimeout>>()
const startedAt = new Map<string, number>()
const remaining = new Map<string, number>()

function clearTimer(id: string): void {
  const timer = timers.get(id)
  if (timer !== undefined) {
    clearTimeout(timer)
    timers.delete(id)
  }
}

function scheduleDismiss(id: string, durationMs: number): void {
  clearTimer(id)
  startedAt.set(id, Date.now())
  timers.set(
    id,
    setTimeout(() => dismiss(id), durationMs),
  )
}

function dismiss(id: string): void {
  clearTimer(id)
  startedAt.delete(id)
  remaining.delete(id)
  const index = toasts.findIndex((toast) => toast.id === id)
  if (index !== -1) toasts.splice(index, 1)
}

function pushToast(kind: ToastKind, message: string, detail?: string): void {
  const id = makeId('toast')
  // Errors are sticky (never auto-dismiss) by default; success/info auto-dismiss.
  const durationMs = kind === 'error' ? null : DEFAULT_TOAST_DURATION_MS
  toasts.push({
    id,
    kind,
    message,
    detail,
    createdAt: Date.now(),
    durationMs,
    detailExpanded: false,
  })

  // Enforce "max 5 visible" by dropping the oldest toast(s) beyond the cap.
  const overflow = toasts.length - MAX_VISIBLE_TOASTS
  for (let i = 0; i < overflow; i += 1) {
    dismiss(toasts[0].id)
  }

  if (durationMs !== null) scheduleDismiss(id, durationMs)
}

function toggleDetail(id: string): void {
  const toast = toasts.find((item) => item.id === id)
  if (toast) toast.detailExpanded = !toast.detailExpanded
}

function pauseTimer(id: string): void {
  const started = startedAt.get(id)
  const toast = toasts.find((item) => item.id === id)
  if (started === undefined || !toast || toast.durationMs === null) return
  const elapsed = Date.now() - started
  remaining.set(id, Math.max(0, toast.durationMs - elapsed))
  clearTimer(id)
}

function resumeTimer(id: string): void {
  const left = remaining.get(id)
  if (left === undefined) return
  remaining.delete(id)
  scheduleDismiss(id, left)
}

function addReceipt(input: ReceiptInput): void {
  receipts.unshift({
    id: makeId('receipt'),
    kind: input.kind,
    summary: input.summary,
    at: input.at ?? Date.now(),
  })
  if (receipts.length > MAX_RECEIPT_ENTRIES) {
    receipts.length = MAX_RECEIPT_ENTRIES
  }
}

const api: FeedbackApi = {
  success(message) {
    pushToast('success', message)
  },
  error(message, detail) {
    pushToast('error', message, detail)
  },
  info(message) {
    pushToast('info', message)
  },
  receipt(entry) {
    addReceipt(entry)
  },
}

provide(FEEDBACK_INJECTION_KEY, api)

onBeforeUnmount(() => {
  for (const timer of timers.values()) clearTimeout(timer)
  timers.clear()
})
</script>

<template>
  <slot />
  <ToastViewport
    :toasts="toasts"
    @dismiss="dismiss"
    @toggle-detail="toggleDetail"
    @pointer-enter="pauseTimer"
    @pointer-leave="resumeTimer"
  />
  <ReceiptTray :receipts="receipts" />
</template>
