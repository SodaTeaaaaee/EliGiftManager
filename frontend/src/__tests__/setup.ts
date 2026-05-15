import { vi } from "vitest";

// Mock naive-ui composition APIs that require provider ancestors
vi.mock("naive-ui", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    useMessage: () => ({
      success: vi.fn(),
      error: vi.fn(),
      warning: vi.fn(),
      info: vi.fn(),
      loading: vi.fn(),
    }),
    useDialog: () => ({
      success: vi.fn(),
      error: vi.fn(),
      warning: vi.fn(),
      info: vi.fn(),
      create: vi.fn(),
    }),
  };
});

vi.mock("@/shared/lib/wails/app.ts", () => ({
  isWailsRuntimeAvailable: vi.fn(() => false),
  assertWailsRuntime: vi.fn(),
  listDemandLines: vi.fn(async () => []),
  listDemandDocuments: vi.fn(async () => []),
  listUnassignedDemandDocuments: vi.fn(async () => []),
  getDemandDocument: vi.fn(async () => ({})),
  importDemandDocument: vi.fn(async () => ({})),
  listWaves: vi.fn(async () => []),
  getWave: vi.fn(async () => ({ id: 1, name: "Test Wave" })),
  createWave: vi.fn(async () => ({ id: 1, name: "Test Wave" })),
  applyAllocationRules: vi.fn(async () => []),
  getWaveOverview: vi.fn(async () => ({
    wave: { id: 1, name: "Test Wave", lifecycleStage: "planning" },
    demandSummary: { totalDocuments: 0, totalLines: 0 },
    allocationSummary: { totalRules: 0, totalFulfillmentLines: 0 },
    exportSummary: { hasSupplierOrder: false },
    shipmentSummary: { totalShipments: 0 },
    channelSyncSummary: { totalJobs: 0 },
    basisDriftSummary: { hasDriftedBasis: false },
  })),
  assignDemandToWave: vi.fn(async () => {}),
  undoWaveAction: vi.fn(async () => ""),
  redoWaveAction: vi.fn(async () => ""),
  exportSupplierOrder: vi.fn(async () => ({})),
  listSupplierOrders: vi.fn(async () => []),
  getSupplierOrderByWave: vi.fn(async () => ({})),
  listLinesBySupplierOrder: vi.fn(async () => []),
  createShipment: vi.fn(async () => ({})),
  listShipmentsByWave: vi.fn(async () => []),
  createChannelSyncJob: vi.fn(async () => ({})),
  listChannelSyncJobsByWave: vi.fn(async () => []),
  planChannelClosure: vi.fn(async () => ({})),
  executeChannelSyncJob: vi.fn(async () => ({})),
  recordChannelClosureDecision: vi.fn(async () => []),
  retryChannelSyncJob: vi.fn(async () => ({})),
  listIntegrationProfiles: vi.fn(async () => []),
  listProfiles: vi.fn(async () => []),
  getProfile: vi.fn(async () => ({})),
  createProfile: vi.fn(async () => ({})),
  updateProfile: vi.fn(async () => ({})),
  deleteProfile: vi.fn(async () => {}),
  seedDefaultProfiles: vi.fn(async () => []),
  createProductMaster: vi.fn(async () => ({})),
  listProductMasters: vi.fn(async () => []),
  updateProductMaster: vi.fn(async () => ({})),
  snapshotProductsForWave: vi.fn(async () => []),
  listProductsByWave: vi.fn(async () => []),
  listAllocationPolicyRules: vi.fn(async () => []),
  createAllocationPolicyRule: vi.fn(async () => ({})),
  updateAllocationPolicyRule: vi.fn(async () => ({})),
  deleteAllocationPolicyRule: vi.fn(async () => {}),
  reconcileWave: vi.fn(async () => ({ successes: [], failures: [] })),
  listAssignedDemandsByWave: vi.fn(async () => []),
  generateParticipants: vi.fn(async () => 0),
  listAdjustmentsByWave: vi.fn(async () => []),
  recordAdjustment: vi.fn(async () => ({})),
  pickCsvFile: vi.fn(async () => ""),
  pickZipFile: vi.fn(async () => ""),
  saveZoom: vi.fn(async () => {}),
}));

vi.mock("@/shared/composables/useUndoRedo.ts", () => ({
  useUndoRedo: vi.fn(() => ({
    canUndo: { value: false },
    canRedo: { value: false },
  })),
}));
