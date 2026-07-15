package dto

// BoundProfileSnapshot holds the execution-relevant fields of an IntegrationProfile
// captured at wave assignment time. Stored as JSON on DemandDocument.BoundProfileSnapshot.
// Only fields that affect wave execution behavior are included — display-only fields are omitted.
type BoundProfileSnapshot struct {
	ProfileID               uint   `json:"profileId"`
	ProfileKey              string `json:"profileKey"`
	SourceSurface           string `json:"sourceSurface"`
	TrackingSyncMode        string `json:"trackingSyncMode"`
	ClosurePolicy           string `json:"closurePolicy"`
	AllowsManualClosure     bool   `json:"allowsManualClosure"`
	RequiresCarrierMapping  bool   `json:"requiresCarrierMapping"`
	RequiresExternalOrderNo bool   `json:"requiresExternalOrderNo"`
	SupportsPartialShipment bool   `json:"supportsPartialShipment"`
	ConnectorKey            string `json:"connectorKey"`
	// FactorySupplierPlatform is the factory-facing platform label written onto
	// SupplierOrder.SupplierPlatform. When empty, export falls back to ConnectorKey.
	FactorySupplierPlatform            string `json:"factorySupplierPlatform"`
	SupportsAPIExport                  bool   `json:"supportsAPIExport"`
	SupportsExportSupplierOrder        bool   `json:"supportsExportSupplierOrder"`
	SupportsImportProductCatalog       bool   `json:"supportsImportProductCatalog"`
	SupportsImportSupplierShipment     bool   `json:"supportsImportSupplierShipment"`
}
