/**
 * Product entity types — domain enums defined here; DTO shapes re-exported
 * from generated Wails models (wailsjs/go/models.ts).
 */
import type { dto } from '@/../wailsjs/go/models'

/** Classification of product form factor. */
export type ProductKind =
  | 'badge'
  | 'standee'
  | 'charm'
  | 'postcard'
  | 'print'
  | 'bundle'
  | 'other'

/** ProductMaster DTO — re-exported from generated model. */
export type ProductMaster = dto.ProductMasterDTO

/** Product DTO — re-exported from generated model. */
export type Product = dto.ProductDTO
