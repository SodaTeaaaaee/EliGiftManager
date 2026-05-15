// Bridge: strong-typed thin wrappers over generated Wails bindings.
// Never import from "wailsjs" directly outside this file.

import {
  GetDemandDocument,
  ImportDemandDocument,
  ListDemandDocuments,
  ListDemandLines,
  ListUnassignedDemandDocuments,
} from "../../../../wailsjs/go/main/DemandController";
import {
  CreateWave,
  ListWaves,
  GetWave,
  GetWaveOverview,
  ApplyAllocationRules,
  AssignDemandToWave,
  GenerateParticipants,
  ListAssignedDemandsByWave,
  UndoWaveAction,
  RedoWaveAction,
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
    routingDisposition: string;
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

export async function getWave(id: number): Promise<dto.WaveDTO> {
  assertWailsRuntime();
  return GetWave(id);
}

export async function createWave(name: string): Promise<dto.WaveDTO> {
  assertWailsRuntime();
  return CreateWave(new dto.CreateWaveInput({ name }));
}

export async function applyAllocationRules(
  waveId: number,
): Promise<dto.FulfillmentLineDTO[]> {
  assertWailsRuntime();
  return ApplyAllocationRules(waveId);
}

export async function getWaveOverview(
  waveId: number,
): Promise<dto.WaveOverviewDTO> {
  assertWailsRuntime();
  return GetWaveOverview(waveId);
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

// ── AllocationPolicyController (runtime fallback — bindings not yet generated) ──

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
  return (window as any).go.main.AllocationPolicyController.ListAllocationPolicyRules(waveID)
}

export async function createAllocationPolicyRule(
  input: CreateAllocationPolicyRuleInput,
): Promise<AllocationPolicyRule> {
  assertWailsRuntime()
  return (window as any).go.main.AllocationPolicyController.CreateAllocationPolicyRule(input)
}

export async function updateAllocationPolicyRule(
  input: UpdateAllocationPolicyRuleInput,
): Promise<AllocationPolicyRule> {
  assertWailsRuntime()
  return (window as any).go.main.AllocationPolicyController.UpdateAllocationPolicyRule(input)
}

export async function deleteAllocationPolicyRule(ruleID: number): Promise<void> {
  assertWailsRuntime()
  return (window as any).go.main.AllocationPolicyController.DeleteAllocationPolicyRule(ruleID)
}

export async function reconcileWave(waveID: number): Promise<ReconcileResult> {
  assertWailsRuntime()
  return (window as any).go.main.AllocationPolicyController.ReconcileWave(waveID)
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
  },
): Promise<dto.FulfillmentAdjustmentDTO> {
  assertWailsRuntime();
  return RecordAdjustment(dto.RecordAdjustmentInput.createFrom(input));
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
