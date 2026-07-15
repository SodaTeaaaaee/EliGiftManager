package dto

import "time"

// MarkSupplierOrderSubmittedInput is the input for transitioning a supplier
// order from draft to submitted, recording the factory-assigned external
// order number and submission timestamp.
type MarkSupplierOrderSubmittedInput struct {
	OrderID         uint       `json:"orderId"`
	ExternalOrderNo string     `json:"externalOrderNo"`
	SubmittedAt     *time.Time `json:"submittedAt" ts_type:"string"`
}

// SupplierOrderLineAcceptanceEntry is a single line's factory-accepted
// quantity, part of RecordSupplierOrderAcceptanceInput.
type SupplierOrderLineAcceptanceEntry struct {
	LineID           uint `json:"lineId"`
	AcceptedQuantity int  `json:"acceptedQuantity"`
}

// RecordSupplierOrderAcceptanceInput is the input for transitioning a
// supplier order from submitted to accepted, recording the factory-accepted
// quantity for each line.
type RecordSupplierOrderAcceptanceInput struct {
	OrderID uint                               `json:"orderId"`
	Lines   []SupplierOrderLineAcceptanceEntry `json:"lines"`
}

// SupplierOrderFileResultDTO is returned by GenerateSupplierOrderFile,
// pointing the caller at the freshly written factory file.
type SupplierOrderFileResultDTO struct {
	OrderID     uint      `json:"orderId"`
	FilePath    string    `json:"filePath"`
	LineCount   int       `json:"lineCount"`
	GeneratedAt time.Time `json:"generatedAt" ts_type:"string"`
	// Warnings carries non-fatal notices (e.g. legacy JSON fallback when no
	// export_supplier_order template is bound). Empty when the preferred path ran.
	Warnings []string `json:"warnings,omitempty"`
}
