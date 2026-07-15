/**
 * Product entity types — domain enums defined here; DTO shapes re-exported
 * from generated Wails models (wailsjs/go/models.ts).
 */
import type { dto } from '@/../wailsjs/go/models'
import type { ProductKind as DomainProductKind } from '@/shared/api/generated/enums'

/** Classification of product form factor. */
export type ProductKind = DomainProductKind

/** ProductMaster DTO — re-exported from generated model. */
export type ProductMaster = dto.ProductMasterDTO
export type ProductMasterDTO = dto.ProductMasterDTO

/** Product DTO — re-exported from generated model. */
export type Product = dto.ProductDTO
export type ProductDTO = dto.ProductDTO

/** Batch-stock-to-wave (dedup-aware) result types. */
export type SnapshotProductDetailItem = dto.SnapshotProductDetailItem
export type SnapshotProductsDetailedResult = dto.SnapshotProductsDetailedResult
