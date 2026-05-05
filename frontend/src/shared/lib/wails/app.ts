import {
  PickCSVFile as pickCSVFileBinding,
  PickDataFile as pickDataFileBinding,
  PickFolder as pickFolderBinding,
  PickZIPFile as pickZIPFileBinding,
  PreviewArchive as previewArchiveBinding,
  PreviewCSVSample as previewCSVSampleBinding,
} from "../../../../wailsjs/go/main/App";
import {
  AddMemberAddress as addMemberAddressBinding,
  DeleteMemberAddress as deleteMemberAddressBinding,
  ListMembers as listMembersBinding,
  ListWaveMembers as listWaveMembersBinding,
  RemoveMemberFromWave as removeMemberFromWaveBinding,
  SetDefaultAddress as setDefaultAddressBinding,
  UpdateMemberAddress as updateMemberAddressBinding,
} from "../../../../wailsjs/go/main/MemberController";
import {
  GetProductImages as getProductImagesBinding,
  ListProducts as listProductsBinding,
  ListProductsWithTags as listProductsWithTagsBinding,
  ListProductTags as listProductTagsBinding,
  RemoveLevelTag as removeLevelTagBinding,
  RemoveUserTag as removeUserTagBinding,
  UpdateProduct as updateProductBinding,
  UpsertLevelTag as upsertLevelTagBinding,
  UpsertUserTag as upsertUserTagBinding,
} from "../../../../wailsjs/go/main/ProductController";
import {
  AddDispatchToMember as addDispatchToMemberBinding,
  AllocateByTags as allocateByTagsBinding,
  BindDefaultAddresses as bindDefaultAddressesBinding,
  CreateWave as createWaveBinding,
  DeleteWave as deleteWaveBinding,
  ExportOrderCSV as exportOrderCSVBinding,
  ImportDispatchWave as importDispatchWaveBinding,
  ImportToWave as importToWaveBinding,
  ListDispatchRecords as listDispatchRecordsBinding,
  ListWaves as listWavesBinding,
  PreviewExport as previewExportBinding,
  RemoveDispatchFromMember as removeDispatchFromMemberBinding,
  RemoveProductFromWave as removeProductFromWaveBinding,
  SetDispatchAddress as setDispatchAddressBinding,
  SyncUserTagForTargetQuantity as syncUserTagForTargetQuantityBinding,
  UpdateDispatchQuantity as updateDispatchQuantityBinding,
} from "../../../../wailsjs/go/main/WaveController";
import {
  BackupDatabase as backupDatabaseBinding,
  Bootstrap as bootstrapBinding,
  GetDashboard as getDashboardBinding,
  PingDB as pingDatabaseBinding,
  RestoreDatabase as restoreDatabaseBinding,
  SaveZoom as saveZoomBinding,
} from "../../../../wailsjs/go/main/SystemController";
import {
  AddFromPreset as addFromPresetBinding,
  CreateTemplate as createTemplateBinding,
  DeleteTemplate as deleteTemplateBinding,
  GetPresetContent as getPresetContentBinding,
  ListBuiltinPresets as listBuiltinPresetsBinding,
  ListTemplates as listTemplatesBinding,
  ListUserPresets as listUserPresetsBinding,
  UpdateTemplate as updateTemplateBinding,
  ValidateTemplate as validateTemplateBinding,
} from "../../../../wailsjs/go/main/TemplateController";
import type { BootstrapPayload } from "@/shared/types/app";
import { model } from "../../../../wailsjs/go/models";
import type { main } from "../../../../wailsjs/go/models";

export const WAILS_PREVIEW_MESSAGE =
  "当前处于浏览器预览模式，Wails 后端尚未连接";

type WindowWithWails = Window & { go?: unknown };
export function isWailsRuntimeAvailable() {
  return typeof window !== "undefined" &&
    Boolean((window as WindowWithWails).go);
}
function assertWailsRuntime() {
  if (!isWailsRuntimeAvailable()) throw new Error(WAILS_PREVIEW_MESSAGE);
}

