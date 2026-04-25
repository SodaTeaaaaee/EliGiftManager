import {
  AutoAllocateWave as autoAllocateWaveBinding,
  BackupDatabase as backupDatabaseBinding,
  Bootstrap as bootstrapBinding,
  CreateTemplate as createTemplateBinding,
  CreateWave as createWaveBinding,
  DeleteWave as deleteWaveBinding,
  ExportWaveByPlatform as exportWaveByPlatformBinding,
  GetDashboard as getDashboardBinding,
  ImportToWave as importToWaveBinding,
  ListDispatchRecords as listDispatchRecordsBinding,
  ListMembers as listMembersBinding,
  ListProducts as listProductsBinding,
  ListTemplates as listTemplatesBinding,
  ListWaves as listWavesBinding,
  PingDB as pingDatabaseBinding,
  RestoreDatabase as restoreDatabaseBinding,
  SetDefaultAddress as setDefaultAddressBinding,
  UpdateProduct as updateProductBinding,
} from '../../../../wailsjs/go/main/App'
import type { BootstrapPayload } from '@/shared/types/app'
import { model } from '../../../../wailsjs/go/models'
import type { main } from '../../../../wailsjs/go/models'

export const WAILS_PREVIEW_MESSAGE = '当前处于浏览器预览模式，Wails 后端尚未连接'

type WindowWithWails = Window & { go?: unknown }
export function isWailsRuntimeAvailable() { return typeof window !== 'undefined' && Boolean((window as WindowWithWails).go) }
function assertWailsRuntime() { if (!isWailsRuntimeAvailable()) throw new Error(WAILS_PREVIEW_MESSAGE) }

export function bootstrapApp(): Promise<BootstrapPayload> { assertWailsRuntime(); return bootstrapBinding() }
export function pingDatabase() { assertWailsRuntime(); return pingDatabaseBinding() }
export function getDashboard(): Promise<main.DashboardPayload> { assertWailsRuntime(); return getDashboardBinding() }
export function createWave(name: string): Promise<model.Wave> { assertWailsRuntime(); return createWaveBinding(name) }
export function deleteWave(waveId: number): Promise<void> { assertWailsRuntime(); return deleteWaveBinding(waveId) }
export function listWaves(): Promise<main.WaveItem[]> { assertWailsRuntime(); return listWavesBinding() }
export function importToWave(waveId: number, csvPath: string, templateId: number): Promise<void> { assertWailsRuntime(); return importToWaveBinding(waveId, csvPath, templateId) }
export function autoAllocateWave(waveId: number, mappingTemplateId: number): Promise<void> { assertWailsRuntime(); return autoAllocateWaveBinding(waveId, mappingTemplateId) }
export function exportWaveByPlatform(waveId: number, platform: string, exportTemplateId: number): Promise<string> { assertWailsRuntime(); return exportWaveByPlatformBinding(waveId, platform, exportTemplateId) }
export function listMembers(page = 1, pageSize = 50, keyword = '', platform = ''): Promise<main.MemberListPayload> { assertWailsRuntime(); return listMembersBinding(page, pageSize, keyword, platform) }
export function listProducts(page = 1, pageSize = 50, keyword = '', platform = ''): Promise<main.ProductListPayload> { assertWailsRuntime(); return listProductsBinding(page, pageSize, keyword, platform) }
export function listDispatchRecords(waveId = 0): Promise<main.DispatchRecordItem[]> { assertWailsRuntime(); return listDispatchRecordsBinding(waveId) }
export function createTemplate(platform: string, templateType: string, name: string, mappingRules: string): Promise<main.TemplateItem> { assertWailsRuntime(); return createTemplateBinding(platform, templateType, name, mappingRules) }
export function listTemplates(): Promise<main.TemplateItem[]> { assertWailsRuntime(); return listTemplatesBinding() }
export function setDefaultAddress(memberId: number, addressId: number): Promise<void> { assertWailsRuntime(); return setDefaultAddressBinding(memberId, addressId) }
export type ProductUpdateInput = {
  id: number
  platform: string
  factory: string
  factorySku: string
  name: string
  coverImage: string
  extraData: string
}

export function updateProduct(product: ProductUpdateInput): Promise<void> {
  assertWailsRuntime()
  return updateProductBinding(model.Product.createFrom(product))
}
export function backupDatabase(): Promise<string> { assertWailsRuntime(); return backupDatabaseBinding() }
export function restoreDatabase(): Promise<void> { assertWailsRuntime(); return restoreDatabaseBinding() }

export type DashboardPayload = main.DashboardPayload
export type WaveItem = main.WaveItem
export type MemberItem = main.MemberItem
export type MemberListPayload = main.MemberListPayload
export type ProductItem = main.ProductItem
export type ProductListPayload = main.ProductListPayload
export type DispatchRecordItem = main.DispatchRecordItem
export type TemplateItem = main.TemplateItem
