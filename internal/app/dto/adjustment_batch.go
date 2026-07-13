package dto

// BatchRecordAdjustmentsInput carries a batch of adjustment entries to record in one call.
// Each entry is processed independently (partial-success semantics) — a failure in one
// entry does not prevent the remaining entries from being recorded.
type BatchRecordAdjustmentsInput struct {
	Entries []RecordAdjustmentInput `json:"entries"`
}

// BatchAdjustmentItemResult reports the outcome of a single entry within a batch call.
// Index mirrors the entry's position in the input slice so callers can correlate
// results back to the request without relying on ordering guarantees alone.
type BatchAdjustmentItemResult struct {
	Index      int                       `json:"index"`
	Success    bool                      `json:"success"`
	Adjustment *FulfillmentAdjustmentDTO `json:"adjustment"`
	Error      string                    `json:"error"`
}

// BatchRecordAdjustmentsResult aggregates per-entry results plus summary counts.
type BatchRecordAdjustmentsResult struct {
	Results      []BatchAdjustmentItemResult `json:"results"`
	SuccessCount int                          `json:"successCount"`
	FailureCount int                          `json:"failureCount"`
}
