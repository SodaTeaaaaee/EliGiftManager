import {
  PickCSVFile as pickCSVFileBinding,
  PickZIPFile as pickZIPFileBinding,
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
  AssignProductTag as assignProductTagBinding,
  GetProductImages as getProductImagesBinding,
  ListProducts as listProductsBinding,
  ListProductsWithTags as listProductsWithTagsBinding,
  ListProductTags as listProductTagsBinding,
  RemoveProductTag as removeProductTagBinding,
  UpdateProduct as updateProductBinding,
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
} from "../../../../wailsjs/go/main/SystemController";
import {
  CreateTemplate as createTemplateBinding,
  ListDefaultTemplates as listDefaultTemplatesBinding,
  ListTemplates as listTemplatesBinding,
  UpdateTemplate as updateTemplateBinding,
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
): Promise<void> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve();
  return importDispatchWaveBinding(waveId, csvPath, importTemplateId);
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
  return syncUserTagForTargetQuantityBinding(waveId, memberId, productId, targetQty);
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
export function assignProductTag(
  productId: number,
  platform: string,
  tagName: string,
  quantity: number,
  tagType: string,
): Promise<void> {
  assertWailsRuntime();
  return assignProductTagBinding(
    productId,
    platform,
    tagName,
    quantity,
    tagType,
  );
}
export function removeProductTag(
  productId: number,
  platform: string,
  tagName: string,
  tagType: string,
): Promise<void> {
  assertWailsRuntime();
  return removeProductTagBinding(productId, platform, tagName, tagType);
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
export function listDefaultTemplates(): Promise<main.TemplateItem[]> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve([]);
  return listDefaultTemplatesBinding();
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
export function pickZIPFile(): Promise<string> {
  if (!isWailsRuntimeAvailable()) return Promise.resolve("");
  return pickZIPFileBinding();
}
export function restoreDatabase(): Promise<void> {
  assertWailsRuntime();
  return restoreDatabaseBinding();
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
