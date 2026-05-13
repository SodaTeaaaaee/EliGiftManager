// Bridge: strong-typed thin wrappers over generated Wails bindings.
// Never import from "wailsjs" directly outside this file.

import {
  GetDemandDocument,
  ImportDemandDocument,
  ListDemandDocuments,
} from "../../../../wailsjs/go/main/DemandController";
import {
  CreateWave,
  ListWaves,
  GetWave,
  GetWaveOverview,
  ApplyAllocationRules,
} from "../../../../wailsjs/go/main/WaveController";
import {
  ExportSupplierOrder,
  ListSupplierOrders,
} from "../../../../wailsjs/go/main/ExportController";
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