export function bootstrapApp(): Promise<BootstrapPayload> {
  assertWailsRuntime();
  return bootstrapBinding();
}
export function pingDatabase() {
  assertWailsRuntime();
  return pingDatabaseBinding();
}
export function getDashboard(): Promise<main.DashboardPayload> {
  assertWailsRuntime();
  return getDashboardBinding();
}
export function createWave(name: string): Promise<model.Wave> {
  assertWailsRuntime();
  return createWaveBinding(name);
}
export function deleteWave(waveId: number): Promise<void> {
  assertWailsRuntime();
  return deleteWaveBinding(waveId);
}
export function listWaves(status = ""): Promise<main.WaveItem[]> {
  assertWailsRuntime();
  return listWavesBinding(status);
}
export function importToWave(
  waveId: number,
  csvPath: string,
  templateId: number,
): Promise<void> {
  assertWailsRuntime();
  return importToWaveBinding(waveId, csvPath, templateId);
}
export function importDispatchWave(
  waveId: number,
  csvPath: string,
  importTemplateId: number,
  setDefault: boolean,
): Promise<void> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve();
  return importDispatchWaveBinding(
    waveId,
    csvPath,
    importTemplateId,
    setDefault,
  );
}
export function allocateByTags(waveId: number): Promise<number> {
  assertWailsRuntime();
  return allocateByTagsBinding(waveId);
}
export function updateDispatchQuantity(
  dispatchId: number,
  quantity: number,
): Promise<void> {
  assertWailsRuntime();
  return updateDispatchQuantityBinding(dispatchId, quantity);
}
export function syncUserTagForTargetQuantity(
  waveId: number,
  memberId: number,
  productId: number,
  targetQty: number,
): Promise<void> {
  assertWailsRuntime();
  return syncUserTagForTargetQuantityBinding(
    waveId,
    memberId,
    productId,
    targetQty,
  );
}
export function addDispatchToMember(
  waveId: number,
  memberId: number,
  productId: number,
  quantity: number,
): Promise<void> {
  assertWailsRuntime();
  return addDispatchToMemberBinding(waveId, memberId, productId, quantity);
}
export function removeDispatchFromMember(dispatchId: number): Promise<void> {
  assertWailsRuntime();
  return removeDispatchFromMemberBinding(dispatchId);
}
export function removeProductFromWave(
  waveId: number,
  productId: number,
): Promise<void> {
  assertWailsRuntime();
  return removeProductFromWaveBinding(waveId, productId);
}
export function setDispatchAddress(
  waveId: number,
  memberId: number,
  addressId: number,
): Promise<void> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve();
  return setDispatchAddressBinding(waveId, memberId, addressId);
}
export function listProductTags(platform: string): Promise<string[]> {
  assertWailsRuntime();
  return listProductTagsBinding(platform);
}
export function listProductsWithTags(
  waveId: number,
  platform = "",
  page = 1,
  pageSize = 50,
): Promise<main.ProductListWithTagsPayload> {
  assertWailsRuntime();
  return listProductsWithTagsBinding(waveId, platform, page, pageSize);
}
export function upsertLevelTag(
  productId: number,
  memberPlatform: string,
  levelName: string,
  quantity: number,
): Promise<void> {
  assertWailsRuntime();
  return upsertLevelTagBinding(productId, memberPlatform, levelName, quantity);
}
export function upsertUserTag(
  productId: number,
  waveMemberId: number,
  quantity: number,
): Promise<void> {
  assertWailsRuntime();
  return upsertUserTagBinding(productId, waveMemberId, quantity);
}
export function removeLevelTag(
  productId: number,
  platform: string,
  tagName: string,
): Promise<void> {
  assertWailsRuntime();
  return removeLevelTagBinding(productId, platform, tagName);
}
export function removeUserTag(
  productId: number,
  waveMemberId: number,
): Promise<void> {
  assertWailsRuntime();
  return removeUserTagBinding(productId, waveMemberId);
}
export function listMembers(
  page = 1,
  pageSize = 50,
  keyword = "",
  platform = "",
): Promise<main.MemberListPayload> {
  assertWailsRuntime();
  return listMembersBinding(page, pageSize, keyword, platform);
}
export function listProducts(
  page = 1,
  pageSize = 50,
  keyword = "",
  platform = "",
): Promise<main.ProductListPayload> {
  assertWailsRuntime();
  return listProductsBinding(page, pageSize, keyword, platform);
}
export function getProductImages(
  productId: number,
): Promise<model.ProductImage[]> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve([]);
  return getProductImagesBinding(productId);
}
export function listDispatchRecords(
  waveId = 0,
): Promise<main.DispatchRecordItem[]> {
  assertWailsRuntime();
  return listDispatchRecordsBinding(waveId);
}
export function createTemplate(
  platform: string,
  templateType: string,
  name: string,
  mappingRules: string,
): Promise<main.TemplateItem> {
  assertWailsRuntime();
  return createTemplateBinding(platform, templateType, name, mappingRules);
}
export function listTemplates(): Promise<main.TemplateItem[]> {
  assertWailsRuntime();
  return listTemplatesBinding();
}
export function listBuiltinPresets(): Promise<PresetInfo[]> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve([]);
  return listBuiltinPresetsBinding();
}
export function listUserPresets(): Promise<PresetInfo[]> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve([]);
  return listUserPresetsBinding();
}
export function getPresetContent(source: string, id: string): Promise<PresetContent> {
  assertWailsRuntime();
  return getPresetContentBinding(source, id);
}
export function addFromPreset(source: string, id: string): Promise<main.TemplateItem> {
  assertWailsRuntime();
  return addFromPresetBinding(source, id);
}
export function deleteTemplate(id: number): Promise<void> {
  assertWailsRuntime();
  return deleteTemplateBinding(id);
}
export function updateTemplate(
  id: number,
  platform: string,
  templateType: string,
  name: string,
  mappingRules: string,
): Promise<void> {
  assertWailsRuntime();
  return updateTemplateBinding(id, platform, templateType, name, mappingRules);
}
export type TemplateValidationResult = {
  valid: boolean
  errors: string[]
  warnings: string[]
}
export function validateTemplate(
  templateType: string,
  mappingRules: string,
): Promise<TemplateValidationResult> {
  assertWailsRuntime();
  return validateTemplateBinding(templateType, mappingRules);
}
export function setDefaultAddress(
  memberId: number,
  addressId: number,
): Promise<void> {
  assertWailsRuntime();
  return setDefaultAddressBinding(memberId, addressId);
}
export function addMemberAddress(
  memberId: number,
  recipientName: string,
  phone: string,
  address: string,
): Promise<model.MemberAddress> {
  assertWailsRuntime();
  return addMemberAddressBinding(memberId, recipientName, phone, address);
}
export function updateMemberAddress(
  addressId: number,
  recipientName: string,
  phone: string,
  address: string,
): Promise<void> {
  assertWailsRuntime();
  return updateMemberAddressBinding(addressId, recipientName, phone, address);
}
export function deleteMemberAddress(addressId: number): Promise<void> {
  assertWailsRuntime();
  return deleteMemberAddressBinding(addressId);
}
export function removeMemberFromWave(
  waveId: number,
  memberId: number,
): Promise<void> {
  assertWailsRuntime();
  return removeMemberFromWaveBinding(waveId, memberId);
}
export function listWaveMembers(waveId: number): Promise<MemberItem[]> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve([]);
  return listWaveMembersBinding(waveId);
}
export type ProductUpdateInput = {
  id: number;
  platform: string;
  factory: string;
  factorySku: string;
  name: string;
  coverImage: string;
  extraData: string;
};

