// Bridge: strong-typed thin wrappers over generated Wails bindings.
// Never import from "wailsjs" directly outside this file.

import {
  ApplyRevision as _ApplyRevision,
  AssignLines as _AssignLines,
  AttachIdentity as _AttachIdentity,
  CloseWave as _CloseWave,
  DecideDuplicate as _DecideDuplicate,
  DismissRevision as _DismissRevision,
  MoveLines as _MoveLines,
  CreateAddress as _CreateAddress,
  CreateAlias as _CreateAlias,
  CreateBundleComponent as _CreateBundleComponent,
  CreateCarrierMapping as _CreateCarrierMapping,
  CreateCustomer as _CreateCustomer,
  CreateGrant as _CreateGrant,
  CreatePlatform as _CreatePlatform,
  CreateProduct as _CreateProduct,
  CreateTemplate as _CreateTemplate,
  CreateWave as _CreateWave,
  AddException as _AddException,
  DeleteCarrierMapping as _DeleteCarrierMapping,
  DeleteException as _DeleteException,
  DeleteQuantitySplitRule as _DeleteQuantitySplitRule,
  DeleteRule as _DeleteRule,
  DeleteTemplate as _DeleteTemplate,
  DocumentTypeCatalog as _DocumentTypeCatalog,
  ListQuantitySplitRules as _ListQuantitySplitRules,
  UpsertQuantitySplitRule as _UpsertQuantitySplitRule,
  ExportFactoryOrderFile as _ExportFactoryOrderFile,
  ExportWritebackFile as _ExportWritebackFile,
  GenerateFactoryOrder as _GenerateFactoryOrder,
  GenerateFactoryOrderForResults as _GenerateFactoryOrderForResults,
  GenerateWritebacks as _GenerateWritebacks,
  GetCustomer as _GetCustomer,
  GetSettings as _GetSettings,
  GetTemplate as _GetTemplate,
  GetWave as _GetWave,
  Home as _Home,
  ImportCarrierMappings as _ImportCarrierMappings,
  ImportFile as _ImportFile,
  ImportShipment as _ImportShipment,
  ImportShipmentFile as _ImportShipmentFile,
  IngestDocument as _IngestDocument,
  InspectSampleFile as _InspectSampleFile,
  ListAddresses as _ListAddresses,
  ListAliases as _ListAliases,
  ListCarrierMappings as _ListCarrierMappings,
  ListCustomers as _ListCustomers,
  ListEntitlementInstances as _ListEntitlementInstances,
  ListExceptions as _ListExceptions,
  ListInboxRows as _ListInboxRows,
  ListPlatforms as _ListPlatforms,
  ListProducts as _ListProducts,
  ListResultViews as _ListResultViews,
  ListRules as _ListRules,
  ListSupplierOrderLines as _ListSupplierOrderLines,
  ListSupplierOrders as _ListSupplierOrders,
  ListTemplates as _ListTemplates,
  ListWaves as _ListWaves,
  ListWritebacksByWave as _ListWritebacksByWave,
  MarkWritebackFailed as _MarkWritebackFailed,
  MarkWritebackSent as _MarkWritebackSent,
  NamedTransformers as _NamedTransformers,
  ProductTotals as _ProductTotals,
  PreviewMapping as _PreviewMapping,
  PreviewTemplate as _PreviewTemplate,
  ReopenWave as _ReopenWave,
  SaveSettings as _SaveSettings,
  SemanticDictionary as _SemanticDictionary,
  SetResultAddress as _SetResultAddress,
  UpdateAlias as _UpdateAlias,
  UpdateCarrierMapping as _UpdateCarrierMapping,
  UpdateCustomer as _UpdateCustomer,
  UpdateTemplate as _UpdateTemplate,
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
  ChannelWritebackItem,
  CustomerProfile,
  DocumentTypeInfo,
  DuplicateObservation,
  EntitlementException,
  EntitlementRule,
  ExceptionView,
  ExportFileResult,
  FulfillmentResult,
  GenerateFactoryOrderResult,
  HomeBuckets,
  ImportCarrierMappingsResult,
  ImportFileResult,
  ImportShipmentFileResult,
  InboxRow,
  IngestDocumentResult,
  IngestFactInput,
  InputDocument,
  InstanceView,
  ParseIssue,
  Platform,
  PreviewRow,
  ProductAlias,
  ProductBundleComponent,
  ProductItem,
  ProductTotal,
  QuantitySplitRule,
  RecipientAddress,
  ResultView,
  SampleFileInfo,
  Shipment,
  SkippedShipment,
  SupplierOrder,
  SupplierOrderLine,
  TemplateConfig,
  TemplatePreview,
  Wave,
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

/** Register a new source/factory platform manually. */
export async function createPlatform(input: Partial<Platform>): Promise<void> {
  assertWailsRuntime()
  await _CreatePlatform(input as wailsDomain.Platform)
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

/** Persist edits to an existing customer profile. */
export async function updateCustomer(input: Partial<CustomerProfile>): Promise<void> {
  assertWailsRuntime()
  await _UpdateCustomer(input as wailsDomain.CustomerProfile)
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

export async function updateAlias(aliasID: number, productItemID: number): Promise<void> {
  assertWailsRuntime()
  await _UpdateAlias(aliasID, productItemID)
}

/** Attach one internal product to an alias as a bundle component. */
export async function createBundleComponent(
  input: Partial<ProductBundleComponent>,
): Promise<void> {
  assertWailsRuntime()
  await _CreateBundleComponent(input as wailsDomain.ProductBundleComponent)
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

export async function getTemplate(id: number): Promise<TemplateConfig> {
  assertWailsRuntime()
  const res = await _GetTemplate(id)
  return res as unknown as TemplateConfig
}

/** Edit an active template in place; the backend bumps Version itself. */
export async function updateTemplate(input: Partial<TemplateConfig>): Promise<TemplateConfig> {
  assertWailsRuntime()
  const res = await _UpdateTemplate(input as wailsDomain.TemplateConfig)
  return res as unknown as TemplateConfig
}

/** Delete an active template; built-ins are refused by the backend. */
export async function deleteTemplate(id: number): Promise<void> {
  assertWailsRuntime()
  await _DeleteTemplate(id)
}

/** Closed set of document types with their locked direction and platform kind. */
export async function getDocumentTypeCatalog(): Promise<DocumentTypeInfo[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _DocumentTypeCatalog()
  return (res ?? []) as unknown as DocumentTypeInfo[]
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

/**
 * Read the raw first rows of a sample file (no mapping applied). Pass an
 * empty sheetName for the first sheet; Records[0] is the candidate header.
 */
export async function inspectSampleFile(
  filePath: string,
  sheetName: string,
  limit: number,
): Promise<SampleFileInfo> {
  assertWailsRuntime()
  const res = await _InspectSampleFile(filePath, sheetName, limit)
  return {
    Format: res?.Format ?? '',
    Sheets: (res?.Sheets ?? []) as string[],
    Records: ((res?.Records ?? []) as unknown as string[][]).map((row) =>
      (row ?? []).map((cell) => (cell == null ? '' : String(cell))),
    ),
    Total: res?.Total ?? 0,
  }
}

function toTemplatePreview(res: unknown): TemplatePreview {
  const raw = (res ?? {}) as {
    Rows?: unknown[]
    Issues?: unknown[]
    TotalRows?: number
    DroppedRows?: number
  }
  return {
    Rows: (raw.Rows ?? []).map((row) => {
      const r = (row ?? {}) as Partial<PreviewRow>
      return {
        LineNo: r.LineNo ?? 0,
        SourceRow: r.SourceRow ?? 0,
        Values: (r.Values ?? {}) as Record<string, string>,
        Fingerprint: r.Fingerprint ?? '',
      }
    }),
    Issues: (raw.Issues ?? []) as ParseIssue[],
    TotalRows: raw.TotalRows ?? 0,
    DroppedRows: raw.DroppedRows ?? 0,
  }
}

/** Parse a sample file through an unsaved mapping config (editor validation). */
export async function previewMapping(
  mappingJSON: string,
  documentType: string,
  filePath: string,
  limit: number,
): Promise<TemplatePreview> {
  assertWailsRuntime()
  const res = await _PreviewMapping(mappingJSON, documentType, filePath, limit)
  return toTemplatePreview(res)
}

/** Parse a sample file through a saved template's mapping config without ingesting. */
export async function previewTemplate(
  templateID: number,
  filePath: string,
  limit: number,
): Promise<TemplatePreview> {
  assertWailsRuntime()
  const res = await _PreviewTemplate(templateID, filePath, limit)
  return toTemplatePreview(res)
}

/** Import a platform export file through a template into inbox facts. */
export async function importFile(
  platformID: number,
  templateID: number,
  filePath: string,
): Promise<ImportFileResult> {
  assertWailsRuntime()
  const res = await _ImportFile(platformID, templateID, filePath)
  return {
    Document: res.Document as unknown as InputDocument,
    FactsCreated: res.FactsCreated ?? 0,
    LinesCreated: res.LinesCreated ?? 0,
    Duplicates: (res.Duplicates ?? []) as unknown as DuplicateObservation[],
    Issues: (res.Issues ?? []) as unknown as ParseIssue[],
  }
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

export async function updateCarrierMapping(input: Partial<CarrierMapping>): Promise<CarrierMapping> {
  assertWailsRuntime()
  const res = await _UpdateCarrierMapping(input as wailsDomain.CarrierMapping)
  return res as unknown as CarrierMapping
}

export async function deleteCarrierMapping(id: number): Promise<void> {
  assertWailsRuntime()
  await _DeleteCarrierMapping(id)
}

/**
 * Bulk-load carrier mappings from a file: `nameHeader` is the column holding
 * the carrier name/description, `codeHeader` the platform carrier ID.
 */
export async function importCarrierMappings(
  platformID: number,
  filePath: string,
  nameHeader: string,
  codeHeader: string,
): Promise<ImportCarrierMappingsResult> {
  assertWailsRuntime()
  const res = await _ImportCarrierMappings(platformID, filePath, nameHeader, codeHeader)
  return {
    Created: res?.Created ?? 0,
    Updated: res?.Updated ?? 0,
    Skipped: res?.Skipped ?? 0,
    Issues: (res?.Issues ?? []) as unknown as ParseIssue[],
  }
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

/**
 * Decide one duplicate observation. `accept` keeps the record as-is (no new
 * responsibility); rejecting replays the stored input as a new fact.
 */
export async function decideDuplicate(observationID: number, accept: boolean): Promise<void> {
  assertWailsRuntime()
  await _DecideDuplicate(observationID, accept)
}

/** Move fact lines into another open wave; frozen lines are refused. */
export async function moveLines(lineIDs: number[], targetWaveID: number): Promise<void> {
  assertWailsRuntime()
  await _MoveLines(lineIDs, targetWaveID)
}

/** Apply a pending revision fact onto the fact it revises. */
export async function applyRevision(factID: number): Promise<void> {
  assertWailsRuntime()
  await _ApplyRevision(factID)
}

/** Dismiss a pending revision fact without applying it. */
export async function dismissRevision(factID: number): Promise<void> {
  assertWailsRuntime()
  await _DismissRevision(factID)
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

/** Delete one rule and recompute the wave's entitlements. */
export async function deleteRule(ruleID: number): Promise<void> {
  assertWailsRuntime()
  await _DeleteRule(ruleID)
}

/** Add a per-instance entitlement exception and recompute the wave. */
export async function addException(input: Partial<EntitlementException>): Promise<void> {
  assertWailsRuntime()
  await _AddException(input as wailsDomain.EntitlementException)
}

/** Delete one entitlement exception and recompute the wave. */
export async function deleteException(exceptionID: number): Promise<void> {
  assertWailsRuntime()
  await _DeleteException(exceptionID)
}

/** List the wave's exceptions joined with customer and product display names. */
export async function listExceptions(waveID: number): Promise<ExceptionView[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListExceptions(waveID)
  return (res ?? []) as unknown as ExceptionView[]
}

/** List the wave's membership instances with display fields for pickers. */
export async function listEntitlementInstances(waveID: number): Promise<InstanceView[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListEntitlementInstances(waveID)
  return (res ?? []) as unknown as InstanceView[]
}

/** Store a wave quantity split (components replaced wholesale) and re-derive covered lines. */
export async function upsertQuantitySplitRule(rule: Partial<QuantitySplitRule>): Promise<void> {
  assertWailsRuntime()
  await _UpsertQuantitySplitRule(rule as wailsDomain.QuantitySplitRule)
}

/** Remove a wave quantity split; covered lines fall back to plain alignment. */
export async function deleteQuantitySplitRule(id: number): Promise<void> {
  assertWailsRuntime()
  await _DeleteQuantitySplitRule(id)
}

export async function listQuantitySplitRules(waveID: number): Promise<QuantitySplitRule[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListQuantitySplitRules(waveID)
  return (res ?? []) as unknown as QuantitySplitRule[]
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

/**
 * Partial submission: aggregate only the selected fulfillment results into
 * the factory order. Unselected results are left untouched; the
 * one-open-order slot per (wave, factory) still applies.
 */
export async function generateFactoryOrderForResults(
  waveID: number,
  factoryID: number,
  resultIDs: number[],
): Promise<GenerateFactoryOrderResult> {
  assertWailsRuntime()
  const res = await _GenerateFactoryOrderForResults(waveID, factoryID, resultIDs)
  return {
    Order: res.Order as unknown as SupplierOrder,
    Lines: (res.Lines ?? []) as unknown as SupplierOrderLine[],
  }
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

/** Render a factory order into its export file and return the written path. */
export async function exportFactoryOrderFile(orderID: number): Promise<ExportFileResult> {
  assertWailsRuntime()
  const res = await _ExportFactoryOrderFile(orderID)
  return {
    Order: res.Order as unknown as SupplierOrder,
    Path: res.Path ?? '',
    Rows: (res.Rows ?? []) as Record<string, string>[],
  }
}

/** Import a factory shipment-return file (one row per parcel) through an explicit template. */
export async function importShipmentFile(
  platformID: number,
  templateID: number,
  filePath: string,
): Promise<ImportShipmentFileResult> {
  assertWailsRuntime()
  const res = await _ImportShipmentFile(platformID, templateID, filePath)
  return {
    Imported: res.Imported ?? 0,
    Skipped: (res.Skipped ?? []) as unknown as SkippedShipment[],
    Shipments: (res.Shipments ?? []) as unknown as Shipment[],
    Issues: (res.Issues ?? []) as unknown as ParseIssue[],
  }
}

export async function generateWritebacks(factID: number): Promise<ChannelWritebackItem[]> {
  assertWailsRuntime()
  const res = await _GenerateWritebacks(factID)
  return (res ?? []) as unknown as ChannelWritebackItem[]
}

/** List every writeback item behind the wave's fact lines, ordered by id. */
export async function listWritebacksByWave(waveID: number): Promise<ChannelWritebackItem[]> {
  if (!isWailsRuntimeAvailable()) return []
  const res = await _ListWritebacksByWave(waveID)
  return (res ?? []) as unknown as ChannelWritebackItem[]
}

/** Record a successful channel writeback for one parcel. */
export async function markWritebackSent(writebackID: number): Promise<void> {
  assertWailsRuntime()
  await _MarkWritebackSent(writebackID)
}

/** Record a failed channel writeback attempt (retry counter advances). */
export async function markWritebackFailed(writebackID: number, errMsg: string): Promise<void> {
  assertWailsRuntime()
  await _MarkWritebackFailed(writebackID, errMsg)
}

/** Write one writeback item's stored payload to the exports dir; returns the path. */
export async function exportWritebackFile(writebackID: number): Promise<string> {
  assertWailsRuntime()
  const res = await _ExportWritebackFile(writebackID)
  return res?.Path ?? ''
}

// ── WorkspaceController: Home ──

export async function getHomeBuckets(): Promise<HomeBuckets> {
  if (!isWailsRuntimeAvailable()) {
    return {
      Unassigned: 0,
      DuplicateAsk: 0,
      AlignmentConflict: 0,
      IdentityUnattached: 0,
      PendingRevisions: 0,
      BlockedResults: 0,
      WritebackFailed: 0,
      ResidualClose: 0,
      RevisionFrozenConflicts: 0,
      RecentWaves: [],
    }
  }
  const res = await _Home()
  return res as unknown as HomeBuckets
}

// ── Wails runtime: native dialogs ──

/** Minimal mirror of the Wails v2 runtime OpenFileDialog options. */
interface WailsFileDialogFilter {
  displayName: string
  pattern: string
}

interface WailsRuntimeDialogApi {
  OpenFileDialog?: (options?: { title?: string; filters?: WailsFileDialogFilter[] }) => Promise<string>
}

/**
 * Open a native single-file picker through the Wails runtime dialog API.
 * The committed copy of wailsjs/runtime predates the generated dialog
 * wrappers, so the runtime object Wails injects on `window` is called
 * directly — still only ever from inside the bridge. Returns null when the
 * runtime (or dialog) is unavailable or the user cancels.
 */
export async function pickFile(
  filters?: WailsFileDialogFilter[],
): Promise<string | null> {
  if (!isWailsRuntimeAvailable()) return null
  const runtime = (window as unknown as { runtime?: WailsRuntimeDialogApi }).runtime
  if (!runtime?.OpenFileDialog) return null
  const path = await runtime.OpenFileDialog(filters ? { filters } : undefined)
  return path && path.trim() !== '' ? path : null
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
