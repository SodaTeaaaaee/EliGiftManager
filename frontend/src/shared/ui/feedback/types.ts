/**
 * Shared types for the feedback kit (plan 4.3). This is the app's ONLY
 * feedback path — toasts, the undo/action receipt log, page-level error
 * banners, and the global bridge-disconnected banner all live here.
 */

/** The three toast tones the app can raise. Errors are sticky by default. */
export type ToastKind = 'success' | 'error' | 'info'

/** Options accepted by `useFeedback()`'s `error()` call. */
export interface ToastErrorOptions {
  /** Optional copyable technical detail (stack trace, raw error message, request id, ...). */
  detail?: string
}

/** A live toast instance tracked by `FeedbackProvider`. */
export interface ToastRecord {
  id: string
  kind: ToastKind
  message: string
  detail?: string
  /** epoch ms this toast was created — used only for internal timer bookkeeping. */
  createdAt: number
  /** null = sticky (never auto-dismisses; errors default to this). */
  durationMs: number | null
  detailExpanded: boolean
}

/** The three kinds of entries the receipt tray can log (plan 3.3.3 / docs decision #25). */
export type ReceiptKind = 'undo' | 'redo' | 'action'

/** Input accepted by `useFeedback().receipt(...)`. `at` defaults to `Date.now()`. */
export interface ReceiptInput {
  kind: ReceiptKind
  summary: string
  at?: number
}

/** A logged receipt entry, as stored/rendered by `ReceiptTray`. Read-only once created. */
export interface ReceiptEntry {
  id: string
  kind: ReceiptKind
  summary: string
  /** epoch ms. */
  at: number
}

/** The api surface `FeedbackProvider` provides and `useFeedback()` returns. */
export interface FeedbackApi {
  success(message: string): void
  error(message: string, detail?: string): void
  info(message: string): void
  receipt(entry: ReceiptInput): void
}

/** Max simultaneously-visible toasts (plan 4.3). */
export const MAX_VISIBLE_TOASTS = 5

/** Max session-scoped receipt-tray entries (plan 3.3.3 / docs decision #25). */
export const MAX_RECEIPT_ENTRIES = 12

/** Default auto-dismiss duration for success/info toasts. Errors are sticky (`null`). */
export const DEFAULT_TOAST_DURATION_MS = 4600
