/** Classification of product form factor. */
export type ProductKind =
  | "badge"
  | "standee"
  | "charm"
  | "postcard"
  | "print"
  | "bundle"
  | "other";

/**
 * ProductMaster (aligned to Go dto.ProductMasterDTO).
 * Canonical product definition independent of any wave.
 */
export interface ProductMaster {
  id: number;
  supplierPlatform: string;
  factorySku: string;
  supplierProductRef: string;
  name: string;
  productKind: string;
  archived: boolean;
  extraData: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * Product (aligned to Go dto.ProductDTO).
 * Wave-scoped product instance, optionally linked to a ProductMaster.
 */
export interface Product {
  id: number;
  waveId: number;
  productMasterId: number | null;
  supplierPlatform: string;
  factorySku: string;
  name: string;
  extraData: string;
  createdAt: string;
  updatedAt: string;
}
