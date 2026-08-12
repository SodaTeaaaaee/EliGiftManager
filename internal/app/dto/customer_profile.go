package dto

import "time"

type CustomerIdentityDTO struct {
	ID                uint      `json:"id"`
	CustomerProfileID uint      `json:"customerProfileId"`
	IdentityPlatform  string    `json:"identityPlatform"`
	IdentityValue     string    `json:"identityValue"`
	IdentityType      string    `json:"identityType"` // platform_uid, email, username, external_buyer_id
	IsPrimary         bool      `json:"isPrimary"`
	ExtraData         string    `json:"extraData"`
	CreatedAt         time.Time `json:"createdAt" ts_type:"string"`
	UpdatedAt         time.Time `json:"updatedAt" ts_type:"string"`
}

type CustomerProfileDTO struct {
	ID                       uint                  `json:"id"`
	DisplayName              string                `json:"displayName"`
	ProfileType              string                `json:"profileType"` // member, buyer, mixed, manual
	Status                   string                `json:"status"`
	MergedIntoProfileID      *uint                 `json:"mergedIntoProfileId"`
	RowVersion               uint64                `json:"rowVersion"`
	DisplayNameMode          string                `json:"displayNameMode"`
	DisplayNameObservationID *uint                 `json:"displayNameObservationId"`
	MatchedHistoricalName    string                `json:"matchedHistoricalName,omitempty"`
	ExtraData                string                `json:"extraData"`
	CreatedAt                time.Time             `json:"createdAt" ts_type:"string"`
	UpdatedAt                time.Time             `json:"updatedAt" ts_type:"string"`
	Identities               []CustomerIdentityDTO `json:"identities"`
	Addresses                []CustomerAddressDTO  `json:"addresses"`
	ActiveAddressCount       int                   `json:"activeAddressCount"`
}

type CreateCustomerProfileInput struct {
	DisplayName string `json:"displayName"`
	ProfileType string `json:"profileType"` // member, buyer, mixed, manual
	ExtraData   string `json:"extraData"`
}

type UpdateCustomerProfileInput struct {
	ID                 uint   `json:"id"`
	DisplayName        string `json:"displayName"`
	ProfileType        string `json:"profileType"` // member, buyer, mixed, manual
	ExtraData          string `json:"extraData"`
	ExpectedRowVersion uint64 `json:"expectedRowVersion"`
	ActorRef           string `json:"actorRef"`
	IdempotencyKey     string `json:"idempotencyKey"`
}

type CustomerNameObservationDTO struct {
	ID                  uint       `json:"id"`
	Kind                string     `json:"kind"`
	Value               string     `json:"value"`
	Source              string     `json:"source"`
	FirstSeenAt         *time.Time `json:"firstSeenAt" ts_type:"string"`
	LastSeenAt          *time.Time `json:"lastSeenAt" ts_type:"string"`
	Count               uint       `json:"count"`
	IsDisplayNameSource bool       `json:"isDisplayNameSource"`
	OriginProfileID     uint       `json:"originProfileId"`
}

type CustomerProfileOriginDTO struct {
	ID                         uint       `json:"id"`
	CustomerProfileID          uint       `json:"customerProfileId"`
	OriginKind                 string     `json:"originKind"`
	SourceIntegrationProfileID *uint      `json:"sourceIntegrationProfileId"`
	ExternalRef                string     `json:"externalRef"`
	SourceDocumentID           *uint      `json:"sourceDocumentId"`
	LastSeenAt                 *time.Time `json:"lastSeenAt" ts_type:"string"`
	CreatedAt                  time.Time  `json:"createdAt" ts_type:"string"`
}

type PinCustomerDisplayNameInput struct {
	ProfileID          uint   `json:"profileId"`
	Name               string `json:"name"`
	ExpectedRowVersion uint64 `json:"expectedRowVersion"`
	ActorRef           string `json:"actorRef"`
	IdempotencyKey     string `json:"idempotencyKey"`
}

type UnpinCustomerDisplayNameInput struct {
	ProfileID          uint   `json:"profileId"`
	ExpectedRowVersion uint64 `json:"expectedRowVersion"`
	ActorRef           string `json:"actorRef"`
	IdempotencyKey     string `json:"idempotencyKey"`
}

type CreateCustomerIdentityInput struct {
	CustomerProfileID uint   `json:"customerProfileId"`
	IdentityPlatform  string `json:"identityPlatform"`
	IdentityValue     string `json:"identityValue"`
	IdentityType      string `json:"identityType"`
	IsPrimary         bool   `json:"isPrimary"`
	ExtraData         string `json:"extraData"`
}

type MergeSuggestionDTO struct {
	ID              uint               `json:"id"`
	SourceProfileID uint               `json:"sourceProfileId"`
	TargetProfileID uint               `json:"targetProfileId"`
	Reason          string             `json:"reason"`
	Status          string             `json:"status"` // pending, dismissed, merged
	SourceProfile   CustomerProfileDTO `json:"sourceProfile"`
	TargetProfile   CustomerProfileDTO `json:"targetProfile"`
}

type SystemSettingsDTO struct {
	AutoMergeCrossPlatform bool `json:"autoMergeCrossPlatform"`
	AutoMergeByEmail       bool `json:"autoMergeByEmail"`
	AutoMergeByPhone       bool `json:"autoMergeByPhone"`
}
