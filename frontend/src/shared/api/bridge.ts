// Bridge: strong-typed thin wrappers over generated Wails bindings.
// Never import from "wailsjs" directly outside this file.

import {
  GetDemandDocument,
  ImportDemandCSV,
  ImportDemandDocument,
  ImportDemandFromCSV,
  ListDemandInboxRows,
  ListDemandDocuments,
  ListDemandLines,
  ListUnassignedDemandDocuments,
  ParseCSVFile,
  ParseTabularFile,
  UpdateDemandLineRouting,
  BatchUpdateDemandLineRouting,
  GetWaveRoutingStats,
} from "../../../wailsjs/go/main/DemandController";
import {
  CreateWave,
  ListWaves,
  ListWaveDashboardRows,
  ListWavesFiltered,
  GetWave,
  GetWaveOverview,
  GetWaveWorkspaceSnapshot,
  ListWaveFulfillmentRows,
  ListWaveFulfillmentRowsFiltered,
  ListWaveParticipantRows,
  MapDemandLines,
  AssignDemandToWave,
  BatchAssignDemandToWave,
  BatchUnassignDemandFromWave,
  UnassignDemandFromWave,
  GenerateParticipants,
  ListAssignedDemandsByWave,
  UndoWaveAction,
  RedoWaveAction,
  ListRecentHistory,
  GetHistoryGraph,
  RunHistoryGC,
  ListWavesPaginated,
  ValidateStepAccess,
  UpdateWave,
  CloseWave,
} from "../../../wailsjs/go/main/WaveController";
import { GetActionCenterSummary } from "../../../wailsjs/go/main/ActionCenterController";
import { RevealInFolder, GetDataDir } from "../../../wailsjs/go/main/FileSystemController";
import {
  ExportSupplierOrder,
  ExportSupplierOrderForProfile,
  GenerateSupplierOrderFile,
  GetSupplierOrderByWave,
  ListLinesBySupplierOrder,
  ListSupplierOrders,
  MarkSupplierOrderSubmitted,
  RecordSupplierOrderAcceptance,
} from "../../../wailsjs/go/main/ExportController";
import {
  ListAdjustmentsByWave,
  RecordAdjustment,
  BatchRecordAdjustments,
} from "../../../wailsjs/go/main/AdjustmentController";
import {
  CreateShipment,
  GetSupplierOrderLineShippedSummary,
  ImportShipments,
  ListShipmentsByWave,
  MapAndReconcileShipments,
  UpdateShipment,
  VoidShipment,
} from "../../../wailsjs/go/main/ShipmentController";
import {
  BindInternalCarrier,
  CreateChannelSyncJob,
  CreateCarrierMapping,
  DeleteCarrierMapping,
  ExecuteChannelSyncJob,
  ImportCarrierMappings,
  ListCarrierMappings,
  ListChannelSyncJobsByWave,
  ListConnectorCapabilities,
  ListExternalCarriers,
  ListIntegrationProfiles,
  PlanChannelClosure,
  RecordChannelClosureDecision,
  RetryChannelSyncJob,
  RegisterExternalCarrier,
} from "../../../wailsjs/go/main/ChannelSyncController";
import {
  CreateProfile,
  DeleteProfile,
  GetProfile,
  ListProfiles,
  SeedDefaultProfiles,
  UpdateProfile,
} from "../../../wailsjs/go/main/ProfileController";
import {
  CreateProductMaster,
  ImportProductCatalog,
  ListProductMasters,
  ListProductsByWave,
  SnapshotProductsForWave,
  SnapshotProductsForWaveDetailed,
  UpdateProductMaster,
} from "../../../wailsjs/go/main/ProductController";
import {
  CreateAddress,
  DeleteAddress,
  GetAddress,
  ListAddressesByProfile,
  UpdateAddress,
  BindAddressToLine,
  UnbindAddressFromLine,
  BatchBindAddressToLines,
  BindDefaultAddressesForWave,
} from "../../../wailsjs/go/main/AddressController";
import {
  PickCSVFile,
  PickCatalogImportFile,
  PickTabularFile,
  PickZIPFile,
  SaveZoom,
} from "../../../wailsjs/go/main/App";
import {
  ListCustomerProfiles,
  GetCustomerProfile,
  CreateCustomerProfile,
  UpdateCustomerProfile,
  DeleteCustomerProfile,
  AddCustomerIdentity,
  DeleteCustomerIdentity,
  ListCustomerNameObservations,
  ListCustomerProfileOrigins,
  PinCustomerDisplayName,
  UnpinCustomerDisplayName,
  GetCustomerFulfillmentHistory,
} from "../../../wailsjs/go/main/CustomerProfileController";
import {
  DismissMergeCandidate,
  GetMergeCandidate,
  GetMergePolicy,
  GetMergeScanRun,
  ListMergeCandidates,
  ScanMergeCandidates,
  UpdateMergePolicy,
} from "../../../wailsjs/go/main/MergeGovernanceController";
import {
  GetCustomerResolutionFeaturePolicy,
  UpdateCustomerResolutionFeaturePolicy,
} from '../../../wailsjs/go/main/CustomerResolutionFeaturePolicyController'
import {
  GetImportEvidenceRetention,
  GetImportRunDetail,
  ListImportRuns,
  ListImportRunsPage,
  PruneExpiredImportEvidence,
  SetImportEvidenceRetention,
} from '../../../wailsjs/go/main/ImportEvidenceController'
import {
  ExecuteCustomerSplit,
  GetCustomerSplitHistory,
  ListCustomerSplitHistory,
  PreviewCustomerSplit,
} from '../../../wailsjs/go/main/SplitController'
import { dto } from "../../../wailsjs/go/models";
import {
  ListCustomerIdentityPlatforms,
  ListCustomerProfilesPage,
  ListDemandInboxRowsPage,
  ListProductMastersPage,
  ListShipmentsByWavePage,
} from "../../../wailsjs/go/main/ListPaginationController";
import { markBridgeMissing, markBridgeSeen } from "./health";

// ── Guards ──

function isWailsRuntimeAvailable(): boolean {
  const available = typeof window !== "undefined" && !!(window as any).go;
  if (available) {
    markBridgeSeen();
  } else {
    markBridgeMissing();
  }
  return available;
}

function assertWailsRuntime(): void {
  if (!isWailsRuntimeAvailable()) {
    throw new Error(
      "Wails backend not connected — is the app running inside Wails?",
    );
  }
}

// ── DemandController ──

export async function listDemandLines(documentID: number): Promise<dto.DemandLineDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListDemandLines(documentID);
}

export async function listDemandDocuments(): Promise<dto.DemandDocumentDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListDemandDocuments();
}

/** Server-side filtered + paginated demand-inbox rows. Soft-fail — returns an empty page. */
export async function listDemandInboxRows(input: {
  assignment?: string;
  demandKind?: string;
  integrationProfileId?: number;
  pagination?: { page?: number; pageSize?: number };
}): Promise<dto.DemandInboxRowListDTO> {
  if (!isWailsRuntimeAvailable()) {
    return dto.DemandInboxRowListDTO.createFrom({
      rows: [],
      pagination: { page: 1, pageSize: 50, totalCount: 0, totalPages: 0 },
    });
  }
  return ListDemandInboxRows(
    dto.DemandInboxFilterInput.createFrom(input),
    dto.PaginationInput.createFrom(input.pagination ?? {}),
  );
}

export async function listDemandInboxRowsPage(input: {
  assignment?: string
  demandKind?: string
  demandKinds?: string[]
  routingDispositions?: string[]
  integrationProfileId?: number
  waveId?: number
  sortBy?: string
  sortDir?: 'asc' | 'desc'
  limit: number
  offset: number
}): Promise<dto.DemandInboxPageResult> {
  if (!isWailsRuntimeAvailable()) {
    return dto.DemandInboxPageResult.createFrom({ items: [], totalCount: 0 })
  }
  return ListDemandInboxRowsPage(dto.DemandInboxFilterInput.createFrom(input))
}

