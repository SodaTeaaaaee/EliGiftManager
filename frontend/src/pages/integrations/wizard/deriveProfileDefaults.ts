/**
 * Pure derivation helpers for fields `CreateProfileInput` requires that the
 * intake wizard's 5 named steps (platformPreset / businessSurface /
 * sampleUpload / capabilities / confirm — see the P4 foundations i18n key
 * tree) have no dedicated UI for: the 7 profile "strategy" enums
 * (`initialAllocationStrategy`, `identityStrategy`,
 * `entitlementAuthorityMode`, `recipientInputMode`, `referenceStrategy`,
 * `trackingSyncMode`, `closurePolicy`) and `documentType` (which the backend
 * derives 1:1 from `demandKind`, see `internal/app/template_usecase.go`'s
 * `validDocumentTypes` + `controller_demand_csv_import.go`'s default).
 *
 * Rather than adding a 6th wizard step, these are deterministically derived
 * from answers the operator already gave (business surface + capability
 * toggles) — every value produced here is still shown to the operator on
 * the confirm step via `StatusBadge`, never silently applied. This is a
 * deliberate scope decision, flagged in the unit's deviations report; a
 * later iteration could promote any of these to an explicit wizard field if
 * the derived default proves wrong often enough in practice.
 */
import type { IntakeProfileCapabilities } from '@/shared/lib/demand-intake/platform-presets'

export type DemandKind = 'membership_entitlement' | 'retail_order'

/** `documentType` used for both `createDocumentTemplate`/`bindTemplateToProfile` and later
 * `importDemandCSV`/`getDefaultTemplateForProfile` calls — must match `validDocumentTypes`
 * in `internal/app/template_usecase.go` exactly. */
export function documentTypeForDemandKind(demandKind: DemandKind): string {
  return demandKind === 'retail_order' ? 'import_sales_order' : 'import_entitlement'
}

export interface DerivedProfileStrategy {
  initialAllocationStrategy: string
  identityStrategy: string
  entitlementAuthorityMode: string
  recipientInputMode: string
  referenceStrategy: string
  trackingSyncMode: string
  closurePolicy: string
}

export interface DeriveProfileStrategyOptions {
  /** Whether the operator picked a `connectorKey` in the capabilities step. */
  hasConnectorKey: boolean
  /**
   * Explicit `trackingSyncMode` choice, ONLY meaningful when `hasConnectorKey` is true.
   * `api_push`/`document_export` are gated server-side by
   * `internal/app/profile_usecase.go`'s `validateExecutionReadiness`: both REQUIRE a
   * non-empty `connectorKey` AND that the (`trackingSyncMode`, `connectorKey`) pair
   * resolves against the backend's executor registry — a pairing the frontend has no
   * way to verify ahead of time (`listConnectorCapabilities()` lists registered
   * connector keys and their capability flags, but not which `trackingSyncMode` each
   * is registered under). This function therefore never silently guesses `api_push`
   * or `document_export` — the wizard only sets `hasConnectorKey: true` once the
   * operator has explicitly picked a connector AND a sync mode, and any backend
   * rejection (wrong pairing) is surfaced to the operator as-is via the real error
   * message, not swallowed.
   */
  trackingSyncModeOverride?: string
}

/**
 * Derives the 7 profile-strategy enum fields from `demandKind` + the operator's capability
 * toggles. Every branch below is a plain, auditable heuristic — no hidden magic:
 * - `trackingSyncMode`: when no `connectorKey` is configured, defaults to the SAFE, always-
 *   valid pair of options that never require a connector (`manual_confirmation` if manual
 *   closure is allowed, else `unsupported`) — `api_push`/`document_export` are only used when
 *   the operator explicitly opted into a connector + sync mode (see
 *   `DeriveProfileStrategyOptions.trackingSyncModeOverride` above).
 * - `closurePolicy`: `close_after_sync` for the two automated sync modes; otherwise
 *   `close_after_manual_confirmation` if manual closure is allowed, else
 *   `close_after_shipment` (the line closes the moment it ships, since neither automatic
 *   nor manual sync-back is available).
 * - `recipientInputMode` / `referenceStrategy`: membership demand collects recipient info
 *   via a platform claim flow and is tracked at the member level; retail demand already
 *   carries recipient info on the order itself (`'none'`) and is tracked per order.
 * - `initialAllocationStrategy` / `identityStrategy` / `entitlementAuthorityMode`: fixed,
 *   generically-sound defaults (policy-driven allocation, platform-UID identity, local-policy
 *   entitlement authority) — the common case for a freshly-onboarded integration.
 */
export function deriveProfileStrategyDefaults(
  demandKind: DemandKind,
  capabilities: IntakeProfileCapabilities,
  options: DeriveProfileStrategyOptions = { hasConnectorKey: false },
): DerivedProfileStrategy {
  const isMembership = demandKind === 'membership_entitlement'

  const trackingSyncMode = options.hasConnectorKey
    ? (options.trackingSyncModeOverride ?? 'document_export')
    : capabilities.allowsManualClosure
      ? 'manual_confirmation'
      : 'unsupported'

  const closurePolicy =
    trackingSyncMode === 'api_push' || trackingSyncMode === 'document_export'
      ? 'close_after_sync'
      : capabilities.allowsManualClosure
        ? 'close_after_manual_confirmation'
        : 'close_after_shipment'

  return {
    initialAllocationStrategy: 'policy_driven',
    identityStrategy: 'platform_uid',
    entitlementAuthorityMode: 'local_policy',
    recipientInputMode: isMembership ? 'platform_claim' : 'none',
    referenceStrategy: isMembership ? 'member_level' : 'order_level',
    trackingSyncMode,
    closurePolicy,
  }
}
