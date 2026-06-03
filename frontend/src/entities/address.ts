/**
 * Address entity types — re-exported from generated Wails DTO.
 *
 * The generated `dto.*` classes (wailsjs/go/models.ts) are the authoritative
 * definitions. This file re-exports them as type aliases so that existing
 * `import type { CustomerAddressDTO } from '@/entities/address'` continues
 * to work without importing from wailsjs directly.
 */
import type { dto } from '@/../wailsjs/go/models'

export type CustomerAddressDTO = dto.CustomerAddressDTO
export type CreateAddressInput = dto.CreateAddressInput
export type UpdateAddressInput = dto.UpdateAddressInput
export type BindAddressInput = dto.BindAddressInput
