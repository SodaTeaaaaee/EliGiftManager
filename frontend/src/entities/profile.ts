/**
 * Profile entity types — re-exported from generated Wails DTO.
 *
 * The generated `dto.*` classes (wailsjs/go/models.ts) are the authoritative
 * definitions. This file re-exports them as type aliases so that existing
 * imports from '@/entities/profile' continue to work.
 */
import type { dto } from '@/../wailsjs/go/models'

/** IntegrationProfile DTO — re-exported from generated model. */
export type IntegrationProfile = dto.IntegrationProfileDTO

/** Input for creating a new IntegrationProfile. */
export type CreateProfileInput = dto.CreateProfileInput

/** Input for updating an existing IntegrationProfile. */
export type UpdateProfileInput = dto.UpdateProfileInput
