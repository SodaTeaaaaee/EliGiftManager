package dto

// WaveFulfillmentFilterInput carries the fulfillment grid's multi-select state-dim
// filters, review/drift filters, keyword search, and pagination (plan 3.3.2 / 5.4).
// Each populated *States/*Requirements/*Statuses slice is OR'd internally and AND'd
// against the other dimensions ("address ready AND supplier not submitted" is two
// non-empty single-value slices). An empty slice for a dimension means "no filter
// on that dimension" (matches everything).
type WaveFulfillmentFilterInput struct {
	WaveID             uint            `json:"waveId"`
	AllocationStates   []string        `json:"allocationStates"`
	AddressStates      []string        `json:"addressStates"`
	SupplierStates     []string        `json:"supplierStates"`
	ChannelSyncStates  []string        `json:"channelSyncStates"`
	ReviewRequirements []string        `json:"reviewRequirements"`
	DriftStatuses      []string        `json:"driftStatuses"`
	Keyword            string          `json:"keyword"`
	Pagination         PaginationInput `json:"pagination"`
}

// WaveFulfillmentRowsPage wraps a filtered/paginated page of fulfillment rows.
type WaveFulfillmentRowsPage struct {
	Items      []WaveFulfillmentRowDTO `json:"items"`
	Pagination PaginationResult        `json:"pagination"`
}

// WavesPage wraps a typed, sorted, paginated page of waves — replaces the untyped
// map[string]any previously returned by controller_wave.go's ListWavesPaginated.
type WavesPage struct {
	Items      []WaveDTO        `json:"items"`
	Pagination PaginationResult `json:"pagination"`
}

// WaveListFilterInput extends pagination with wave-list filters for the assign-to-wave
// picker and the waves list page (plan §4.5 of the round-2 spec).
type WaveListFilterInput struct {
	PaginationInput
	LifecycleStage string `json:"lifecycleStage"`
	NameKeyword    string `json:"nameKeyword"`
	WaveType       string `json:"waveType"`
}