export async function listUnassignedDemandDocuments(): Promise<dto.DemandDocumentDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListUnassignedDemandDocuments();
}

export async function getDemandDocument(
  id: number,
): Promise<dto.DemandDocumentDTO> {
  assertWailsRuntime();
  return GetDemandDocument(id);
}

/**
 * Parse a spreadsheet on disk (CSV / XLSX / XLS via extension detection) into
 * headers + header-keyed row maps. Preferred entry for all import previews.
 *
 * @param hasHeader when false, the first physical row stays in Rows (headerless
 *   positional sheets such as bilibili membership). Defaults to true.
 */
export async function parseTabularFile(
  path: string,
  hasHeader = true,
): Promise<dto.CSVFilePreviewDTO> {
  assertWailsRuntime();
  try {
    if (typeof ParseTabularFile === 'function') {
      return await ParseTabularFile(path, hasHeader);
    }
  } catch {
    // Runtime may not expose ParseTabularFile yet — fall through to CSV path.
  }
  return ParseCSVFile(path);
}

/** @deprecated Prefer `parseTabularFile` — kept for compatibility; forwards to multi-format parse. */
export async function parseCSVFile(path: string): Promise<dto.CSVFilePreviewDTO> {
  return parseTabularFile(path, true);
}

/**
 * Dual-mode (reject_all / skip_invalid) template-driven demand CSV import. `rows` are
 * header-keyed maps, typically produced by `parseTabularFile`. Partial-success in
 * `skip_invalid` mode — check `result.errors` / `result.errorCount`.
 */
export async function importDemandCSV(input: {
  integrationProfileId: number;
  documentType: string;
  sourceDocumentNo: string;
  sourceCustomerRef: string;
  importMode: string;
  rows?: Record<string, string>[];
  /** Preferred for positional / multi-format sheets — backend re-reads with hasHeader from mapping rules. */
  filePath?: string;
  /** Optional one-request mapping override. It is validated but never persisted as a template/binding. */
  mappingRules?: string;
}): Promise<dto.ImportDemandCSVResult> {
  assertWailsRuntime();
  const request = dto.ImportDemandCSVInput.createFrom({
    ...input,
    rows: input.rows ?? [],
    filePath: input.filePath ?? "",
  }) as dto.ImportDemandCSVInput & { mappingRules: string }
  // Keep this explicit assignment compatible with runtimes generated before the
  // optional DTO field existed: createFrom() otherwise drops unknown properties.
  request.mappingRules = input.mappingRules ?? ""
  return ImportDemandCSV(request)
}

/** Import a demand document. Accepts a plain object matching CreateDemandInput shape. */
export async function importDemandDocument(input: {
  kind: string;
  captureMode: string;
  sourceChannel: string;
  sourceSurface?: string;
  sourceDocumentNo: string;
  sourceCustomerRef?: string;
  customerProfileId?: number;
  integrationProfileId?: number;
  lines: Array<{
    lineType: string;
    obligationTriggerKind: string;
    entitlementAuthority: string;
    recipientInputState?: string;
    routingDisposition: string;
    routingReasonCode?: string;
    eligibilityContextRef?: string;
    entitlementCode?: string;
    giftLevelSnapshot?: string;
    productMasterId?: number;
    recipientInputPayload?: string;
    externalTitle: string;
    requestedQuantity: number;
  }>;
}): Promise<dto.DemandDocumentDTO> {
  assertWailsRuntime();
  const req = dto.CreateDemandInput.createFrom(input);
  return ImportDemandDocument(req);
}

// ── ActionCenterController ──

export async function getActionCenterSummary(): Promise<dto.ActionCenterSummaryDTO> {
  if (!isWailsRuntimeAvailable()) {
    return dto.ActionCenterSummaryDTO.createFrom({
      waves: [],
      inboxPendingIntakeCount: 0,
      navBadges: [],
    });
  }
  return GetActionCenterSummary();
}

// ── WaveController ──

export async function listWaves(): Promise<dto.WaveDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListWaves();
}

export async function listWaveDashboardRows(): Promise<dto.WaveDashboardRowDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListWaveDashboardRows();
}

export async function getWave(id: number): Promise<dto.WaveDTO> {
  assertWailsRuntime();
  return GetWave(id);
}

/** Create a wave. `waveType`/`notes`/`levelTags` are optional — callable with just a name. */
export async function createWave(input: {
  name: string;
  waveType?: string;
  notes?: string;
  levelTags?: string;
}): Promise<dto.WaveDTO> {
  assertWailsRuntime();
  return CreateWave(
    new dto.CreateWaveInput({
      name: input.name,
      waveType: input.waveType ?? "",
      notes: input.notes ?? "",
      levelTags: input.levelTags ?? "",
    }),
  );
}

/** Rename/update a wave's name/notes/levelTags. */
export async function updateWave(input: {
  waveId: number;
  name: string;
  notes: string;
  levelTags: string;
}): Promise<dto.WaveDTO> {
  assertWailsRuntime();
  return UpdateWave(dto.UpdateWaveInput.createFrom(input));
}

/** Close a wave. `force: true` closes despite residual open items (requires `note`). */
export async function closeWave(input: {
  waveId: number;
  note: string;
  force: boolean;
}): Promise<dto.CloseWaveResult> {
  assertWailsRuntime();
  return CloseWave(dto.CloseWaveInput.createFrom(input));
}

export async function listWavesFiltered(input: {
  page: number
  pageSize: number
  sortBy?: string
  sortDesc?: boolean
  lifecycleStage?: string
  nameKeyword?: string
  waveType?: string
}): Promise<dto.WavesPage> {
  if (!isWailsRuntimeAvailable()) return dto.WavesPage.createFrom({ items: [], pagination: {} })
  return ListWavesFiltered(dto.WaveListFilterInput.createFrom(input))
}

/** Unassign a demand document from a wave. */
export async function unassignDemandFromWave(input: {
  waveId: number;
  demandDocumentId: number;
}): Promise<void> {
  assertWailsRuntime();
  return UnassignDemandFromWave(dto.UnassignDemandInput.createFrom(input));
}

/** Batch-return demand documents to the unassigned pool (single undo node). Hard-fail. */
export async function batchUnassignDemandFromWave(input: {
  waveId: number
  docIds: number[]
}): Promise<dto.BatchUnassignDemandResult> {
  assertWailsRuntime()
  return BatchUnassignDemandFromWave(dto.BatchUnassignDemandInput.createFrom(input))
}

export async function mapDemandLines(
  waveId: number,
): Promise<dto.DemandMappingResult> {
  assertWailsRuntime();
  return MapDemandLines(waveId);
}

export async function getWaveOverview(
  waveId: number,
): Promise<dto.WaveOverviewDTO> {
  assertWailsRuntime();
  return GetWaveOverview(waveId);
}

export async function getWaveWorkspaceSnapshot(
  waveId: number,
): Promise<dto.WaveWorkspaceSnapshotDTO> {
  assertWailsRuntime();
  return GetWaveWorkspaceSnapshot(waveId);
}

export async function listWaveFulfillmentRows(
  waveId: number,
): Promise<dto.WaveFulfillmentRowDTO[]> {
  assertWailsRuntime();
  return ListWaveFulfillmentRows(waveId);
}

