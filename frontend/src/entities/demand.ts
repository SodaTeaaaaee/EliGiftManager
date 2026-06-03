/**
 * Demand entity types — domain enums defined here; DTO shapes re-exported
 * from generated Wails models (wailsjs/go/models.ts).
 */
import type { dto } from '@/../wailsjs/go/models'

/** Kinds of demand documents imported into the system. */
export type DemandKind = 'membership_entitlement' | 'retail_order'

/** How the demand document was captured. */
export type DemandCaptureMode = 'document_import' | 'api_ingest' | 'manual_entry'

/** Granular reason that created a demand line obligation. */
export type ObligationTriggerKind =
  | 'periodic_membership'
  | 'loyalty_membership'
  | 'supporter_only_purchase'
  | 'member_only_discount_purchase'
  | 'campaign_reward'
  | 'manual_compensation'

/** Who or what system asserts the entitlement is valid. */
export type EntitlementAuthority =
  | 'local_policy'
  | 'upstream_platform'
  | 'manual_grant'

/** Whether recipient input (address, size, etc.) has been collected. */
export type RecipientInputState =
  | 'not_required'
  | 'waiting_for_input'
  | 'partially_collected'
  | 'ready'
  | 'waived'
  | 'expired'

/** Whether this system accepts the line for processing. */
export type RoutingDisposition =
  | 'pending_intake'
  | 'accepted'
  | 'deferred'
  | 'excluded_manual'
  | 'excluded_duplicate'
  | 'excluded_revoked'

/** DemandDocument DTO — re-exported from generated model. */
export type DemandDocument = dto.DemandDocumentDTO

/** DemandLine DTO — re-exported from generated model. */
export type DemandLine = dto.DemandLineDTO
