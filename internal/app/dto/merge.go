package dto

type MergeProfilesInput struct {
	SourceProfileID uint `json:"sourceProfileId"`
	TargetProfileID uint `json:"targetProfileId"`
}

type MergeProfilesResult struct {
	MigratedIdentityCount   int `json:"migratedIdentityCount"`
	MigratedAddressCount    int `json:"migratedAddressCount"`
	UpdatedDemandDocs       int `json:"updatedDemandDocs"`
	UpdatedParticipants     int `json:"updatedParticipants"`
	UpdatedFulfillmentLines int `json:"updatedFulfillmentLines"`
}