/** Server-side filtered + paginated fulfillment-line grid rows for a wave. */
export async function listWaveFulfillmentRowsFiltered(input: {
  waveId: number;
  allocationStates: string[];
  addressStates: string[];
  supplierStates: string[];
  channelSyncStates: string[];
  reviewRequirements: string[];
  driftStatuses: string[];
  keyword: string;
  pagination: PaginationInput;
}): Promise<dto.WaveFulfillmentRowsPage> {
  assertWailsRuntime();
  return ListWaveFulfillmentRowsFiltered(
    dto.WaveFulfillmentFilterInput.createFrom(input),
  );
}

export async function listWaveParticipantRows(
  waveId: number,
): Promise<dto.WaveParticipantRowDTO[]> {
  assertWailsRuntime();
  return ListWaveParticipantRows(waveId);
}

/** Assign a demand document to a wave. */
export async function assignDemandToWave(
  waveId: number,
  demandDocumentId: number,
): Promise<void> {
  assertWailsRuntime();
  return AssignDemandToWave(waveId, demandDocumentId);
}

/** Batch-assign demand documents to a wave. Partial-success — check each item's `success` flag. */
export async function batchAssignDemandToWave(input: {
  waveId: number;
  docIds: number[];
}): Promise<dto.BatchAssignDemandResult> {
  assertWailsRuntime();
  return BatchAssignDemandToWave(dto.BatchAssignDemandInput.createFrom(input));
}

/** Undo the last action for a wave. Returns the command summary of the undone action. */
export async function undoWaveAction(waveId: number): Promise<string> {
  assertWailsRuntime();
  return UndoWaveAction(waveId);
}

/** Redo the last undone action for a wave. Returns the command summary of the redone action. */
export async function redoWaveAction(waveId: number): Promise<string> {
  assertWailsRuntime();
  return RedoWaveAction(waveId);
}

// ── ExportController ──

export async function exportSupplierOrder(
  waveId: number,
): Promise<dto.SupplierOrderDTO[]> {
  assertWailsRuntime();
  return ExportSupplierOrder(waveId);
}

export async function exportSupplierOrderForProfile(
  waveId: number,
  factoryProfileId: number,
): Promise<dto.SupplierOrderDTO[]> {
  assertWailsRuntime();
  return ExportSupplierOrderForProfile(waveId, factoryProfileId);
}

export async function listSupplierOrders(): Promise<dto.SupplierOrderDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListSupplierOrders();
}

export async function getSupplierOrderByWave(
  waveId: number,
): Promise<dto.SupplierOrderDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return GetSupplierOrderByWave(waveId);
}

export async function listLinesBySupplierOrder(
  orderId: number,
): Promise<dto.SupplierOrderLineDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListLinesBySupplierOrder(orderId);
}

/**
 * Generate a downloadable factory order file for a supplier order. NOT
 * idempotent/versioned — every call writes a fresh timestamped file rather
 * than overwriting; the returned `filePath` is the latest artifact.
 */
export async function generateSupplierOrderFile(
  orderId: number,
): Promise<dto.SupplierOrderFileResultDTO> {
  assertWailsRuntime();
  return GenerateSupplierOrderFile(orderId);
}

/** Mark a `draft` supplier order as submitted to the factory. `externalOrderNo` is required. */
export async function markSupplierOrderSubmitted(input: {
  orderId: number;
  externalOrderNo: string;
  submittedAt?: string;
}): Promise<dto.SupplierOrderDTO> {
  assertWailsRuntime();
  return MarkSupplierOrderSubmitted(
    dto.MarkSupplierOrderSubmittedInput.createFrom(input),
  );
}

/**
 * Record factory acceptance for a `submitted` supplier order. Must include
 * every line of the order in one call — there is no partial/incremental
 * acceptance; the order transitions straight to `accepted`.
 */
export async function recordSupplierOrderAcceptance(input: {
  orderId: number;
  lines: Array<{ lineId: number; acceptedQuantity: number }>;
}): Promise<dto.SupplierOrderDTO> {
  assertWailsRuntime();
  return RecordSupplierOrderAcceptance(
    dto.RecordSupplierOrderAcceptanceInput.createFrom(input),
  );
}

// ── ShipmentController ──

export async function createShipment(input: {
  supplierOrderId: number
  supplierPlatform: string
  shipmentNo: string
  externalShipmentNo: string
  carrierCode: string
  carrierName: string
  trackingNo: string
  status: string
  shippedAt: string
  basisPayloadSnapshot: string
  lines: Array<{
    supplierOrderLineId: number
    fulfillmentLineId: number
    quantity: number
  }>
}): Promise<dto.ShipmentDTO> {
  assertWailsRuntime()
  const req = dto.CreateShipmentInput.createFrom(input)
  return CreateShipment(req)
}

export async function listShipmentsByWave(waveId: number): Promise<dto.ShipmentDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListShipmentsByWave(waveId)
}

export async function listShipmentsByWavePage(input: {
  waveId: number
  sortBy?: string
  sortDir?: 'asc' | 'desc'
  limit: number
  offset: number
}): Promise<dto.ShipmentPageResult> {
  if (!isWailsRuntimeAvailable()) {
    return dto.ShipmentPageResult.createFrom({ items: [], totalCount: 0 })
  }
  return ListShipmentsByWavePage(dto.ShipmentByWavePageFilterInput.createFrom(input))
}

export type ImportShipmentEntry = dto.ImportShipmentEntry

export type ImportShipmentsInput = dto.ImportShipmentInput

export type ImportShipmentsResult = dto.ImportShipmentResult

export async function importShipments(input: ImportShipmentsInput): Promise<ImportShipmentsResult> {
  assertWailsRuntime()
  return ImportShipments(dto.ImportShipmentInput.createFrom(input))
}

/**
 * Template-driven factory-return sheet import: maps external rows via the
 * profile's default shipment-import template, reconciles to internal line
 * IDs, then runs ImportShipments. Prefer `filePath` for multi-format sheets.
 */
export async function mapAndReconcileShipments(input: {
  waveId: number
  integrationProfileId: number
  importMode: string
  filePath?: string
  rows?: Record<string, string>[]
}): Promise<ImportShipmentsResult> {
  assertWailsRuntime()
  return MapAndReconcileShipments(
    dto.MapAndReconcileShipmentsInput.createFrom({
      waveId: input.waveId,
      integrationProfileId: input.integrationProfileId,
      importMode: input.importMode,
      filePath: input.filePath ?? '',
      rows: input.rows ?? [],
    }),
  )
}

/**
 * Per-line shipped/remaining quantity summary for a supplier order — feeds
 * the shipment line selector's over-ship-aware display. Server-side
 * over-ship blocking already applies unconditionally in `createShipment`/
 * `importShipments`; this is purely a pre-submission UI display.
 */
export async function getSupplierOrderLineShippedSummary(
  orderId: number,
): Promise<dto.SupplierOrderLineShippedDTO[]> {
  assertWailsRuntime()
  return GetSupplierOrderLineShippedSummary(orderId)
}

/** Correct a previously recorded shipment's header fields. Outside the undo/redo history by design. */
export async function updateShipment(input: {
  id: number
  supplierPlatform: string
  shipmentNo: string
  externalShipmentNo: string
  carrierCode: string
  carrierName: string
  trackingNo: string
  shippedAt?: string
}): Promise<dto.ShipmentDTO> {
  assertWailsRuntime()
  return UpdateShipment(dto.UpdateShipmentInput.createFrom(input))
}

/** Void a wrongly-entered shipment. Outside the undo/redo history by design. */
export async function voidShipment(input: {
  id: number
  note: string
  operatorId: string
}): Promise<dto.ShipmentDTO> {
  assertWailsRuntime()
  return VoidShipment(dto.VoidShipmentInput.createFrom(input))
}

// ── ChannelSyncController ──

