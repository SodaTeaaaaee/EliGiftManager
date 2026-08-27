// Bridge: strong-typed thin wrappers over generated Wails bindings.
// Never import from "wailsjs" directly outside this file.

import {
  AssignLines as _AssignLines,
  AttachIdentity as _AttachIdentity,
  CloseWave as _CloseWave,
  CreateAddress as _CreateAddress,
  CreateAlias as _CreateAlias,
  CreateCarrierMapping as _CreateCarrierMapping,
  CreateCustomer as _CreateCustomer,
  CreateGrant as _CreateGrant,
  CreateProduct as _CreateProduct,
  CreateTemplate as _CreateTemplate,
  CreateWave as _CreateWave,
  ExportFactoryOrder as _ExportFactoryOrder,
  GenerateFactoryOrder as _GenerateFactoryOrder,
  GenerateWritebacks as _GenerateWritebacks,
  GetCustomer as _GetCustomer,
  GetSettings as _GetSettings,
  GetWave as _GetWave,
  Home as _Home,
  ImportShipment as _ImportShipment,
  IngestDocument as _IngestDocument,
  ListAddresses as _ListAddresses,
  ListAliases as _ListAliases,
  ListCarrierMappings as _ListCarrierMappings,
  ListCustomers as _ListCustomers,
  ListInboxRows as _ListInboxRows,
  ListPlatforms as _ListPlatforms,
  ListProducts as _ListProducts,
  ListResultViews as _ListResultViews,
  ListRules as _ListRules,
  ListSupplierOrderLines as _ListSupplierOrderLines,
  ListSupplierOrders as _ListSupplierOrders,
  ListTemplates as _ListTemplates,
  ListWaves as _ListWaves,
  NamedTransformers as _NamedTransformers,
  ProductTotals as _ProductTotals,
  ReopenWave as _ReopenWave,
  SaveSettings as _SaveSettings,
  SemanticDictionary as _SemanticDictionary,
  SetResultAddress as _SetResultAddress,
  UpsertRule as _UpsertRule,
  VoidFactoryOrder as _VoidFactoryOrder,
} from '../../../wailsjs/go/controller/WorkspaceController'

import {
  GetDataDir as _GetDataDir,
  RevealInFolder as _RevealInFolder,
} from '../../../wailsjs/go/controller/FileSystemController'

import type {
  AppSettings,
  CarrierMapping,
  CustomerProfile,
  DuplicateObservation,
  EntitlementRule,
  FulfillmentResult,
  GenerateFactoryOrderResult,
  HomeBuckets,
  InboxRow,
  IngestDocumentResult,
  IngestFactInput,
  InputDocument,
  Platform,
  ProductAlias,
  ProductItem,
  ProductTotal,
  RecipientAddress,
  ResultView,
  Shipment,
  SupplierOrder,
  SupplierOrderLine,
  TemplateConfig,
  Wave,
  ChannelWritebackItem,
} from '@/entities/models'

import type { app as wailsApp, domain as wailsDomain } from '../../../wailsjs/go/models'

import { markBridgeMissing, markBridgeSeen } from './health'

// ── Guards ──

function isWailsRuntimeAvailable(): boolean {
  if (typeof window === 'undefined') return false
  const w = window as unknown as { go?: { controller?: unknown } }
  const ok = Boolean(w.go?.controller)
  if (ok) {
    markBridgeSeen()
  } else {
    markBridgeMissing()
  }
  return ok
}

function assertWailsRuntime(): void {
  if (!isWailsRuntimeAvailable()) {
    throw new Error('Wails runtime is not available')
  }
}

// ── WorkspaceController: Platforms & System ──

export async function listPlatforms(): Promise<Platform[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListPlatforms()
  return (res ?? []) as unknown as Platform[]
}

// ── WorkspaceController: Customers & Addresses ──

export async function createCustomer(name: string, notes = ''): Promise<CustomerProfile> {
  assertWailsRuntime()
  const res = await _CreateCustomer(name, notes)
  return res as unknown as CustomerProfile
}

export async function listCustomers(): Promise<CustomerProfile[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListCustomers()
  return (res ?? []) as unknown as CustomerProfile[]
}

export async function getCustomer(id: number): Promise<CustomerProfile> {
  assertWailsRuntime()
  const res = await _GetCustomer(id)
  return res as unknown as CustomerProfile
}

