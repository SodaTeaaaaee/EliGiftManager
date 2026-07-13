package dto

type MergeProfilesInput struct {
	SourceProfileID uint `json:"sourceProfileId"`
	TargetProfileID uint `json:"targetProfileId"`
}

type MergeProfilesResult struct {
	MigratedIdentityCount int `json:"migratedIdentityCount"`
	MigratedAddressCount  int `json:"migratedAddressCount"`
	UpdatedDemandDocs     int `json:"updatedDemandDocs"`
	// Deprecated: historical wave participant snapshots are never rewritten.
	UpdatedParticipants int `json:"updatedParticipants"`
	// Deprecated: historical fulfillment lines are never rewritten.
	UpdatedFulfillmentLines int  `json:"updatedFulfillmentLines"`
	MergeID                 uint `json:"mergeId"`
	UndoAvailable           bool `json:"undoAvailable"`
}