export function updateProduct(product: ProductUpdateInput): Promise<void> {
  assertWailsRuntime();
  return updateProductBinding(model.Product.createFrom(product));
}
export function backupDatabase(): Promise<string> {
  assertWailsRuntime();
  return backupDatabaseBinding();
}
export function pickCSVFile(): Promise<string> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve("");
  return pickCSVFileBinding();
}
export function pickDataFile(): Promise<string> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve("");
  return pickDataFileBinding();
}
export function pickZIPFile(): Promise<string> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve("");
  return pickZIPFileBinding();
}
export function pickFolder(): Promise<string> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve("");
  return pickFolderBinding();
}
export type ArchivePreview = { extractDir: string; csvFiles: string[]; dirs: { name: string; fileCount: number }[] };
export function previewArchive(path: string): Promise<ArchivePreview | null> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve(null);
  return previewArchiveBinding(path);
}
export function previewCSVSample(path: string): Promise<string[][]> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve([]);
  return previewCSVSampleBinding(path);
}
export function restoreDatabase(): Promise<void> {
  assertWailsRuntime();
  return restoreDatabaseBinding();
}

export function saveZoom(percent: number): Promise<void> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve();
  return saveZoomBinding(percent);
}

export type AddressBindingResult = { updated: number; skipped: number };
export function bindDefaultAddresses(
  waveId: number,
): Promise<AddressBindingResult> {
  if (!isWailsRuntimeAvailable()) {
    return Promise.resolve({ updated: 0, skipped: 0 });
  }
  return bindDefaultAddressesBinding(waveId) as Promise<AddressBindingResult>;
}

export function exportOrderCSV(
  waveId: number,
  exportTemplateId: number,
): Promise<string> {
  if (!isWailsRuntimeAvailable()) {
    return Promise.resolve("/mock/path/eligift-factory-order.csv");
  }
  return exportOrderCSVBinding(waveId, exportTemplateId);
}

export type ExportPreview = {
  totalRecords: number;
  missingAddressCount: number;
};
export function previewExport(waveId: number): Promise<ExportPreview> {
  if (!isWailsRuntimeAvailable()) {
    return Promise.resolve({ totalRecords: 0, missingAddressCount: 0 });
  }
  return previewExportBinding(waveId) as Promise<ExportPreview>;
}

export type DashboardPayload = main.DashboardPayload;
export type WaveItem = main.WaveItem;
export type MemberItem = main.MemberItem;
export type MemberListPayload = main.MemberListPayload;
export type ProductItem = main.ProductItem;
export type ProductListPayload = main.ProductListPayload;
export type DispatchRecordItem = main.DispatchRecordItem;
export type TemplateItem = main.TemplateItem;
export type PresetInfo = { id: string; platform: string; type: string; name: string };
export type PresetContent = {
  id: string;
  platform: string;
  type: string;
  name: string;
  mappingRules: any;
};
