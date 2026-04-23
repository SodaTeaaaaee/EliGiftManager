import {
  Bootstrap as bootstrapBinding,
  PingDB as pingDatabaseBinding,
  ValidateBatch as validateBatchBinding,
} from '../../../../wailsjs/go/main/App'
import type { BootstrapPayload } from '@/shared/types/app'

export const WAILS_PREVIEW_MESSAGE = '当前处于浏览器预览模式，Wails 后端尚未连接'

type WindowWithWails = Window & {
  go?: unknown
}

export function isWailsRuntimeAvailable() {
  return typeof window !== 'undefined' && Boolean((window as WindowWithWails).go)
}

function assertWailsRuntime() {
  if (!isWailsRuntimeAvailable()) {
    throw new Error(WAILS_PREVIEW_MESSAGE)
  }
}

export function bootstrapApp(): Promise<BootstrapPayload> {
  assertWailsRuntime()
  return bootstrapBinding()
}

export function pingDatabase() {
  assertWailsRuntime()
  return pingDatabaseBinding()
}

export function validateBatch(batchName: string) {
  assertWailsRuntime()
  return validateBatchBinding(batchName)
}
