/** Kinds of demand documents imported into the system. */
export type DemandKind = "membership_entitlement" | "retail_order";

/** How the demand document was captured. */
export type DemandCaptureMode = "document_import" | "api_ingest" | "manual_entry";

/** Granular reason that created a demand line obligation. */
export type ObligationTriggerKind =
  | "periodic_membership"
  | "loyalty_membership"
  | "supporter_only_purchase"
  | "member_only_discount_purchase"
  | "campaign_reward"
  | "manual_compensation";

/** Who or what system asserts the entitlement is valid. */
export type EntitlementAuthority =
  | "local_policy"
  | "upstream_platform"
  | "manual_grant";

/** Whether recipient input (address, size, etc.) has been collected. */
export type RecipientInputState =
  | "not_required"
  | "waiting_for_input"
  | "partially_collected"
  | "ready"
  | "waived"
  | "expired";

/** Whether this system accepts the line for processing. */
export type RoutingDisposition =
  | "pending_intake"
  | "accepted"
  | "deferred"
  | "excluded_manual"
  | "excluded_duplicate"
  | "excluded_revoked";

/**
 * DemandDocument (aligned to Go dto.DemandDocumentDTO).
 * Go `*uint` fields → TS `number | null`.
 * Go `string` fields → TS `string` (not `Record<string, unknown>`).
 */
export interface DemandDocument {
  id: number;
  kind: string;
  captureMode: string;
  sourceChannel: string;
  sourceSurface: string;
  integrationProfileId: number | null;
  sourceDocumentNo: string;
  sourceCustomerRef: string;
  customerProfileId: number | null;
  sourceCreatedAt: string;
  sourcePaidAt: string;
  currency: string;
  authoritySnapshotAt: string;
  rawPayload: string;
  extraData: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * DemandLine (aligned to Go dto.DemandLineDTO).
 * Go `*int` fields → TS `number | null`.
 * Go `string` fields → TS `string`.
 * `routingDisposition` is required string in Go → TS `string`.
 */
export interface DemandLine {
  id: number;
  demandDocumentId: number;
  sourceLineNo: number | null;
  lineType: string;
  obligationTriggerKind: string;
  entitlementAuthority: string;
  recipientInputState: string;
  routingDisposition: string;
  routingReasonCode: string;
  eligibilityContextRef: string;
  productMasterId: number | null;
  externalTitle: string;
  requestedQuantity: number;
  entitlementCode: string;
  giftLevelSnapshot: string;
  recipientInputPayload: string;
  rawPayload: string;
  extraData: string;
  createdAt: string;
  updatedAt: string;
}

/** Demand document with its lines eagerly loaded. */
export interface DemandDocumentWithLines extends DemandDocument {
  lines: DemandLine[];
}
