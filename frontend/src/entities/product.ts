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

/**
 * Build a browser-loadable URL for a product asset stored under the local
 * asset middleware (`/local-images/...`). Empty / missing paths return "".
 */
export function localImageUrl(relativePath: string | undefined | null): string {
  if (!relativePath) return ''
  const rel = relativePath.replace(/^\/+/, '')
  if (!rel) return ''
  return `/local-images/${rel}`
}

/**
 * Parse `ProductMaster.detailImagePaths` (JSON-encoded `[]string`) into a
 * string array. Invalid / empty payloads yield `[]`.
 */
export function parseDetailImagePaths(raw: string | undefined | null): string[] {
  if (!raw || !raw.trim()) return []
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((item): item is string => typeof item === 'string' && item.length > 0)
  } catch {
    return []
  }
}
