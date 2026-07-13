package dto

// BindAddressEntry is one line->address binding request within a batch.
type BindAddressEntry struct {
	FulfillmentLineID uint `json:"fulfillmentLineId"`
	CustomerAddressID uint `json:"customerAddressId"`
}

// AddressBatchItemResult is the per-entry outcome of a batch address-binding operation.
type AddressBatchItemResult struct {
	FulfillmentLineID uint   `json:"fulfillmentLineId"`
	CustomerAddressID *uint  `json:"customerAddressId,omitempty"`
	Success           bool   `json:"success"`
	ErrorMessage      string `json:"errorMessage,omitempty"`
}
