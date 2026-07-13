/**
 * Customer (CustomerProfile) entity types — re-exported from generated Wails
 * DTO.
 *
 * NOTE: `@/entities/profile` is a DIFFERENT concept (IntegrationProfile, the
 * §3.4 connector/integration profile) — do not confuse the two. This file
 * covers the CustomerProfile domain (§3.6 customer detail / merge / address
 * / fulfillment history).
 *
 * The generated `dto.*` classes (wailsjs/go/models.ts) are the authoritative
 * definitions. This file re-exports them as type aliases so that
 * `import type { CustomerProfileDTO } from '@/entities/customer'` continues
 * to work without importing from wailsjs directly.
 */
import type { dto } from '@/../wailsjs/go/models'

export type CustomerProfileDTO = dto.CustomerProfileDTO
export type CustomerIdentityDTO = dto.CustomerIdentityDTO
export type CreateCustomerProfileInput = dto.CreateCustomerProfileInput
export type UpdateCustomerProfileInput = dto.UpdateCustomerProfileInput
export type CreateCustomerIdentityInput = dto.CreateCustomerIdentityInput
export type MergeSuggestionDTO = dto.MergeSuggestionDTO
export type SystemSettingsDTO = dto.SystemSettingsDTO
export type CustomerFulfillmentHistoryRowDTO = dto.CustomerFulfillmentHistoryRowDTO
// NOTE: CustomerAddressDTO is already re-exported by `./address.ts` — do not
// duplicate it here (index.ts's `export type *` would collide).

// Merge-preview types (consumed by MergePreviewDialog.vue).
export type MergeProfilesPreviewResult = dto.MergeProfilesPreviewResult
export type MergePreviewProfileSide = dto.MergePreviewProfileSide
export type MergePreviewConflict = dto.MergePreviewConflict
export type MergePreviewIdentity = dto.MergePreviewIdentity
export type MergePreviewAddress = dto.MergePreviewAddress
