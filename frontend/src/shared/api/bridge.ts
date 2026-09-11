// Bridge: strong-typed thin wrappers over the Wails v3 generated bindings.
// This is the only file in src/ allowed to import from frontend/bindings/ or
// @wailsio/runtime; pages go through '@/shared/api/bridge' exclusively.

import { Dialogs } from '@wailsio/runtime'

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
} from '../../../bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/controller/workspacecontroller'

import {
  GetDataDir as _GetDataDir,
  RevealInFolder as _RevealInFolder,
} from '../../../bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/controller/filesystemcontroller'

// ── Generated model types: wire shapes for the entities facade ──
// The bridge is the only file in src/ allowed to import frontend/bindings,
// so the type facade (src/entities/models.ts) builds on these re-exports.
// Type-only: erased at runtime, so this adds no code to the bundle. The
// `Wire` suffix marks the raw generated forms; the facade narrows them into
// the UI-facing names. Note BlockReason/WorkState enum objects are left out
// on purpose — the UI keeps the string-literal unions from
// @/shared/api/generated/enums (values are identical over the wire).

export type {
  AddressSnapshot as AddressSnapshotWire,
  AppSettings as AppSettingsWire,
  CarrierMapping as CarrierMappingWire,
  ChannelWritebackItem as ChannelWritebackItemWire,
  CustomerProfile as CustomerProfileWire,
  DuplicateObservation as DuplicateObservationWire,
  EntitlementException as EntitlementExceptionWire,
  EntitlementRule as EntitlementRuleWire,
  EntitlementSelector as EntitlementSelectorWire,
  FulfillmentResult as FulfillmentResultWire,
  InputDocument as InputDocumentWire,
  InputFact as InputFactWire,
  InputFactLine as InputFactLineWire,
  Platform as PlatformWire,
  ProductAlias as ProductAliasWire,
  ProductBundleComponent as ProductBundleComponentWire,
  ProductItem as ProductItemWire,
  QuantitySplitRule as QuantitySplitRuleWire,
  RecipientAddress as RecipientAddressWire,
  Shipment as ShipmentWire,
  SupplierOrder as SupplierOrderWire,
  SupplierOrderLine as SupplierOrderLineWire,
  TemplateConfig as TemplateConfigWire,
  Wave as WaveWire,
} from '../../../bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/domain/models'

export type {
  DocumentTypeInfo as DocumentTypeInfoWire,
  ExceptionView as ExceptionViewWire,
  ExportFileResult as ExportFileResultWire,
  GenerateFactoryOrderResult as GenerateFactoryOrderResultWire,
  HomeBuckets as HomeBucketsWire,
  ImportCarrierMappingsResult as ImportCarrierMappingsResultWire,
  ImportFileResult as ImportFileResultWire,
  ImportShipmentFileResult as ImportShipmentFileResultWire,
  IngestDocumentResult as IngestDocumentResultWire,
  IngestFactInput as IngestFactInputWire,
  IngestLine as IngestLineWire,
  InboxRow as InboxRowWire,
  InstanceView as InstanceViewWire,
  ProductTotal as ProductTotalWire,
  ResultView as ResultViewWire,
  SampleFileInfo as SampleFileInfoWire,
  SkippedShipment as SkippedShipmentWire,
} from '../../../bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/app/models'

export type {
  ParseIssue as ParseIssueWire,
  PreviewRow as PreviewRowWire,
  TemplatePreview as TemplatePreviewWire,
} from '../../../bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment/models'

import type {
  AppSettings,
  CarrierMapping,
  ChannelWritebackItem,
  CustomerProfile,
  DocumentTypeInfo,
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
  Platform,
  ProductAlias,
  ProductBundleComponent,
  ProductItem,
  ProductTotal,
  QuantitySplitRule,
  RecipientAddress,
  ResultView,
  SampleFileInfo,
  Shipment,
  SupplierOrder,
  SupplierOrderLine,
  TemplateConfig,
  TemplatePreview,
  Wave,
} from '@/entities/models'

// Wire shapes the facade deliberately loosens (optional ids/timestamps); the
// generated controller signatures want the strict forms.
import type {
  AppSettings as AppSettingsWire,
  QuantitySplitRule as QuantitySplitRuleWire,
} from '../../../bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/domain/models'

