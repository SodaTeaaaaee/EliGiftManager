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
	ID                 uint                  `json:"id"`
	DisplayName        string                `json:"displayName"`
	ProfileType        string                `json:"profileType"` // member, buyer, mixed, manual
	ExtraData          string                `json:"extraData"`
	CreatedAt          time.Time             `json:"createdAt" ts_type:"string"`
	UpdatedAt          time.Time             `json:"updatedAt" ts_type:"string"`
	Identities         []CustomerIdentityDTO `json:"identities"`
	Addresses          []CustomerAddressDTO  `json:"addresses"`
	ActiveAddressCount int                   `json:"activeAddressCount"`
}

type CreateCustomerProfileInput struct {
	DisplayName string `json:"displayName"`
	ProfileType string `json:"profileType"` // member, buyer, mixed, manual
	ExtraData   string `json:"extraData"`
}

type UpdateCustomerProfileInput struct {
	ID          uint   `json:"id"`
	DisplayName string `json:"displayName"`
	ProfileType string `json:"profileType"` // member, buyer, mixed, manual
	ExtraData   string `json:"extraData"`
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
