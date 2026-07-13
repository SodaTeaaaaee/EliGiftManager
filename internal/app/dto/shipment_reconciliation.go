package dto

// SupplierOrderLineShippedDTO reports the shipped/remaining quantity for a single
// supplier-order line so the shipment-backfill UI can display 已发/剩余 before
// submission. Over-ship BLOCKING is already enforced server-side (see
// ShipmentRepository.SumShippedQuantityBySOL usage in ShipmentUseCase /
// ShipmentImportUseCase); this DTO is a read-only display projection only.
type SupplierOrderLineShippedDTO struct {
	LineID            uint   `json:"lineId"`
	FulfillmentLineID uint   `json:"fulfillmentLineId"`
	BatchNo           string `json:"batchNo"`
	SupplierLineNo    int    `json:"supplierLineNo"`
	SupplierSKU       string `json:"supplierSku"`
	SubmittedQuantity int    `json:"submittedQuantity"`
	AcceptedQuantity  int    `json:"acceptedQuantity"`
	ShippedQuantity   int    `json:"shippedQuantity"`
	RemainingQuantity int    `json:"remainingQuantity"`
}