import type {
  ExportFileResult as ExportFileResultWire,
  ImportFileResult as ImportFileResultWire,
  ImportShipmentFileResult as ImportShipmentFileResultWire,
  IngestDocumentResult as IngestDocumentResultWire,
  GenerateFactoryOrderResult as GenerateFactoryOrderResultWire,
  SampleFileInfo as SampleFileInfoWire,
} from '../../../bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/app/models'

import type {
  TemplatePreview as TemplatePreviewWire,
} from '../../../bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/app/alignment/models'

import { markBridgeMissing, markBridgeSeen } from './health'

// ── Guards ──

/**
 * Host bridge objects the Wails v3 runtime itself probes to tell a WebView
 * from a plain browser (see @wailsio/runtime dist/system.js, `_invoke`):
 * `window.chrome.webview` (WebView2), `window.webkit.messageHandlers.external`
 * (WKWebView), `window.wails` (Android WebView). Unlike the post-navigation
 * `window._wails` init script these exist before any page script runs, so the
 * check is synchronous and race-free.
 *
 * Semantics differ from the v2 guard, which tested whether the bound Go
 * objects existed: this guard only proves the host WebView bridge exists. In
 * the extreme case where the WebView is present but the runtime fails to
 * initialize, the old guard would soft-fail while this one lets the call
 * through and the invocation itself errors. That matches how @wailsio/runtime
 * performs its own WebView detection, and is a deliberate choice.
 */
interface WailsHostGlobals {
  chrome?: { webview?: { postMessage?: unknown } }
  webkit?: { messageHandlers?: Record<string, { postMessage?: unknown } | undefined> }
  wails?: { invoke?: unknown }
}

function isWailsRuntimeAvailable(): boolean {
  if (typeof window === 'undefined') return false
  const w = window as unknown as WailsHostGlobals
  const ok = Boolean(
    w.chrome?.webview?.postMessage ??
      w.webkit?.messageHandlers?.['external']?.postMessage ??
      w.wails?.invoke,
  )
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
  return (await _ListPlatforms()) ?? []
}

/** Register a new source/factory platform manually. */
export async function createPlatform(input: Partial<Platform>): Promise<void> {
  assertWailsRuntime()
  await _CreatePlatform(input as Platform)
}

// ── WorkspaceController: Customers & Addresses ──

export async function createCustomer(name: string, notes = ''): Promise<CustomerProfile> {
  assertWailsRuntime()
  return (await _CreateCustomer(name, notes)) as CustomerProfile
}

export async function listCustomers(): Promise<CustomerProfile[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListCustomers()) ?? []
}

export async function getCustomer(id: number): Promise<CustomerProfile> {
  assertWailsRuntime()
  return (await _GetCustomer(id)) as CustomerProfile
}

/** Persist edits to an existing customer profile. */
export async function updateCustomer(input: Partial<CustomerProfile>): Promise<void> {
  assertWailsRuntime()
  await _UpdateCustomer(input as CustomerProfile)
}

export async function createAddress(input: Partial<RecipientAddress>): Promise<RecipientAddress> {
  assertWailsRuntime()
  return (await _CreateAddress(input as RecipientAddress)) as RecipientAddress
}

export async function listAddresses(customerID: number): Promise<RecipientAddress[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListAddresses(customerID)) ?? []
}

// ── WorkspaceController: Products & Aliases ──

export async function createProduct(input: Partial<ProductItem>): Promise<ProductItem> {
  assertWailsRuntime()
  return (await _CreateProduct(input as ProductItem)) as ProductItem
}

export async function listProducts(): Promise<ProductItem[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListProducts()) ?? []
}

export async function createAlias(input: Partial<ProductAlias>): Promise<ProductAlias> {
  assertWailsRuntime()
  return (await _CreateAlias(input as ProductAlias)) as ProductAlias
}

export async function listAliases(productID: number): Promise<ProductAlias[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListAliases(productID)) ?? []
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
  await _CreateBundleComponent(input as ProductBundleComponent)
}

// ── WorkspaceController: Templates & Carrier Mappings ──

export async function createTemplate(input: Partial<TemplateConfig>): Promise<TemplateConfig> {
  assertWailsRuntime()
  return (await _CreateTemplate(input as TemplateConfig)) as TemplateConfig
}

export async function listTemplates(): Promise<TemplateConfig[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListTemplates()) ?? []
}