export async function createChannelSyncJob(input: {
  waveId: number
  integrationProfileId: number
  direction: string
  items: Array<{
    fulfillmentLineId: number
    shipmentId: number
    externalDocumentNo: string
    externalLineNo: string
    trackingNo: string
    carrierCode: string
  }>
}): Promise<dto.ChannelSyncJobDTO> {
  assertWailsRuntime()
  const req = dto.CreateChannelSyncJobInput.createFrom(input)
  return CreateChannelSyncJob(req)
}

export async function listChannelSyncJobsByWave(waveId: number): Promise<dto.ChannelSyncJobDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListChannelSyncJobsByWave(waveId)
}

export async function planChannelClosure(input: {
  waveId: number
  integrationProfileId: number
}): Promise<dto.PlanChannelClosureResult> {
  assertWailsRuntime()
  const req = dto.PlanChannelClosureInput.createFrom(input)
  return PlanChannelClosure(req)
}

export async function executeChannelSyncJob(
  jobId: number,
): Promise<dto.ExecuteSyncResult> {
  assertWailsRuntime()
  return ExecuteChannelSyncJob(jobId)
}

export async function recordChannelClosureDecision(input: {
  waveId: number
  integrationProfileId: number
  entries: Array<{
    fulfillmentLineId: number
    decisionKind: string
    reasonCode: string
    note: string
    evidenceRef: string
    operatorId: string
  }>
}): Promise<dto.ClosureDecisionRecordDTO[]> {
  assertWailsRuntime()
  const req = dto.RecordClosureDecisionInput.createFrom(input)
  return RecordChannelClosureDecision(req)
}

export async function retryChannelSyncJob(
  jobId: number,
): Promise<dto.ExecuteSyncResult> {
  assertWailsRuntime()
  return RetryChannelSyncJob(jobId)
}

export async function listIntegrationProfiles(): Promise<dto.IntegrationProfileSummaryDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListIntegrationProfiles()
}

// ── CarrierMapping ──

export async function createCarrierMapping(input: {
  integrationProfileId: number
  internalCarrierCode: string
  externalCarrierCode: string
  externalCarrierName: string
  /** JSON string array, e.g. `["SF","顺丰"]`. */
  aliases?: string
  isDefault: boolean
}): Promise<dto.CarrierMappingDTO> {
  assertWailsRuntime()
  return CreateCarrierMapping(
    dto.CreateCarrierMappingInput.createFrom({
      ...input,
      aliases: input.aliases ?? '',
    }),
  )
}

export async function listCarrierMappings(profileId: number): Promise<dto.CarrierMappingDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListCarrierMappings(profileId)
}

export async function deleteCarrierMapping(id: number): Promise<void> {
  assertWailsRuntime()
  return DeleteCarrierMapping(id)
}

/**
 * Template-mapped carrier mapping upsert (`carrier.*` dest keys). Prefer
 * `filePath` so the backend honours hasHeader/positional rules.
 */
export async function importCarrierMappings(input: {
  integrationProfileId: number
  importMode: string
  filePath?: string
  rows?: Record<string, string>[]
}): Promise<dto.ImportCarrierMappingsResult> {
  assertWailsRuntime()
  return ImportCarrierMappings(
    dto.ImportCarrierMappingsInput.createFrom({
      integrationProfileId: input.integrationProfileId,
      importMode: input.importMode,
      filePath: input.filePath ?? '',
      rows: input.rows ?? [],
    }),
  )
}

export async function listExternalCarriers(profileId: number): Promise<dto.ExternalCarrierDTO[]> {
  assertWailsRuntime()
  return ListExternalCarriers(profileId)
}

export async function registerExternalCarrier(input: {
  integrationProfileId: number
  externalCarrierCode: string
  externalCarrierName: string
}): Promise<dto.ExternalCarrierDTO> {
  assertWailsRuntime()
  return RegisterExternalCarrier(dto.RegisterExternalCarrierInput.createFrom(input))
}

export async function bindInternalCarrier(input: {
  externalCarrierId: number
  internalCarrierCode: string
}): Promise<dto.ExternalCarrierDTO> {
  assertWailsRuntime()
  return BindInternalCarrier(dto.BindInternalCarrierInput.createFrom(input))
}

export async function listConnectorCapabilities(): Promise<Record<string, any>> {
  if (!isWailsRuntimeAvailable()) return {}
  return ListConnectorCapabilities()
}

// ── ProfileController ──

export async function listProfiles(): Promise<dto.IntegrationProfileDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListProfiles()
}

export async function getProfile(id: number): Promise<dto.IntegrationProfileDTO> {
  assertWailsRuntime()
  return GetProfile(id)
}

export async function createProfile(input: {
  profileKey: string
  sourceChannel: string
  sourceSurface: string
  demandKind: string
  initialAllocationStrategy: string
  identityStrategy: string
  entitlementAuthorityMode: string
  recipientInputMode: string
  referenceStrategy: string
  trackingSyncMode: string
  closurePolicy: string
  supportsPartialShipment: boolean
  supportsApiImport: boolean
  supportsApiExport: boolean
  requiresCarrierMapping: boolean
  requiresExternalOrderNo: boolean
  allowsManualClosure: boolean
  /** Factory-surface capability: export supplier orders. */
  supportsExportSupplierOrder?: boolean
  /** Factory-surface capability: import product catalog. */
  supportsImportProductCatalog?: boolean
  /** Factory-surface capability: import supplier shipment returns. */
  supportsImportSupplierShipment?: boolean
  connectorKey: string
  /** Factory-facing platform label (supplier orders / product catalog fallback). */
  factorySupplierPlatform?: string
  supportedLocales: string
  defaultLocale: string
  extraData: string
}): Promise<dto.IntegrationProfileDTO> {
  assertWailsRuntime()
  const req = dto.CreateProfileInput.createFrom({
    ...input,
    supportsExportSupplierOrder: input.supportsExportSupplierOrder ?? false,
    supportsImportProductCatalog: input.supportsImportProductCatalog ?? false,
    supportsImportSupplierShipment: input.supportsImportSupplierShipment ?? false,
    factorySupplierPlatform: input.factorySupplierPlatform ?? '',
  })
  return CreateProfile(req)
}

export async function updateProfile(input: {
  id: number
  profileKey: string
  sourceChannel: string
  sourceSurface: string
  demandKind: string
  initialAllocationStrategy: string
  identityStrategy: string
  entitlementAuthorityMode: string
  recipientInputMode: string
  referenceStrategy: string
  trackingSyncMode: string
  closurePolicy: string
  supportsPartialShipment: boolean
  supportsApiImport: boolean
  supportsApiExport: boolean
  requiresCarrierMapping: boolean
  requiresExternalOrderNo: boolean
  allowsManualClosure: boolean
  supportsExportSupplierOrder?: boolean
  supportsImportProductCatalog?: boolean
  supportsImportSupplierShipment?: boolean
  connectorKey: string
  factorySupplierPlatform?: string
  supportedLocales: string
  defaultLocale: string
  extraData: string
}): Promise<dto.IntegrationProfileDTO> {
  assertWailsRuntime()
  const req = dto.UpdateProfileInput.createFrom({
    ...input,
    supportsExportSupplierOrder: input.supportsExportSupplierOrder ?? false,
    supportsImportProductCatalog: input.supportsImportProductCatalog ?? false,
    supportsImportSupplierShipment: input.supportsImportSupplierShipment ?? false,
    factorySupplierPlatform: input.factorySupplierPlatform ?? '',
  })
  return UpdateProfile(req)
}

export async function deleteProfile(id: number): Promise<void> {
  assertWailsRuntime()
  return DeleteProfile(id)
}

export async function seedDefaultProfiles(): Promise<dto.IntegrationProfileDTO[]> {
  assertWailsRuntime()
  return SeedDefaultProfiles()
}

// ── ProductController ──

