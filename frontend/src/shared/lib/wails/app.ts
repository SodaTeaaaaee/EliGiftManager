// Bridge: strong-typed thin wrappers over generated Wails bindings.
// Never import from "wailsjs" directly outside this file.

import {
  GetDemandDocument,
  ImportDemandDocument,
  ListDemandInboxRows,
  ListDemandDocuments,
  ListDemandLines,
  ListUnassignedDemandDocuments,
} from "../../../../wailsjs/go/main/DemandController";
import {
  CreateWave,
  ListWaves,
  ListWaveDashboardRows,
  GetWave,
  GetWaveOverview,
  GetWaveWorkspaceSnapshot,
  ListWaveFulfillmentRows,
  ListWaveParticipantRows,
  MapDemandLines,
  AssignDemandToWave,
  GenerateParticipants,
  ListAssignedDemandsByWave,
  UndoWaveAction,
  RedoWaveAction,
  ListRecentHistory,
} from "../../../../wailsjs/go/main/WaveController";
import {
  ExportSupplierOrder,
  GetSupplierOrderByWave,
  ListLinesBySupplierOrder,
  ListSupplierOrders,
} from "../../../../wailsjs/go/main/ExportController";
import {
  ListAdjustmentsByWave,
  RecordAdjustment,
} from "../../../../wailsjs/go/main/AdjustmentController";
import {
  CreateShipment,
  ListShipmentsByWave,
} from "../../../../wailsjs/go/main/ShipmentController";

// ImportShipments is not yet in the generated binding; call via runtime bridge directly.
function _ImportShipments(req: unknown): Promise<unknown> {
  return (window as any).go.main.ShipmentController.ImportShipments(req);
}
import {
  CreateChannelSyncJob,
  ExecuteChannelSyncJob,
  ListChannelSyncJobsByWave,
  ListIntegrationProfiles,
  PlanChannelClosure,
  RecordChannelClosureDecision,
  RetryChannelSyncJob,
} from "../../../../wailsjs/go/main/ChannelSyncController";
import {
  CreateProfile,
  DeleteProfile,
  GetProfile,
  ListProfiles,
  SeedDefaultProfiles,
  UpdateProfile,
} from "../../../../wailsjs/go/main/ProfileController";
import {
  CreateProductMaster,
  ListProductMasters,
  ListProductsByWave,
  SnapshotProductsForWave,
  UpdateProductMaster,
} from "../../../../wailsjs/go/main/ProductController";
import {
  PickCSVFile,
  PickZIPFile,
  SaveZoom,
} from "../../../../wailsjs/go/main/App";
import { dto } from "../../../../wailsjs/go/models";

// ── Guards ──

function isWailsRuntimeAvailable(): boolean {
  return typeof window !== "undefined" && !!(window as any).go;
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

export async function listDemandInboxRows(input: {
  assignment?: string;
  demandKind?: string;
}): Promise<dto.DemandInboxRowDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListDemandInboxRows(dto.DemandInboxFilterInput.createFrom(input));
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

export async function createWave(name: string): Promise<dto.WaveDTO> {
  assertWailsRuntime();
  return CreateWave(new dto.CreateWaveInput({ name }));
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
): Promise<dto.SupplierOrderDTO> {
  assertWailsRuntime();
  return ExportSupplierOrder(waveId);
}

export async function listSupplierOrders(): Promise<dto.SupplierOrderDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListSupplierOrders();
}

export async function getSupplierOrderByWave(
  waveId: number,
): Promise<dto.SupplierOrderDTO> {
  if (!isWailsRuntimeAvailable()) return new dto.SupplierOrderDTO();
  return GetSupplierOrderByWave(waveId);
}

export async function listLinesBySupplierOrder(
  orderId: number,
): Promise<dto.SupplierOrderLineDTO[]> {
  if (!isWailsRuntimeAvailable()) return [];
  return ListLinesBySupplierOrder(orderId);
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

export interface ImportShipmentEntry {
  supplierOrderLineId: number
  fulfillmentLineId: number
  externalShipmentNo: string
  carrierCode: string
  carrierName: string
  trackingNo: string
  quantity: number
  shippedAt: string
}

export interface ImportShipmentsInput {
  waveId: number
  integrationProfileId: number
  entries: ImportShipmentEntry[]
}

export interface ImportShipmentsResult {
  createdShipments: dto.ShipmentDTO[]
  errors: Array<{ entryIndex: number; reason: string }>
  totalProcessed: number
  successCount: number
  errorCount: number
}

export async function importShipments(input: ImportShipmentsInput): Promise<ImportShipmentsResult> {
  assertWailsRuntime()
  return _ImportShipments(input) as Promise<ImportShipmentsResult>
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
  isDefault: boolean
}): Promise<any> {
  assertWailsRuntime()
  return (window as any).go.main.ChannelSyncController.CreateCarrierMapping(input)
}

export async function listCarrierMappings(profileId: number): Promise<any[]> {
  if (!isWailsRuntimeAvailable()) return []
  return (window as any).go.main.ChannelSyncController.ListCarrierMappings(profileId)
}

export async function deleteCarrierMapping(id: number): Promise<void> {
  assertWailsRuntime()
  return (window as any).go.main.ChannelSyncController.DeleteCarrierMapping(id)
}