export async function createAddress(input: Partial<RecipientAddress>): Promise<RecipientAddress> {
  assertWailsRuntime()
  const res = await _CreateAddress(input as wailsDomain.RecipientAddress)
  return res as unknown as RecipientAddress
}

export async function listAddresses(customerID: number): Promise<RecipientAddress[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListAddresses(customerID)
  return (res ?? []) as unknown as RecipientAddress[]
}

// ── WorkspaceController: Products & Aliases ──

export async function createProduct(input: Partial<ProductItem>): Promise<ProductItem> {
  assertWailsRuntime()
  const res = await _CreateProduct(input as wailsDomain.ProductItem)
  return res as unknown as ProductItem
}

export async function listProducts(): Promise<ProductItem[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListProducts()
  return (res ?? []) as unknown as ProductItem[]
}

export async function createAlias(input: Partial<ProductAlias>): Promise<ProductAlias> {
  assertWailsRuntime()
  const res = await _CreateAlias(input as wailsDomain.ProductAlias)
  return res as unknown as ProductAlias
}

export async function listAliases(productID: number): Promise<ProductAlias[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListAliases(productID)
  return (res ?? []) as unknown as ProductAlias[]
}

// ── WorkspaceController: Templates & Carrier Mappings ──

export async function createTemplate(input: Partial<TemplateConfig>): Promise<TemplateConfig> {
  assertWailsRuntime()
  const res = await _CreateTemplate(input as wailsDomain.TemplateConfig)
  return res as unknown as TemplateConfig
}

export async function listTemplates(): Promise<TemplateConfig[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListTemplates()
  return (res ?? []) as unknown as TemplateConfig[]
}

export async function getSemanticDictionary(): Promise<string[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _SemanticDictionary()
  return res ?? []
}

export async function getNamedTransformers(): Promise<string[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _NamedTransformers()
  return res ?? []
}

export async function createCarrierMapping(input: Partial<CarrierMapping>): Promise<CarrierMapping> {
  assertWailsRuntime()
  const res = await _CreateCarrierMapping(input as wailsDomain.CarrierMapping)
  return res as unknown as CarrierMapping
}

export async function listCarrierMappings(platformID: number): Promise<CarrierMapping[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListCarrierMappings(platformID)
  return (res ?? []) as unknown as CarrierMapping[]
}

// ── WorkspaceController: Settings ──

export async function getSettings(): Promise<AppSettings> {
  if (!isWailsRuntimeAvailable()) {
    return {
      Locale: 'zh-CN',
      Theme: 'system',
      Density: 'comfortable',
      DuplicateRecordMinutes: 10,
      DuplicateAskDays: 10,
    }
  }
  const res = await _GetSettings()
  return res as unknown as AppSettings
}

export async function saveSettings(input: Partial<AppSettings>): Promise<void> {
  assertWailsRuntime()
  await _SaveSettings(input as wailsDomain.AppSettings)
}

// ── WorkspaceController: Inbox & Ingestion ──

export async function ingestDocument(
  doc: Partial<InputDocument>,
  facts: IngestFactInput[],
): Promise<IngestDocumentResult> {
  assertWailsRuntime()
  const res = await _IngestDocument(doc as wailsDomain.InputDocument, facts as wailsApp.IngestFactInput[])
  return {
    Document: res.Document as unknown as InputDocument,
    Duplicates: (res.Duplicates ?? []) as unknown as DuplicateObservation[],
  }
}

export async function attachIdentity(identityID: number, customerID: number): Promise<void> {
  assertWailsRuntime()
  await _AttachIdentity(identityID, customerID)
}

export async function assignLines(waveID: number, lineIDs: number[]): Promise<void> {
  assertWailsRuntime()
  await _AssignLines(waveID, lineIDs)
}

export async function listInboxRows(): Promise<InboxRow[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListInboxRows()
  return (res ?? []) as unknown as InboxRow[]
}

// ── WorkspaceController: Waves ──

export async function createWave(name: string, notes = ''): Promise<Wave> {
  assertWailsRuntime()
  const res = await _CreateWave(name, notes)
  return res as unknown as Wave
}

export async function listWaves(): Promise<Wave[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListWaves()
  return (res ?? []) as unknown as Wave[]
}