export async function createProductMaster(input: {
  supplierPlatform: string
  factorySku: string
  supplierProductRef: string
  name: string
  productKind: string
  coverImagePath?: string
  detailImagePaths?: string
  extraData?: string
}): Promise<dto.ProductMasterDTO> {
  assertWailsRuntime()
  const req = dto.CreateProductMasterInput.createFrom({
    ...input,
    coverImagePath: input.coverImagePath ?? '',
    detailImagePaths: input.detailImagePaths ?? '',
    extraData: input.extraData ?? '',
  })
  return CreateProductMaster(req)
}

export async function listProductMasters(): Promise<dto.ProductMasterDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListProductMasters()
}

export async function listProductMastersPage(input: {
  keyword?: string
  productKinds?: string[]
  archivedOnly?: boolean
  sortBy?: string
  sortDir?: 'asc' | 'desc'
  limit: number
  offset: number
}): Promise<dto.ProductMasterPageResult> {
  if (!isWailsRuntimeAvailable()) {
    return dto.ProductMasterPageResult.createFrom({ items: [], totalCount: 0 })
  }
  return ListProductMastersPage(dto.ProductMasterPageFilterInput.createFrom(input))
}

export async function updateProductMaster(input: {
  id: number
  supplierPlatform: string
  factorySku: string
  supplierProductRef: string
  name: string
  productKind: string
  archived: boolean
  coverImagePath?: string
  detailImagePaths?: string
  extraData?: string
}): Promise<dto.ProductMasterDTO> {
  assertWailsRuntime()
  const req = dto.UpdateProductMasterInput.createFrom({
    ...input,
    coverImagePath: input.coverImagePath ?? '',
    detailImagePaths: input.detailImagePaths ?? '',
    extraData: input.extraData ?? '',
  })
  return UpdateProductMaster(req)
}

export async function snapshotProductsForWave(input: {
  waveId: number
  masterIds: number[]
}): Promise<dto.ProductDTO[]> {
  assertWailsRuntime()
  const req = dto.SnapshotProductsInput.createFrom(input)
  return SnapshotProductsForWave(req)
}

export async function listProductsByWave(waveId: number): Promise<dto.ProductDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListProductsByWave(waveId)
}

/**
 * Template-mapped ProductMaster upsert (`product.*` dest keys, document type
 * `import_product_catalog`). Prefer `filePath` for multi-format sheets.
 */
export async function importProductCatalog(input: {
  integrationProfileId: number
  importMode: string
  filePath?: string
  rows?: Record<string, string>[]
}): Promise<dto.ImportProductCatalogResult> {
  assertWailsRuntime()
  return ImportProductCatalog(
    dto.ImportProductCatalogInput.createFrom({
      integrationProfileId: input.integrationProfileId,
      importMode: input.importMode,
      filePath: input.filePath ?? '',
      rows: input.rows ?? [],
    }),
  )
}

/**
 * Batch-stock-to-wave with per-item created/skipped detail (dedup-aware).
 * Prefer this over `snapshotProductsForWave` for any UI that needs to show
 * "N created, M already existed" feedback.
 */
export async function snapshotProductsForWaveDetailed(input: {
  waveId: number
  masterIds: number[]
}): Promise<dto.SnapshotProductsDetailedResult> {
  assertWailsRuntime()
  const req = dto.SnapshotProductsInput.createFrom(input)
  return SnapshotProductsForWaveDetailed(req)
}

// ── AllocationPolicyController ──

import {
  CreateAllocationPolicyRule,
  UpdateAllocationPolicyRule,
  DeleteAllocationPolicyRule,
  ListAllocationPolicyRules,
  ReconcileWave,
} from "../../../wailsjs/go/main/AllocationPolicyController";

import type {
  AllocationPolicyRule,
  CreateAllocationPolicyRuleInput,
  UpdateAllocationPolicyRuleInput,
  ReconcileResult,
} from "@/entities/allocation-policy"

export async function listAllocationPolicyRules(
  waveID: number,
): Promise<AllocationPolicyRule[]> {
  if (!isWailsRuntimeAvailable()) return []
  // DTO.selectorPayload is codegen'd as number[] (json.RawMessage → []byte) but
  // is a SelectorPayload object at runtime, so go through unknown.
  return ListAllocationPolicyRules(waveID) as unknown as Promise<AllocationPolicyRule[]>
}

export async function createAllocationPolicyRule(
  input: CreateAllocationPolicyRuleInput,
): Promise<AllocationPolicyRule> {
  assertWailsRuntime()
  return CreateAllocationPolicyRule(input as any) as unknown as Promise<AllocationPolicyRule>
}

export async function updateAllocationPolicyRule(
  input: UpdateAllocationPolicyRuleInput,
): Promise<AllocationPolicyRule> {
  assertWailsRuntime()
  return UpdateAllocationPolicyRule(input as any) as unknown as Promise<AllocationPolicyRule>
}

export async function deleteAllocationPolicyRule(ruleID: number): Promise<void> {
  assertWailsRuntime()
  return DeleteAllocationPolicyRule(ruleID)
}

export async function reconcileWave(waveID: number): Promise<ReconcileResult> {
  assertWailsRuntime()
  return ReconcileWave(waveID) as Promise<ReconcileResult>
}

export async function listAssignedDemandsByWave(
  waveId: number,
): Promise<dto.DemandDocumentDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListAssignedDemandsByWave(waveId);
}

export async function generateParticipants(waveID: number): Promise<number> {
  assertWailsRuntime()
  return GenerateParticipants(waveID)
}

// ── AdjustmentController ──

export async function listAdjustmentsByWave(
  waveId: number,
): Promise<dto.FulfillmentAdjustmentDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListAdjustmentsByWave(waveId);
}

// ── History ──

export async function listRecentHistory(
  waveId: number,
  limit: number = 10,
): Promise<dto.HistoryNodeDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListRecentHistory(waveId, limit)
}

export async function getHistoryGraph(waveId: number): Promise<dto.HistoryGraphDTO> {
  assertWailsRuntime()
  return GetHistoryGraph(waveId)
}


export async function runHistoryGC(waveId: number): Promise<number> {
  assertWailsRuntime()
  return RunHistoryGC(waveId)
}

export async function recordAdjustment(
  input: {
    waveId: number;
    targetKind: string;
    fulfillmentLineId?: number | null;
    waveParticipantSnapshotId?: number | null;
    adjustmentKind: string;
    quantityDelta: number;
    reasonCode: string;
    operatorId: string;
    note: string;
    evidenceRef: string;
    fromProductId?: number | null;
    toProductId?: number | null;
  },
): Promise<dto.FulfillmentAdjustmentDTO> {
  assertWailsRuntime();
  return RecordAdjustment(dto.RecordAdjustmentInput.createFrom(input));
}

/** Batch-record adjustments. Partial-success — check each item's `success` flag. */
export async function batchRecordAdjustments(input: {
  entries: Array<{
    waveId: number;
    targetKind: string;
    fulfillmentLineId?: number | null;
    waveParticipantSnapshotId?: number | null;
    adjustmentKind: string;
    quantityDelta: number;
    reasonCode: string;
    operatorId: string;
    note: string;
    evidenceRef: string;
    fromProductId?: number | null;
    toProductId?: number | null;
  }>;
}): Promise<dto.BatchRecordAdjustmentsResult> {
  assertWailsRuntime();
  return BatchRecordAdjustments(
    dto.BatchRecordAdjustmentsInput.createFrom(input),
  );
}

// ── TemplateController ──

import {
  CreateDocumentTemplate,
  UpdateDocumentTemplate,
  DeleteDocumentTemplate,
  ListDocumentTemplates,
  BindTemplateToProfile,
  UnbindTemplate,
  SetDefaultBinding,
  ListBindingsByProfile,
  GetDefaultTemplateForProfile,
} from "../../../wailsjs/go/main/TemplateController";

