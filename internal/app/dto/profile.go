package dto

// ---- IntegrationProfile (full) ----

type IntegrationProfileDTO struct {
	ID                        uint   `json:"id"`
	ProfileKey                string `json:"profileKey"`
	SourceChannel             string `json:"sourceChannel"`
	SourceSurface             string `json:"sourceSurface"`
	DemandKind                string `json:"demandKind"`
	InitialAllocationStrategy string `json:"initialAllocationStrategy"`
	IdentityStrategy          string `json:"identityStrategy"`
	EntitlementAuthorityMode  string `json:"entitlementAuthorityMode"`
	RecipientInputMode        string `json:"recipientInputMode"`
	ReferenceStrategy         string `json:"referenceStrategy"`
	TrackingSyncMode          string `json:"trackingSyncMode"`
	ClosurePolicy             string `json:"closurePolicy"`
	SupportsPartialShipment   bool   `json:"supportsPartialShipment"`
	SupportsAPIImport         bool   `json:"supportsApiImport"`
	SupportsAPIExport         bool   `json:"supportsApiExport"`
	RequiresCarrierMapping    bool   `json:"requiresCarrierMapping"`
	RequiresExternalOrderNo   bool   `json:"requiresExternalOrderNo"`
	AllowsManualClosure       bool   `json:"allowsManualClosure"`
	ConnectorKey              string `json:"connectorKey"`
	SupportedLocales          string `json:"supportedLocales"`
	DefaultLocale             string `json:"defaultLocale"`
	ExtraData                 string `json:"extraData"`
	CreatedAt                 string `json:"createdAt"`
	UpdatedAt                 string `json:"updatedAt"`
}

type CreateProfileInput struct {
	ProfileKey                string `json:"profileKey"`
	SourceChannel             string `json:"sourceChannel"`
	SourceSurface             string `json:"sourceSurface"`
	DemandKind                string `json:"demandKind"`
	InitialAllocationStrategy string `json:"initialAllocationStrategy"`
	IdentityStrategy          string `json:"identityStrategy"`
	EntitlementAuthorityMode  string `json:"entitlementAuthorityMode"`
	RecipientInputMode        string `json:"recipientInputMode"`
	ReferenceStrategy         string `json:"referenceStrategy"`
	TrackingSyncMode          string `json:"trackingSyncMode"`
	ClosurePolicy             string `json:"closurePolicy"`
	SupportsPartialShipment   bool   `json:"supportsPartialShipment"`
	SupportsAPIImport         bool   `json:"supportsApiImport"`
	SupportsAPIExport         bool   `json:"supportsApiExport"`
	RequiresCarrierMapping    bool   `json:"requiresCarrierMapping"`
	RequiresExternalOrderNo   bool   `json:"requiresExternalOrderNo"`
	AllowsManualClosure       bool   `json:"allowsManualClosure"`
	ConnectorKey              string `json:"connectorKey"`
	SupportedLocales          string `json:"supportedLocales"`
	DefaultLocale             string `json:"defaultLocale"`
	ExtraData                 string `json:"extraData"`
}

type UpdateProfileInput struct {
	ID                        uint   `json:"id"`
	ProfileKey                string `json:"profileKey"`
	SourceChannel             string `json:"sourceChannel"`
	SourceSurface             string `json:"sourceSurface"`
	DemandKind                string `json:"demandKind"`
	InitialAllocationStrategy string `json:"initialAllocationStrategy"`
	IdentityStrategy          string `json:"identityStrategy"`
	EntitlementAuthorityMode  string `json:"entitlementAuthorityMode"`
	RecipientInputMode        string `json:"recipientInputMode"`
	ReferenceStrategy         string `json:"referenceStrategy"`
	TrackingSyncMode          string `json:"trackingSyncMode"`
	ClosurePolicy             string `json:"closurePolicy"`
	SupportsPartialShipment   bool   `json:"supportsPartialShipment"`
	SupportsAPIImport         bool   `json:"supportsApiImport"`
	SupportsAPIExport         bool   `json:"supportsApiExport"`
	RequiresCarrierMapping    bool   `json:"requiresCarrierMapping"`
	RequiresExternalOrderNo   bool   `json:"requiresExternalOrderNo"`
	AllowsManualClosure       bool   `json:"allowsManualClosure"`
	ConnectorKey              string `json:"connectorKey"`
	SupportedLocales          string `json:"supportedLocales"`
	DefaultLocale             string `json:"defaultLocale"`
	ExtraData                 string `json:"extraData"`
}
