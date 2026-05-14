// Bridge: strong-typed thin wrappers over generated Wails bindings.
// Never import from "wailsjs" directly outside this file.

import {
  GetDemandDocument,
  ImportDemandDocument,
  ListDemandDocuments,
  ListDemandLines,
} from "../../../../wailsjs/go/main/DemandController";
import {
  CreateWave,
  ListWaves,
  GetWave,
  GetWaveOverview,
  ApplyAllocationRules,
  AssignDemandToWave,
} from "../../../../wailsjs/go/main/WaveController";
import {
  ExportSupplierOrder,
  ListSupplierOrders,
} from "../../../../wailsjs/go/main/ExportController";
import {
  CreateShipment,
  ListShipmentsByWave,
} from "../../../../wailsjs/go/main/ShipmentController";
import {
  CreateChannelSyncJob,
  ListChannelSyncJobsByWave,
  PlanChannelClosure,
} from "../../../../wailsjs/go/main/ChannelSyncController";
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
  sourceDocumentNo: string;
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
  basisHistoryNodeId: string
  basisProjectionHash: string
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