export async function getTemplate(id: number): Promise<TemplateConfig> {
  assertWailsRuntime()
  return (await _GetTemplate(id)) as TemplateConfig
}

/** Edit an active template in place; the backend bumps Version itself. */
export async function updateTemplate(input: Partial<TemplateConfig>): Promise<TemplateConfig> {
  assertWailsRuntime()
  return (await _UpdateTemplate(input as TemplateConfig)) as TemplateConfig
}

/** Delete an active template; built-ins are refused by the backend. */
export async function deleteTemplate(id: number): Promise<void> {
  assertWailsRuntime()
  await _DeleteTemplate(id)
}

/** Closed set of document types with their locked direction and platform kind. */
export async function getDocumentTypeCatalog(): Promise<DocumentTypeInfo[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _DocumentTypeCatalog()) ?? []
}

export async function getSemanticDictionary(): Promise<string[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _SemanticDictionary()) ?? []
}

export async function getNamedTransformers(): Promise<string[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _NamedTransformers()) ?? []
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
  const res = (await _InspectSampleFile(filePath, sheetName, limit)) as SampleFileInfoWire
  // Optional chains mirror the v2 baseline: today the Go side returns nil
  // only alongside an error, so these are pure defensive fallbacks.
  return {
    Format: res?.Format ?? '',
    Sheets: res?.Sheets ?? [],
    Records: (res?.Records ?? []).map((row) =>
      (row ?? []).map((cell) => (cell == null ? '' : String(cell))),
    ),
    Total: res?.Total ?? 0,
  }
}

function toTemplatePreview(res: TemplatePreviewWire | null): TemplatePreview {
  return {
    Rows: (res?.Rows ?? []).map((row) => ({
      LineNo: row.LineNo ?? 0,
      SourceRow: row.SourceRow ?? 0,
      Values: (row.Values ?? {}) as Record<string, string>,
      Fingerprint: row.Fingerprint ?? '',
    })),
    Issues: res?.Issues ?? [],
    TotalRows: res?.TotalRows ?? 0,
    DroppedRows: res?.DroppedRows ?? 0,
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
  return toTemplatePreview(await _PreviewMapping(mappingJSON, documentType, filePath, limit))
}

/** Parse a sample file through a saved template's mapping config without ingesting. */
export async function previewTemplate(
  templateID: number,
  filePath: string,
  limit: number,
): Promise<TemplatePreview> {
  assertWailsRuntime()
  return toTemplatePreview(await _PreviewTemplate(templateID, filePath, limit))
}

/** Import a platform export file through a template into inbox facts. */
export async function importFile(
  platformID: number,
  templateID: number,
  filePath: string,
): Promise<ImportFileResult> {
  assertWailsRuntime()
  const res = (await _ImportFile(platformID, templateID, filePath)) as ImportFileResultWire
  return {
    Document: res.Document,
    FactsCreated: res.FactsCreated ?? 0,
    LinesCreated: res.LinesCreated ?? 0,
    Duplicates: res.Duplicates ?? [],
    Issues: res.Issues ?? [],
  }
}

export async function createCarrierMapping(input: Partial<CarrierMapping>): Promise<CarrierMapping> {
  assertWailsRuntime()
  return (await _CreateCarrierMapping(input as CarrierMapping)) as CarrierMapping
}

export async function listCarrierMappings(platformID: number): Promise<CarrierMapping[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListCarrierMappings(platformID)) ?? []
}

export async function updateCarrierMapping(input: Partial<CarrierMapping>): Promise<CarrierMapping> {
  assertWailsRuntime()
  return (await _UpdateCarrierMapping(input as CarrierMapping)) as CarrierMapping
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
    Issues: res?.Issues ?? [],
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
  return (await _GetSettings()) as AppSettings
}

export async function saveSettings(input: Partial<AppSettings>): Promise<void> {
  assertWailsRuntime()
  await _SaveSettings(input as AppSettingsWire)
}

// ── WorkspaceController: Inbox & Ingestion ──

export async function ingestDocument(
  doc: Partial<InputDocument>,
  facts: Partial<IngestFactInput>[],
): Promise<IngestDocumentResult> {
  assertWailsRuntime()
  const res = (await _IngestDocument(
    doc as InputDocument,
    facts as IngestFactInput[],
  )) as IngestDocumentResultWire
  return {
    Document: res.Document,
    Duplicates: res.Duplicates ?? [],
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
  return (await _ListInboxRows()) ?? []
}

// ── WorkspaceController: Waves ──

export async function createWave(name: string, notes = ''): Promise<Wave> {
  assertWailsRuntime()
  return (await _CreateWave(name, notes)) as Wave
}

export async function listWaves(): Promise<Wave[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListWaves()) ?? []
}

