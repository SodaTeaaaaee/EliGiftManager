package dto

// MergePreviewIdentity mirrors the identity fields shown in a merge preview
// side-by-side comparison.
type MergePreviewIdentity struct {
	ID               uint   `json:"id"`
	IdentityPlatform string `json:"identityPlatform"`
	IdentityValue    string `json:"identityValue"`
	IdentityType     string `json:"identityType"`
	IsPrimary        bool   `json:"isPrimary"`
}

// MergePreviewAddress mirrors the address fields shown in a merge preview
// side-by-side comparison.
type MergePreviewAddress struct {
	ID            uint   `json:"id"`
	Label         string `json:"label"`
	RecipientName string `json:"recipientName"`
	Phone         string `json:"phone"`
	Country       string `json:"country"`
	Province      string `json:"province"`
	City          string `json:"city"`
	District      string `json:"district"`
	AddressLine1  string `json:"addressLine1"`
	AddressLine2  string `json:"addressLine2"`
	PostalCode    string `json:"postalCode"`
	IsDefault     bool   `json:"isDefault"`
}

// MergePreviewProfileSide is one side (source or target) of a merge preview:
// the profile's display fields plus its identities/addresses, so the
// operator can eyeball both sides before committing.
type MergePreviewProfileSide struct {
	ProfileID   uint                   `json:"profileId"`
	DisplayName string                 `json:"displayName"`
	ProfileType string                 `json:"profileType"`
	Identities  []MergePreviewIdentity `json:"identities"`
	Addresses   []MergePreviewAddress  `json:"addresses"`
}

// MergePreviewConflict flags a profile-level field where source and target
// disagree, so the merge dialog can highlight it rather than silently
// discarding the source's value.
type MergePreviewConflict struct {
	Field       string `json:"field"`
	SourceValue string `json:"sourceValue"`
	TargetValue string `json:"targetValue"`
}

// MergeProfilesPreviewResult is the read-only conflict-detail response for
// PreviewMergeProfiles (plan 5.2): both sides' identities/addresses, a
// highlighted conflict list, and duplicate-identity warnings — shown before
// MergeProfiles is actually invoked.
type MergeProfilesPreviewResult struct {
	Source                  MergePreviewProfileSide `json:"source"`
	Target                  MergePreviewProfileSide `json:"target"`
	Conflicts               []MergePreviewConflict  `json:"conflicts"`
	MovedIdentityCount      int                     `json:"movedIdentityCount"`
	MovedAddressCount       int                     `json:"movedAddressCount"`
	DuplicateIdentityValues []string                `json:"duplicateIdentityValues"`
}