export async function createDocumentTemplate(input: {
  templateKey: string
  documentType: string
  format: string
  mappingRules: string
  extraData: string
}): Promise<dto.DocumentTemplateDTO> {
  assertWailsRuntime()
  const req = dto.CreateDocumentTemplateInput.createFrom(input)
  return CreateDocumentTemplate(req)
}

export async function updateDocumentTemplate(input: {
  id: number
  format: string
  mappingRules: string
  extraData: string
}): Promise<dto.DocumentTemplateDTO> {
  assertWailsRuntime()
  const req = dto.UpdateDocumentTemplateInput.createFrom(input)
  return UpdateDocumentTemplate(req)
}

export async function deleteDocumentTemplate(id: number): Promise<void> {
  assertWailsRuntime()
  return DeleteDocumentTemplate(id)
}

export async function listDocumentTemplates(): Promise<dto.DocumentTemplateDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListDocumentTemplates()
}

export async function bindTemplateToProfile(input: {
  integrationProfileId: number
  documentType: string
  templateId: number
  isDefault: boolean
}): Promise<dto.ProfileTemplateBindingDTO> {
  assertWailsRuntime()
  const req = dto.BindTemplateToProfileInput.createFrom(input)
  return BindTemplateToProfile(req)
}

export async function unbindTemplate(bindingId: number): Promise<void> {
  assertWailsRuntime()
  return UnbindTemplate(bindingId)
}

export async function setDefaultBinding(bindingId: number): Promise<void> {
  assertWailsRuntime()
  return SetDefaultBinding(bindingId)
}

export async function listBindingsByProfile(profileId: number): Promise<dto.ProfileTemplateBindingDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListBindingsByProfile(profileId)
}

export async function getDefaultTemplateForProfile(profileId: number, docType: string): Promise<dto.DocumentTemplateDTO | null> {
  assertWailsRuntime()
  // Codegen types this as non-null dto.DocumentTemplateDTO, but the Go method returns a typed
  // nil pointer (no default binding exists), which serializes to a real JSON `null` — cast through.
  return GetDefaultTemplateForProfile(profileId, docType) as unknown as Promise<dto.DocumentTemplateDTO | null>
}

// ── DemandController (CSV import) ──

export async function importDemandFromCSV(input: {
  integrationProfileId: number
  documentType: string
  sourceDocumentNo: string
  sourceCustomerRef: string
  rows: Record<string, string>[]
}): Promise<dto.DemandDocumentDTO> {
  assertWailsRuntime()
  return ImportDemandFromCSV(dto.ImportDemandTemplateInput.createFrom(input))
}

// ── DemandController (routing management) ──

export async function updateDemandLineRouting(input: {
  demandLineId: number
  routingDisposition: string
  recipientInputState: string
  routingReasonCode: string
}): Promise<void> {
  assertWailsRuntime()
  return UpdateDemandLineRouting(dto.UpdateDemandLineRoutingInput.createFrom(input))
}

export async function batchUpdateDemandLineRouting(input: {
  updates: Array<{
    demandLineId: number
    routingDisposition: string
    recipientInputState: string
    routingReasonCode: string
  }>
}): Promise<dto.BatchUpdateDemandLineRoutingResult> {
  assertWailsRuntime()
  return BatchUpdateDemandLineRouting(dto.BatchUpdateDemandLineRoutingInput.createFrom(input))
}

export async function getWaveRoutingStats(waveId: number): Promise<dto.WaveRoutingStatsDTO> {
  assertWailsRuntime()
  return GetWaveRoutingStats(waveId)
}

// ── App (utility) ──

export async function pickCsvFile(): Promise<string> {
  assertWailsRuntime();
  return PickCSVFile();
}

/**
 * Native dialog for CSV / XLSX / XLS. Falls back to `PickCSVFile` when the
 * generated binding is missing (older runtime without `PickTabularFile`).
 */
export async function pickTabularFile(): Promise<string> {
  assertWailsRuntime();
  try {
    if (typeof PickTabularFile === 'function') {
      return await PickTabularFile();
    }
  } catch {
    // Runtime may not expose PickTabularFile yet — fall through.
  }
  // Fallback: CSV-only picker.
  return PickCSVFile();
}

export async function pickZipFile(): Promise<string> {
  assertWailsRuntime();
  return PickZIPFile();
}

/**
 * Native dialog for product-catalog imports: ZIP (with images) or CSV / XLSX / XLS.
 * Falls back to `pickTabularFile` when the generated binding is missing.
 */
export async function pickCatalogImportFile(): Promise<string> {
  assertWailsRuntime();
  try {
    if (typeof PickCatalogImportFile === 'function') {
      return await PickCatalogImportFile();
    }
  } catch {
    // Runtime may not expose PickCatalogImportFile yet — fall through.
  }
  return pickTabularFile();
}

export async function saveZoom(zoomPercent: number): Promise<void> {
  if (!isWailsRuntimeAvailable()) return;
  await SaveZoom(zoomPercent);
}

// ── FileSystemController ──

/**
 * Reveal a server-generated file in the OS file manager (Explorer/Finder),
 * pre-selected. `path` always originates server-side (e.g.
 * `SupplierOrderFileResultDTO.filePath`, or the `output_file` key parsed out
 * of a `ChannelSyncJob.responsePayload`) — never raw user input.
 */
export async function revealInFolder(path: string): Promise<void> {
  assertWailsRuntime();
  return RevealInFolder(path);
}

/**
 * Resolved data directory path (see `internal/service/path_service.go`'s
 * three-tier dev/portable/system resolution). Hard-fail — a missing/
 * unresolvable data dir is a real error state, not a soft-fail candidate.
 */
export async function getDataDir(): Promise<string> {
  assertWailsRuntime();
  return GetDataDir();
}

// ── AddressController ──

export async function createAddress(input: {
  customerProfileId: number
  label: string
  recipientName: string
  phone: string
  country: string
  province: string
  city: string
  district: string
  addressLine1: string
  addressLine2: string
  postalCode: string
  isDefault: boolean
  isTest: boolean
  validationStatus: string
  validationDetail: string
  extraData: string
}): Promise<dto.CustomerAddressDTO> {
  assertWailsRuntime()
  const req = dto.CreateAddressInput.createFrom(input)
  return CreateAddress(req)
}

export async function updateAddress(input: {
  id: number
  customerProfileId: number
  label: string
  recipientName: string
  phone: string
  country: string
  province: string
  city: string
  district: string
  addressLine1: string
  addressLine2: string
  postalCode: string
  isDefault: boolean
  isTest: boolean
  validationStatus: string
  validationDetail: string
  extraData: string
}): Promise<dto.CustomerAddressDTO> {
  assertWailsRuntime()
  const req = dto.UpdateAddressInput.createFrom(input)
  return UpdateAddress(req)
}

export async function deleteAddress(id: number): Promise<void> {
  assertWailsRuntime()
  return DeleteAddress(id)
}

export async function getAddress(id: number): Promise<dto.CustomerAddressDTO> {
  assertWailsRuntime()
  return GetAddress(id)
}

export async function listAddressesByProfile(profileID: number): Promise<dto.CustomerAddressDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListAddressesByProfile(profileID)
}

export async function bindAddressToLine(input: {
  fulfillmentLineId: number
  customerAddressId: number
}): Promise<dto.CustomerAddressDTO> {
  assertWailsRuntime()
  const req = dto.BindAddressInput.createFrom(input)
  return BindAddressToLine(req)
}

export async function unbindAddressFromLine(fulfillmentLineID: number): Promise<void> {
  assertWailsRuntime()
  return UnbindAddressFromLine(fulfillmentLineID)
}

