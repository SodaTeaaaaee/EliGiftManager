package dto

// BoundProfileSnapshot holds the execution-relevant fields of an IntegrationProfile
// captured at wave assignment time. Stored as JSON on DemandDocument.BoundProfileSnapshot.
// Only fields that affect wave execution behavior are included — display-only fields are omitted.
type BoundProfileSnapshot struct {
	ProfileID               uint   `json:"profileId"`
	ProfileKey              string `json:"profileKey"`
	TrackingSyncMode        string `json:"trackingSyncMode"`
	ClosurePolicy           string `json:"closurePolicy"`
	AllowsManualClosure     bool   `json:"allowsManualClosure"`
	RequiresCarrierMapping  bool   `json:"requiresCarrierMapping"`
	RequiresExternalOrderNo bool   `json:"requiresExternalOrderNo"`
	SupportsPartialShipment bool   `json:"supportsPartialShipment"`
	ConnectorKey            string `json:"connectorKey"`
	SupportsAPIExport       bool   `json:"supportsAPIExport"`
}
