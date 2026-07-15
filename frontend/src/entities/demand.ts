/**
 * Demand entity types — domain enums defined here; DTO shapes re-exported
 * from generated Wails models (wailsjs/go/models.ts).
 */
import type { dto } from '@/../wailsjs/go/models'
import type {
  CaptureMode,
  DemandKind as DomainDemandKind,
  EntitlementAuthority as DomainEntitlementAuthority,
  ObligationTriggerKind as DomainObligationTriggerKind,
  RecipientInputState as DomainRecipientInputState,
  RoutingDisposition as DomainRoutingDisposition,
} from '@/shared/api/generated/enums'

/** Kinds of demand documents imported into the system. */
export type DemandKind = DomainDemandKind

/** How the demand document was captured. */
export type DemandCaptureMode = CaptureMode

/** Granular reason that created a demand line obligation. */
export type ObligationTriggerKind = DomainObligationTriggerKind

/** Who or what system asserts the entitlement is valid. */
export type EntitlementAuthority = DomainEntitlementAuthority

/** Whether recipient input (address, size, etc.) has been collected. */
export type RecipientInputState = DomainRecipientInputState

/** Whether this system accepts the line for processing. */
export type RoutingDisposition = DomainRoutingDisposition

/** DemandDocument DTO — re-exported from generated model. */
export type DemandDocument = dto.DemandDocumentDTO

/** DemandLine DTO — re-exported from generated model. */
export type DemandLine = dto.DemandLineDTO

/** CSV file preview (headers + header-keyed row maps) — re-exported from generated model. */
export type CSVFilePreview = dto.CSVFilePreviewDTO

/** Outcome of a dual-mode demand CSV import — re-exported from generated model. */
export type ImportDemandCSVResult = dto.ImportDemandCSVResult

/** One row of the demand-inbox grid — re-exported from generated model. */
export type DemandInboxRow = dto.DemandInboxRowDTO

/** Server-paginated demand-inbox row page — re-exported from generated model. */
export type DemandInboxRowList = dto.DemandInboxRowListDTO