export async function getWave(id: number): Promise<Wave> {
  assertWailsRuntime()
  return (await _GetWave(id)) as Wave
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
  return (await _UpsertRule(rule as EntitlementRule)) as EntitlementRule
}

export async function listRules(waveID: number): Promise<EntitlementRule[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListRules(waveID)) ?? []
}

/** Delete one rule and recompute the wave's entitlements. */
export async function deleteRule(ruleID: number): Promise<void> {
  assertWailsRuntime()
  await _DeleteRule(ruleID)
}

/** Add a per-instance entitlement exception and recompute the wave. */
export async function addException(input: Partial<EntitlementException>): Promise<void> {
  assertWailsRuntime()
  await _AddException(input as EntitlementException)
}

/** Delete one entitlement exception and recompute the wave. */
export async function deleteException(exceptionID: number): Promise<void> {
  assertWailsRuntime()
  await _DeleteException(exceptionID)
}

/** List the wave's exceptions joined with customer and product display names. */
export async function listExceptions(waveID: number): Promise<ExceptionView[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListExceptions(waveID)) ?? []
}

/** List the wave's membership instances with display fields for pickers. */
export async function listEntitlementInstances(waveID: number): Promise<InstanceView[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListEntitlementInstances(waveID)) ?? []
}

/** Store a wave quantity split (components replaced wholesale) and re-derive covered lines. */
export async function upsertQuantitySplitRule(rule: Partial<QuantitySplitRule>): Promise<void> {
  assertWailsRuntime()
  await _UpsertQuantitySplitRule(rule as QuantitySplitRuleWire)
}

/** Remove a wave quantity split; covered lines fall back to plain alignment. */
export async function deleteQuantitySplitRule(id: number): Promise<void> {
  assertWailsRuntime()
  await _DeleteQuantitySplitRule(id)
}

export async function listQuantitySplitRules(waveID: number): Promise<QuantitySplitRule[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ((await _ListQuantitySplitRules(waveID)) ?? []) as QuantitySplitRule[]
}

export async function createGrant(
  waveID: number,
  customerID: number,
  productID: number,
  qty: number,
): Promise<FulfillmentResult> {
  assertWailsRuntime()
  return (await _CreateGrant(waveID, customerID, productID, qty)) as FulfillmentResult
}

export async function listResultViews(waveID: number): Promise<ResultView[]> {
  if (!isWailsRuntimeAvailable()) return []
  // The generated ResultView types WorkState/Blocks as TS enum objects and a
  // nullable array; over the wire Go always sends the raw snake_case strings
  // and a Blocks array, which is exactly what the facade union types model.
  return ((await _ListResultViews(waveID)) ?? []) as unknown as ResultView[]
}

export async function getProductTotals(waveID: number): Promise<ProductTotal[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ProductTotals(waveID)) ?? []
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
  const res = (await _GenerateFactoryOrder(waveID, factoryID)) as GenerateFactoryOrderResultWire
  return {
    Order: res.Order,
    Lines: res.Lines ?? [],
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
  const res = (await _GenerateFactoryOrderForResults(
    waveID,
    factoryID,
    resultIDs,
  )) as GenerateFactoryOrderResultWire
  return {
    Order: res.Order,
    Lines: res.Lines ?? [],
  }
}

export async function voidFactoryOrder(orderID: number): Promise<void> {
  assertWailsRuntime()
  await _VoidFactoryOrder(orderID)
}

export async function listSupplierOrders(waveID: number): Promise<SupplierOrder[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListSupplierOrders(waveID)) ?? []
}

export async function listSupplierOrderLines(orderID: number): Promise<SupplierOrderLine[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListSupplierOrderLines(orderID)) ?? []
}

export async function importShipment(
  trackingID: string,
  trackingNo: string,
  carrierCode: string,
  carrierName: string,
  qty: number,
): Promise<Shipment> {
  assertWailsRuntime()
  return (await _ImportShipment(trackingID, trackingNo, carrierCode, carrierName, qty)) as Shipment
}

