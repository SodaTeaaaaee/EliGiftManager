package dto

type CustomerAddressDTO struct {
	ID                uint   `json:"id"`
	CustomerProfileID uint   `json:"customerProfileId"`
	Label             string `json:"label"`
	RecipientName     string `json:"recipientName"`
	Phone             string `json:"phone"`
	Country           string `json:"country"`
	Province          string `json:"province"`
	City              string `json:"city"`
	District          string `json:"district"`
	AddressLine1      string `json:"addressLine1"`
	AddressLine2      string `json:"addressLine2"`
	PostalCode        string `json:"postalCode"`
	IsDefault         bool   `json:"isDefault"`
	IsTest            bool   `json:"isTest"`
	ValidationStatus  string `json:"validationStatus"`
	ValidationDetail  string `json:"validationDetail"`
	ExtraData         string `json:"extraData"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type CreateAddressInput struct {
	CustomerProfileID uint   `json:"customerProfileId"`
	Label             string `json:"label"`
	RecipientName     string `json:"recipientName"`
	Phone             string `json:"phone"`
	Country           string `json:"country"`
	Province          string `json:"province"`
	City              string `json:"city"`
	District          string `json:"district"`
	AddressLine1      string `json:"addressLine1"`
	AddressLine2      string `json:"addressLine2"`
	PostalCode        string `json:"postalCode"`
	IsDefault         bool   `json:"isDefault"`
	IsTest            bool   `json:"isTest"`
	ValidationStatus  string `json:"validationStatus"`
	ValidationDetail  string `json:"validationDetail"`
	ExtraData         string `json:"extraData"`
}

type UpdateAddressInput struct {
	ID                uint   `json:"id"`
	CustomerProfileID uint   `json:"customerProfileId"`
	Label             string `json:"label"`
	RecipientName     string `json:"recipientName"`
	Phone             string `json:"phone"`
	Country           string `json:"country"`
	Province          string `json:"province"`
	City              string `json:"city"`
	District          string `json:"district"`
	AddressLine1      string `json:"addressLine1"`
	AddressLine2      string `json:"addressLine2"`
	PostalCode        string `json:"postalCode"`
	IsDefault         bool   `json:"isDefault"`
	IsTest            bool   `json:"isTest"`
	ValidationStatus  string `json:"validationStatus"`
	ValidationDetail  string `json:"validationDetail"`
	ExtraData         string `json:"extraData"`
}

type BindAddressInput struct {
	FulfillmentLineID uint `json:"fulfillmentLineId"`
	CustomerAddressID uint `json:"customerAddressId"`
}