export async function getWave(id: number): Promise<Wave> {
  assertWailsRuntime()
  const res = await _GetWave(id)
  return res as unknown as Wave
}

export async function closeWave(id: number, result: string, note = ''): Promise<void> {
  assertWailsRuntime()
  await _CloseWave(id, result, note)
}

export async function reopenWave(id: number): Promise<void> {
  assertWailsRuntime()
  await _ReopenWave(id)
}

// ── WorkspaceController: Wave Rules & Grants ──

export async function upsertRule(rule: Partial<EntitlementRule>): Promise<EntitlementRule> {
  assertWailsRuntime()
  const res = await _UpsertRule(rule as wailsDomain.EntitlementRule)
  return res as unknown as EntitlementRule
}

export async function listRules(waveID: number): Promise<EntitlementRule[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListRules(waveID)
  return (res ?? []) as unknown as EntitlementRule[]
}

export async function createGrant(
  waveID: number,
  customerID: number,
  productID: number,
  qty: number,
): Promise<FulfillmentResult> {
  assertWailsRuntime()
  const res = await _CreateGrant(waveID, customerID, productID, qty)
  return res as unknown as FulfillmentResult
}

export async function listResultViews(waveID: number): Promise<ResultView[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListResultViews(waveID)
  return (res ?? []) as unknown as ResultView[]
}

export async function getProductTotals(waveID: number): Promise<ProductTotal[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ProductTotals(waveID)
  return (res ?? []) as unknown as ProductTotal[]
}

export async function setResultAddress(resultID: number, addressID: number): Promise<void> {
  assertWailsRuntime()
  await _SetResultAddress(resultID, addressID)
}

// ── WorkspaceController: Factory, Shipments & Writebacks ──

export async function generateFactoryOrder(
  waveID: number,
  factoryID: number,
): Promise<GenerateFactoryOrderResult> {
  assertWailsRuntime()
  const res = await _GenerateFactoryOrder(waveID, factoryID)
  return {
    Order: res.Order as unknown as SupplierOrder,
    Lines: (res.Lines ?? []) as unknown as SupplierOrderLine[],
  }
}

export async function exportFactoryOrder(orderID: number): Promise<SupplierOrder> {
  assertWailsRuntime()
  const res = await _ExportFactoryOrder(orderID)
  return res as unknown as SupplierOrder
}

export async function voidFactoryOrder(orderID: number): Promise<void> {
  assertWailsRuntime()
  await _VoidFactoryOrder(orderID)
}

export async function listSupplierOrders(waveID: number): Promise<SupplierOrder[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListSupplierOrders(waveID)
  return (res ?? []) as unknown as SupplierOrder[]
}

export async function listSupplierOrderLines(orderID: number): Promise<SupplierOrderLine[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListSupplierOrderLines(orderID)
  return (res ?? []) as unknown as SupplierOrderLine[]
}

export async function importShipment(
  trackingID: string,
  trackingNo: string,
  carrierCode: string,
  carrierName: string,
  qty: number,
): Promise<Shipment> {
  assertWailsRuntime()
  const res = await _ImportShipment(trackingID, trackingNo, carrierCode, carrierName, qty)
  return res as unknown as Shipment
}

export async function generateWritebacks(factID: number): Promise<ChannelWritebackItem[]> {
  assertWailsRuntime()
  const res = await _GenerateWritebacks(factID)
  return (res ?? []) as unknown as ChannelWritebackItem[]
}

// ── WorkspaceController: Home ──

export async function getHomeBuckets(): Promise<HomeBuckets> {
  if (!isWailsRuntimeAvailable()) {
    return {
      Unassigned: 0,
      DuplicateAsk: 0,
      AlignmentConflict: 0,
      IdentityUnattached: 0,
      BlockedResults: 0,
      WritebackFailed: 0,
      ResidualClose: 0,
      RecentWaves: [],
    }
  }
  const res = await _Home()
  return res as unknown as HomeBuckets
}

// ── FileSystemController ──

export async function revealInFolder(path: string): Promise<void> {
  if (!isWailsRuntimeAvailable()) return
  await _RevealInFolder(path)
}

export async function getDataDir(): Promise<string> {
  assertWailsRuntime()
  return await _GetDataDir()
}

export { isWailsRuntimeAvailable, assertWailsRuntime }