/** Render a factory order into its export file and return the written path. */
export async function exportFactoryOrderFile(orderID: number): Promise<ExportFileResult> {
  assertWailsRuntime()
  const res = (await _ExportFactoryOrderFile(orderID)) as ExportFileResultWire
  return {
    Order: res.Order,
    Path: res.Path ?? '',
    Rows: (res.Rows ?? []).map((row) => row ?? {}) as Record<string, string>[],
  }
}

/** Import a factory shipment-return file (one row per parcel) through an explicit template. */
export async function importShipmentFile(
  platformID: number,
  templateID: number,
  filePath: string,
): Promise<ImportShipmentFileResult> {
  assertWailsRuntime()
  const res = (await _ImportShipmentFile(
    platformID,
    templateID,
    filePath,
  )) as ImportShipmentFileResultWire
  return {
    Imported: res.Imported ?? 0,
    Skipped: res.Skipped ?? [],
    Shipments: res.Shipments ?? [],
    Issues: res.Issues ?? [],
  }
}

export async function generateWritebacks(factID: number): Promise<ChannelWritebackItem[]> {
  assertWailsRuntime()
  return (await _GenerateWritebacks(factID)) ?? []
}

/** List every writeback item behind the wave's fact lines, ordered by id. */
export async function listWritebacksByWave(waveID: number): Promise<ChannelWritebackItem[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (await _ListWritebacksByWave(waveID)) ?? []
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
  return (await _Home()) as HomeBuckets
}

// ── Wails runtime: native dialogs ──

/** File-picker filter as pages express it (v2 dialog option shape). */
interface WailsFileDialogFilter {
  displayName: string
  pattern: string
}

/**
 * Open a native single-file picker through the Wails v3 runtime Dialogs API
 * (`@wailsio/runtime` Dialogs.OpenFile). Returns null when the runtime is
 * unavailable or the user cancels.
 *
 * Callers pass filters as `{displayName, pattern}` pairs; they map 1:1 onto
 * v3's `{Filters: [{DisplayName, Pattern}]}`. The filter specs the v2 Go-side
 * pickers offered (a1cbe78 app.go Pick*File, deleted by the v3 migration) were:
 *   CSV:     [CSV Files: *.csv]
 *   Tabular: [Tabular Files: *.csv;*.xlsx;*.xls, CSV Files: *.csv,
 *             Excel Files: *.xlsx;*.xls]
 *   ZIP:     [ZIP Files: *.zip]
 *   Catalog: [Catalog Files: *.zip;*.csv;*.xlsx;*.xls, ZIP Files: *.zip,
 *             Tabular Files: *.csv;*.xlsx;*.xls]
 * — kept here so future pickers can restore the same file-kind breakdowns.
 */
export async function pickFile(
  filters?: WailsFileDialogFilter[],
): Promise<string | null> {
  if (!isWailsRuntimeAvailable()) return null
  let picked: string | string[] | null
  try {
    picked = await Dialogs.OpenFile({
      CanChooseFiles: true,
      ...(filters
        ? { Filters: filters.map((f) => ({ DisplayName: f.displayName, Pattern: f.pattern })) }
        : {}),
    })
  } catch (err) {
    // A user cancel reaches us as a rejected promise, in one of two shapes:
    //  - the runtime's own "cancelled" error (matched by /cancel/i), or
    //  - the wails wrapper's locale-independent English prefix
    //    "Dialog.OpenFile failed, error getting selection" wrapping the OS
    //    error. On Windows the cancel HRESULT (ERROR_CANCELLED) is described
    //    by the OS in the system locale (e.g. 「操作已被用户取消。」 on zh-CN,
    //    which contains no "cancel"), so matching the message text alone
    //    breaks on non-English systems. The wrapper prefix is hardcoded in
    //    @wailsio/runtime and locale-free; genuine non-cancel failures under
    //    that prefix are vanishingly rare, and mapping them to null matches
    //    the v2 behaviour of degrading silently at the dialog layer.
    if (
      err instanceof Error &&
      (/cancel/i.test(err.message) ||
        /Dialog\.OpenFile failed, error getting selection/i.test(err.message))
    ) {
      return null
    }
    throw err
  }
  // Single selection resolves to a string; guard the array shape anyway —
  // the runtime wrapper defaults unresolved selections to [].
  const path = Array.isArray(picked) ? (picked[0] ?? null) : picked
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
