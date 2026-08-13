package dto

import "time"

// ---- IntegrationProfile (full) ----

type IntegrationProfileDTO struct {
	ID            uint   `json:"id"`
	ProfileKey    string `json:"profileKey"`
	SourceChannel string `json:"sourceChannel"`
	SourceSurface string `json:"sourceSurface"`
	// DemandKind is a leftover optional hint, not the unique document type.
	// One source platform may bind both demand import types.
	DemandKind                string `json:"demandKind"`
	InitialAllocationStrategy string `json:"initialAllocationStrategy"`
	// IdentityStrategy is leftover; import-time identity follows
	// InterpretDemandImportDocumentType.
	IdentityStrategy               string    `json:"identityStrategy"`
	EntitlementAuthorityMode       string    `json:"entitlementAuthorityMode"`
	RecipientInputMode             string    `json:"recipientInputMode"`
	ReferenceStrategy              string    `json:"referenceStrategy"`
	TrackingSyncMode               string    `json:"trackingSyncMode"`
	ClosurePolicy                  string    `json:"closurePolicy"`
	SupportsPartialShipment        bool      `json:"supportsPartialShipment"`
	SupportsAPIImport              bool      `json:"supportsApiImport"`
	SupportsAPIExport              bool      `json:"supportsApiExport"`
	RequiresCarrierMapping         bool      `json:"requiresCarrierMapping"`
	RequiresExternalOrderNo        bool      `json:"requiresExternalOrderNo"`
	AllowsManualClosure            bool      `json:"allowsManualClosure"`
	SupportsExportSupplierOrder    bool      `json:"supportsExportSupplierOrder"`
	SupportsImportProductCatalog   bool      `json:"supportsImportProductCatalog"`
	SupportsImportSupplierShipment bool      `json:"supportsImportSupplierShipment"`
	ConnectorKey                   string    `json:"connectorKey"`
	FactorySupplierPlatform        string    `json:"factorySupplierPlatform"`
	SupportedLocales               string    `json:"supportedLocales"`
	DefaultLocale                  string    `json:"defaultLocale"`
	ExtraData                      string    `json:"extraData"`
	CreatedAt                      time.Time `json:"createdAt" ts_type:"string"`
	UpdatedAt                      time.Time `json:"updatedAt" ts_type:"string"`
}

type CreateProfileInput struct {
	ProfileKey    string `json:"profileKey"`
	SourceChannel string `json:"sourceChannel"`
	SourceSurface string `json:"sourceSurface"`
	// DemandKind is a leftover optional hint, not the unique document type.
	// One source platform may bind both demand import types.
	DemandKind                string `json:"demandKind"`
	InitialAllocationStrategy string `json:"initialAllocationStrategy"`
	// IdentityStrategy is leftover; import-time identity follows
	// InterpretDemandImportDocumentType.
	IdentityStrategy               string `json:"identityStrategy"`
	EntitlementAuthorityMode       string `json:"entitlementAuthorityMode"`
	RecipientInputMode             string `json:"recipientInputMode"`
	ReferenceStrategy              string `json:"referenceStrategy"`
	TrackingSyncMode               string `json:"trackingSyncMode"`
	ClosurePolicy                  string `json:"closurePolicy"`
	SupportsPartialShipment        bool   `json:"supportsPartialShipment"`
	SupportsAPIImport              bool   `json:"supportsApiImport"`
	SupportsAPIExport              bool   `json:"supportsApiExport"`
	RequiresCarrierMapping         bool   `json:"requiresCarrierMapping"`
	RequiresExternalOrderNo        bool   `json:"requiresExternalOrderNo"`
	AllowsManualClosure            bool   `json:"allowsManualClosure"`
	SupportsExportSupplierOrder    bool   `json:"supportsExportSupplierOrder"`
	SupportsImportProductCatalog   bool   `json:"supportsImportProductCatalog"`
	SupportsImportSupplierShipment bool   `json:"supportsImportSupplierShipment"`
	ConnectorKey                   string `json:"connectorKey"`
	FactorySupplierPlatform        string `json:"factorySupplierPlatform"`
	SupportedLocales               string `json:"supportedLocales"`
	DefaultLocale                  string `json:"defaultLocale"`
	ExtraData                      string `json:"extraData"`
}

type UpdateProfileInput struct {
	ID            uint   `json:"id"`
	ProfileKey    string `json:"profileKey"`
	SourceChannel string `json:"sourceChannel"`
	SourceSurface string `json:"sourceSurface"`
	// DemandKind is a leftover optional hint, not the unique document type.
	// One source platform may bind both demand import types.
	DemandKind                string `json:"demandKind"`
	InitialAllocationStrategy string `json:"initialAllocationStrategy"`
	// IdentityStrategy is leftover; import-time identity follows
	// InterpretDemandImportDocumentType.
	IdentityStrategy               string `json:"identityStrategy"`
	EntitlementAuthorityMode       string `json:"entitlementAuthorityMode"`
	RecipientInputMode             string `json:"recipientInputMode"`
	ReferenceStrategy              string `json:"referenceStrategy"`
	TrackingSyncMode               string `json:"trackingSyncMode"`
	ClosurePolicy                  string `json:"closurePolicy"`
	SupportsPartialShipment        bool   `json:"supportsPartialShipment"`
	SupportsAPIImport              bool   `json:"supportsApiImport"`
	SupportsAPIExport              bool   `json:"supportsApiExport"`
	RequiresCarrierMapping         bool   `json:"requiresCarrierMapping"`
	RequiresExternalOrderNo        bool   `json:"requiresExternalOrderNo"`
	AllowsManualClosure            bool   `json:"allowsManualClosure"`
	SupportsExportSupplierOrder    bool   `json:"supportsExportSupplierOrder"`
	SupportsImportProductCatalog   bool   `json:"supportsImportProductCatalog"`
	SupportsImportSupplierShipment bool   `json:"supportsImportSupplierShipment"`
	ConnectorKey                   string `json:"connectorKey"`
	FactorySupplierPlatform        string `json:"factorySupplierPlatform"`
	SupportedLocales               string `json:"supportedLocales"`
	DefaultLocale                  string `json:"defaultLocale"`
	ExtraData                      string `json:"extraData"`
}