/** Batch-bind an existing address to a set of fulfillment lines. Partial-success. */
export async function batchBindAddressToLines(
  entries: Array<{ fulfillmentLineId: number; customerAddressId: number }>,
): Promise<dto.AddressBatchItemResult[]> {
  assertWailsRuntime()
  return BatchBindAddressToLines(
    entries.map((e) => dto.BindAddressEntry.createFrom(e)),
  )
}

/** One-click: bind the default recipient address to every address-missing line in the wave. */
export async function bindDefaultAddressesForWave(
  waveId: number,
): Promise<dto.AddressBatchItemResult[]> {
  assertWailsRuntime()
  return BindDefaultAddressesForWave(waveId)
}

// ── Paginated waves ──

export interface PaginationInput {
  page: number
  pageSize: number
  sortBy: string
  sortDesc: boolean
}

export interface PaginationResult {
  page: number
  pageSize: number
  totalCount: number
  totalPages: number
}

export async function listWavesPaginated(input: PaginationInput): Promise<{
  items: dto.WaveDTO[]
  pagination: PaginationResult
}> {
  if (!isWailsRuntimeAvailable()) return { items: [], pagination: { page: 1, pageSize: 50, totalCount: 0, totalPages: 0 } }
  return ListWavesPaginated(dto.PaginationInput.createFrom(input)) as Promise<{
    items: dto.WaveDTO[]
    pagination: PaginationResult
  }>
}

export async function validateStepAccess(waveId: number, stepKey: string): Promise<void> {
  assertWailsRuntime()
  return ValidateStepAccess(waveId, stepKey)
}

// ── MergeController ──

import {
  ExecuteCustomerMerge as _ExecuteCustomerMerge,
  GetCustomerMergeHistory as _GetCustomerMergeHistory,
  ListCustomerMergeHistory as _ListCustomerMergeHistory,
  PreviewCustomerMerge as _PreviewCustomerMerge,
} from "../../../wailsjs/go/main/MergeController";
import {
  DryRunCustomerMergeUndo as _DryRunCustomerMergeUndo,
  ExecuteCustomerMergeUndo as _ExecuteCustomerMergeUndo,
} from "../../../wailsjs/go/main/MergeUndoController";

export interface CustomerMergePreviewRequest {
  sourceProfileId: number
  targetProfileId: number
  candidateId?: number
  primaryIdentitySelections?: Array<{ namespace: string; identityType: string; identityId: number }>
  defaultAddressId?: number
  displayNameResolution?: string
}

export async function previewCustomerMerge(input: CustomerMergePreviewRequest): Promise<dto.CustomerMergePreviewResult> {
  assertWailsRuntime()
  return _PreviewCustomerMerge(dto.CustomerMergePreviewInput.createFrom({
    ...input,
    primaryIdentitySelections: input.primaryIdentitySelections ?? [],
    displayNameResolution: input.displayNameResolution ?? 'keep_target',
  }))
}

export interface ExecuteCustomerMergeRequest {
  operationKey: string
  previewToken: string
  sourceProfileId: number
  targetProfileId: number
  expectedSourceRowVersion: number
  expectedTargetRowVersion: number
  candidateId?: number
  expectedCandidateRowVersion: number
  expectedEvidenceHash: string
  expectedPolicyVersion: number
  expectedPolicyRevisionId?: number
  primaryIdentitySelections?: Array<{ namespace: string; identityType: string; identityId: number }>
  defaultAddressId?: number
  displayNameResolution: string
  actorRef: string
  decisionReason: string
}

export async function executeCustomerMerge(input: ExecuteCustomerMergeRequest): Promise<dto.ExecuteCustomerMergeResult> {
  assertWailsRuntime()
  return _ExecuteCustomerMerge(dto.ExecuteCustomerMergeInput.createFrom({
    ...input,
    primaryIdentitySelections: input.primaryIdentitySelections ?? [],
  }))
}

export async function listCustomerMergeHistory(input: {
  profileId: number
  candidateId?: number
  status?: string
  beforeCreatedAt?: string
  beforeId?: number
  limit?: number
}): Promise<dto.CustomerMergeHistoryPage> {
  assertWailsRuntime()
  return _ListCustomerMergeHistory(dto.CustomerMergeHistoryQuery.createFrom({
    profileId: input.profileId,
    candidateId: input.candidateId ?? 0,
    status: input.status ?? '',
    beforeCreatedAt: input.beforeCreatedAt,
    beforeId: input.beforeId ?? 0,
    limit: input.limit ?? 50,
  }))
}

export async function getCustomerMergeHistory(mergeId: number): Promise<dto.CustomerMergeHistoryDetail> {
  assertWailsRuntime()
  return _GetCustomerMergeHistory(mergeId)
}

export async function dryRunCustomerMergeUndo(mergeId: number): Promise<dto.CustomerMergeUndoDryRunResult> {
  assertWailsRuntime()
  return _DryRunCustomerMergeUndo(dto.CustomerMergeUndoDryRunInput.createFrom({ mergeId }))
}

export async function executeCustomerMergeUndo(input: {
  mergeId: number
  undoOperationKey: string
  eligibilityToken: string
  expectedSourceRowVersion: number
  expectedTargetRowVersion: number
  actorRef: string
  reason: string
}): Promise<dto.ExecuteCustomerMergeUndoResult> {
  assertWailsRuntime()
  return _ExecuteCustomerMergeUndo(dto.ExecuteCustomerMergeUndoInput.createFrom(input))
}

// ── CustomerProfileController ──

export async function listCustomerProfiles(keyword: string = "", platform: string = "", missingAddressOnly: boolean = false): Promise<dto.CustomerProfileDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListCustomerProfiles(keyword, platform, missingAddressOnly)
}

export async function listCustomerProfilesPage(input: {
  keyword?: string
  platform?: string
  missingAddressOnly?: boolean
  sortBy?: string
  sortDir?: 'asc' | 'desc'
  limit: number
  offset: number
}): Promise<dto.CustomerProfilePageResult> {
  if (!isWailsRuntimeAvailable()) {
    return dto.CustomerProfilePageResult.createFrom({ items: [], totalCount: 0 })
  }
  return ListCustomerProfilesPage(dto.CustomerProfilePageFilterInput.createFrom(input))
}

export async function listCustomerIdentityPlatforms(): Promise<string[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListCustomerIdentityPlatforms()
}

export async function getCustomerProfile(id: number): Promise<dto.CustomerProfileDTO> {
  assertWailsRuntime()
  return GetCustomerProfile(id)
}

export async function createCustomerProfile(input: {
  displayName: string
  profileType: string
  extraData: string
}): Promise<dto.CustomerProfileDTO> {
  assertWailsRuntime()
  return CreateCustomerProfile(dto.CreateCustomerProfileInput.createFrom(input))
}

export async function updateCustomerProfile(input: {
  id: number
  displayName: string
  profileType: string
  extraData: string
  expectedRowVersion: number
  actorRef: string
  idempotencyKey: string
}): Promise<dto.CustomerProfileDTO> {
  assertWailsRuntime()
  return UpdateCustomerProfile(dto.UpdateCustomerProfileInput.createFrom(input))
}

export async function deleteCustomerProfile(id: number): Promise<void> {
  assertWailsRuntime()
  return DeleteCustomerProfile(id)
}

export async function addCustomerIdentity(input: {
  customerProfileId: number
  identityPlatform: string
  identityValue: string
  identityType: string
  isPrimary: boolean
  extraData: string
}): Promise<dto.CustomerIdentityDTO> {
  assertWailsRuntime()
  return AddCustomerIdentity(dto.CreateCustomerIdentityInput.createFrom(input))
}

export async function deleteCustomerIdentity(id: number): Promise<void> {
  assertWailsRuntime()
  return DeleteCustomerIdentity(id)
}

export async function listCustomerNameObservations(profileId: number): Promise<dto.CustomerNameObservationDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListCustomerNameObservations(profileId)
}

