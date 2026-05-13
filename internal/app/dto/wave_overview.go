package dto

type WaveOverviewDTO struct {
	Wave               WaveDTO `json:"wave"`
	DemandCount        int     `json:"demandCount"`
	FulfillmentCount   int     `json:"fulfillmentCount"`
	SupplierOrderCount int     `json:"supplierOrderCount"`
}