export async function listConnectorCapabilities(): Promise<Record<string, any>> {
  if (!isWailsRuntimeAvailable()) return {}
  return (window as any).go.main.ChannelSyncController.ListConnectorCapabilities()
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
  connectorKey: string
  supportedLocales: string
  defaultLocale: string
  extraData: string
}): Promise<dto.IntegrationProfileDTO> {
  assertWailsRuntime()
  const req = dto.CreateProfileInput.createFrom(input)
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
  connectorKey: string
  supportedLocales: string
  defaultLocale: string
  extraData: string
}): Promise<dto.IntegrationProfileDTO> {
  assertWailsRuntime()
  const req = dto.UpdateProfileInput.createFrom(input)
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
}): Promise<dto.ProductMasterDTO> {
  assertWailsRuntime()
  const req = dto.CreateProductMasterInput.createFrom(input)
  return CreateProductMaster(req)
}

export async function listProductMasters(): Promise<dto.ProductMasterDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListProductMasters()
}

export async function updateProductMaster(input: {
  id: number
  supplierPlatform: string
  factorySku: string
  supplierProductRef: string
  name: string
  productKind: string
  archived: boolean
}): Promise<dto.ProductMasterDTO> {
  assertWailsRuntime()
  const req = dto.UpdateProductMasterInput.createFrom(input)
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

// ── AllocationPolicyController ──

import {
  CreateAllocationPolicyRule,
  UpdateAllocationPolicyRule,
  DeleteAllocationPolicyRule,
  ListAllocationPolicyRules,
  ReconcileWave,
} from "../../../../wailsjs/go/main/AllocationPolicyController";

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
  return ListAllocationPolicyRules(waveID) as Promise<AllocationPolicyRule[]>
}

export async function createAllocationPolicyRule(
  input: CreateAllocationPolicyRuleInput,
): Promise<AllocationPolicyRule> {
  assertWailsRuntime()
  return CreateAllocationPolicyRule(input as any) as Promise<AllocationPolicyRule>
}

export async function updateAllocationPolicyRule(
  input: UpdateAllocationPolicyRuleInput,
): Promise<AllocationPolicyRule> {
  assertWailsRuntime()
  return UpdateAllocationPolicyRule(input as any) as Promise<AllocationPolicyRule>
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

export interface HistoryNodeDTO {
  id: number
  parentNodeId: number
  preferredRedoChildId: number
  commandKind: string
  commandSummary: string
  projectionHash: string
  checkpointHint: boolean
  createdAt: string
  createdBy: string
}

export async function listRecentHistory(
  waveId: number,
  limit: number = 10,
): Promise<HistoryNodeDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListRecentHistory(waveId, limit) as Promise<HistoryNodeDTO[]>
}

export interface HistoryGraphNodeDTO {
  id: number
  parentNodeId: number
  preferredRedoChildId: number
  commandKind: string
  commandSummary: string
  projectionHash: string
  checkpointHint: boolean
  createdAt: string
  createdBy: string
  isCurrentHead: boolean
  isPinned: boolean
  childCount: number
}

export interface HistoryGraphDTO {
  scopeId: number
  currentHeadId: number
  nodes: HistoryGraphNodeDTO[]
}

export async function getHistoryGraph(waveId: number): Promise<HistoryGraphDTO> {
  assertWailsRuntime()
  return (window as any).go.main.WaveController.GetHistoryGraph(waveId)
}

export async function runHistoryGC(waveId: number): Promise<number> {
  assertWailsRuntime()
  return (window as any).go.main.WaveController.RunHistoryGC(waveId)
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

// ── TemplateController ──

import {
  CreateDocumentTemplate,
  ListDocumentTemplates,
  BindTemplateToProfile,
  ListBindingsByProfile,
  GetDefaultTemplateForProfile,
} from "../../../../wailsjs/go/main/TemplateController";

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

export async function listBindingsByProfile(profileId: number): Promise<dto.ProfileTemplateBindingDTO[]> {
  if (!isWailsRuntimeAvailable()) return []
  return ListBindingsByProfile(profileId)
}

export async function getDefaultTemplateForProfile(profileId: number, docType: string): Promise<dto.DocumentTemplateDTO> {
  assertWailsRuntime()
  return GetDefaultTemplateForProfile(profileId, docType)
}

// ── DemandController (routing management) ──

export async function updateDemandLineRouting(input: {
  demandLineId: number
  routingDisposition: string
  recipientInputState: string
  routingReasonCode: string
}): Promise<void> {
  assertWailsRuntime()
  return (window as any).go.main.DemandController.UpdateDemandLineRouting(input)
}

export async function batchUpdateDemandLineRouting(input: {
  updates: Array<{
    demandLineId: number
    routingDisposition: string
    recipientInputState: string
    routingReasonCode: string
  }>
}): Promise<{
  updatedCount: number
  errors: Array<{ demandLineId: number; reason: string }>
}> {
  assertWailsRuntime()
  return (window as any).go.main.DemandController.BatchUpdateDemandLineRouting(input)
}

export async function getWaveRoutingStats(waveId: number): Promise<{
  totalLines: number
  acceptedReadyCount: number
  acceptedWaitingCount: number
  acceptedPartialCount: number
  deferredCount: number
  excludedManualCount: number
  excludedDuplicateCount: number
  excludedRevokedCount: number
  pendingIntakeCount: number
}> {
  assertWailsRuntime()
  return (window as any).go.main.DemandController.GetWaveRoutingStats(waveId)
}

// ── App (utility) ──

export async function pickCsvFile(): Promise<string> {
  assertWailsRuntime();
  return PickCSVFile();
}

export async function pickZipFile(): Promise<string> {
  assertWailsRuntime();
  return PickZIPFile();
}

export async function saveZoom(zoomPercent: number): Promise<void> {
  if (!isWailsRuntimeAvailable()) return;
  await SaveZoom(zoomPercent);
}

export { isWailsRuntimeAvailable, assertWailsRuntime };