export async function listCustomerProfileOrigins(profileId: number): Promise<dto.CustomerProfileOriginDTO[]> {
  assertWailsRuntime()
  return ListCustomerProfileOrigins(profileId)
}

export async function pinCustomerDisplayName(input: {
  profileId: number
  name: string
  expectedRowVersion: number
  actorRef: string
  idempotencyKey: string
}): Promise<dto.CustomerProfileDTO> {
  assertWailsRuntime()
  return PinCustomerDisplayName(dto.PinCustomerDisplayNameInput.createFrom(input))
}

export async function unpinCustomerDisplayName(input: {
  profileId: number
  expectedRowVersion: number
  actorRef: string
  idempotencyKey: string
}): Promise<dto.CustomerProfileDTO> {
  assertWailsRuntime()
  return UnpinCustomerDisplayName(dto.UnpinCustomerDisplayNameInput.createFrom(input))
}

// ── MergeGovernanceController ──

export async function getMergePolicy(): Promise<dto.MergePolicyDTO> {
  assertWailsRuntime()
  return GetMergePolicy()
}

export async function updateMergePolicy(input: {
  expectedRevision: number
  rules: {
    candidateDetectionEnabled: boolean
    emailEvidenceMode: string
    phoneEvidenceMode: string
    executionMode: string
  }
  actorRef: string
}): Promise<dto.MergePolicyDTO> {
  assertWailsRuntime()
  return UpdateMergePolicy(dto.UpdateMergePolicyInput.createFrom(input))
}

export async function scanMergeCandidates(): Promise<dto.MergeScanRunDTO> {
  assertWailsRuntime()
  return ScanMergeCandidates()
}

export async function getMergeScanRun(id: number): Promise<dto.MergeScanRunDTO> {
  assertWailsRuntime()
  return GetMergeScanRun(id)
}

export async function listMergeCandidates(status: string = ''): Promise<dto.MergeCandidateDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListMergeCandidates(status)
}

export async function getMergeCandidate(id: number): Promise<dto.MergeCandidateDTO> {
  assertWailsRuntime()
  return GetMergeCandidate(id)
}

export async function dismissMergeCandidate(input: {
  id: number
  evidenceHash: string
  policyVersion: number
}): Promise<void> {
  assertWailsRuntime()
  return DismissMergeCandidate(dto.DismissMergeCandidateInput.createFrom(input))
}

// ── CustomerResolutionFeaturePolicyController ──

export async function getCustomerResolutionFeaturePolicy(): Promise<dto.CustomerResolutionFeaturePolicyDTO> {
  assertWailsRuntime()
  return GetCustomerResolutionFeaturePolicy()
}

export async function updateCustomerResolutionFeaturePolicy(input: {
  expectedRevision: number
  customerResolutionWritesEnabled: boolean
  candidateScanEnabled: boolean
  mergeExecutionEnabled: boolean
  splitExecutionEnabled: boolean
  importEvidenceEnabled: boolean
  carrierRegistryWritesEnabled: boolean
  actorRef: string
  reason: string
}): Promise<dto.CustomerResolutionFeaturePolicyDTO> {
  assertWailsRuntime()
  return UpdateCustomerResolutionFeaturePolicy(dto.UpdateCustomerResolutionFeaturePolicyInput.createFrom(input))
}

// ── ImportEvidenceController ──

export async function listImportRuns(limit = 100): Promise<dto.ImportRunSummaryDTO[]> {
  assertWailsRuntime()
  return ListImportRuns(limit)
}

export async function listImportRunsPage(input: {
  limit?: number
  cursor?: string
  status?: string
  profileId?: number
  documentType?: string
} = {}): Promise<dto.ImportRunPageDTO> {
  assertWailsRuntime()
  return ListImportRunsPage(dto.ListImportRunsPageInput.createFrom({
    limit: input.limit ?? 50,
    cursor: input.cursor ?? '',
    status: input.status ?? '',
    profileId: input.profileId,
    documentType: input.documentType ?? '',
  }))
}

export async function getImportRunDetail(id: number): Promise<dto.ImportRunDetailDTO> {
  assertWailsRuntime()
  return GetImportRunDetail(id)
}

export async function getImportEvidenceRetention(): Promise<dto.ImportEvidenceRetentionDTO> {
  assertWailsRuntime()
  return GetImportEvidenceRetention()
}

export async function setImportEvidenceRetention(retentionDays: number): Promise<dto.ImportEvidenceRetentionDTO> {
  assertWailsRuntime()
  return SetImportEvidenceRetention(dto.SetImportEvidenceRetentionInput.createFrom({ retentionDays }))
}

export async function pruneExpiredImportEvidence(): Promise<Record<string, number>> {
  assertWailsRuntime()
  return PruneExpiredImportEvidence()
}

// ── SplitController ──

export interface CustomerSplitPlanRequest {
  sourceProfileId: number
  targetStrategy: string
  newProfileDisplayName: string
  newProfileType: string
  targetPrimaryIdentityIds?: number[]
  targetDefaultAddressId?: number
  targetDisplayNameObservationId?: number
  sourceDisplayNameResolution: string
  selection: {
    identityIds?: number[]
    addressIds?: number[]
    demandDocumentIds?: number[]
    nameObservationIds?: number[]
    originIds?: number[]
  }
}

function splitPlanDTO(input: CustomerSplitPlanRequest): dto.CustomerSplitPreviewInput {
  return dto.CustomerSplitPreviewInput.createFrom({
    ...input,
    targetPrimaryIdentityIds: input.targetPrimaryIdentityIds ?? [],
    selection: dto.CustomerSplitSelection.createFrom({
      identityIds: input.selection.identityIds ?? [],
      addressIds: input.selection.addressIds ?? [],
      demandDocumentIds: input.selection.demandDocumentIds ?? [],
      nameObservationIds: input.selection.nameObservationIds ?? [],
      originIds: input.selection.originIds ?? [],
    }),
  })
}

export async function previewCustomerSplit(input: CustomerSplitPlanRequest): Promise<dto.CustomerSplitPreviewResult> {
  assertWailsRuntime()
  return PreviewCustomerSplit(splitPlanDTO(input))
}

export async function executeCustomerSplit(input: {
  operationKey: string
  planToken: string
  expectedSourceRowVersion: number
  expectedTargetRowVersion: number
  actorRef: string
  decisionReason: string
  plan: CustomerSplitPlanRequest
}): Promise<dto.ExecuteCustomerSplitResult> {
  assertWailsRuntime()
  return ExecuteCustomerSplit(dto.ExecuteCustomerSplitInput.createFrom({
    ...input,
    plan: splitPlanDTO(input.plan),
  }))
}

export async function listCustomerSplitHistory(input: {
  profileId: number
  status?: string
  beforeCreatedAt?: string
  beforeId?: number
  limit?: number
}): Promise<dto.CustomerSplitHistoryPage> {
  assertWailsRuntime()
  return ListCustomerSplitHistory(dto.CustomerSplitHistoryQuery.createFrom({
    profileId: input.profileId,
    status: input.status ?? '',
    beforeCreatedAt: input.beforeCreatedAt,
    beforeId: input.beforeId ?? 0,
    limit: input.limit ?? 50,
  }))
}

export async function getCustomerSplitHistory(splitId: number): Promise<dto.CustomerSplitHistoryDetail> {
  assertWailsRuntime()
  return GetCustomerSplitHistory(splitId)
}

/** Cross-wave fulfillment history for a customer profile. Soft-fail — returns []. */
export async function getCustomerFulfillmentHistory(
  customerProfileId: number,
): Promise<dto.CustomerFulfillmentHistoryRowDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return GetCustomerFulfillmentHistory(customerProfileId)
}

export { isWailsRuntimeAvailable, assertWailsRuntime };
