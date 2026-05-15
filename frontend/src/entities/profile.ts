/**
 * IntegrationProfile (aligned to Go dto.IntegrationProfileDTO).
 * Defines how a specific channel/surface integrates with the system.
 */
export interface IntegrationProfile {
  id: number;
  profileKey: string;
  sourceChannel: string;
  sourceSurface: string;
  demandKind: string;
  initialAllocationStrategy: string;
  identityStrategy: string;
  entitlementAuthorityMode: string;
  recipientInputMode: string;
  referenceStrategy: string;
  trackingSyncMode: string;
  closurePolicy: string;
  supportsPartialShipment: boolean;
  supportsApiImport: boolean;
  supportsApiExport: boolean;
  requiresCarrierMapping: boolean;
  requiresExternalOrderNo: boolean;
  allowsManualClosure: boolean;
  connectorKey: string;
  supportedLocales: string;
  defaultLocale: string;
  extraData: string;
  createdAt: string;
  updatedAt: string;
}

/** Input for creating a new IntegrationProfile. */
export interface CreateProfileInput {
  profileKey: string;
  sourceChannel: string;
  sourceSurface: string;
  demandKind: string;
  initialAllocationStrategy: string;
  identityStrategy: string;
  entitlementAuthorityMode: string;
  recipientInputMode: string;
  referenceStrategy: string;
  trackingSyncMode: string;
  closurePolicy: string;
  supportsPartialShipment: boolean;
  supportsApiImport: boolean;
  supportsApiExport: boolean;
  requiresCarrierMapping: boolean;
  requiresExternalOrderNo: boolean;
  allowsManualClosure: boolean;
  connectorKey: string;
  supportedLocales: string;
  defaultLocale: string;
  extraData: string;
}

/** Input for updating an existing IntegrationProfile. */
export interface UpdateProfileInput {
  id: number;
  profileKey: string;
  sourceChannel: string;
  sourceSurface: string;
  demandKind: string;
  initialAllocationStrategy: string;
  identityStrategy: string;
  entitlementAuthorityMode: string;
  recipientInputMode: string;
  referenceStrategy: string;
  trackingSyncMode: string;
  closurePolicy: string;
  supportsPartialShipment: boolean;
  supportsApiImport: boolean;
  supportsApiExport: boolean;
  requiresCarrierMapping: boolean;
  requiresExternalOrderNo: boolean;
  allowsManualClosure: boolean;
  connectorKey: string;
  supportedLocales: string;
  defaultLocale: string;
  extraData: string;
}
