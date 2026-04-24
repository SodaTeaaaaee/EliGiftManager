import {
  Bootstrap as bootstrapBinding,
  GetDashboard as getDashboardBinding,
  ListDispatchRecords as listDispatchRecordsBinding,
  ListMembers as listMembersBinding,
  ListProducts as listProductsBinding,
  ListTemplates as listTemplatesBinding,
  PingDB as pingDatabaseBinding,
  ValidateBatch as validateBatchBinding,
} from '../../../../wailsjs/go/main/App'
import type { BootstrapPayload } from '@/shared/types/app'
import type { main, model } from '../../../../wailsjs/go/models'

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

export function getDashboard(): Promise<main.DashboardPayload> {
  assertWailsRuntime()
  return getDashboardBinding()
}

export function listMembers(): Promise<main.MemberItem[]> {
  assertWailsRuntime()
  return listMembersBinding()
}

export function listProducts(): Promise<main.ProductItem[]> {
  assertWailsRuntime()
  return listProductsBinding()
}

export function listDispatchRecords(): Promise<main.DispatchRecordItem[]> {
  assertWailsRuntime()
  return listDispatchRecordsBinding()
}

export function listTemplates(): Promise<main.TemplateItem[]> {
  assertWailsRuntime()
  return listTemplatesBinding()
}

export type DashboardPayload = main.DashboardPayload
export type MemberItem = main.MemberItem
export type ProductItem = main.ProductItem
export type DispatchRecordItem = main.DispatchRecordItem
export type TemplateItem = main.TemplateItem
export type BatchValidationResult = model.BatchValidationResult
